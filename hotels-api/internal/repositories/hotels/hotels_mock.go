package hotels

import (
	"context"
	"fmt"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/google/uuid"
)

// countRoomsByNight suma las habitaciones ocupadas por noche para un hotel
// (checkout excluido, canceladas salteadas) — la misma semántica que el
// inventario de Mongo y la caché real (D4).
func countRoomsByNight(reservas []hotelsDAO.Reservation, hotelID string, from, to time.Time) map[time.Time]int {
	roomsByDay := make(map[time.Time]int)
	for _, reservation := range reservas {
		if reservation.HotelID != hotelID || reservation.Status == hotelsDAO.StatusCancelled {
			continue
		}
		rooms := reservation.NumRooms
		if rooms < 1 {
			rooms = 1 // reservas legacy sin num_rooms
		}
		resCheckIn := normalizeDate(reservation.CheckIn)
		resCheckOut := normalizeDate(reservation.CheckOut)
		if resCheckOut.After(from) && resCheckIn.Before(to) {
			for date := resCheckIn; date.Before(resCheckOut); date = date.AddDate(0, 0, 1) {
				if !date.Before(from) && date.Before(to) {
					roomsByDay[date] += rooms
				}
			}
		}
	}
	return roomsByDay
}

// Mock simula un repositorio en memoria para hoteles y reservas (REPOSITORIO PRINCIPAL)
type Mock struct {
	hotels   map[string]hotelsDAO.Hotel
	reservas map[string]hotelsDAO.Reservation
}

// MockCache simula la cache real con su MISMA semántica de listas (RV5): los
// getters de listas solo aciertan si la lista agregada fue seteada completa, y
// toda escritura de reserva la invalida. (La versión anterior escaneaba las
// reservas como una DB — idealizada: no podía detectar RV1/RV3.)
type MockCache struct {
	hotels         map[string]hotelsDAO.Hotel
	reservas       map[string]hotelsDAO.Reservation
	hotelLists     map[string][]hotelsDAO.Reservation // key: hotelID
	userLists      map[string][]hotelsDAO.Reservation // key: userID
	userHotelLists map[string][]hotelsDAO.Reservation // key: hotelID+":"+userID
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
		hotels:         make(map[string]hotelsDAO.Hotel),
		reservas:       make(map[string]hotelsDAO.Reservation),
		hotelLists:     make(map[string][]hotelsDAO.Reservation),
		userLists:      make(map[string][]hotelsDAO.Reservation),
		userHotelLists: make(map[string][]hotelsDAO.Reservation),
	}
}

// ===== MOCK PRINCIPAL (comportamiento normal) =====

// allReservations devuelve las reservas del mock como slice (countRoomsByNight
// opera sobre slices para compartir la lógica con las listas del MockCache).
func (m Mock) allReservations() []hotelsDAO.Reservation {
	result := make([]hotelsDAO.Reservation, 0, len(m.reservas))
	for _, r := range m.reservas {
		result = append(result, r)
	}
	return result
}

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

// CRUD de reservas. CreateReservation replica el no-overbooking del repo real
// (D1): si alguna noche del rango no tiene cupo devuelve ErrNoAvailability.
func (m Mock) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	hotel, ok := m.hotels[reservation.HotelID]
	if !ok {
		return "", fmt.Errorf("hotel with ID %s not found", reservation.HotelID)
	}

	rooms := reservation.NumRooms
	if rooms < 1 {
		rooms = 1
	}
	if rooms > hotel.AvaiableRooms {
		return "", hotelsDomain.ErrNoAvailability
	}

	from := normalizeDate(reservation.CheckIn)
	to := normalizeDate(reservation.CheckOut)
	roomsByDay := countRoomsByNight(m.allReservations(), reservation.HotelID, from, to)
	for date := from; date.Before(to); date = date.AddDate(0, 0, 1) {
		if roomsByDay[date]+rooms > hotel.AvaiableRooms {
			return "", hotelsDomain.ErrNoAvailability
		}
	}

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

