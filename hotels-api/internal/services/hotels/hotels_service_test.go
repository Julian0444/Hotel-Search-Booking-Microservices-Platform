package hotels

import (
	"context"
	"errors"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"
)

type MockQueue struct {
	hotelEvents *[]hotelsDomain.HotelNew
	fail        bool
}

func NewMockQueue() MockQueue { return MockQueue{hotelEvents: &[]hotelsDomain.HotelNew{}} }
func (mq MockQueue) Publish(_ context.Context, event hotelsDomain.HotelNew) error {
	*mq.hotelEvents = append(*mq.hotelEvents, event)
	if mq.fail {
		return errors.New("broker down")
	}
	return nil
}
func getTestService() (Service, hotels.Mock, MockQueue) {
	main := hotels.NewMock()
	queue := NewMockQueue()
	return NewService(main, queue), main, queue
}
func validHotel(h hotelsDomain.Hotel) hotelsDomain.Hotel {
	if h.Address == "" {
		h.Address = "Main 1"
	}
	if h.City == "" {
		h.City = "Córdoba"
	}
	if h.Country == "" {
		h.Country = "Argentina"
	}
	h.CheckInTime = "15:00"
	h.CheckOutTime = "11:00"
	return h
}

// futureDate devuelve una fecha futura estable en el formato canónico
// "YYYY-MM-DD" del contrato (RV20); la validación de "check-in no pasado"
// exige fechas dinámicas.
func futureDate(t *testing.T, daysFromNow int) string {
	t.Helper()
	return time.Now().UTC().AddDate(0, 0, daysFromNow).Format(hotelsDomain.DateFormat)
}

func TestCreateAndGetHotel(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := validHotel(hotelsDomain.Hotel{
		Name: "Test Hotel",
		City: "Test City",
	})
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

// RV14: el sentinel ErrHotelNotFound sobrevive el wrap del service (errors.Is)
func TestGetHotelByID_NotFoundTyped(t *testing.T) {
	service, _, _ := getTestService()

	_, err := service.GetHotelByID(context.Background(), "missing")
	if !errors.Is(err, hotelsDomain.ErrHotelNotFound) {
		t.Fatalf("expected ErrHotelNotFound through the service wrap, got %v", err)
	}
}

// E3: listado paginado con total, sin pasar por caché
func TestGetHotels(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	for _, name := range []string{"Alfa", "Beta", "Gamma"} {
		if _, err := service.Create(ctx, validHotel(hotelsDomain.Hotel{Name: name})); err != nil {
			t.Fatalf("error creating hotel %s: %v", name, err)
		}
	}

	page, total, err := service.GetHotels(ctx, 2, 0)
	if err != nil {
		t.Fatalf("error getting hotels: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(page) != 2 {
		t.Errorf("expected page of 2, got %d", len(page))
	}

	rest, total, err := service.GetHotels(ctx, 2, 2)
	if err != nil {
		t.Fatalf("error getting second page: %v", err)
	}
	if total != 3 || len(rest) != 1 {
		t.Errorf("expected second page of 1 (total 3), got %d (total %d)", len(rest), total)
	}
	// Sin solapamiento entre páginas (orden estable por id)
	seen := map[string]bool{}
	for _, h := range append(page, rest...) {
		if seen[h.ID] {
			t.Errorf("hotel %s repeated across pages", h.ID)
		}
		seen[h.ID] = true
	}
}

func TestUpdateHotel(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := validHotel(hotelsDomain.Hotel{Name: "Old Name"})
	id, _ := service.Create(ctx, hotel)
	updated := validHotel(hotelsDomain.Hotel{ID: id, Name: "New Name"})
	err := service.Update(ctx, updated)
	if err != nil {
		t.Fatalf("error updating hotel: %v", err)
	}
	got, _ := service.GetHotelByID(ctx, id)
	if got.Name != "New Name" {
		t.Errorf("expected updated name, got %s", got.Name)
	}
}

// Actualizar persiste antes de publicar un evento.
func TestUpdateHotel_PersistsAndPublishes(t *testing.T) {
	mainRepo := hotels.NewMock()
	queue := NewMockQueue()
	service := NewService(mainRepo, queue)
	ctx := context.Background()

	// Hotel creado directamente en el repositorio.
	hotelID, err := mainRepo.Create(ctx, hotelsDAO.Hotel{Name: "Uncached"})
	if err != nil {
		t.Fatalf("error creating hotel in main repo: %v", err)
	}

	if err := service.Update(ctx, validHotel(hotelsDomain.Hotel{ID: hotelID, Name: "Renamed"})); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	events := *queue.hotelEvents
	if len(events) != 1 || events[0].Operation != "UPDATE" || events[0].HotelID != hotelID {
		t.Errorf("expected 1 UPDATE event for %s, got %+v", hotelID, events)
	}
}

func TestDeleteHotel(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := validHotel(hotelsDomain.Hotel{Name: "ToDelete"})
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

	hotel := validHotel(hotelsDomain.Hotel{Name: "HotelRes", AvailableRooms: 2, PricePerNight: 100})
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user1",
		CheckIn:  futureDate(t, 10),
		CheckOut: futureDate(t, 12),
		NumRooms: 1,
	}
	resID, err := service.CreateReservation(ctx, res)
	if err != nil {
		t.Fatalf("error creating reservation: %v", err)
	}
	// Verifica que la reserva existe
	resList, err := service.GetReservationsByHotelID(ctx, hotelID, 20, 0)
	if err != nil || len(resList) == 0 {
		t.Fatalf("reservation not found after creation")
	}
	found := false
	for _, r := range resList {
		if r.ID == resID {
			found = true
			// DM1/DM2: campos derivados por el service
			if r.Status != hotelsDAO.StatusConfirmed {
				t.Errorf("expected status confirmed, got %q", r.Status)
			}
			if r.HotelName != "HotelRes" {
				t.Errorf("hotel name must derive from the hotel, got %q", r.HotelName)
			}
			// 100 USD/noche × 2 noches × 1 habitación = 20000 centavos
			if r.TotalPrice != 20000 {
				t.Errorf("expected total price 20000 cents, got %d", r.TotalPrice)
			}
			if r.Currency != "USD" {
				t.Errorf("expected currency USD, got %q", r.Currency)
			}
		}
	}
	if !found {
		t.Errorf("created reservation not found in list")
	}
}

// D1: sin cupo devuelve ErrNoAvailability (el controller lo mapea a 409)
func TestCreateReservation_NoAvailability(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotelID, _ := service.Create(ctx, validHotel(hotelsDomain.Hotel{Name: "Full", AvailableRooms: 1, PricePerNight: 50}))

	res := hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user1",
		CheckIn:  futureDate(t, 10),
		CheckOut: futureDate(t, 12),
		NumRooms: 1,
	}
	if _, err := service.CreateReservation(ctx, res); err != nil {
		t.Fatalf("first reservation should succeed: %v", err)
	}

	res.UserID = "user2"
	_, err := service.CreateReservation(ctx, res)
	if !errors.Is(err, hotelsDomain.ErrNoAvailability) {
		t.Fatalf("expected ErrNoAvailability, got %v", err)
	}
}

