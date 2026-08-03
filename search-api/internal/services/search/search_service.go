package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	// Search devuelve la página y el total de matches (numFound) para el
	// meta del envelope (A5/RV22)
	Search(ctx context.Context, query string, limit int, offset int) ([]hotelsDAO.Hotel, int, error)
}

// Funcion de la API de hoteles
type ExternalRepository interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error)
	GetHotels(ctx context.Context, limit, offset int) ([]hotelsDomain.Hotel, int, error)
}

type Service struct {
	repository Repository         // Este seria nuestro repositorio de solr
	hotelsAPI  ExternalRepository // Este seria nuestro repositorio de la API de hoteles
}

// Funcion para crear un nuevo servicio
func NewService(repository Repository, hotelsAPI ExternalRepository) Service {
	return Service{
		repository: repository,
		hotelsAPI:  hotelsAPI,
	}
}

// Funcion para buscar hoteles en Solr
func (service Service) Search(ctx context.Context, query string, offset int, limit int) ([]hotelsDomain.Hotel, int, error) {
	// Llama al metodo Search del repositorio
	hotelsDAOList, total, err := service.repository.Search(ctx, query, limit, offset)
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

// HandleHotelNew procesa un evento de hotel. Recibe el context del consumer
// con un request_id por mensaje (O1) y devuelve error si el evento debe
// reintentarse (E1: el consumer nackea — 1er fallo requeue, 2º DLQ).
// Un hotel que ya no existe en hotels-api NO es error (RV14): se descarta el
// evento y se limpia el doc del índice — reintentarlo jamás lo resolvería.
func (service Service) HandleHotelNew(ctx context.Context, hotelNew hotelsDomain.HotelNew) error {
	logger := slog.With(
		"request_id", utils.RequestIDFromContext(ctx),
		"operation", hotelNew.Operation,
		"hotel_id", hotelNew.HotelID,
	)
	logger.Info("hotel event received")
	// Hacemos un switch para manejar las operaciones de creacion, actualizacion y eliminacion
	switch hotelNew.Operation {
	// Caso en el que se crea o actualiza un hotel
	case "CREATE", "UPDATE":
		// Obtenemos el hotel de la API de hoteles
		hotel, err := service.hotelsAPI.GetHotelByID(ctx, hotelNew.HotelID)
		if err != nil {
			if errors.Is(err, hotelsDomain.ErrHotelNotFound) {
				logger.Warn("hotel no longer exists in hotels-api, removing stale doc from index")
				if err := service.repository.Delete(ctx, hotelNew.HotelID); err != nil {
					logger.Error("error deleting stale hotel from solr", "error", err)
					return fmt.Errorf("error deleting stale hotel from solr: %w", err)
				}
				return nil
			}
			logger.Error("error fetching hotel from hotels-api", "error", err)
			return fmt.Errorf("error fetching hotel from hotels-api: %w", err)
		}
		logger.Info("hotel fetched from hotels-api", "hotel_name", hotel.Name)

		hotelDAO := toHotelDAO(hotel)

		// Caso en el que se crea un hotel
		if hotelNew.Operation == "CREATE" {
			// Llama al metodo Index del repositorio para indexar el hotel en Solr
			if _, err := service.repository.Index(ctx, hotelDAO); err != nil {
				logger.Error("error indexing hotel in solr", "error", err)
				return fmt.Errorf("error indexing hotel in solr: %w", err)
			}
			logger.Info("hotel indexed in solr")
		} else { // Caso en el que se actualiza un hotel
			// Llama al metodo Update del repositorio para actualizar el hotel en Solr
			if err := service.repository.Update(ctx, hotelDAO); err != nil {
				logger.Error("error updating hotel in solr", "error", err)
				return fmt.Errorf("error updating hotel in solr: %w", err)
			}
			logger.Info("hotel updated in solr")
		}
		return nil
	// Caso en el que se elimina un hotel
	case "DELETE":
		// Llama al metodo Delete del repositorio para eliminar el hotel de Solr
		if err := service.repository.Delete(ctx, hotelNew.HotelID); err != nil {
			logger.Error("error deleting hotel from solr", "error", err)
			return fmt.Errorf("error deleting hotel from solr: %w", err)
		}
		logger.Info("hotel deleted from solr")
		return nil
	default:
		// Operación desconocida = mensaje no procesable: el error lo manda a
		// la DLQ (tras el retry de rigor) en vez de perderlo en silencio
		logger.Warn("unknown operation")
		return fmt.Errorf("unknown hotel event operation %q", hotelNew.Operation)
	}
}

// Backfill reindexa el catálogo completo paginando GET /hotels de hotels-api
// (E3): corre al arranque (el índice de Solr es derivado y se reconstruye
// desde la fuente de verdad) y on-demand vía POST /reindex. Es idempotente:
// `id` es la uniqueKey del core, re-indexar pisa el documento.
func (service Service) Backfill(ctx context.Context) (int, error) {
	indexed := 0
	for offset := 0; ; {
		page, total, err := service.hotelsAPI.GetHotels(ctx, backfillPageSize, offset)
		if err != nil {
			return indexed, fmt.Errorf("error fetching hotels page (offset %d): %w", offset, err)
		}
		for _, hotel := range page {
			if _, err := service.repository.Index(ctx, toHotelDAO(hotel)); err != nil {
				return indexed, fmt.Errorf("error indexing hotel %s: %w", hotel.ID, err)
			}
			indexed++
		}
		offset += len(page)
		if len(page) == 0 || offset >= total {
			return indexed, nil
		}
	}
}
