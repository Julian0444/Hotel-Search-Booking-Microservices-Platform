package services

import (
	"context"
	"fmt"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

// Estas funciones salen de los repositorios, se encargan de interactuar tanto de la base de datos como de la cache, ambas tienen las mismas funciones pero con diferentes implementaciones para cada cosa
type Repository interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error)
	Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error)
	Update(ctx context.Context, hotel hotelsDAO.Hotel) error
	Delete(ctx context.Context, id string) error
	CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error)
	GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error)
	CancelReservation(ctx context.Context, id string) error
	GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDAO.Reservation, error)
	GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string) ([]hotelsDAO.Reservation, error)
	GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDAO.Reservation, error)
	DeleteReservationsByHotelID(ctx context.Context, hotelID string) error
	GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error)
}

type Queue interface {
	Publish(hotelNew hotelsDomain.HotelNew) error
}

type Service struct {
	mainRepository  Repository
	cacheRepository Repository
	eventsQueue     Queue
}

// Funcion que se encarga de crear un nuevo servicio con los repositorios y la cola de eventos
func NewService(mainRepository Repository, cacheRepository Repository, eventsQueue Queue) Service {
	return Service{
		mainRepository:  mainRepository,
		cacheRepository: cacheRepository,
		eventsQueue:     eventsQueue,
	}
}

// Funcion que se encarga de obtener un hotel por su ID, primero se intenta obtener de la cache, si no se encuentra se obtiene de la base de datos principal y se guarda en la cache
func (service Service) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	// Se intenta obtener el hotel de la cache
	hotelDAO, err := service.cacheRepository.GetHotelByID(ctx, id)
	if err != nil {
		// Si no se encuentra en la cache, se obtiene de la base de datos principal
		hotelDAO, err = service.mainRepository.GetHotelByID(ctx, id)
		if err != nil {
			return hotelsDomain.Hotel{}, fmt.Errorf("error getting hotel from repository: %v", err)
		}
		// Se guarda el hotel en la cache
		if _, err := service.cacheRepository.Create(ctx, hotelDAO); err != nil {
			return hotelsDomain.Hotel{}, fmt.Errorf("error creating hotel in cache: %w", err)
		}
	}

	// Lo pasa de formato de base de datos a formato de dominio para las respuestas
	//Lo devuelve en formato de dominio
	return hotelsDomain.Hotel{
		ID:            hotelDAO.ID,
		Name:          hotelDAO.Name,
		Description:   hotelDAO.Description,
		Address:       hotelDAO.Address,
		City:          hotelDAO.City,
		State:         hotelDAO.State,
		Country:       hotelDAO.Country,
		Phone:         hotelDAO.Phone,
		Email:         hotelDAO.Email,
		PricePerNight: hotelDAO.PricePerNight,
		Rating:        hotelDAO.Rating,
		AvaiableRooms: hotelDAO.AvaiableRooms,
		CheckInTime:   hotelDAO.CheckInTime,
		CheckOutTime:  hotelDAO.CheckOutTime,
		Amenities:     hotelDAO.Amenities,
		Images:        hotelDAO.Images,
	}, nil
}

// Funcion que se encarga de crear un nuevo hotel, primero se crea en la base de datos principal, luego en la cache y por ultimo se publica un evento para notificar que se creo un nuevo hotel
func (service Service) Create(ctx context.Context, hotel hotelsDomain.Hotel) (string, error) {
	// Convierte el modelo de dominio a modelo DAO
	//Modelo de como viene -> modelo base de datos
	record := hotelsDAO.Hotel{
		Name:          hotel.Name,
		Description:   hotel.Description,
		Address:       hotel.Address,
		City:          hotel.City,
		State:         hotel.State,
		Country:       hotel.Country,
		Phone:         hotel.Phone,
		Email:         hotel.Email,
		PricePerNight: hotel.PricePerNight,
		Rating:        hotel.Rating,
		AvaiableRooms: hotel.AvaiableRooms,
		CheckInTime:   hotel.CheckInTime,
		CheckOutTime:  hotel.CheckOutTime,
		Amenities:     hotel.Amenities,
		Images:        hotel.Images,
	}
	// Crea el hotel en el repositorio principal (base de datos -> MongoDB)
	id, err := service.mainRepository.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("error creating hotel in main repository: %w", err)
	}
	// Crea el hotel en el repositorio de cache
	//El id que usan es el ObjectId de MongoDB
	record.ID = id
	if _, err := service.cacheRepository.Create(ctx, record); err != nil {
		return "", fmt.Errorf("error creating hotel in cache: %w", err)
	}
	// Publica un evento para notificar la creación del hotel (RabbitMQ)
	if err := service.eventsQueue.Publish(hotelsDomain.HotelNew{
		Operation: "CREATE",
		HotelID:   id,
	}); err != nil {
		return "", fmt.Errorf("error publishing hotel new: %w", err)
	}

	return id, nil
}

