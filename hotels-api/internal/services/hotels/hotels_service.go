package hotels

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

type Repository interface {
	GetHotelByID(context.Context, string) (hotelsDAO.Hotel, error)
	GetHotels(context.Context, int64, int64) ([]hotelsDAO.Hotel, error)
	GetHotelsAfter(context.Context, string, int64) ([]hotelsDAO.Hotel, error)
	CountHotels(context.Context) (int64, error)
	Create(context.Context, hotelsDAO.Hotel) (string, error)
	Update(context.Context, hotelsDAO.Hotel) error
	Delete(context.Context, string) error
	CreateReservation(context.Context, hotelsDAO.Reservation) (string, error)
	GetReservationByID(context.Context, string) (hotelsDAO.Reservation, error)
	CancelReservation(context.Context, string) (hotelsDAO.Reservation, error)
	GetReservationsByHotelID(context.Context, string, int64, int64) ([]hotelsDAO.Reservation, error)
	GetReservationsByUserAndHotelID(context.Context, string, string, int64, int64) ([]hotelsDAO.Reservation, error)
	GetReservationsByUserID(context.Context, string, int64, int64) ([]hotelsDAO.Reservation, error)
	GetAvailability(context.Context, []string, string, string) (map[string]bool, error)
}
type Queue interface {
	Publish(context.Context, hotelsDomain.HotelNew) error
}
type Service struct {
	mainRepository Repository
	eventsQueue    Queue
}

func NewService(repository Repository, queue Queue) Service { return Service{repository, queue} }

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

func hotelToDAO(h hotelsDomain.Hotel) hotelsDAO.Hotel {
	return hotelsDAO.Hotel{ID: h.ID, Name: h.Name, Description: h.Description, Address: h.Address, City: h.City, State: h.State, Country: h.Country, Phone: h.Phone, Email: h.Email, PricePerNight: h.PricePerNight, Rating: h.Rating, AvailableRooms: h.AvailableRooms, CheckInTime: h.CheckInTime, CheckOutTime: h.CheckOutTime, Amenities: h.Amenities, Images: h.Images}
}
func (s Service) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	h, err := s.mainRepository.GetHotelByID(ctx, id)
	return hotelToDomain(h), err
}
func hotelsToDomain(rows []hotelsDAO.Hotel) []hotelsDomain.Hotel {
	result := make([]hotelsDomain.Hotel, 0, len(rows))
	for _, row := range rows {
		result = append(result, hotelToDomain(row))
	}
	return result
}
func (s Service) GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDomain.Hotel, int64, error) {
	rows, err := s.mainRepository.GetHotels(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.mainRepository.CountHotels(ctx)
	return hotelsToDomain(rows), total, err
}
func (s Service) GetHotelsAfter(ctx context.Context, afterID string, limit int64) ([]hotelsDomain.Hotel, error) {
	rows, err := s.mainRepository.GetHotelsAfter(ctx, afterID, limit)
	return hotelsToDomain(rows), err
}

// Mongo success defines HTTP success. Broker failures are bounded, logged and
// recovered by periodic reconciliation; publishing is not atomic with Mongo.
func (s Service) publish(ctx context.Context, operation, id string) {
	if err := s.eventsQueue.Publish(ctx, hotelsDomain.HotelNew{Operation: operation, HotelID: strings.ToLower(id)}); err != nil {
		slog.Warn("hotel persisted; search update pending reconciliation", "hotel_id", id, "operation", operation, "error", err)
	}
}
func (s Service) Create(ctx context.Context, hotel hotelsDomain.Hotel) (string, error) {
	if err := hotelsDomain.ValidateHotel(hotel); err != nil {
		return "", err
	}
	id, err := s.mainRepository.Create(ctx, hotelToDAO(hotel))
	if err != nil {
		return "", err
	}
	s.publish(ctx, "CREATE", id)
	return id, nil
}
func (s Service) Update(ctx context.Context, hotel hotelsDomain.Hotel) error {
	if err := hotelsDomain.ValidateHotel(hotel); err != nil {
		return err
	}
	if err := s.mainRepository.Update(ctx, hotelToDAO(hotel)); err != nil {
		return err
	}
	s.publish(ctx, "UPDATE", hotel.ID)
	return nil
}
func (s Service) Delete(ctx context.Context, id string) error {
	if err := s.mainRepository.Delete(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, "DELETE", id)
	return nil
}

// The repository rereads hotel price/capacity under the transaction lock.
// Dates are civil UTC, checkout excluded; requests may reserve at most a year.
func (s Service) CreateReservation(ctx context.Context, r hotelsDomain.Reservation) (string, error) {
	if r.NumRooms == 0 {
		r.NumRooms = 1
	}
	if r.NumGuests == 0 {
		r.NumGuests = 1
	}
	checkIn, err := time.Parse(hotelsDomain.DateFormat, r.CheckIn)
	if err != nil {
		return "", fmt.Errorf("check_in must use YYYY-MM-DD: %w", hotelsDomain.ErrInvalidReservation)
	}
	checkOut, err := time.Parse(hotelsDomain.DateFormat, r.CheckOut)
	if err != nil {
		return "", fmt.Errorf("check_out must use YYYY-MM-DD: %w", hotelsDomain.ErrInvalidReservation)
	}
	if !checkOut.After(checkIn) || checkOut.Sub(checkIn) > 366*24*time.Hour || r.NumRooms < 1 || r.NumGuests < 1 || r.UserID == "" {
		return "", hotelsDomain.ErrInvalidReservation
	}
	return s.mainRepository.CreateReservation(ctx, hotelsDAO.Reservation{HotelID: r.HotelID, UserID: r.UserID, CheckIn: checkIn, CheckOut: checkOut, NumRooms: r.NumRooms, NumGuests: r.NumGuests})
}
func (s Service) GetReservationByID(ctx context.Context, id string) (hotelsDomain.Reservation, error) {
	r, err := s.mainRepository.GetReservationByID(ctx, id)
	return reservationToDomain(r), err
}
func (s Service) CancelReservation(ctx context.Context, id string) error {
	_, err := s.mainRepository.CancelReservation(ctx, id)
	return err
}
func reservationsToDomain(rows []hotelsDAO.Reservation) []hotelsDomain.Reservation {
	result := make([]hotelsDomain.Reservation, 0, len(rows))
	for _, r := range rows {
		result = append(result, reservationToDomain(r))
	}
	return result
}
func (s Service) GetReservationsByHotelID(ctx context.Context, id string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	rows, err := s.mainRepository.GetReservationsByHotelID(ctx, id, limit, offset)
	return reservationsToDomain(rows), err
}
func (s Service) GetReservationsByUserID(ctx context.Context, id string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	rows, err := s.mainRepository.GetReservationsByUserID(ctx, id, limit, offset)
	return reservationsToDomain(rows), err
}
func (s Service) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	rows, err := s.mainRepository.GetReservationsByUserAndHotelID(ctx, hotelID, userID, limit, offset)
	return reservationsToDomain(rows), err
}
func (s Service) GetAvailability(ctx context.Context, ids []string, checkIn, checkOut string) (map[string]bool, error) {
	return s.mainRepository.GetAvailability(ctx, ids, checkIn, checkOut)
}
