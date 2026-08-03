package hotels

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

// Moneda de los precios de la plataforma: los price_per_night del seed y la
// demo están expresados en USD. TotalPrice viaja en centavos (int64).
const reservationCurrency = "USD"

// Estas funciones salen de los repositorios, se encargan de interactuar tanto de la base de datos como de la cache, ambas tienen las mismas funciones pero con diferentes implementaciones para cada cosa
type Repository interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error)
	GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDAO.Hotel, error)
	CountHotels(ctx context.Context) (int64, error)
	Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error)
	Update(ctx context.Context, hotel hotelsDAO.Hotel) error
	Delete(ctx context.Context, id string) error
	CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error)
	GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error)
	CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error)
	GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDAO.Reservation, error)
	GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error)
	GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error)
	DeleteReservationsByHotelID(ctx context.Context, hotelID string) error
	GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error)
}

// CacheRepository extiende Repository con los setters de listas completas:
// las listas agregadas solo se escriben ENTERAS (desde el service, que sabe
// cuándo una página es la lista completa) y se invalidan en cada escritura de
// reserva — nunca se editan por-ítem (RV1/RV2).
type CacheRepository interface {
	Repository
	SetReservationsByHotelID(ctx context.Context, hotelID string, reservations []hotelsDAO.Reservation)
	SetReservationsByUserID(ctx context.Context, userID string, reservations []hotelsDAO.Reservation)
	SetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, reservations []hotelsDAO.Reservation)
}

type Queue interface {
	Publish(hotelNew hotelsDomain.HotelNew) error
	PublishReservation(reservationNew hotelsDomain.ReservationNew) error
}

type Service struct {
	mainRepository  Repository
	cacheRepository CacheRepository
	eventsQueue     Queue
}

// Funcion que se encarga de crear un nuevo servicio con los repositorios y la cola de eventos
func NewService(mainRepository Repository, cacheRepository CacheRepository, eventsQueue Queue) Service {
	return Service{
		mainRepository:  mainRepository,
		cacheRepository: cacheRepository,
		eventsQueue:     eventsQueue,
	}
}

// hotelToDomain convierte el modelo DAO al de dominio para respuestas.
func hotelToDomain(hotelDAO hotelsDAO.Hotel) hotelsDomain.Hotel {
	return hotelsDomain.Hotel{
		ID:             hotelDAO.ID,
		Name:           hotelDAO.Name,
		Description:    hotelDAO.Description,
		Address:        hotelDAO.Address,
		City:           hotelDAO.City,
		State:          hotelDAO.State,
		Country:        hotelDAO.Country,
		Phone:          hotelDAO.Phone,
		Email:          hotelDAO.Email,
		PricePerNight:  hotelDAO.PricePerNight,
		Rating:         hotelDAO.Rating,
		AvailableRooms: hotelDAO.AvailableRooms,
		CheckInTime:    hotelDAO.CheckInTime,
		CheckOutTime:   hotelDAO.CheckOutTime,
		Amenities:      hotelDAO.Amenities,
		Images:         hotelDAO.Images,
	}
}

// Funcion que se encarga de obtener un hotel por su ID, primero se intenta obtener de la cache, si no se encuentra se obtiene de la base de datos principal y se guarda en la cache
func (service Service) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	// Se intenta obtener el hotel de la cache
	hotelDAO, err := service.cacheRepository.GetHotelByID(ctx, id)
	if err != nil {
		// Si no se encuentra en la cache, se obtiene de la base de datos principal.
		// %w para que ErrHotelNotFound sobreviva hasta el controller (RV14).
		hotelDAO, err = service.mainRepository.GetHotelByID(ctx, id)
		if err != nil {
			return hotelsDomain.Hotel{}, fmt.Errorf("error getting hotel from repository: %w", err)
		}
		// Se guarda el hotel en la cache — best-effort (R3): un fallo de caché
		// nunca falla una lectura ya resuelta
		if _, err := service.cacheRepository.Create(ctx, hotelDAO); err != nil {
			slog.Warn("error caching hotel", "hotel_id", id, "error", err)
		}
	}

	// Lo pasa de formato de base de datos a formato de dominio para las respuestas
	return hotelToDomain(hotelDAO), nil
}

