package hotels

import (
	"context"
	"fmt"
	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	"time"

	"github.com/karlseguin/ccache"
)

const (
	keyFormat = "hotel:%s"
)

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
	key := fmt.Sprintf("reservation:%s", reservation.ID)
	repository.client.Set(key, reservation, repository.duration)
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
	key := fmt.Sprintf("reservation:%s", id)
	repository.client.Delete(key)
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
	// Convertir fechas
	checkInTime, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return false, fmt.Errorf("error parsing check-in date: %w", err)
	}
	checkOutTime, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return false, fmt.Errorf("error parsing check-out date: %w", err)
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

	// Contar reservas por día usando un mapa
	reservationsByDay := make(map[time.Time]int)
	for _, reservation := range reservations {
		// Solo considerar reservas que se solapan con el período solicitado
		if !reservation.CheckOut.Before(checkInTime) && !reservation.CheckIn.After(checkOutTime) {
			for date := reservation.CheckIn; !date.After(reservation.CheckOut); date = date.AddDate(0, 0, 1) {
				if !date.Before(checkInTime) && !date.After(checkOutTime) {
					reservationsByDay[date]++
				}
			}
		}
	}

	// Verificar disponibilidad para cada día
	for date := checkInTime; !date.After(checkOutTime); date = date.AddDate(0, 0, 1) {
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
	
	// Eliminar cada reserva individual de la cache
	for _, reservation := range reservations {
		key := fmt.Sprintf("reservation:%s", reservation.ID)
		repository.client.Delete(key)
	}
	
	// Eliminar también la lista de reservas del hotel
	hotelReservationsKey := fmt.Sprintf("reservations:hotel:%s", hotelID)
	repository.client.Delete(hotelReservationsKey)
	
	return nil
}
