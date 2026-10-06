//go:build integration

package hotels

import (
	"context"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
)

// availabilityRepo exercises civil-date overlap against the real Mongo replica set.
type availabilityRepo interface {
	CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error)
	CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error)
	IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error)
}

func mustParseDay(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("invalid date in test: %v", err)
	}
	return parsed
}

// runAvailabilitySuite corre los casos de solapamiento contra una
// implementación con un hotel de capacidad 1 y una reserva 2030-06-10 →
// 2030-06-13 ya creada (confirmed).
func runAvailabilitySuite(t *testing.T, repo availabilityRepo, hotelID string) {
	t.Helper()
	ctx := context.Background()

	if _, err := repo.CreateReservation(ctx, hotelsDAO.Reservation{
		HotelName: "Suite Hotel",
		HotelID:   hotelID,
		UserID:    "suite-user",
		CheckIn:   mustParseDay(t, "2030-06-10"),
		CheckOut:  mustParseDay(t, "2030-06-13"),
		Status:    hotelsDAO.StatusConfirmed,
		NumRooms:  1,
		NumGuests: 1,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("creating base reservation: %v", err)
	}

	cases := []struct {
		name      string
		checkIn   string
		checkOut  string
		available bool
	}{
		{"solapado total (mismo rango)", "2030-06-10", "2030-06-13", false},
		{"solapado parcial por el inicio", "2030-06-08", "2030-06-11", false},
		{"solapado parcial por el final", "2030-06-12", "2030-06-15", false},
		{"contenido dentro del rango", "2030-06-11", "2030-06-12", false},
		{"borde: check-in el día del checkout existente", "2030-06-13", "2030-06-15", true},
		{"borde: checkout el día del check-in existente", "2030-06-08", "2030-06-10", true},
		{"rango disjunto anterior", "2030-06-01", "2030-06-05", true},
		{"rango disjunto posterior", "2030-06-20", "2030-06-25", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			available, err := repo.IsHotelAvailable(ctx, hotelID, tc.checkIn, tc.checkOut)
			if err != nil {
				t.Fatalf("IsHotelAvailable(%s, %s): %v", tc.checkIn, tc.checkOut, err)
			}
			if available != tc.available {
				t.Errorf("IsHotelAvailable(%s, %s) = %v, want %v", tc.checkIn, tc.checkOut, available, tc.available)
			}
		})
	}

	t.Run("check-in igual a checkout es error", func(t *testing.T) {
		if _, err := repo.IsHotelAvailable(ctx, hotelID, "2030-06-10", "2030-06-10"); err == nil {
			t.Error("expected error when check-in equals check-out")
		}
	})

	t.Run("una reserva cancelada no ocupa", func(t *testing.T) {
		id, err := repo.CreateReservation(ctx, hotelsDAO.Reservation{
			HotelName: "Suite Hotel",
			HotelID:   hotelID,
			UserID:    "suite-user-2",
			CheckIn:   mustParseDay(t, "2030-07-01"),
			CheckOut:  mustParseDay(t, "2030-07-03"),
			Status:    hotelsDAO.StatusConfirmed,
			NumRooms:  1,
			NumGuests: 1,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("creating reservation to cancel: %v", err)
		}
		cancelled, err := repo.CancelReservation(ctx, id)
		if err != nil {
			t.Fatalf("cancelling reservation: %v", err)
		}
		if cancelled.Status != hotelsDAO.StatusCancelled {
			t.Fatalf("expected cancelled status, got %q", cancelled.Status)
		}

		available, err := repo.IsHotelAvailable(ctx, hotelID, "2030-07-01", "2030-07-03")
		if err != nil {
			t.Fatalf("IsHotelAvailable after cancel: %v", err)
		}
		if !available {
			t.Error("cancelled reservation must not occupy rooms")
		}
	})
}