// Funcion que se encarga de actualizar un hotel, primero se actualiza en la base de datos principal, luego en la cache y por ultimo se publica un evento para notificar que se actualizo un hotel
func (service Service) Update(ctx context.Context, hotel hotelsDomain.Hotel) error {
	// Convierte el modelo de dominio a modelo DAO
	record := hotelsDAO.Hotel{
		ID:            hotel.ID,
		Name:          hotel.Name,
		Description:   hotel.Description,
		Address:       hotel.Address,
		City:          hotel.City,
		State:         hotel.State,
		Country:       hotel.Country,
		Phone:         hotel.Phone,
		Email:         hotel.Email,
		PricePerNight: hotel.PricePerNight,
		Rating:        hotel.Rating,
		AvaiableRooms: hotel.AvaiableRooms,
		CheckInTime:   hotel.CheckInTime,
		CheckOutTime:  hotel.CheckOutTime,
		Amenities:     hotel.Amenities,
		Images:        hotel.Images,
	}

	// Actualiza el hotel en el repositorio principal (MongoDB)
	err := service.mainRepository.Update(ctx, record)
	if err != nil {
		return fmt.Errorf("error updating hotel in main repository: %w", err)
	}

	//INTENTA actualizar el hotel en el repositorio de cache
	if err := service.cacheRepository.Update(ctx, record); err != nil {
		return fmt.Errorf("error updating hotel in cache: %w", err)
	}

	// Publica un evento para notificar la actualización del hotel (RabbitMQ)
	if err := service.eventsQueue.Publish(hotelsDomain.HotelNew{
		Operation: "UPDATE",
		HotelID:   hotel.ID,
	}); err != nil {
		return fmt.Errorf("error publishing hotel update: %w", err)
	}

	return nil
}

// Funcion que se encarga de eliminar un hotel, primero elimina todas las reservas asociadas, luego el hotel de la base de datos principal, luego de la cache y por ultimo se publica un evento para notificar que se elimino un hotel
func (service Service) Delete(ctx context.Context, id string) error {
	// Primero eliminar todas las reservas asociadas al hotel del repositorio principal (MongoDB)
	if err := service.mainRepository.DeleteReservationsByHotelID(ctx, id); err != nil {
		return fmt.Errorf("error deleting reservations for hotel %s from main repository: %w", id, err)
	}

	// Eliminar todas las reservas asociadas al hotel del repositorio de cache
	if err := service.cacheRepository.DeleteReservationsByHotelID(ctx, id); err != nil {
		return fmt.Errorf("error deleting reservations for hotel %s from cache: %w", id, err)
	}

	// Intenta eliminar el hotel del repositorio principal (MongoDB)
	err := service.mainRepository.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting hotel from main repository: %w", err)
	}

	// Intenta eliminar el hotel del repositorio de cache
	if err := service.cacheRepository.Delete(ctx, id); err != nil {
		return fmt.Errorf("error deleting hotel from cache: %w", err)
	}

	// Publica un evento para notificar la eliminación del hotel (RabbitMQ)
	if err := service.eventsQueue.Publish(hotelsDomain.HotelNew{
		Operation: "DELETE",
		HotelID:   id,
	}); err != nil {
		return fmt.Errorf("error publishing hotel delete: %w", err)
	}

	return nil
}

func (service Service) CreateReservation(ctx context.Context, reservation hotelsDomain.Reservation) (string, error) {
	record := hotelsDAO.Reservation{
		HotelName: reservation.HotelName,
		HotelID:   reservation.HotelID,
		UserID:    reservation.UserID,
		CheckIn:   reservation.CheckIn,
		CheckOut:  reservation.CheckOut,
	}
	// Crea la reserva en el repositorio principal (base de datos -> MongoDB)
	id, err := service.mainRepository.CreateReservation(ctx, record)
	if err != nil {
		return "", fmt.Errorf("error creating reservation in main repository: %w", err)
	}
	// Crea la reserva en el repositorio de cache
	record.ID = id
	if _, err := service.cacheRepository.CreateReservation(ctx, record); err != nil {
		return "", fmt.Errorf("error creating reservation in cache: %w", err)
	}

	return id, nil
}

func (service Service) GetReservationByID(ctx context.Context, id string) (hotelsDomain.Reservation, error) {
	// Se intenta obtener la reserva del repositorio de cache
	reservationDAO, err := service.cacheRepository.GetReservationByID(ctx, id)
	if err != nil {
		// Si no se encuentra en la cache, se obtiene del repositorio principal
		reservationDAO, err = service.mainRepository.GetReservationByID(ctx, id)
		if err != nil {
			return hotelsDomain.Reservation{}, fmt.Errorf("error getting reservation from repository: %w", err)
		}
		// Se guarda la reserva en la cache
		if _, err := service.cacheRepository.CreateReservation(ctx, reservationDAO); err != nil {
			// Log error pero no fallar, ya que tenemos la reserva
			fmt.Printf("Error caching reservation: %v\n", err)
		}
	}

	// Se convierte la reserva de formato de base de datos a formato de dominio
	reservation := hotelsDomain.Reservation{
		ID:        reservationDAO.ID,
		HotelName: reservationDAO.HotelName,
		HotelID:   reservationDAO.HotelID,
		UserID:    reservationDAO.UserID,
		CheckIn:   reservationDAO.CheckIn,
		CheckOut:  reservationDAO.CheckOut,
	}

	return reservation, nil
}