func TestCreateReservation_Validations(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotelID, _ := service.Create(ctx, validHotel(hotelsDomain.Hotel{Name: "Valid", AvailableRooms: 2, PricePerNight: 50}))

	t.Run("checkout before checkin", func(t *testing.T) {
		_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
			HotelID: hotelID, UserID: "u", CheckIn: futureDate(t, 12), CheckOut: futureDate(t, 10), NumRooms: 1,
		})
		// RV19: validación tipada → el controller la mapea a 400
		if !errors.Is(err, hotelsDomain.ErrInvalidReservation) {
			t.Errorf("expected ErrInvalidReservation for check-out before check-in, got %v", err)
		}
	})

	t.Run("past check-in", func(t *testing.T) {
		_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
			HotelID: hotelID, UserID: "u", CheckIn: futureDate(t, -5), CheckOut: futureDate(t, 2), NumRooms: 1,
		})
		if !errors.Is(err, hotelsDomain.ErrInvalidReservation) {
			t.Errorf("expected ErrInvalidReservation for past check-in, got %v", err)
		}
	})

	t.Run("malformed dates", func(t *testing.T) {
		_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
			HotelID: hotelID, UserID: "u", CheckIn: "12/01/2030", CheckOut: futureDate(t, 12), NumRooms: 1,
		})
		if !errors.Is(err, hotelsDomain.ErrInvalidReservation) {
			t.Errorf("expected ErrInvalidReservation for malformed check-in, got %v", err)
		}
	})

	t.Run("rooms above capacity", func(t *testing.T) {
		_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
			HotelID: hotelID, UserID: "u", CheckIn: futureDate(t, 10), CheckOut: futureDate(t, 12), NumRooms: 3,
		})
		if !errors.Is(err, hotelsDomain.ErrNoAvailability) {
			t.Errorf("expected ErrNoAvailability for rooms above capacity, got %v", err)
		}
	})

	t.Run("unknown hotel", func(t *testing.T) {
		_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
			HotelID: "missing", UserID: "u", CheckIn: futureDate(t, 10), CheckOut: futureDate(t, 12), NumRooms: 1,
		})
		// RV19: el sentinel sobrevive el wrap → el controller mapea 404
		if !errors.Is(err, hotelsDomain.ErrHotelNotFound) {
			t.Errorf("expected ErrHotelNotFound for unknown hotel, got %v", err)
		}
	})
}

