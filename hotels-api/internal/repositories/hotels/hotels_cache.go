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

// updateHotelReservationsList mantiene sincronizada la lista agregada de reservas por hotel.
func (repository Cache) updateHotelReservationsList(_ context.Context, reservation hotelsDAO.Reservation, add bool) {
	key := fmt.Sprintf("reservations:hotel:%s", reservation.HotelID)
	item := repository.client.Get(key)

	var reservations []hotelsDAO.Reservation
	if item != nil && !item.Expired() {
		if existingReservations, ok := item.Value().([]hotelsDAO.Reservation); ok {
			reservations = existingReservations
		}
	}

	if add {
		// Agrega o reemplaza la reserva
		found := false
		for i, r := range reservations {
			if r.ID == reservation.ID {
				reservations[i] = reservation
				found = true
				break
			}
		}
		if !found {
			reservations = append(reservations, reservation)
		}
	} else {
		// Elimina la reserva
		for i, r := range reservations {
			if r.ID == reservation.ID {
				reservations = append(reservations[:i], reservations[i+1:]...)
				break
			}
		}
	}

	if len(reservations) > 0 {
		repository.client.Set(key, reservations, repository.duration)
	} else {
		repository.client.Delete(key)
	}
}

// updateUserReservationsList mantiene sincronizada la lista agregada de reservas por usuario.
func (repository Cache) updateUserReservationsList(_ context.Context, reservation hotelsDAO.Reservation, add bool) {
	key := fmt.Sprintf("reservations:user:%s", reservation.UserID)
	item := repository.client.Get(key)

	var reservations []hotelsDAO.Reservation
	if item != nil && !item.Expired() {
		if existingReservations, ok := item.Value().([]hotelsDAO.Reservation); ok {
			reservations = existingReservations
		}
	}

	if add {
		found := false
		for i, r := range reservations {
			if r.ID == reservation.ID {
				reservations[i] = reservation
				found = true
				break
			}
		}
		if !found {
			reservations = append(reservations, reservation)
		}
	} else {
		for i, r := range reservations {
			if r.ID == reservation.ID {
				reservations = append(reservations[:i], reservations[i+1:]...)
				break
			}
		}
	}

	if len(reservations) > 0 {
		repository.client.Set(key, reservations, repository.duration)
	} else {
		repository.client.Delete(key)
	}
}

// updateUserHotelReservationsList mantiene sincronizada la lista combinada por hotel y usuario.
func (repository Cache) updateUserHotelReservationsList(_ context.Context, reservation hotelsDAO.Reservation, add bool) {
	key := fmt.Sprintf("reservations:hotel:%s:user:%s", reservation.HotelID, reservation.UserID)
	item := repository.client.Get(key)

	var reservations []hotelsDAO.Reservation
	if item != nil && !item.Expired() {
		if existingReservations, ok := item.Value().([]hotelsDAO.Reservation); ok {
			reservations = existingReservations
		}
	}

	if add {
		found := false
		for i, r := range reservations {
			if r.ID == reservation.ID {
				reservations[i] = reservation
				found = true
				break
			}
		}
		if !found {
			reservations = append(reservations, reservation)
		}
	} else {
		for i, r := range reservations {
			if r.ID == reservation.ID {
				reservations = append(reservations[:i], reservations[i+1:]...)
				break
			}
		}
	}

	if len(reservations) > 0 {
		repository.client.Set(key, reservations, repository.duration)
	} else {
		repository.client.Delete(key)
	}
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

// Crea una reserva en la cache
func (repository Cache) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	// Guardar la reserva individual
	key := fmt.Sprintf("reservation:%s", reservation.ID)
	repository.client.Set(key, reservation, repository.duration)

	// Actualizar listas agregadas
	repository.updateHotelReservationsList(ctx, reservation, true)
	repository.updateUserReservationsList(ctx, reservation, true)
	repository.updateUserHotelReservationsList(ctx, reservation, true)

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

// Cancela una reserva en la cache
func (repository Cache) CancelReservation(ctx context.Context, id string) error {
	// Obtener la reserva para limpiar listas
	reservation, err := repository.GetReservationByID(ctx, id)
	if err != nil {
		// Si no existe, eliminar la clave individual y salir
		key := fmt.Sprintf("reservation:%s", id)
		repository.client.Delete(key)
		return nil
	}

	key := fmt.Sprintf("reservation:%s", id)
	repository.client.Delete(key)

	// Limpiar listas agregadas
	repository.updateHotelReservationsList(ctx, reservation, false)
	repository.updateUserReservationsList(ctx, reservation, false)
	repository.updateUserHotelReservationsList(ctx, reservation, false)

	return nil
}

// Obtiene las reservas por ID de hotel y usuario de la cache
func (repository Cache) GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string) ([]hotelsDAO.Reservation, error) {
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
	return reservations, nil
}

// Obtiene las reservas por ID de hotel de la cache
func (repository Cache) GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDAO.Reservation, error) {
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
	return reservations, nil
}

// Obtiene las reservas por ID de usuario de la cache
func (repository Cache) GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDAO.Reservation, error) {
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
	return reservations, nil
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
			// En caché, podemos continuar incluso si hay error en un hotel
			availability[r.hotelID] = false
			continue
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
		// Si no hay datos en caché, asumimos que no hay disponibilidad
		// Esta es una decisión conservadora para evitar overboking
		return false, nil
	}

	reservations, ok := item.Value().([]hotelsDAO.Reservation)
	if !ok {
		return false, fmt.Errorf("error converting cached reservations")
	}

	// Contar reservas por día usando un mapa (fechas normalizadas)
	reservationsByDay := make(map[time.Time]int)
	for _, reservation := range reservations {
		resCheckIn := normalizeDate(reservation.CheckIn)
		resCheckOut := normalizeDate(reservation.CheckOut)

		// Se solapan si resCheckIn < checkOutTime y resCheckOut > checkInTime
		if resCheckOut.After(checkInTime) && resCheckIn.Before(checkOutTime) {
			// Iterar noches ocupadas: incluye check-in, excluye check-out
			for date := resCheckIn; date.Before(resCheckOut); date = date.AddDate(0, 0, 1) {
				if !date.Before(checkInTime) && date.Before(checkOutTime) {
					reservationsByDay[date]++
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

// Elimina todas las reservas de un hotel de la cache
func (repository Cache) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	// Obtener todas las reservas del hotel para eliminarlas de la cache
	reservations, err := repository.GetReservationsByHotelID(ctx, hotelID)
	if err != nil {
		// Si no hay reservas, no hay nada que eliminar
		return nil
	}

	// Eliminar cada reserva individual y limpiar listas agregadas
	for _, reservation := range reservations {
		key := fmt.Sprintf("reservation:%s", reservation.ID)
		repository.client.Delete(key)

		repository.updateUserReservationsList(ctx, reservation, false)
		repository.updateUserHotelReservationsList(ctx, reservation, false)
	}

	// Eliminar también la lista de reservas del hotel
	hotelReservationsKey := fmt.Sprintf("reservations:hotel:%s", hotelID)
	repository.client.Delete(hotelReservationsKey)

	return nil
}
