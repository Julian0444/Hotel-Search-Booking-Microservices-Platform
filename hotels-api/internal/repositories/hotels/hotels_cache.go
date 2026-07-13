package hotels

import (
	"context"
	"fmt"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"

	"github.com/karlseguin/ccache"
)

const (
	keyFormat = "hotel:%s"
)

// normalizeDate devuelve la fecha a medianoche (00:00:00) para evitar problemas de comparación por horas.
func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// invalidateReservationLists borra las listas agregadas del par hotel/usuario.
// Las listas nunca se editan in-place (RV1: cachear parciales envenena; RV2:
// mutar slices compartidos con lectores concurrentes es un data race): toda
// escritura invalida y la próxima lectura repobla completa desde Mongo.
func (repository Cache) invalidateReservationLists(reservation hotelsDAO.Reservation) {
	repository.client.Delete(fmt.Sprintf("reservations:hotel:%s", reservation.HotelID))
	repository.client.Delete(fmt.Sprintf("reservations:user:%s", reservation.UserID))
	repository.client.Delete(fmt.Sprintf("reservations:hotel:%s:user:%s", reservation.HotelID, reservation.UserID))
}

// storeReservationsList guarda una lista agregada COMPLETA bajo key, como
// copia (los llamadores conservan su slice; nadie comparte backing array con
// la caché), y cachea también cada reserva individual. Una lista vacía se
// guarda igual: "sé que no hay reservas" también es un hit válido.
func (repository Cache) storeReservationsList(key string, reservations []hotelsDAO.Reservation) {
	list := make([]hotelsDAO.Reservation, len(reservations))
	copy(list, reservations)
	for _, r := range list {
		repository.client.Set(fmt.Sprintf("reservation:%s", r.ID), r, repository.duration)
	}
	repository.client.Set(key, list, repository.duration)
}

func (repository Cache) SetReservationsByHotelID(_ context.Context, hotelID string, reservations []hotelsDAO.Reservation) {
	repository.storeReservationsList(fmt.Sprintf("reservations:hotel:%s", hotelID), reservations)
}

func (repository Cache) SetReservationsByUserID(_ context.Context, userID string, reservations []hotelsDAO.Reservation) {
	repository.storeReservationsList(fmt.Sprintf("reservations:user:%s", userID), reservations)
}

func (repository Cache) SetReservationsByUserAndHotelID(_ context.Context, hotelID, userID string, reservations []hotelsDAO.Reservation) {
	repository.storeReservationsList(fmt.Sprintf("reservations:hotel:%s:user:%s", hotelID, userID), reservations)
}

type CacheConfig struct {
	MaxSize      int64
	ItemsToPrune uint32
	Duration     time.Duration
}

type Cache struct {
	client   *ccache.Cache
	duration time.Duration
}

// Crea una nueva instancia de Cache
func NewCache(config CacheConfig) Cache {
	client := ccache.New(ccache.Configure().
		MaxSize(config.MaxSize).
		ItemsToPrune(config.ItemsToPrune))
	return Cache{
		client:   client,
		duration: config.Duration,
	}
}

// Obtiene un hotel por su ID de la cache
func (repository Cache) GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error) {
	//Crea la llave para buscar el hotel
	key := fmt.Sprintf(keyFormat, id)
	//Obtiene el item de la cache
	item := repository.client.Get(key)
	//Si no se encuentra el item, regresa un error
	if item == nil {
		return hotelsDAO.Hotel{}, fmt.Errorf("not found item with key %s", key)
	}
	//Si el item esta expirado, regresa un error
	if item.Expired() {
		return hotelsDAO.Hotel{}, fmt.Errorf("item with key %s is expired", key)
	}
	hotelDAO, ok := item.Value().(hotelsDAO.Hotel)
	if !ok {
		return hotelsDAO.Hotel{}, fmt.Errorf("error converting item with key %s", key)
	}

	return hotelDAO, nil
}

// Crea un nuevo hotel en la cache
func (repository Cache) Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	key := fmt.Sprintf(keyFormat, hotel.ID)
	//Guarda el hotel en la cache
	repository.client.Set(key, hotel, repository.duration)
	return hotel.ID, nil
}