// DM1: cancelar es soft-delete (queda con status cancelled) e idempotente
func TestCancelReservation(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := validHotel(hotelsDomain.Hotel{Name: "HotelResCancel", AvailableRooms: 1, PricePerNight: 50})
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user2",
		CheckIn:  futureDate(t, 10),
		CheckOut: futureDate(t, 12),
		NumRooms: 1,
	}
	resID, _ := service.CreateReservation(ctx, res)

	// Ahora cancela la reserva
	err := service.CancelReservation(ctx, resID)
	if err != nil {
		t.Fatalf("error canceling reservation: %v", err)
	}

	// Soft-delete: la reserva sigue existiendo pero con status cancelled
	got, err := service.GetReservationByID(ctx, resID)
	if err != nil {
		t.Fatalf("cancelled reservation must still exist (soft delete): %v", err)
	}
	if got.Status != hotelsDAO.StatusCancelled {
		t.Errorf("expected status cancelled, got %q", got.Status)
	}
	if got.CancelledAt == nil {
		t.Error("expected cancelled_at to be set")
	}

	// Idempotente: un segundo cancel no falla
	if err := service.CancelReservation(ctx, resID); err != nil {
		t.Fatalf("second cancel must be idempotent: %v", err)
	}

	// Cancelar libera el cupo: el mismo rango vuelve a estar disponible
	if _, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
		HotelID: hotelID, UserID: "user3", CheckIn: res.CheckIn, CheckOut: res.CheckOut, NumRooms: 1,
	}); err != nil {
		t.Fatalf("cancelled reservation must free up the room: %v", err)
	}
}

func TestGetReservationsByHotelID(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := validHotel(hotelsDomain.Hotel{Name: "HotelRes2", AvailableRooms: 1, PricePerNight: 50})
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user2",
		CheckIn:  futureDate(t, 10),
		CheckOut: futureDate(t, 12),
		NumRooms: 1,
	}
	_, _ = service.CreateReservation(ctx, res)
	resList, err := service.GetReservationsByHotelID(ctx, hotelID, 20, 0)
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

	hotel := validHotel(hotelsDomain.Hotel{Name: "HotelRes3", AvailableRooms: 1, PricePerNight: 50})
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user3",
		CheckIn:  futureDate(t, 10),
		CheckOut: futureDate(t, 12),
		NumRooms: 1,
	}
	_, _ = service.CreateReservation(ctx, res)
	resList, err := service.GetReservationsByUserID(ctx, "user3", 20, 0)
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

	hotel := validHotel(hotelsDomain.Hotel{Name: "HotelRes4", AvailableRooms: 1, PricePerNight: 50})
	hotelID, _ := service.Create(ctx, hotel)
	res := hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user4",
		CheckIn:  futureDate(t, 10),
		CheckOut: futureDate(t, 12),
		NumRooms: 1,
	}
	_, _ = service.CreateReservation(ctx, res)
	resList, err := service.GetReservationsByUserAndHotelID(ctx, hotelID, "user4", 20, 0)
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

	hotel := validHotel(hotelsDomain.Hotel{
		Name:           "HotelAvail",
		AvailableRooms: 1, // necesario para que IsHotelAvailable devuelva true
	})
	hotelID, _ := service.Create(ctx, hotel)
	availability, err := service.GetAvailability(ctx, []string{hotelID}, "2024-01-01", "2024-01-02")
	if err != nil {
		t.Fatalf("error getting availability: %v", err)
	}
	if !availability[hotelID] {
		t.Errorf("expected hotel to be available")
	}
}

// Disponibilidad con reservas que ocupan una noche (checkout excluido)
func TestAvailabilityWithReservation(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotelID, _ := service.Create(ctx, validHotel(hotelsDomain.Hotel{
		Name:           "HotelOcc",
		AvailableRooms: 1,
		PricePerNight:  50,
	}))

	checkIn := futureDate(t, 10)
	checkOut := futureDate(t, 11)

	// La reserva ocupa la noche del check-in
	_, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
		HotelID:  hotelID,
		UserID:   "user-occ",
		CheckIn:  checkIn,
		CheckOut: checkOut,
		NumRooms: 1,
	})
	if err != nil {
		t.Fatalf("error creating reservation: %v", err)
	}

	if _, err := service.GetHotelByID(ctx, hotelID); err != nil {
		t.Fatal(err)
	}

	// Mismo rango debe estar no disponible (las fechas del contrato ya son
	// strings canónicos, van directo)
	availability, err := service.GetAvailability(ctx, []string{hotelID}, checkIn, checkOut)
	if err != nil {
		t.Fatalf("error getting availability: %v", err)
	}
	if availability[hotelID] {
		t.Errorf("expected hotel to be unavailable for occupied night")
	}

	// Checkout excluido: el día del checkout ya está libre para check-in
	availability, err = service.GetAvailability(ctx, []string{hotelID}, checkOut, futureDate(t, 12))
	if err != nil {
		t.Fatalf("error getting availability (checkout exclusion): %v", err)
	}
	if !availability[hotelID] {
		t.Errorf("expected hotel to be available after checkout")
	}
}