// GetHotels lista el catálogo paginado directo del repositorio principal (E3):
// alimenta el backfill/reindex de search-api. No pasa por caché a propósito —
// cachear páginas de listado reintroduciría el veneno de listas parciales (RV1).
func (service Service) GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDomain.Hotel, int64, error) {
	hotelsDAOList, err := service.mainRepository.GetHotels(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting hotels from repository: %w", err)
	}
	total, err := service.mainRepository.CountHotels(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting hotels: %w", err)
	}

	hotelsDomainList := make([]hotelsDomain.Hotel, 0, len(hotelsDAOList))
	for _, hotelDAO := range hotelsDAOList {
		hotelsDomainList = append(hotelsDomainList, hotelToDomain(hotelDAO))
	}
	return hotelsDomainList, total, nil
}

// Funcion que se encarga de crear un nuevo hotel, primero se crea en la base de datos principal, luego en la cache y por ultimo se publica un evento para notificar que se creo un nuevo hotel
func (service Service) Create(ctx context.Context, hotel hotelsDomain.Hotel) (string, error) {
	// Convierte el modelo de dominio a modelo DAO
	//Modelo de como viene -> modelo base de datos
	record := hotelsDAO.Hotel{
		Name:           hotel.Name,
		Description:    hotel.Description,
		Address:        hotel.Address,
		City:           hotel.City,
		State:          hotel.State,
		Country:        hotel.Country,
		Phone:          hotel.Phone,
		Email:          hotel.Email,
		PricePerNight:  hotel.PricePerNight,
		Rating:         hotel.Rating,
		AvailableRooms: hotel.AvailableRooms,
		CheckInTime:    hotel.CheckInTime,
		CheckOutTime:   hotel.CheckOutTime,
		Amenities:      hotel.Amenities,
		Images:         hotel.Images,
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
		ID:             hotel.ID,
		Name:           hotel.Name,
		Description:    hotel.Description,
		Address:        hotel.Address,
		City:           hotel.City,
		State:          hotel.State,
		Country:        hotel.Country,
		Phone:          hotel.Phone,
		Email:          hotel.Email,
		PricePerNight:  hotel.PricePerNight,
		Rating:         hotel.Rating,
		AvailableRooms: hotel.AvailableRooms,
		CheckInTime:    hotel.CheckInTime,
		CheckOutTime:   hotel.CheckOutTime,
		Amenities:      hotel.Amenities,
		Images:         hotel.Images,
	}

	// Actualiza el hotel en el repositorio principal (MongoDB)
	err := service.mainRepository.Update(ctx, record)
	if err != nil {
		return fmt.Errorf("error updating hotel in main repository: %w", err)
	}

	// Caché best-effort (D2): si el hotel expiró de la caché, loguear y seguir.
	// Nunca fallar una escritura de DB exitosa — y el evento se publica SIEMPRE.
	if err := service.cacheRepository.Update(ctx, record); err != nil {
		slog.Warn("error updating hotel in cache (continuing)", "hotel_id", hotel.ID, "error", err)
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

	// Eliminar las reservas del hotel de la cache — best-effort (D2)
	if err := service.cacheRepository.DeleteReservationsByHotelID(ctx, id); err != nil {
		slog.Warn("error deleting hotel reservations from cache (continuing)", "hotel_id", id, "error", err)
	}

	// Intenta eliminar el hotel del repositorio principal (MongoDB)
	err := service.mainRepository.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting hotel from main repository: %w", err)
	}

	// Eliminar el hotel de la cache — best-effort (D2)
	if err := service.cacheRepository.Delete(ctx, id); err != nil {
		slog.Warn("error deleting hotel from cache (continuing)", "hotel_id", id, "error", err)
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

// reservationToDomain convierte el modelo DAO al de dominio para respuestas.
// Las fechas de estadía salen date-only (RV20): serializar el time.Time de
// Mongo como RFC3339-UTC hacía que el frontend las corriera un día en
// timezones al oeste de UTC.
func reservationToDomain(reservationDAO hotelsDAO.Reservation) hotelsDomain.Reservation {
	return hotelsDomain.Reservation{
		ID:          reservationDAO.ID,
		HotelName:   reservationDAO.HotelName,
		HotelID:     reservationDAO.HotelID,
		UserID:      reservationDAO.UserID,
		CheckIn:     reservationDAO.CheckIn.Format(hotelsDomain.DateFormat),
		CheckOut:    reservationDAO.CheckOut.Format(hotelsDomain.DateFormat),
		Status:      reservationDAO.Status,
		NumRooms:    reservationDAO.NumRooms,
		NumGuests:   reservationDAO.NumGuests,
		TotalPrice:  reservationDAO.TotalPrice,
		Currency:    reservationDAO.Currency,
		CreatedAt:   reservationDAO.CreatedAt,
		CancelledAt: reservationDAO.CancelledAt,
	}
}

// normalizeToDay trunca a medianoche para comparar solo la parte fecha.
func normalizeToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CreateReservation valida la reserva contra el hotel (DM2), deriva nombre y
// precio del hotel (nunca del body) y delega el no-overbooking al claim
// atómico del repositorio (D1), que devuelve ErrNoAvailability sin cupo.
// Los errores de validación van tipados con ErrInvalidReservation para que el
// controller responda 400 y no 500 (RV19).
func (service Service) CreateReservation(ctx context.Context, reservation hotelsDomain.Reservation) (string, error) {
	// El hotel tiene que existir (cache-aside vía GetHotelByID). El wrap %w
	// preserva ErrHotelNotFound → el controller mapea 404 (RV19).
	hotel, err := service.GetHotelByID(ctx, reservation.HotelID)
	if err != nil {
		return "", fmt.Errorf("error getting hotel for reservation: %w", err)
	}

	// Defaults y validaciones
	if reservation.NumRooms == 0 {
		reservation.NumRooms = 1
	}
	if reservation.NumGuests == 0 {
		reservation.NumGuests = 1
	}
	checkIn, err := time.Parse(hotelsDomain.DateFormat, reservation.CheckIn)
	if err != nil {
		return "", fmt.Errorf("check_in must be a date in YYYY-MM-DD format: %w", hotelsDomain.ErrInvalidReservation)
	}
	checkOut, err := time.Parse(hotelsDomain.DateFormat, reservation.CheckOut)
	if err != nil {
		return "", fmt.Errorf("check_out must be a date in YYYY-MM-DD format: %w", hotelsDomain.ErrInvalidReservation)
	}
	if !checkOut.After(checkIn) {
		return "", fmt.Errorf("check-out date must be after check-in date: %w", hotelsDomain.ErrInvalidReservation)
	}
	if checkIn.Before(normalizeToDay(time.Now().UTC())) {
		return "", fmt.Errorf("check-in date must not be in the past: %w", hotelsDomain.ErrInvalidReservation)
	}
	if reservation.NumRooms < 1 {
		return "", fmt.Errorf("num_rooms must be at least 1: %w", hotelsDomain.ErrInvalidReservation)
	}
	if reservation.NumRooms > hotel.AvailableRooms {
		return "", fmt.Errorf("num_rooms must be between 1 and the hotel capacity (%d): %w", hotel.AvailableRooms, hotelsDomain.ErrNoAvailability)
	}
	if reservation.NumGuests < 1 {
		return "", fmt.Errorf("num_guests must be at least 1: %w", hotelsDomain.ErrInvalidReservation)
	}

	// Derivar precio total en centavos (nunca float para dinero): precio por
	// noche redondeado a centavos × noches × habitaciones.
	nights := int64(checkOut.Sub(checkIn).Hours() / 24)
	totalPrice := int64(math.Round(hotel.PricePerNight*100)) * nights * int64(reservation.NumRooms)

	record := hotelsDAO.Reservation{
		HotelName:  hotel.Name, // derivado del hotel, no del body
		HotelID:    reservation.HotelID,
		UserID:     reservation.UserID,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
		Status:     hotelsDAO.StatusConfirmed,
		NumRooms:   reservation.NumRooms,
		NumGuests:  reservation.NumGuests,
		TotalPrice: totalPrice,
		Currency:   reservationCurrency,
		CreatedAt:  time.Now().UTC(),
	}

	// Crea la reserva en el repositorio principal: el claim atómico del
	// inventario pasa o devuelve ErrNoAvailability (el controller mapea 409)
	id, err := service.mainRepository.CreateReservation(ctx, record)
	if err != nil {
		return "", fmt.Errorf("error creating reservation in main repository: %w", err)
	}

	// Caché best-effort (R3): la reserva ya está persistida
	record.ID = id
	if _, err := service.cacheRepository.CreateReservation(ctx, record); err != nil {
		slog.Warn("error caching reservation (continuing)", "reservation_id", id, "error", err)
	}

	// Evento de reserva (DM5) por la cola reservations-news. Nadie la consume
	// aún: un fallo de publish no puede tirar una reserva ya confirmada.
	if err := service.eventsQueue.PublishReservation(hotelsDomain.ReservationNew{
		Operation:     "CREATE",
		ReservationID: id,
		HotelID:       record.HotelID,
	}); err != nil {
		slog.Warn("error publishing reservation created event (continuing)", "reservation_id", id, "error", err)
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
		// Se guarda la reserva en la cache — best-effort (R3)
		if _, err := service.cacheRepository.CreateReservation(ctx, reservationDAO); err != nil {
			slog.Warn("error caching reservation (continuing)", "reservation_id", id, "error", err)
		}
	}

	// Se convierte la reserva de formato de base de datos a formato de dominio
	return reservationToDomain(reservationDAO), nil
}

// CancelReservation cancela en el repositorio principal (soft-delete
// idempotente que libera el inventario) y refleja el cambio en la caché y en
// la cola de eventos de forma best-effort.
func (service Service) CancelReservation(ctx context.Context, id string) error {
	cancelled, err := service.mainRepository.CancelReservation(ctx, id)
	if err != nil {
		return fmt.Errorf("error canceling reservation from main repository: %w", err)
	}

	// Caché best-effort: setea la copia cancelada que devolvió Mongo e
	// invalida las listas agregadas — funciona igual con la key individual
	// evicted, cosa que Cache.CancelReservation no podía garantizar (RV3)
	if _, err := service.cacheRepository.CreateReservation(ctx, cancelled); err != nil {
		slog.Warn("error updating cancelled reservation in cache (continuing)", "reservation_id", id, "error", err)
	}

	// Evento de reserva (DM5) — best-effort, ver CreateReservation
	if err := service.eventsQueue.PublishReservation(hotelsDomain.ReservationNew{
		Operation:     "CANCEL",
		ReservationID: id,
		HotelID:       cancelled.HotelID,
	}); err != nil {
		slog.Warn("error publishing reservation cancelled event (continuing)", "reservation_id", id, "error", err)
	}

	return nil
}

func (service Service) GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	// Se intenta obtener las reservas del repositorio de cache
	reservationsDAO, err := service.cacheRepository.GetReservationsByHotelID(ctx, hotelID, limit, offset)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		reservationsDAO, err = service.mainRepository.GetReservationsByHotelID(ctx, hotelID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("error getting reservations from repository: %w", err)
		}
		// Se guarda en la cache SOLO si esta página es la lista completa
		// (offset 0 y menos resultados que el límite); cachear una página
		// parcial envenenaría la lista agregada. El setter guarda la lista
		// entera como copia (RV1/RV2). Best-effort (R3).
		if offset == 0 && int64(len(reservationsDAO)) < limit {
			service.cacheRepository.SetReservationsByHotelID(ctx, hotelID, reservationsDAO)
		}
	}

	// Se convierten las reservas de formato de base de datos a formato de dominio
	reservations := make([]hotelsDomain.Reservation, 0)
	for _, reservationDAO := range reservationsDAO {
		reservations = append(reservations, reservationToDomain(reservationDAO))
	}

	return reservations, nil
}