// CancelReservation replica el soft-delete idempotente del repo real (DM1).
func (m Mock) CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	reservation, ok := m.reservas[id]
	if !ok {
		return hotelsDAO.Reservation{}, fmt.Errorf("reservation with ID %s not found", id)
	}
	if reservation.Status == hotelsDAO.StatusCancelled {
		return reservation, nil // idempotente: ya cancelada
	}
	now := time.Now().UTC()
	reservation.Status = hotelsDAO.StatusCancelled
	reservation.CancelledAt = &now
	m.reservas[id] = reservation
	return reservation, nil
}

func (m Mock) GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.HotelID == hotelID {
			result = append(result, r)
		}
	}
	return paginateReservations(result, limit, offset), nil
}

func (m Mock) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.HotelID == hotelID && r.UserID == userID {
			result = append(result, r)
		}
	}
	return paginateReservations(result, limit, offset), nil
}

func (m Mock) GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	var result []hotelsDAO.Reservation
	for _, r := range m.reservas {
		if r.UserID == userID {
			result = append(result, r)
		}
	}
	return paginateReservations(result, limit, offset), nil
}

func (m Mock) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range hotelIDs {
		available, err := m.IsHotelAvailable(ctx, id, checkIn, checkOut)
		if err != nil {
			return nil, fmt.Errorf("error checking availability for hotel %s: %w", id, err)
		}
		result[id] = available
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

// IsHotelAvailable replica la lógica de disponibilidad del repo real:
// por noche, checkout excluido, canceladas salteadas, contando num_rooms.
func (m Mock) IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error) {
	hotel, ok := m.hotels[hotelID]
	if !ok {
		return false, fmt.Errorf("hotel with ID %s not found", hotelID)
	}

	checkInTime, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return false, fmt.Errorf("error parsing check-in date: %w", err)
	}
	checkOutTime, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return false, fmt.Errorf("error parsing check-out date: %w", err)
	}

	checkInTime = normalizeDate(checkInTime)
	checkOutTime = normalizeDate(checkOutTime)

	if !checkOutTime.After(checkInTime) {
		return false, fmt.Errorf("check-out date must be after check-in date")
	}

	roomsByDay := countRoomsByNight(m.allReservations(), hotelID, checkInTime, checkOutTime)
	for date := checkInTime; date.Before(checkOutTime); date = date.AddDate(0, 0, 1) {
		if roomsByDay[date] >= hotel.AvaiableRooms {
			return false, nil
		}
	}

	return true, nil
}

// ===== MOCK CACHE (comportamiento como la cache real) =====

// La cache NO crea hoteles, solo los almacena
func (m MockCache) Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	m.hotels[hotel.ID] = hotel
	return hotel.ID, nil
}

// invalidateReservationLists espeja la invalidación de la caché real: toda
// escritura de reserva borra las listas agregadas del par hotel/usuario.
func (m MockCache) invalidateReservationLists(reservation hotelsDAO.Reservation) {
	delete(m.hotelLists, reservation.HotelID)
	delete(m.userLists, reservation.UserID)
	delete(m.userHotelLists, reservation.HotelID+":"+reservation.UserID)
}

// storeReservationsList espeja el setter real: guarda la lista COMPLETA como
// copia y cachea también cada reserva individual.
func (m MockCache) storeReservationsList(lists map[string][]hotelsDAO.Reservation, key string, reservations []hotelsDAO.Reservation) {
	list := make([]hotelsDAO.Reservation, len(reservations))
	copy(list, reservations)
	for _, r := range list {
		m.reservas[r.ID] = r
	}
	lists[key] = list
}

func (m MockCache) SetReservationsByHotelID(_ context.Context, hotelID string, reservations []hotelsDAO.Reservation) {
	m.storeReservationsList(m.hotelLists, hotelID, reservations)
}

func (m MockCache) SetReservationsByUserID(_ context.Context, userID string, reservations []hotelsDAO.Reservation) {
	m.storeReservationsList(m.userLists, userID, reservations)
}

func (m MockCache) SetReservationsByUserAndHotelID(_ context.Context, hotelID, userID string, reservations []hotelsDAO.Reservation) {
	m.storeReservationsList(m.userHotelLists, hotelID+":"+userID, reservations)
}

