package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
)

// backfillPageSize es el tamaño de página con el que el backfill recorre
// GET /hotels (hotels-api clampa limit a 100).
const backfillPageSize = 50

// Funciones de solr
type Repository interface {
	Index(ctx context.Context, hotel hotelsDAO.Hotel) (string, error)
	Update(ctx context.Context, hotel hotelsDAO.Hotel) error
	Delete(ctx context.Context, id string) error
	ListIDs(ctx context.Context) ([]string, error)
	// Search devuelve la página y el total de matches (numFound) para el
	// meta del envelope (A5/RV22). sort ya viene whitelisteado del controller;
	// el repo lo mapea a un sort de Solr (plan 13).
	Search(ctx context.Context, query string, sort string, limit int, offset int) ([]hotelsDAO.Hotel, int, error)
}

// Funcion de la API de hoteles
type ExternalRepository interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error)
	GetHotelsAfter(ctx context.Context, limit int, after string) ([]hotelsDomain.Hotel, error)
}

type Service struct {
	// Canal compartido entre copias del Service: serializa eventos y reindex,
	// con espera cancelable. Requiere UNA instancia activa de search-api.
	writer     chan struct{}
	repository Repository         // Este seria nuestro repositorio de solr
	hotelsAPI  ExternalRepository // Este seria nuestro repositorio de la API de hoteles
}

// Funcion para crear un nuevo servicio
func NewService(repository Repository, hotelsAPI ExternalRepository) Service {
	return Service{
		writer:     make(chan struct{}, 1),
		repository: repository,
		hotelsAPI:  hotelsAPI,
	}
}

// Funcion para buscar hoteles en Solr
func (service Service) Search(ctx context.Context, query string, sort string, offset int, limit int) ([]hotelsDomain.Hotel, int, error) {
	// Llama al metodo Search del repositorio
	hotelsDAOList, total, err := service.repository.Search(ctx, query, sort, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching hotels: %w", err)
	}

	// Hace un mapeo de los hoteles de la lista de hoteles de Solr a la lista de hoteles de dominio
	hotelsDomainList := make([]hotelsDomain.Hotel, 0)
	for _, hotel := range hotelsDAOList {
		hotelsDomainList = append(hotelsDomainList, hotelsDomain.Hotel{
			ID:             hotel.ID,
			Name:           hotel.Name,
			Description:    hotel.Description,
			Address:        hotel.Address,
			City:           hotel.City,
			State:          hotel.State,
			Country:        hotel.Country,
			Phone:          hotel.Phone,
			Email:          hotel.Email,
			Rating:         hotel.Rating,
			PricePerNight:  hotel.PricePerNight,
			AvailableRooms: hotel.AvailableRooms,
			CheckInTime:    hotel.CheckInTime,
			CheckOutTime:   hotel.CheckOutTime,
			Amenities:      hotel.Amenities,
			Images:         hotel.Images,
		})
	}

	// Devuelve la lista de hoteles y el total de matches
	return hotelsDomainList, total, nil
}

// toHotelDAO convierte el hotel de dominio (respuesta de hotels-api) al DAO
// que se indexa en Solr.
func toHotelDAO(hotel hotelsDomain.Hotel) hotelsDAO.Hotel {
	return hotelsDAO.Hotel{
		ID:             hotel.ID,
		Name:           hotel.Name,
		Description:    hotel.Description,
		Address:        hotel.Address,
		City:           hotel.City,
		State:          hotel.State,
		Country:        hotel.Country,
		Phone:          hotel.Phone,
		Email:          hotel.Email,
		Rating:         hotel.Rating,
		PricePerNight:  hotel.PricePerNight,
		AvailableRooms: hotel.AvailableRooms,
		CheckInTime:    hotel.CheckInTime,
		CheckOutTime:   hotel.CheckOutTime,
		Amenities:      hotel.Amenities,
		Images:         hotel.Images,
	}
}

// lockWriter impide que un backfill antiguo pise un evento ya aplicado.
func (service Service) lockWriter(ctx context.Context) error {
	select {
	case service.writer <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HandleHotelNew usa el evento como invalidación. Incluso DELETE consulta la
// fuente actual: duplicados o eventos fuera de orden no pisarán estado reciente.
func (service Service) HandleHotelNew(ctx context.Context, event hotelsDomain.HotelNew) error {
	if event.HotelID == "" || (event.Operation != "CREATE" && event.Operation != "UPDATE" && event.Operation != "DELETE") {
		return fmt.Errorf("invalid hotel event: %w", hotelsDomain.ErrInvalidEvent)
	}
	if err := service.lockWriter(ctx); err != nil {
		return err
	}
	defer func() { <-service.writer }()
	_, err := service.syncHotel(ctx, event.HotelID)
	return err
}

// syncHotel nunca indexa snapshots de una página antigua, solamente el GET
// fresco del documento. hotels-api lee Mongo directamente, sin caché.
func (service Service) syncHotel(ctx context.Context, id string) (bool, error) {
	hotel, err := service.hotelsAPI.GetHotelByID(ctx, id)
	if errors.Is(err, hotelsDomain.ErrHotelNotFound) {
		if err := service.repository.Delete(ctx, id); err != nil {
			return false, fmt.Errorf("deleting stale hotel %s: %w", id, err)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("fetching hotel %s: %w", id, err)
	}
	if _, err := service.repository.Index(ctx, toHotelDAO(hotel)); err != nil {
		return false, fmt.Errorf("indexing hotel %s: %w", id, err)
	}
	return true, nil
}

// Backfill reconcilia la unión de IDs del catálogo y del índice, sin vaciar el
// core. Keyset evita saltos al eliminar durante la paginación. Los eventos que
// llegan durante el recorrido esperan el lock y después consultan estado fresco.
// No es un snapshot transaccional entre Mongo y Solr: una alta con un ID previo
// al cursor o una publicación perdida converge en la siguiente pasada periódica.
func (service Service) Backfill(ctx context.Context) (int, error) {
	if err := service.lockWriter(ctx); err != nil {
		return 0, err
	}
	defer func() { <-service.writer }()
	ids, err := service.repository.ListIDs(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing indexed hotels: %w", err)
	}
	candidates := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		candidates[id] = struct{}{}
	}
	after := ""
	for {
		page, err := service.hotelsAPI.GetHotelsAfter(ctx, backfillPageSize, after)
		if err != nil {
			return 0, fmt.Errorf("fetching hotels after %q: %w", after, err)
		}
		if len(page) == 0 {
			break
		}
		for _, hotel := range page {
			if hotel.ID <= after {
				return 0, fmt.Errorf("hotels-api returned non-increasing cursor %q after %q", hotel.ID, after)
			}
			candidates[hotel.ID] = struct{}{}
			after = hotel.ID
		}
	}
	ids = ids[:0]
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	indexed := 0
	for _, id := range ids {
		exists, err := service.syncHotel(ctx, id)
		if err != nil {
			return indexed, err
		}
		if exists {
			indexed++
		}
	}
	slog.Info("catalogue reconciliation complete", "request_id", utils.RequestIDFromContext(ctx), "indexed", indexed, "checked", len(ids))
	return indexed, nil
}