// Actualiza un hotel en la cache
func (repository Cache) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	key := fmt.Sprintf(keyFormat, hotel.ID)

	// Busca el item actual en la cache y regresa un error si no se encuentra o esta expirado
	item := repository.client.Get(key)
	if item == nil {
		return fmt.Errorf("hotel with ID %s not found in cache", hotel.ID)
	}
	if item.Expired() {
		return fmt.Errorf("item with key %s is expired", key)
	}

	// Convierte el item a un hotel
	currentHotel, ok := item.Value().(hotelsDAO.Hotel)
	if !ok {
		return fmt.Errorf("error converting item with key %s", key)
	}

	// Actualiza solo los campos que no son cero o vacios
	if hotel.Name != "" {
		currentHotel.Name = hotel.Name
	}
	if hotel.Description != "" {
		currentHotel.Description = hotel.Description
	}
	if hotel.Address != "" {
		currentHotel.Address = hotel.Address
	}
	if hotel.City != "" {
		currentHotel.City = hotel.City
	}
	if hotel.State != "" {
		currentHotel.State = hotel.State
	}
	if hotel.Country != "" {
		currentHotel.Country = hotel.Country
	}
	if hotel.Phone != "" {
		currentHotel.Phone = hotel.Phone
	}
	if hotel.Email != "" {
		currentHotel.Email = hotel.Email
	}
	if hotel.PricePerNight != 0 {
		currentHotel.PricePerNight = hotel.PricePerNight
	}
	if hotel.AvaiableRooms != 0 {
		currentHotel.AvaiableRooms = hotel.AvaiableRooms
	}
	if !hotel.CheckInTime.IsZero() {
		currentHotel.CheckInTime = hotel.CheckInTime
	}
	if !hotel.CheckOutTime.IsZero() {
		currentHotel.CheckOutTime = hotel.CheckOutTime
	}
	if hotel.Rating != 0 {
		currentHotel.Rating = hotel.Rating
	}
	if len(hotel.Amenities) > 0 {
		currentHotel.Amenities = hotel.Amenities
	}
	if len(hotel.Images) > 0 {
		currentHotel.Images = hotel.Images
	}

	// Guarda el hotel actualizado en la cache y reinicia el tiempo de expiracion
	repository.client.Set(key, currentHotel, repository.duration)

	//Devuelve nil si no hay errores
	return nil
}

// Elimina un hotel de la cache
func (repository Cache) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf(keyFormat, id)
	// Elimina el hotel de la cache
	repository.client.Delete(key)
	return nil
}

// Crea una reserva en la cache: guarda la copia individual e INVALIDA las
// listas agregadas del par hotel/usuario (nunca las edita — RV1/RV2). La
// próxima lectura repobla la lista completa desde Mongo vía los setters.
func (repository Cache) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	key := fmt.Sprintf("reservation:%s", reservation.ID)
	repository.client.Set(key, reservation, repository.duration)
	repository.invalidateReservationLists(reservation)
	return reservation.ID, nil
}

// Obtiene una reserva por ID de la cache
func (repository Cache) GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	key := fmt.Sprintf("reservation:%s", id)
	item := repository.client.Get(key)
	if item == nil {
		return hotelsDAO.Reservation{}, fmt.Errorf("reservation not found with ID %s", id)
	}
	if item.Expired() {
		return hotelsDAO.Reservation{}, fmt.Errorf("reservation with ID %s is expired", id)
	}
	reservation, ok := item.Value().(hotelsDAO.Reservation)
	if !ok {
		return hotelsDAO.Reservation{}, fmt.Errorf("error converting reservation with ID %s", id)
	}
	return reservation, nil
}

