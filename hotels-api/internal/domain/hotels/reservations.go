package hotels

import (
	"errors"
	"time"
)

// ErrNoAvailability es el sentinel del no-overbooking (D1): el repositorio lo
// devuelve cuando alguna noche del rango no tiene cupo y el controller lo
// mapea a 409 Conflict.
var ErrNoAvailability = errors.New("no availability for the requested dates")

// ErrInvalidReservation tipifica los errores de validación de una reserva
// (fechas mal formadas, check-in en el pasado, cantidades inválidas): el
// controller los mapea a 400 — antes cualquier validación del service salía
// como 500 (RV19).
var ErrInvalidReservation = errors.New("invalid reservation")

// ErrReservationNotFound tipifica "la reserva no existe" (mismo patrón que
// ErrHotelNotFound/RV14): el controller lo mapea a 404 y cualquier otro fallo
// del repositorio queda como 500 real.
var ErrReservationNotFound = errors.New("reservation not found")

// DateFormat es el formato canónico de las fechas de reserva en el contrato
// HTTP, tanto en requests como en respuestas (RV20): date-only, sin hora ni
// zona — serializar RFC3339-UTC hacía que el frontend corriera las fechas un
// día en timezones al oeste de UTC.
const DateFormat = "2006-01-02"

// CheckIn/CheckOut viajan como string "YYYY-MM-DD" (ver DateFormat); los
// timestamps de auditoría (created_at/cancelled_at) siguen siendo RFC3339.
type Reservation struct {
	ID          string     `json:"id"`
	HotelID     string     `json:"hotel_id"`
	HotelName   string     `json:"hotel_name"`
	UserID      string     `json:"user_id"`
	CheckIn     string     `json:"check_in"`
	CheckOut    string     `json:"check_out"`
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
