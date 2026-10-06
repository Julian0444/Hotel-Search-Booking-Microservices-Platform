package hotels

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var ErrInvalidHotel = errors.New("invalid hotel")
var ErrCapacityConflict = errors.New("capacity cannot be lower than existing booked rooms")
var ErrHotelHasReservations = errors.New("hotel with reservation history cannot be deleted")

// POST and PUT use the same complete representation. Zero capacity means
// temporarily closed to new reservations; zero price/rating are valid.
func ValidateHotel(h Hotel) error {
	for _, field := range []struct{ name, value string }{{"name", h.Name}, {"address", h.Address}, {"city", h.City}, {"country", h.Country}} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required: %w", field.name, ErrInvalidHotel)
		}
	}
	if h.AvailableRooms < 0 || h.AvailableRooms > 10000 {
		return fmt.Errorf("available_rooms must be between 0 and 10000: %w", ErrInvalidHotel)
	}
	if math.IsNaN(h.PricePerNight) || math.IsInf(h.PricePerNight, 0) || h.PricePerNight < 0 || h.PricePerNight > 1000000 {
		return fmt.Errorf("price_per_night must be between 0 and 1000000: %w", ErrInvalidHotel)
	}
	if math.IsNaN(h.Rating) || math.IsInf(h.Rating, 0) || h.Rating < 0 || h.Rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5: %w", ErrInvalidHotel)
	}
	for _, value := range []string{h.CheckInTime, h.CheckOutTime} {
		if len(value) != 5 {
			return fmt.Errorf("check_in_time and check_out_time require HH:mm: %w", ErrInvalidHotel)
		}
		if _, err := time.Parse("15:04", value); err != nil {
			return fmt.Errorf("check_in_time and check_out_time require HH:mm: %w", ErrInvalidHotel)
		}
	}
	return nil
}
