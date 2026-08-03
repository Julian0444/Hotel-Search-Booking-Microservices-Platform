package hotels

import (
	"context"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"

	"github.com/google/uuid"
)

// availabilityRepo es el subconjunto de Repository que ejercita la suite de
// solapamiento de fechas (C10). La corren la caché (acá) y Mongo (en la suite
// `integration`) para garantizar la MISMA semántica en ambas (D4): por noche,
// checkout excluido, canceladas salteadas.
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

// cacheSuiteHarness adapta la caché real a la suite compartida: con la
// semántica F1 las escrituras INVALIDAN las listas agregadas (nunca las
// editan), así que el harness re-publica la lista completa del hotel tras
// cada escritura — exactamente lo que hace el service al repoblar desde
// Mongo. Así IsHotelAvailable cuenta desde una lista presente y la suite
// ejercita el conteo real (incluido el salteo de canceladas).
type cacheSuiteHarness struct {
	cache        Cache
	hotelID      string
	reservations []hotelsDAO.Reservation
}

func (h *cacheSuiteHarness) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	if reservation.ID == "" {
		reservation.ID = uuid.New().String() // Mongo asigna el ID en producción
	}
	if _, err := h.cache.CreateReservation(ctx, reservation); err != nil {
		return "", err
	}
	h.reservations = append(h.reservations, reservation)
	h.cache.SetReservationsByHotelID(ctx, h.hotelID, h.reservations)
	return reservation.ID, nil
}

func (h *cacheSuiteHarness) CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	cancelled, err := h.cache.CancelReservation(ctx, id)
	if err != nil {
		return hotelsDAO.Reservation{}, err
	}
	for i := range h.reservations {
		if h.reservations[i].ID == id {
			h.reservations[i] = cancelled
		}
	}
	h.cache.SetReservationsByHotelID(ctx, h.hotelID, h.reservations)
	return cancelled, nil
}

func (h *cacheSuiteHarness) IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error) {
	return h.cache.IsHotelAvailable(ctx, hotelID, checkIn, checkOut)
}

// La suite contra la implementación de caché real (ccache), con el hotel y las
// reservas inyectados vía el harness.
func TestCacheAvailabilitySuite(t *testing.T) {
	cache := NewCache(CacheConfig{MaxSize: 1000, ItemsToPrune: 10, Duration: time.Minute})
	ctx := context.Background()

	hotelID := "suite-hotel-1"
	if _, err := cache.Create(ctx, hotelsDAO.Hotel{ID: hotelID, Name: "Suite Hotel", AvailableRooms: 1}); err != nil {
		t.Fatalf("creating hotel in cache: %v", err)
	}

	runAvailabilitySuite(t, &cacheSuiteHarness{cache: cache, hotelID: hotelID}, hotelID)
}

// D3: lista de reservas ausente en caché = cero reservas = disponible
func TestCacheAvailability_NoReservationsListIsAvailable(t *testing.T) {
	cache := NewCache(CacheConfig{MaxSize: 1000, ItemsToPrune: 10, Duration: time.Minute})
	ctx := context.Background()

	hotelID := "fresh-hotel"
	if _, err := cache.Create(ctx, hotelsDAO.Hotel{ID: hotelID, Name: "Fresh", AvailableRooms: 2}); err != nil {
		t.Fatalf("creating hotel in cache: %v", err)
	}

	available, err := cache.IsHotelAvailable(ctx, hotelID, "2030-06-10", "2030-06-12")
	if err != nil {
		t.Fatalf("IsHotelAvailable: %v", err)
	}
	if !available {
		t.Error("hotel with no cached reservations list must be available (D3)")
	}
}