// La cache NO crea reservas: almacena la copia individual e invalida las
// listas agregadas (misma semántica que la real — RV1/RV2)
func (m MockCache) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	m.reservas[reservation.ID] = reservation
	m.invalidateReservationLists(reservation)
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

func (m MockCache) CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	// La cache real no devuelve error si no existe
	reservation, ok := m.reservas[id]
	if !ok {
		return hotelsDAO.Reservation{}, nil
	}

	// Igual que la cache real: marca cancelled la copia individual e
	// invalida las listas agregadas
	now := time.Now().UTC()
	reservation.Status = hotelsDAO.StatusCancelled
	reservation.CancelledAt = &now
	m.reservas[id] = reservation
	m.invalidateReservationLists(reservation)

	return reservation, nil
}

// Los getters de listas solo aciertan si la lista agregada fue seteada
// completa — como la caché real: "la key existe" implica "lista completa"
func (m MockCache) GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	list, ok := m.hotelLists[hotelID]
	if !ok {
		return nil, fmt.Errorf("not found item with key reservations:hotel:%s", hotelID)
	}
	return paginateReservations(list, limit, offset), nil
}

func (m MockCache) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	list, ok := m.userHotelLists[hotelID+":"+userID]
	if !ok {
		return nil, fmt.Errorf("not found item with key reservations:hotel:%s:user:%s", hotelID, userID)
	}
	return paginateReservations(list, limit, offset), nil
}

func (m MockCache) GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	list, ok := m.userLists[userID]
	if !ok {
		return nil, fmt.Errorf("not found item with key reservations:user:%s", userID)
	}
	return paginateReservations(list, limit, offset), nil
}

func (m MockCache) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range hotelIDs {
		if _, ok := m.hotels[id]; !ok {
			return nil, fmt.Errorf("hotel with ID %s not found or expired in cache", id)
		}

		available, err := m.IsHotelAvailable(ctx, id, checkIn, checkOut)
		if err != nil {
			return nil, fmt.Errorf("error checking availability for hotel %s: %w", id, err)
		}
		result[id] = available
	}
	return result, nil
}

// IsHotelAvailable replica la lógica de la caché real: cuenta SOLO desde la
// lista agregada del hotel si está seteada (lista ausente = cero reservas
// conocidas = disponible, D3); excluye el día de checkout, saltea canceladas
// y cuenta num_rooms (D4).
func (m MockCache) IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error) {
	hotel, ok := m.hotels[hotelID]
	if !ok {
		return false, fmt.Errorf("error getting hotel from cache: not found item with key hotel:%s", hotelID)
	}

	checkInTime, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return false, fmt.Errorf("error parsing check-in date: %w", err)
	}
	checkOutTime, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return false, fmt.Errorf("error parsing check-out date: %w", err)
	}

	checkInTime = normalizeDate(checkInTime)
	checkOutTime = normalizeDate(checkOutTime)

	if !checkOutTime.After(checkInTime) {
		return false, fmt.Errorf("check-out date must be after check-in date")
	}

	list, ok := m.hotelLists[hotelID]
	if !ok {
		// Lista ausente = disponible (D3), igual que la caché real
		return true, nil
	}

	roomsByDay := countRoomsByNight(list, hotelID, checkInTime, checkOutTime)
	for date := checkInTime; date.Before(checkOutTime); date = date.AddDate(0, 0, 1) {
		if roomsByDay[date] >= hotel.AvaiableRooms {
			return false, nil
		}
	}

	return true, nil
}

// Elimina todas las reservas de un hotel del mock cache — espejo del real:
// borra las copias individuales que la lista del hotel conozca e invalida las
// listas de los usuarios involucrados.
func (m MockCache) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	if list, ok := m.hotelLists[hotelID]; ok {
		for _, r := range list {
			delete(m.reservas, r.ID)
			delete(m.userLists, r.UserID)
			delete(m.userHotelLists, hotelID+":"+r.UserID)
		}
	}
	delete(m.hotelLists, hotelID)
	return nil
}