func TestInvalidHotelsNeverPersistOrPublish(t *testing.T) {
	for _, mutate := range []struct {
		name   string
		change func(*hotelsDomain.Hotel)
	}{
		{"empty", func(h *hotelsDomain.Hotel) { *h = hotelsDomain.Hotel{} }},
		{"blank name", func(h *hotelsDomain.Hotel) { h.Name = "  " }},
		{"negative price", func(h *hotelsDomain.Hotel) { h.PricePerNight = -1 }},
		{"negative capacity", func(h *hotelsDomain.Hotel) { h.AvailableRooms = -1 }},
		{"rating beyond five", func(h *hotelsDomain.Hotel) { h.Rating = 5.1 }},
		{"empty schedule", func(h *hotelsDomain.Hotel) { h.CheckInTime = "" }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			service, repo, queue := getTestService()
			original := validHotel(hotelsDomain.Hotel{Name: "Original", AvailableRooms: 2})
			id, err := service.Create(context.Background(), original)
			if err != nil {
				t.Fatal(err)
			}
			*queue.hotelEvents = nil
			invalid := original
			invalid.ID = id
			mutate.change(&invalid)
			if _, err := service.Create(context.Background(), invalid); !errors.Is(err, hotelsDomain.ErrInvalidHotel) {
				t.Fatalf("create accepted invalid hotel: %v", err)
			}
			if err := service.Update(context.Background(), invalid); !errors.Is(err, hotelsDomain.ErrInvalidHotel) {
				t.Fatalf("update accepted invalid hotel: %v", err)
			}
			count, _ := repo.CountHotels(context.Background())
			stored, _ := repo.GetHotelByID(context.Background(), id)
			if count != 1 || stored.Name != "Original" || stored.AvailableRooms != 2 || len(*queue.hotelEvents) != 0 {
				t.Fatalf("invalid request had side effects: %+v events=%+v", stored, *queue.hotelEvents)
			}
		})
	}
}

func TestPublishFailureDoesNotTurnPersistedCRUDIntoHTTPFailure(t *testing.T) {
	repo := hotels.NewMock()
	queue := NewMockQueue()
	queue.fail = true
	service := NewService(repo, queue)
	ctx := context.Background()
	hotel := validHotel(hotelsDomain.Hotel{Name: "Persisted", AvailableRooms: 2})
	id, err := service.Create(ctx, hotel)
	if err != nil || id == "" {
		t.Fatalf("persisted create must succeed: %s %v", id, err)
	}
	hotel.ID = id
	hotel.AvailableRooms = 0
	hotel.PricePerNight = 0
	hotel.Rating = 0
	hotel.Amenities = []string{}
	hotel.Images = []string{}
	if err := service.Update(ctx, hotel); err != nil {
		t.Fatal(err)
	}
	stored, err := service.GetHotelByID(ctx, id)
	if err != nil || stored.AvailableRooms != 0 {
		t.Fatalf("update not persisted %+v %v", stored, err)
	}
	if err := service.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetHotelByID(ctx, id); !errors.Is(err, hotelsDomain.ErrHotelNotFound) {
		t.Fatalf("delete not persisted: %v", err)
	}
	if len(*queue.hotelEvents) != 3 {
		t.Fatalf("want three publish attempts got %v", *queue.hotelEvents)
	}
}

func TestReservationsDoNotCallPublisher(t *testing.T) {
	service, _, queue := getTestService()
	ctx := context.Background()
	hotel, err := service.Create(ctx, validHotel(hotelsDomain.Hotel{Name: "No broker reservation", AvailableRooms: 1}))
	if err != nil {
		t.Fatal(err)
	}
	*queue.hotelEvents = nil
	id, err := service.CreateReservation(ctx, hotelsDomain.Reservation{HotelID: hotel, UserID: "u", CheckIn: futureDate(t, 10), CheckOut: futureDate(t, 12)})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.CancelReservation(ctx, id); err != nil {
		t.Fatal(err)
	}
	if len(*queue.hotelEvents) != 0 {
		t.Fatalf("reservation touched publisher: %v", *queue.hotelEvents)
	}
}
