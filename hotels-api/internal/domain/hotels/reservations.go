package hotels

import (
	"errors"
	"time"
)

// ErrNoAvailability es el sentinel del no-overbooking (D1): el repositorio lo
// devuelve cuando alguna noche del rango no tiene cupo y el controller lo
// mapea a 409 Conflict.
var ErrNoAvailability = errors.New("no availability for the requested dates")

type Reservation struct {
	ID          string     `json:"id"`
	HotelID     string     `json:"hotel_id"`
	HotelName   string     `json:"hotel_name"`
	UserID      string     `json:"user_id"`
	CheckIn     time.Time  `json:"check_in"`
	CheckOut    time.Time  `json:"check_out"`
	Status      string     `json:"status"`
	NumRooms    int        `json:"num_rooms"`
	NumGuests   int        `json:"num_guests"`
	TotalPrice  int64      `json:"total_price"` // centavos
	Currency    string     `json:"currency"`
	CreatedAt   time.Time  `json:"created_at"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
}

// ReservationNew se publica en la cola `reservations-news` (distinta de
// `hotels-news`: search-api espera HotelNew ahí y un ReservationNew rompería
// su Unmarshal). Hoy nadie la consume; queda disponible como stretch.
type ReservationNew struct {
	Operation     string `json:"operation"` // CREATE | CANCEL
	ReservationID string `json:"reservation_id"`
	HotelID       string `json:"hotel_id"`
}
