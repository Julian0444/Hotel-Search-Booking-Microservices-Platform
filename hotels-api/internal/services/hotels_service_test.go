package services

import (
	"context"
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"
)

// Mock de la cola (no hace nada)
type MockQueue struct{}

func (mq MockQueue) Publish(hotelNew hotelsDomain.HotelNew) error { return nil }

// Helper para crear el service con mocks separados
func getTestService() Service {
	mainRepo := hotels.NewMock()       // Repositorio principal
	cacheRepo := hotels.NewMockCache() // Cache
	return NewService(mainRepo, cacheRepo, MockQueue{})
}

func TestCreateAndGetHotel(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{
		Name: "Test Hotel",
		City: "Test City",
	}
	id, err := service.Create(ctx, hotel)
	if err != nil {
		t.Fatalf("error creating hotel: %v", err)
	}
	got, err := service.GetHotelByID(ctx, id)
	if err != nil {
		t.Fatalf("error getting hotel: %v", err)
	}
	if got.Name != hotel.Name {
		t.Errorf("expected name %s, got %s", hotel.Name, got.Name)
	}
}

func TestUpdateHotel(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "Old Name"}
	id, _ := service.Create(ctx, hotel)
	updated := hotelsDomain.Hotel{ID: id, Name: "New Name"}
	err := service.Update(ctx, updated)
	if err != nil {
		t.Fatalf("error updating hotel: %v", err)
	}
	got, _ := service.GetHotelByID(ctx, id)
	if got.Name != "New Name" {
		t.Errorf("expected updated name, got %s", got.Name)
	}
}

func TestDeleteHotel(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "ToDelete"}
	id, _ := service.Create(ctx, hotel)

	// Verificar que existe antes de eliminar
	_, err := service.GetHotelByID(ctx, id)
	if err != nil {
		t.Fatalf("hotel not found before deletion: %v", err)
	}

	err = service.Delete(ctx, id)
	if err != nil {
		t.Fatalf("error deleting hotel: %v", err)
	}
	_, err = service.GetHotelByID(ctx, id)
	if err == nil {
		t.Error("expected error for deleted hotel, got nil")
	}
}

func TestCreateReservation(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelRes"}
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID: hotelID,
		UserID:  "user1",
	}
	resID, err := service.CreateReservation(ctx, res)
	if err != nil {
		t.Fatalf("error creating reservation: %v", err)
	}
	// Verifica que la reserva existe
	resList, err := service.GetReservationsByHotelID(ctx, hotelID)
	if err != nil || len(resList) == 0 {
		t.Fatalf("reservation not found after creation")
	}
	found := false
	for _, r := range resList {
		if r.ID == resID {
			found = true
		}
	}
	if !found {
		t.Errorf("created reservation not found in list")
	}
}

func TestCancelReservation(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelResCancel"}
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID: hotelID,
		UserID:  "user2",
	}
	resID, _ := service.CreateReservation(ctx, res)
	// Ahora cancela la reserva
	err := service.CancelReservation(ctx, resID)
	if err != nil {
		t.Fatalf("error canceling reservation: %v", err)
	}
	// Verifica que la reserva ya no existe
	resList, _ := service.GetReservationsByHotelID(ctx, hotelID)
	for _, r := range resList {
		if r.ID == resID {
			t.Errorf("reservation was not canceled")
		}
	}
}

func TestGetReservationsByHotelID(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelRes2"}
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID: hotelID,
		UserID:  "user2",
	}
	service.CreateReservation(ctx, res)
	resList, err := service.GetReservationsByHotelID(ctx, hotelID)
	if err != nil {
		t.Fatalf("error getting reservations: %v", err)
	}
	if len(resList) != 1 {
		t.Errorf("expected 1 reservation, got %d", len(resList))
	}
}

func TestGetReservationsByUserID(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelRes3"}
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID: hotelID,
		UserID:  "user3",
	}
	service.CreateReservation(ctx, res)
	resList, err := service.GetReservationsByUserID(ctx, "user3")
	if err != nil {
		t.Fatalf("error getting reservations: %v", err)
	}
	if len(resList) != 1 {
		t.Errorf("expected 1 reservation, got %d", len(resList))
	}
}

func TestGetReservationsByUserAndHotelID(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelRes4"}
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID: hotelID,
		UserID:  "user4",
	}
	service.CreateReservation(ctx, res)
	resList, err := service.GetReservationsByUserAndHotelID(ctx, hotelID, "user4")
	if err != nil {
		t.Fatalf("error getting reservations: %v", err)
	}
	if len(resList) != 1 {
		t.Errorf("expected 1 reservation, got %d", len(resList))
	}
}

func TestGetAvailability(t *testing.T) {
	service := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelAvail"}
	hotelID, _ := service.Create(ctx, hotel)
	availability, err := service.GetAvailability(ctx, []string{hotelID}, "2024-01-01", "2024-01-02")
	if err != nil {
		t.Fatalf("error getting availability: %v", err)
	}
	if !availability[hotelID] {
		t.Errorf("expected hotel to be available")
	}
}
