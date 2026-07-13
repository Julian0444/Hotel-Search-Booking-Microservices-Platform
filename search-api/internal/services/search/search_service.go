package search

import (
	"context"
	"fmt"
	"log/slog"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
)

// Funciones de solr
type Repository interface {
	Index(ctx context.Context, hotel hotelsDAO.Hotel) (string, error)
	Update(ctx context.Context, hotel hotelsDAO.Hotel) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, limit int, offset int) ([]hotelsDAO.Hotel, error) // Updated signature
}

// Funcion de la API de hoteles
type ExternalRepository interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error)
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
func (service Service) Search(ctx context.Context, query string, offset int, limit int) ([]hotelsDomain.Hotel, error) {
	// Llama al metodo Search del repositorio
	hotelsDAOList, err := service.repository.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error searching hotels: %w", err)
	}

	// Hace un mapeo de los hoteles de la lista de hoteles de Solr a la lista de hoteles de dominio
	hotelsDomainList := make([]hotelsDomain.Hotel, 0)
	for _, hotel := range hotelsDAOList {
		hotelsDomainList = append(hotelsDomainList, hotelsDomain.Hotel{
			ID:            hotel.ID,
			Name:          hotel.Name,
			Description:   hotel.Description,
			Address:       hotel.Address,
			City:          hotel.City,
			State:         hotel.State,
			Country:       hotel.Country,
			Phone:         hotel.Phone,
			Email:         hotel.Email,
			Rating:        hotel.Rating,
			PricePerNight: hotel.PricePerNight,
			AvaiableRooms: hotel.AvaiableRooms,
			CheckInTime:   hotel.CheckInTime,
			CheckOutTime:  hotel.CheckOutTime,
			Amenities:     hotel.Amenities,
			Images:        hotel.Images,
		})
	}

	// Devuelve la lista de hoteles
	return hotelsDomainList, nil
}

// Funcion para manejar la creacion y eliminacion de hoteles. Recibe el
// context del consumer con un request_id por mensaje (O1): se loguea acá y
// viaja como header en el fetch a hotels-api para correlacionar el hop.
func (service Service) HandleHotelNew(ctx context.Context, hotelNew hotelsDomain.HotelNew) {
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
			logger.Error("error fetching hotel from hotels-api", "error", err)
			return
		}
		logger.Info("hotel fetched from hotels-api", "hotel_name", hotel.Name)

		hotelDAO := hotelsDAO.Hotel{
			ID:            hotel.ID,
			Name:          hotel.Name,
			Description:   hotel.Description,
			Address:       hotel.Address,
			City:          hotel.City,
			State:         hotel.State,
			Country:       hotel.Country,
			Phone:         hotel.Phone,
			Email:         hotel.Email,
			Rating:        hotel.Rating,
			PricePerNight: hotel.PricePerNight,
			AvaiableRooms: hotel.AvaiableRooms,
			CheckInTime:   hotel.CheckInTime,
			CheckOutTime:  hotel.CheckOutTime,
			Amenities:     hotel.Amenities,
			Images:        hotel.Images,
		}

		// Caso en el que se crea un hotel
		if hotelNew.Operation == "CREATE" {
			// Llama al metodo Index del repositorio para indexar el hotel en Solr
			if _, err := service.repository.Index(ctx, hotelDAO); err != nil {
				logger.Error("error indexing hotel in solr", "error", err)
			} else {
				logger.Info("hotel indexed in solr")
			}
		} else { // Caso en el que se actualiza un hotel
			// Llama al metodo Update del repositorio para actualizar el hotel en Solr
			if err := service.repository.Update(ctx, hotelDAO); err != nil {
				logger.Error("error updating hotel in solr", "error", err)
			} else {
				logger.Info("hotel updated in solr")
			}
		}
	// Caso en el que se elimina un hotel
	case "DELETE":
		// Llama al metodo Delete del repositorio para eliminar el hotel de Solr
		if err := service.repository.Delete(ctx, hotelNew.HotelID); err != nil {
			logger.Error("error deleting hotel from solr", "error", err)
		} else {
			logger.Info("hotel deleted from solr")
		}
	default:
		logger.Warn("unknown operation")
	}
}