// Cancela una reserva en la cache. Sin call-sites de producción: el service
// refleja cancelaciones con CreateReservation(copia cancelada de Mongo), que
// funciona igual con la key individual evicted (RV3). Queda por la interfaz
// Repository: marca la copia individual si está e invalida las listas.
func (repository Cache) CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	reservation, err := repository.GetReservationByID(ctx, id)
	if err != nil {
		// Miss: sin la reserva no se conocen hotel/usuario para invalidar sus
		// listas — exactamente por eso el service usa CreateReservation (RV3).
		repository.client.Delete(fmt.Sprintf("reservation:%s", id))
		return hotelsDAO.Reservation{}, nil
	}

	now := time.Now().UTC()
	reservation.Status = hotelsDAO.StatusCancelled
	reservation.CancelledAt = &now

	repository.client.Set(fmt.Sprintf("reservation:%s", id), reservation, repository.duration)
	repository.invalidateReservationLists(reservation)

	return reservation, nil
}

// paginateReservations aplica limit/offset sobre una lista cacheada, con la
// misma semántica que SetLimit/SetSkip en Mongo.
func paginateReservations(reservations []hotelsDAO.Reservation, limit, offset int64) []hotelsDAO.Reservation {
	if offset >= int64(len(reservations)) {
		return []hotelsDAO.Reservation{}
	}
	end := offset + limit
	if end > int64(len(reservations)) {
		end = int64(len(reservations))
	}
	return reservations[offset:end]
}

// Obtiene las reservas por ID de hotel y usuario de la cache
func (repository Cache) GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	key := fmt.Sprintf("reservations:hotel:%s:user:%s", hotelID, userID)
	item := repository.client.Get(key)
	if item == nil {
		return nil, fmt.Errorf("not found item with key %s", key)
	}
	if item.Expired() {
		return nil, fmt.Errorf("item with key %s is expired", key)
	}
	reservations, ok := item.Value().([]hotelsDAO.Reservation)
	if !ok {
		return nil, fmt.Errorf("error converting item with key %s", key)
	}
	return paginateReservations(reservations, limit, offset), nil
}

// Obtiene las reservas por ID de hotel de la cache
func (repository Cache) GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	key := fmt.Sprintf("reservations:hotel:%s", hotelID)
	item := repository.client.Get(key)
	if item == nil {
		return nil, fmt.Errorf("not found item with key %s", key)
	}
	if item.Expired() {
		return nil, fmt.Errorf("item with key %s is expired", key)
	}
	reservations, ok := item.Value().([]hotelsDAO.Reservation)
	if !ok {
		return nil, fmt.Errorf("error converting item with key %s", key)
	}
	return paginateReservations(reservations, limit, offset), nil
}

// Obtiene las reservas por ID de usuario de la cache
func (repository Cache) GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	key := fmt.Sprintf("reservations:user:%s", userID)
	item := repository.client.Get(key)
	if item == nil {
		return nil, fmt.Errorf("not found item with key %s", key)
	}
	if item.Expired() {
		return nil, fmt.Errorf("item with key %s is expired", key)
	}
	reservations, ok := item.Value().([]hotelsDAO.Reservation)
	if !ok {
		return nil, fmt.Errorf("error converting item with key %s", key)
	}
	return paginateReservations(reservations, limit, offset), nil
}

// GetAvailability verifica la disponibilidad de múltiples hoteles en caché
func (repository Cache) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	if len(hotelIDs) == 0 {
		return map[string]bool{}, nil
	}

	// Verificar si todos los hoteles están en la caché
	for _, id := range hotelIDs {
		key := fmt.Sprintf(keyFormat, id)
		item := repository.client.Get(key)
		if item == nil || item.Expired() {
			return nil, fmt.Errorf("hotel with ID %s not found or expired in cache", id)
		}
	}
	type result struct {
		hotelID   string
		available bool
		err       error
	}

	results := make(chan result, len(hotelIDs))

	for _, id := range hotelIDs {
		go func(hotelID string) {
			available, err := repository.IsHotelAvailable(ctx, hotelID, checkIn, checkOut)
			results <- result{
				hotelID:   hotelID,
				available: available,
				err:       err,
			}
		}(id)
	}

	availability := make(map[string]bool)
	for i := 0; i < len(hotelIDs); i++ {
		r := <-results
		if r.err != nil {
			// La caché no puede responder por este hotel: se propaga y el
			// service cae a Mongo (RV4 — antes se mapeaba a available=false
			// con error nil, mintiendo ante input inválido). El canal tiene
			// buffer: las goroutines restantes no quedan colgadas.
			return nil, fmt.Errorf("cache availability failed for hotel %s: %w", r.hotelID, r.err)
		}
		availability[r.hotelID] = r.available
	}

	return availability, nil
}

