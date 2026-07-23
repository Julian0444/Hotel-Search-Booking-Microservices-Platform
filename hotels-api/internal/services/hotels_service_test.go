package services

import (
	"context"
	"errors"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"
)

// Mock de la cola: registra lo publicado para asserts
type MockQueue struct {
	hotelEvents       *[]hotelsDomain.HotelNew
	reservationEvents *[]hotelsDomain.ReservationNew
}

func NewMockQueue() MockQueue {
	return MockQueue{
		hotelEvents:       &[]hotelsDomain.HotelNew{},
		reservationEvents: &[]hotelsDomain.ReservationNew{},
	}
}

func (mq MockQueue) Publish(hotelNew hotelsDomain.HotelNew) error {
	*mq.hotelEvents = append(*mq.hotelEvents, hotelNew)
	return nil
}

func (mq MockQueue) PublishReservation(reservationNew hotelsDomain.ReservationNew) error {
	*mq.reservationEvents = append(*mq.reservationEvents, reservationNew)
	return nil
}

// Helper para crear el service con mocks reutilizables
func getTestService() (Service, hotels.Mock, hotels.MockCache) {
	mainRepo := hotels.NewMock()       // Repositorio principal
	cacheRepo := hotels.NewMockCache() // Cache
	return NewService(mainRepo, cacheRepo, NewMockQueue()), mainRepo, cacheRepo
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
		if _, err := service.Create(ctx, hotelsDomain.Hotel{Name: name}); err != nil {
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

// D2: un miss de caché en Update no falla la escritura y el evento sale igual
func TestUpdateHotel_CacheMissStillPublishes(t *testing.T) {
	mainRepo := hotels.NewMock()
	cacheRepo := hotels.NewMockCache()
	queue := NewMockQueue()
	service := NewService(mainRepo, cacheRepo, queue)
	ctx := context.Background()

	// Hotel creado SOLO en main: la caché no lo conoce (simula expiración)
	hotelID, err := mainRepo.Create(ctx, hotelsDAO.Hotel{Name: "Uncached"})
	if err != nil {
		t.Fatalf("error creating hotel in main repo: %v", err)
	}

	if err := service.Update(ctx, hotelsDomain.Hotel{ID: hotelID, Name: "Renamed"}); err != nil {
		t.Fatalf("update must not fail on cache miss: %v", err)
	}

	events := *queue.hotelEvents
	if len(events) != 1 || events[0].Operation != "UPDATE" || events[0].HotelID != hotelID {
		t.Errorf("expected 1 UPDATE event for %s, got %+v", hotelID, events)
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

	hotel := hotelsDomain.Hotel{Name: "HotelRes", AvaiableRooms: 2, PricePerNight: 100}
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

	hotelID, _ := service.Create(ctx, hotelsDomain.Hotel{Name: "Full", AvaiableRooms: 1, PricePerNight: 50})

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

	hotelID, _ := service.Create(ctx, hotelsDomain.Hotel{Name: "Valid", AvaiableRooms: 2, PricePerNight: 50})

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

// DM5: crear y cancelar publican ReservationNew en la cola de reservas
func TestReservationEventsPublished(t *testing.T) {
	mainRepo := hotels.NewMock()
	cacheRepo := hotels.NewMockCache()
	queue := NewMockQueue()
	service := NewService(mainRepo, cacheRepo, queue)
	ctx := context.Background()

	hotelID, _ := service.Create(ctx, hotelsDomain.Hotel{Name: "Events", AvaiableRooms: 1, PricePerNight: 10})
	resID, err := service.CreateReservation(ctx, hotelsDomain.Reservation{
		HotelID: hotelID, UserID: "u", CheckIn: futureDate(t, 10), CheckOut: futureDate(t, 11), NumRooms: 1,
	})
	if err != nil {
		t.Fatalf("error creating reservation: %v", err)
	}
	if err := service.CancelReservation(ctx, resID); err != nil {
		t.Fatalf("error canceling reservation: %v", err)
	}

	events := *queue.reservationEvents
	if len(events) != 2 {
		t.Fatalf("expected 2 reservation events, got %d", len(events))
	}
	if events[0].Operation != "CREATE" || events[0].ReservationID != resID || events[0].HotelID != hotelID {
		t.Errorf("unexpected CREATE event: %+v", events[0])
	}
	if events[1].Operation != "CANCEL" || events[1].ReservationID != resID || events[1].HotelID != hotelID {
		t.Errorf("unexpected CANCEL event: %+v", events[1])
	}
}

// DM1: cancelar es soft-delete (queda con status cancelled) e idempotente
func TestCancelReservation(t *testing.T) {
	service, _, _ := getTestService()
	ctx := context.Background()

	hotel := hotelsDomain.Hotel{Name: "HotelResCancel", AvaiableRooms: 1, PricePerNight: 50}
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

	hotel := hotelsDomain.Hotel{Name: "HotelRes2", AvaiableRooms: 1, PricePerNight: 50}
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

	hotel := hotelsDomain.Hotel{Name: "HotelRes3", AvaiableRooms: 1, PricePerNight: 50}
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

	hotel := hotelsDomain.Hotel{Name: "HotelRes4", AvaiableRooms: 1, PricePerNight: 50}
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
	service := NewService(mainRepo, cacheRepo, NewMockQueue())
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
	service := NewService(mainRepo, cacheRepo, NewMockQueue())
	ctx := context.Background()

	// Crear hotel en main para asociar reserva (con capacidad para el mock)
	hotelID, err := mainRepo.Create(ctx, hotelsDAO.Hotel{Name: "HotelForReservation", AvaiableRooms: 1})
	if err != nil {
		t.Fatalf("error creating hotel in main repo: %v", err)
	}
	// Crear reserva solo en main
	resID, err := mainRepo.CreateReservation(ctx, hotelsDAO.Reservation{
		HotelID:  hotelID,
		UserID:   "user-cache",
		NumRooms: 1,
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
		PricePerNight: 50,
	})

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

	// La escritura invalidó las listas agregadas de la caché (F1): una lectura
	// repuebla la lista completa — como en producción — para que el conteo de
	// disponibilidad de la caché vea la reserva.
	if _, err := service.GetReservationsByHotelID(ctx, hotelID, 20, 0); err != nil {
		t.Fatalf("error repopulating reservations list: %v", err)
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