func (service Service) CancelReservation(ctx context.Context, id string) error {
	// Intenta eliminar la reserva del repositorio principal (MongoDB)
	err := service.mainRepository.CancelReservation(ctx, id)
	if err != nil {
		return fmt.Errorf("error canceling reservation from main repository: %w", err)
	}

	// Intenta eliminar la reserva del repositorio de cache
	if err := service.cacheRepository.CancelReservation(ctx, id); err != nil {
		return fmt.Errorf("error canceling reservation from cache: %w", err)
	}

	return nil
}

func (service Service) GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDomain.Reservation, error) {
	// Se intenta obtener las reservas del repositorio de cache
	reservationsDAO, err := service.cacheRepository.GetReservationsByHotelID(ctx, hotelID)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		reservationsDAO, err = service.mainRepository.GetReservationsByHotelID(ctx, hotelID)
		if err != nil {
			return nil, fmt.Errorf("error getting reservations from repository: %v", err)
		}
		// Se guardan las reservas en la cache
		for _, reservationDAO := range reservationsDAO {
			if _, err := service.cacheRepository.CreateReservation(ctx, reservationDAO); err != nil {
				return nil, fmt.Errorf("error creating reservation in cache: %w", err)
			}
		}
	}

	// Se convierten las reservas de formato de base de datos a formato de dominio
	reservations := make([]hotelsDomain.Reservation, 0)
	for _, reservationDAO := range reservationsDAO {
		reservations = append(reservations, hotelsDomain.Reservation{
			ID:        reservationDAO.ID,
			HotelName: reservationDAO.HotelName,
			HotelID:   reservationDAO.HotelID,
			UserID:    reservationDAO.UserID,
			CheckIn:   reservationDAO.CheckIn,
			CheckOut:  reservationDAO.CheckOut,
		})
	}

	return reservations, nil
}

func (service Service) GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string) ([]hotelsDomain.Reservation, error) {
	// Se intenta obtener las reservas del repositorio de cache
	reservationsDAO, err := service.cacheRepository.GetReservationsByUserAndHotelID(ctx, hotelID, userID)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		reservationsDAO, err = service.mainRepository.GetReservationsByUserAndHotelID(ctx, hotelID, userID)
		if err != nil {
			return nil, fmt.Errorf("error getting reservations from repository: %v", err)
		}
		// Se guardan las reservas en la cache
		for _, reservationDAO := range reservationsDAO {
			if _, err := service.cacheRepository.CreateReservation(ctx, reservationDAO); err != nil {
				return nil, fmt.Errorf("error creating reservation in cache: %w", err)
			}
		}
	}

	// Se convierten las reservas de formato de base de datos a formato de dominio
	reservations := make([]hotelsDomain.Reservation, 0)
	for _, reservationDAO := range reservationsDAO {
		reservations = append(reservations, hotelsDomain.Reservation{
			ID:        reservationDAO.ID,
			HotelName: reservationDAO.HotelName,
			HotelID:   reservationDAO.HotelID,
			UserID:    reservationDAO.UserID,
			CheckIn:   reservationDAO.CheckIn,
			CheckOut:  reservationDAO.CheckOut,
		})
	}

	return reservations, nil
}

func (service Service) GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDomain.Reservation, error) {
	// Se intenta obtener las reservas del repositorio de cache
	reservationsDAO, err := service.cacheRepository.GetReservationsByUserID(ctx, userID)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		reservationsDAO, err = service.mainRepository.GetReservationsByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("error getting reservations from repository: %v", err)
		}
		// Se guardan las reservas en la cache
		for _, reservationDAO := range reservationsDAO {
			if _, err := service.cacheRepository.CreateReservation(ctx, reservationDAO); err != nil {
				return nil, fmt.Errorf("error creating reservation in cache: %w", err)
			}
		}
	}

	// Se convierten las reservas de formato de base de datos a formato de dominio
	reservations := make([]hotelsDomain.Reservation, 0)
	for _, reservationDAO := range reservationsDAO {
		reservations = append(reservations, hotelsDomain.Reservation{
			ID:        reservationDAO.ID,
			HotelName: reservationDAO.HotelName,
			HotelID:   reservationDAO.HotelID,
			UserID:    reservationDAO.UserID,
			CheckIn:   reservationDAO.CheckIn,
			CheckOut:  reservationDAO.CheckOut,
		})
	}

	return reservations, nil
}

// Hay que ver lo de hacerlo desde la cache
func (service Service) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	// Se intenta obtener la disponibilidad de los hoteles del repositorio de cache
	availability, err := service.cacheRepository.GetAvailability(ctx, hotelIDs, checkIn, checkOut)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		availability, err = service.mainRepository.GetAvailability(ctx, hotelIDs, checkIn, checkOut)
		if err != nil {
			return nil, fmt.Errorf("error getting availability from repository: %v", err)
		}
	}

	return availability, nil
}