// IsHotelAvailable verifica la disponibilidad de un hotel en caché
func (repository Cache) IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error) {
	// Convertir y normalizar fechas
	checkInTime, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return false, fmt.Errorf("error parsing check-in date: %w", err)
	}
	checkInTime = normalizeDate(checkInTime)

	checkOutTime, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return false, fmt.Errorf("error parsing check-out date: %w", err)
	}
	checkOutTime = normalizeDate(checkOutTime)

	if !checkOutTime.After(checkInTime) {
		return false, fmt.Errorf("check-out date must be after check-in date")
	}

	// Obtener hotel de caché
	hotel, err := repository.GetHotelByID(ctx, hotelID)
	if err != nil {
		return false, fmt.Errorf("error getting hotel from cache: %w", err)
	}

	// Obtener reservas de caché
	key := fmt.Sprintf("reservations:hotel:%s", hotelID)
	item := repository.client.Get(key)
	if item == nil || item.Expired() {
		// Lista ausente = cero reservas conocidas = disponible (D3). El falso
		// negativo anterior reportaba hoteles libres como "no disponibles";
		// el no-overbooking real lo garantiza el claim atómico en Mongo.
		return true, nil
	}

	reservations, ok := item.Value().([]hotelsDAO.Reservation)
	if !ok {
		return false, fmt.Errorf("error converting cached reservations")
	}

	// Contar habitaciones ocupadas por noche (fechas normalizadas), salteando
	// canceladas — misma semántica que el inventario de Mongo (D4).
	reservationsByDay := make(map[time.Time]int)
	for _, reservation := range reservations {
		if reservation.Status == hotelsDAO.StatusCancelled {
			continue
		}
		rooms := reservation.NumRooms
		if rooms < 1 {
			rooms = 1 // reservas pre-plan-04 sin num_rooms
		}
		resCheckIn := normalizeDate(reservation.CheckIn)
		resCheckOut := normalizeDate(reservation.CheckOut)

		// Se solapan si resCheckIn < checkOutTime y resCheckOut > checkInTime
		if resCheckOut.After(checkInTime) && resCheckIn.Before(checkOutTime) {
			// Iterar noches ocupadas: incluye check-in, excluye check-out
			for date := resCheckIn; date.Before(resCheckOut); date = date.AddDate(0, 0, 1) {
				if !date.Before(checkInTime) && date.Before(checkOutTime) {
					reservationsByDay[date] += rooms
				}
			}
		}
	}

	// Verificar disponibilidad para cada noche solicitada (excluye día de checkout)
	for date := checkInTime; date.Before(checkOutTime); date = date.AddDate(0, 0, 1) {
		if reservationsByDay[date] >= hotel.AvaiableRooms {
			return false, nil
		}
	}

	return true, nil
}

// Elimina todas las reservas de un hotel de la cache: borra las copias
// individuales que la lista del hotel conozca e invalida las listas de los
// usuarios involucrados (borrar, no editar — misma regla que RV1/RV2).
func (repository Cache) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	// Lista completa si está cacheada (limit alto: no una página)
	reservations, err := repository.GetReservationsByHotelID(ctx, hotelID, int64(^uint64(0)>>1), 0)
	if err == nil {
		for _, r := range reservations {
			repository.client.Delete(fmt.Sprintf("reservation:%s", r.ID))
			repository.client.Delete(fmt.Sprintf("reservations:user:%s", r.UserID))
			repository.client.Delete(fmt.Sprintf("reservations:hotel:%s:user:%s", hotelID, r.UserID))
		}
	}
	repository.client.Delete(fmt.Sprintf("reservations:hotel:%s", hotelID))
	return nil
}
