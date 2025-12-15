package services

import (
	"context"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"
)

// Mock de la cola (no hace nada)
type MockQueue struct{}

func (mq MockQueue) Publish(hotelNew hotelsDomain.HotelNew) error { return nil }

// Helper para crear el service con mocks reutilizables
func getTestService() (Service, hotels.Mock, hotels.MockCache) {
	mainRepo := hotels.NewMock()       // Repositorio principal
	cacheRepo := hotels.NewMockCache() // Cache
	return NewService(mainRepo, cacheRepo, MockQueue{}), mainRepo, cacheRepo
}

func TestCreateAndGetHotel(t *testing.T) {
	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
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

	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
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
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{
		Name:          "HotelAvail",
		AvaiableRooms: 1, // necesario para que IsHotelAvailable devuelva true
	}
	hotelID, _ := service.Create(ctx, hotel)
	availability, err := service.GetAvailability(ctx, []string{hotelID}, "2024-01-01", "2024-01-02")
	if err != nil {
		t.Fatalf("error getting availability: %v", err)
	}
	if !availability[hotelID] {
		t.Errorf("expected hotel to be available")
	}
}

// Cache-miss: obtiene de main y luego queda en cache
func TestGetHotelByID_PopulatesCache(t *testing.T) {
	// Crear repos separados para inyectarlos y reusarlos
	mainRepo := hotels.NewMock()
	cacheRepo := hotels.NewMockCache()
	service := NewService(mainRepo, cacheRepo, MockQueue{})
	ctx := context.Background()

	// Crear hotel solo en el repo principal (no en cache)
	hotelID, err := mainRepo.Create(ctx, hotelsDAO.Hotel{Name: "Cacheable Hotel"})
	if err != nil {
		t.Fatalf("error creating hotel in main repo: %v", err)
	}

	// Primer acceso: debería leer de main y poblar cache
	got, err := service.GetHotelByID(ctx, hotelID)
	if err != nil {
		t.Fatalf("error getting hotel: %v", err)
	}
	if got.ID != hotelID {
		t.Fatalf("expected hotel ID %s, got %s", hotelID, got.ID)
	}

	// Segundo acceso: debe estar en cache
	_, err = cacheRepo.GetHotelByID(ctx, hotelID)
	if err != nil {
		t.Fatalf("expected hotel to be cached, got error: %v", err)
	}
}

// Cache-miss en reserva: obtiene de main y luego queda en cache
func TestGetReservationByID_PopulatesCache(t *testing.T) {
	mainRepo := hotels.NewMock()
	cacheRepo := hotels.NewMockCache()
	service := NewService(mainRepo, cacheRepo, MockQueue{})
	ctx := context.Background()

	// Crear hotel en main para asociar reserva
	hotelID, err := mainRepo.Create(ctx, hotelsDAO.Hotel{Name: "HotelForReservation"})
	if err != nil {
		t.Fatalf("error creating hotel in main repo: %v", err)
	}
	// Crear reserva solo en main
	resID, err := mainRepo.CreateReservation(ctx, hotelsDAO.Reservation{
		HotelID: hotelID,
		UserID:  "user-cache",
	})
	if err != nil {
		t.Fatalf("error creating reservation in main repo: %v", err)
	}

	// Primer acceso: debería leer de main y poblar cache
	got, err := service.GetReservationByID(ctx, resID)
	if err != nil {
		t.Fatalf("error getting reservation: %v", err)
	}
	if got.ID != resID {
		t.Fatalf("expected reservation ID %s, got %s", resID, got.ID)
	}

	// Segundo acceso: debe estar en cache
	_, err = cacheRepo.GetReservationByID(ctx, resID)
	if err != nil {
		t.Fatalf("expected reservation to be cached, got error: %v", err)
	}
}

// Disponibilidad con reservas que ocupan una noche (checkout excluido)
func TestAvailabilityWithReservation(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotelID, _ := service.Create(ctx, hotelsDomain.Hotel{
		Name:          "HotelOcc",
		AvaiableRooms: 1,
	})

	// Reserva 2024-01-01 a 2024-01-02 ocupa la noche del 1
	_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user-occ",
		CheckIn:  parseDate(t, "2024-01-01"),
		CheckOut: parseDate(t, "2024-01-02"),
	})
	if err != nil {
		t.Fatalf("error creating reservation: %v", err)
	}

	// Mismo rango debe estar no disponible
	availability, err := service.GetAvailability(ctx, []string{hotelID}, "2024-01-01", "2024-01-02")
	if err != nil {
		t.Fatalf("error getting availability: %v", err)
	}
	if availability[hotelID] {
		t.Errorf("expected hotel to be unavailable for occupied night")
	}

	// Checkout excluido: el día 2024-01-02 ya libre para check-in
	availability, err = service.GetAvailability(ctx, []string{hotelID}, "2024-01-02", "2024-01-03")
	if err != nil {
		t.Fatalf("error getting availability (checkout exclusion): %v", err)
	}
	if !availability[hotelID] {
		t.Errorf("expected hotel to be available after checkout")
	}
}

// parseDate helper para tests de fechas
func parseDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("invalid date in test: %v", err)
	}
	return parsed
}
