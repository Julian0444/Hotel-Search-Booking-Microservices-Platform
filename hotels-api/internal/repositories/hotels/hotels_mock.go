package hotels

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/google/uuid"
)

// countRoomsByNight suma las habitaciones ocupadas por noche para un hotel
// (checkout excluido, canceladas salteadas) — la misma semántica que el
// inventario de Mongo (D4).
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

// Constructor del mock principal
func NewMock() Mock {
	return Mock{
		hotels:   make(map[string]hotelsDAO.Hotel),
		reservas: make(map[string]hotelsDAO.Reservation),
	}
}

// ===== MOCK PRINCIPAL (comportamiento normal) =====

// allReservations devuelve las reservas del mock como slice (countRoomsByNight
// opera sobre slices).
func (m Mock) allReservations() []hotelsDAO.Reservation {
	result := make([]hotelsDAO.Reservation, 0, len(m.reservas))
	for _, r := range m.reservas {
		result = append(result, r)
	}
	return result
}

// CRUD de hoteles. GetHotelByID devuelve el sentinel tipado como el repo real
// (RV14) para que los tests de service/controller ejerciten el mapeo a 404.
func (m Mock) GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error) {
	hotel, ok := m.hotels[id]
	if !ok {
		return hotelsDAO.Hotel{}, fmt.Errorf("hotel with ID %s: %w", id, hotelsDomain.ErrHotelNotFound)
	}
	return hotel, nil
}

// GetHotels pagina como el repo real: orden estable (por ID) + limit/offset.
func (m Mock) GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDAO.Hotel, error) {
	ids := make([]string, 0, len(m.hotels))
	for id := range m.hotels {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	if offset >= int64(len(ids)) {
		return []hotelsDAO.Hotel{}, nil
	}
	end := offset + limit
	if end > int64(len(ids)) {
		end = int64(len(ids))
	}
	page := make([]hotelsDAO.Hotel, 0, end-offset)
	for _, id := range ids[offset:end] {
		page = append(page, m.hotels[id])
	}
	return page, nil
}

func (m Mock) CountHotels(ctx context.Context) (int64, error) {
	return int64(len(m.hotels)), nil
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
	for _, rooms := range countRoomsByNight(m.allReservations(), hotel.ID, time.Time{}, time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)) {
		if rooms > hotel.AvailableRooms {
			return hotelsDomain.ErrCapacityConflict
		}
	}
	m.hotels[hotel.ID] = hotel
	return nil
}

func (m Mock) Delete(ctx context.Context, id string) error {
	_, ok := m.hotels[id]
	if !ok {
		return fmt.Errorf("hotel with ID %s not found", id)
	}
	for _, r := range m.reservas {
		if r.HotelID == id {
			return hotelsDomain.ErrHotelHasReservations
		}
	}
	delete(m.hotels, id)
	return nil
}

// CRUD de reservas. CreateReservation replica el no-overbooking del repo real
// (D1): si alguna noche del rango no tiene cupo devuelve ErrNoAvailability.
func (m Mock) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	hotel, ok := m.hotels[reservation.HotelID]
	if !ok {
		return "", hotelsDomain.ErrHotelNotFound
	}

	if reservation.CheckIn.Before(normalizeDate(time.Now().UTC())) {
		return "", hotelsDomain.ErrInvalidReservation
	}
	reservation.Status = hotelsDAO.StatusConfirmed
	reservation.HotelName = hotel.Name
	reservation.Currency = "USD"
	reservation.TotalPrice = int64(math.Round(hotel.PricePerNight*100)) * int64(len(nightsBetween(reservation.CheckIn, reservation.CheckOut))) * int64(reservation.NumRooms)
	rooms := reservation.NumRooms
	if rooms < 1 {
		rooms = 1
	}
	if rooms > hotel.AvailableRooms {
		return "", hotelsDomain.ErrNoAvailability
	}

	from := normalizeDate(reservation.CheckIn)
	to := normalizeDate(reservation.CheckOut)
	roomsByDay := countRoomsByNight(m.allReservations(), reservation.HotelID, from, to)
	for date := from; date.Before(to); date = date.AddDate(0, 0, 1) {
		if roomsByDay[date]+rooms > hotel.AvailableRooms {
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
		if roomsByDay[date] >= hotel.AvailableRooms {
			return false, nil
		}
	}

	return true, nil
}

func paginateReservations(rows []hotelsDAO.Reservation, limit, offset int64) []hotelsDAO.Reservation {
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	if offset >= int64(len(rows)) {
		return []hotelsDAO.Reservation{}
	}
	end := offset + limit
	if end > int64(len(rows)) {
		end = int64(len(rows))
	}
	return rows[offset:end]
}
func (m Mock) GetHotelsAfter(ctx context.Context, afterID string, limit int64) ([]hotelsDAO.Hotel, error) {
	rows, _ := m.GetHotels(ctx, int64(len(m.hotels)), 0)
	result := []hotelsDAO.Hotel{}
	for _, h := range rows {
		if h.ID > afterID && int64(len(result)) < limit {
			result = append(result, h)
		}
	}
	return result, nil
}
