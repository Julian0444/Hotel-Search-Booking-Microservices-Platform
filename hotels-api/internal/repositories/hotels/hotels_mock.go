package hotels

import (
	"context"
	"fmt"
	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"

	"github.com/google/uuid"
)

// Mock simula un repositorio en memoria para hoteles y reservas (REPOSITORIO PRINCIPAL)
type Mock struct {
	hotels   map[string]hotelsDAO.Hotel
	reservas map[string]hotelsDAO.Reservation
}

// MockCache simula la cache (siempre devuelve error si no encuentra)
type MockCache struct {
	hotels   map[string]hotelsDAO.Hotel
	reservas map[string]hotelsDAO.Reservation
}

// Constructor del mock principal
func NewMock() Mock {
	return Mock{
		hotels:   make(map[string]hotelsDAO.Hotel),
		reservas: make(map[string]hotelsDAO.Reservation),
	}
}

// Constructor del mock de cache
func NewMockCache() MockCache {
	return MockCache{
		hotels:   make(map[string]hotelsDAO.Hotel),
		reservas: make(map[string]hotelsDAO.Reservation),
	}
}

// ===== MOCK PRINCIPAL (comportamiento normal) =====

// CRUD de hoteles
func (m Mock) GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error) {
	hotel, ok := m.hotels[id]
	if !ok {
		return hotelsDAO.Hotel{}, fmt.Errorf("hotel with ID %s not found", id)
	}
	return hotel, nil
}

func (m Mock) Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	id := uuid.New().String()
	hotel.ID = id
	m.hotels[id] = hotel
	return id, nil
}

func (m Mock) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	_, ok := m.hotels[hotel.ID]
	if !ok {
		return fmt.Errorf("hotel with ID %s not found", hotel.ID)
	}
	m.hotels[hotel.ID] = hotel
	return nil
}

func (m Mock) Delete(ctx context.Context, id string) error {
	_, ok := m.hotels[id]
	if !ok {
		return fmt.Errorf("hotel with ID %s not found", id)
	}
	delete(m.hotels, id)
	return nil
}

// CRUD de reservas
func (m Mock) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	id := uuid.New().String()
	reservation.ID = id
	m.reservas[id] = reservation
	return id, nil
}

func (m Mock) GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	reservation, ok := m.reservas[id]
	if !ok {
		return hotelsDAO.Reservation{}, fmt.Errorf("reservation with ID %s not found", id)
	}
	return reservation, nil
}

func (m Mock) CancelReservation(ctx context.Context, id string) error {
	_, ok := m.reservas[id]
	if !ok {
		return fmt.Errorf("reservation with ID %s not found", id)
	}
	delete(m.reservas, id)
	return nil
}

func (m Mock) GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.HotelID == hotelID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m Mock) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.HotelID == hotelID && r.UserID == userID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m Mock) GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.UserID == userID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m Mock) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range hotelIDs {
		_, ok := m.hotels[id]
		result[id] = ok
	}
	return result, nil
}

// Elimina todas las reservas de un hotel del mock
func (m Mock) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	// Eliminar todas las reservas que pertenezcan al hotel especificado
	var reservationsToDelete []string
	for id, reservation := range m.reservas {
		if reservation.HotelID == hotelID {
			reservationsToDelete = append(reservationsToDelete, id)
		}
	}

	// Eliminar las reservas encontradas
	for _, id := range reservationsToDelete {
		delete(m.reservas, id)
	}

	return nil
}

// ===== MOCK CACHE (comportamiento como la cache real) =====

// La cache NO crea hoteles, solo los almacena
func (m MockCache) Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	m.hotels[hotel.ID] = hotel
	return hotel.ID, nil
}

// La cache NO crea reservas, solo las almacena
func (m MockCache) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	m.reservas[reservation.ID] = reservation
	return reservation.ID, nil
}

func (m MockCache) GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	reservation, ok := m.reservas[id]
	if !ok {
		return hotelsDAO.Reservation{}, fmt.Errorf("reservation not found with ID %s", id)
	}
	return reservation, nil
}

// La cache SIEMPRE devuelve error si no encuentra
func (m MockCache) GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error) {
	hotel, ok := m.hotels[id]
	if !ok {
		return hotelsDAO.Hotel{}, fmt.Errorf("not found item with key hotel:%s", id)
	}
	return hotel, nil
}

func (m MockCache) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	_, ok := m.hotels[hotel.ID]
	if !ok {
		return fmt.Errorf("hotel with ID %s not found in cache", hotel.ID)
	}
	m.hotels[hotel.ID] = hotel
	return nil
}

func (m MockCache) Delete(ctx context.Context, id string) error {
	// La cache real no devuelve error si no existe
	delete(m.hotels, id)
	return nil
}

func (m MockCache) CancelReservation(ctx context.Context, id string) error {
	// La cache real no devuelve error si no existe
	delete(m.reservas, id)
	return nil
}

func (m MockCache) GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.HotelID == hotelID {
			result = append(result, r)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("not found item with key reservations:hotel:%s", hotelID)
	}
	return result, nil
}

func (m MockCache) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.HotelID == hotelID && r.UserID == userID {
			result = append(result, r)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("not found item with key reservations:hotel:%s:user:%s", hotelID, userID)
	}
	return result, nil
}

func (m MockCache) GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.UserID == userID {
			result = append(result, r)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("not found item with key reservations:user:%s", userID)
	}
	return result, nil
}

func (m MockCache) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range hotelIDs {
		_, ok := m.hotels[id]
		if !ok {
			return nil, fmt.Errorf("hotel with ID %s not found or expired in cache", id)
		}
		result[id] = true
	}
	return result, nil
}

// Elimina todas las reservas de un hotel del mock cache
func (m MockCache) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	// Eliminar todas las reservas que pertenezcan al hotel especificado
	var reservationsToDelete []string
	for id, reservation := range m.reservas {
		if reservation.HotelID == hotelID {
			reservationsToDelete = append(reservationsToDelete, id)
		}
	}

	// Eliminar las reservas encontradas
	for _, id := range reservationsToDelete {
		delete(m.reservas, id)
	}

	return nil
}