func (service Service) GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	// Se intenta obtener las reservas del repositorio de cache
	reservationsDAO, err := service.cacheRepository.GetReservationsByUserAndHotelID(ctx, hotelID, userID, limit, offset)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		reservationsDAO, err = service.mainRepository.GetReservationsByUserAndHotelID(ctx, hotelID, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("error getting reservations from repository: %w", err)
		}
		// Ver GetReservationsByHotelID: solo se cachea la lista completa; best-effort (R3)
		if offset == 0 && int64(len(reservationsDAO)) < limit {
			service.cacheRepository.SetReservationsByUserAndHotelID(ctx, hotelID, userID, reservationsDAO)
		}
	}

	// Se convierten las reservas de formato de base de datos a formato de dominio
	reservations := make([]hotelsDomain.Reservation, 0)
	for _, reservationDAO := range reservationsDAO {
		reservations = append(reservations, reservationToDomain(reservationDAO))
	}

	return reservations, nil
}

func (service Service) GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	// Se intenta obtener las reservas del repositorio de cache
	reservationsDAO, err := service.cacheRepository.GetReservationsByUserID(ctx, userID, limit, offset)
	if err != nil {
		// Si no se encuentran en la cache, se obtienen del repositorio principal
		reservationsDAO, err = service.mainRepository.GetReservationsByUserID(ctx, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("error getting reservations from repository: %w", err)
		}
		// Ver GetReservationsByHotelID: solo se cachea la lista completa; best-effort (R3)
		if offset == 0 && int64(len(reservationsDAO)) < limit {
			service.cacheRepository.SetReservationsByUserID(ctx, userID, reservationsDAO)
		}
	}

	// Se convierten las reservas de formato de base de datos a formato de dominio
	reservations := make([]hotelsDomain.Reservation, 0)
	for _, reservationDAO := range reservationsDAO {
		reservations = append(reservations, reservationToDomain(reservationDAO))
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
			return nil, fmt.Errorf("error getting availability from repository: %w", err)
		}
	}

	return availability, nil
}
