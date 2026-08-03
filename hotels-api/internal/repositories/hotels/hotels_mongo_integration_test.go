//go:build integration

// Suite de integración contra un Mongo real (testcontainers).
// Correr con: go test -tags=integration ./...
// Cubre el claim atómico de inventario (D1), la cancelación idempotente (DM1)
// y la suite de solapamiento de fechas compartida con la caché (C10/D4).
package hotels

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
)

// startMongoContainer levanta un mongo:6 (misma imagen que el compose) y
// devuelve la config lista para NewMongo.
func startMongoContainer(t *testing.T) MongoConfig {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:6",
			ExposedPorts: []string{"27017/tcp"},
			Env: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": "root",
				"MONGO_INITDB_ROOT_PASSWORD": "root",
			},
			WaitingFor: wait.ForLog("Waiting for connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("starting mongo container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("getting container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "27017/tcp")
	if err != nil {
		t.Fatalf("getting mapped port: %v", err)
	}

	return MongoConfig{
		Host:                    host,
		Port:                    port.Port(),
		Username:                "root",
		Password:                "root",
		Database:                "hotels-api-integration",
		Collection_hotels:       "hotels",
		Collection_reservations: "reservations",
		Collection_inventory:    "reservation_inventory",
		// Sin esto EnsureIndexes (A3, plan 07) panickea con InvalidNamespace
		Collection_idempotency: "idempotency_keys",
	}
}

// confirmedReservation arma una reserva válida para el repo (el service ya
// setea estos campos en producción).
func confirmedReservation(t *testing.T, hotelID, userID, checkIn, checkOut string, rooms int) hotelsDAO.Reservation {
	t.Helper()
	return hotelsDAO.Reservation{
		HotelName: "Integration Hotel",
		HotelID:   hotelID,
		UserID:    userID,
		CheckIn:   mustParseDay(t, checkIn),
		CheckOut:  mustParseDay(t, checkOut),
		Status:    hotelsDAO.StatusConfirmed,
		NumRooms:  rooms,
		NumGuests: rooms,
		CreatedAt: time.Now().UTC(),
	}
}

// Ejercita el path real Create → CreateReservation → IsHotelAvailable:
// un hotel de 1 habitación con una reserva deja de estar disponible en ese
// rango y sigue disponible en un rango disjunto.
func TestMongo_CreateReservationAvailability(t *testing.T) {
	ctx := context.Background()
	repository := NewMongo(startMongoContainer(t))

	hotelID, err := repository.Create(ctx, hotelsDAO.Hotel{
		Name:           "Integration Hotel",
		City:           "Córdoba",
		AvailableRooms: 1,
	})
	if err != nil {
		t.Fatalf("creating hotel: %v", err)
	}

	if _, err := repository.CreateReservation(ctx, confirmedReservation(t, hotelID, "user-1", "2030-08-01", "2030-08-03", 1)); err != nil {
		t.Fatalf("creating reservation: %v", err)
	}

	available, err := repository.IsHotelAvailable(ctx, hotelID, "2030-08-01", "2030-08-03")
	if err != nil {
		t.Fatalf("checking availability (overlapping): %v", err)
	}
	if available {
		t.Error("hotel with 1 room and 1 reservation should NOT be available in the same range")
	}

	available, err = repository.IsHotelAvailable(ctx, hotelID, "2030-09-01", "2030-09-03")
	if err != nil {
		t.Fatalf("checking availability (disjoint): %v", err)
	}
	if !available {
		t.Error("hotel should be available in a disjoint range")
	}
}

// La suite de solapamiento compartida con la caché, contra Mongo real (D4).
func TestMongo_AvailabilitySuite(t *testing.T) {
	ctx := context.Background()
	repository := NewMongo(startMongoContainer(t))

	hotelID, err := repository.Create(ctx, hotelsDAO.Hotel{
		Name:           "Suite Hotel",
		AvailableRooms: 1,
	})
	if err != nil {
		t.Fatalf("creating hotel: %v", err)
	}

	runAvailabilitySuite(t, repository, hotelID)
}

// D1: N goroutines reservando la última habitación → exactamente 1 gana y el
// inventario nunca supera la capacidad.
func TestMongo_ConcurrentClaimLastRoom(t *testing.T) {
	ctx := context.Background()
	config := startMongoContainer(t)
	repository := NewMongo(config)

	hotelID, err := repository.Create(ctx, hotelsDAO.Hotel{
		Name:           "Race Hotel",
		AvailableRooms: 1,
	})
	if err != nil {
		t.Fatalf("creating hotel: %v", err)
	}

	const attempts = 20
	var wg sync.WaitGroup
	results := make(chan error, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := repository.CreateReservation(ctx, confirmedReservation(t, hotelID, "racer", "2030-08-01", "2030-08-03", 1))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	var created, conflicts, unexpected int
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, hotelsDomain.ErrNoAvailability):
			conflicts++
		default:
			unexpected++
			t.Errorf("unexpected error from concurrent claim: %v", err)
		}
	}

	if created != 1 || conflicts != attempts-1 || unexpected != 0 {
		t.Fatalf("expected 1 winner and %d conflicts, got created=%d conflicts=%d unexpected=%d",
			attempts-1, created, conflicts, unexpected)
	}

	// El inventario nunca supera capacity
	inventory := repository.client.Database(config.Database).Collection(config.Collection_inventory)
	count, err := inventory.CountDocuments(ctx, bson.M{"$expr": bson.M{"$gt": bson.A{"$booked", "$capacity"}}})
	if err != nil {
		t.Fatalf("counting overbooked inventory: %v", err)
	}
	if count != 0 {
		t.Fatalf("found %d inventory entries with booked > capacity", count)
	}
}

// R4/RV24: el batch de disponibilidad responde parcial — un ID inválido o
// inexistente se reporta available=false en vez de tumbar el mapa entero con
// error (y el fan-out corre acotado por el bulkhead).
func TestMongo_GetAvailabilityPartialOnBadID(t *testing.T) {
	ctx := context.Background()
	repository := NewMongo(startMongoContainer(t))

	hotelID, err := repository.Create(ctx, hotelsDAO.Hotel{
		Name:           "Partial Hotel",
		AvailableRooms: 1,
	})
	if err != nil {
		t.Fatalf("creating hotel: %v", err)
	}

	// "garbage" ni siquiera es un ObjectID; el hex válido no existe
	ids := []string{hotelID, "garbage", "bfbfbfbfbfbfbfbfbfbfbfbf"}
	availability, err := repository.GetAvailability(ctx, ids, "2030-08-01", "2030-08-03")
	if err != nil {
		t.Fatalf("availability batch must not fail on a bad ID (RV24): %v", err)
	}
	if len(availability) != len(ids) {
		t.Fatalf("expected %d entries, got %d: %+v", len(ids), len(availability), availability)
	}
	if !availability[hotelID] {
		t.Error("existing hotel without reservations should be available")
	}
	if availability["garbage"] || availability["bfbfbfbfbfbfbfbfbfbfbfbf"] {
		t.Error("unknown hotels must report available=false, not true")
	}
}

// DM1: cancelar es soft-delete, libera el cupo y es idempotente (un segundo
// cancel no doble-libera).
func TestMongo_CancelReservationIdempotent(t *testing.T) {
	ctx := context.Background()
	config := startMongoContainer(t)
	repository := NewMongo(config)

	hotelID, err := repository.Create(ctx, hotelsDAO.Hotel{
		Name:           "Cancel Hotel",
		AvailableRooms: 1,
	})
	if err != nil {
		t.Fatalf("creating hotel: %v", err)
	}

	resID, err := repository.CreateReservation(ctx, confirmedReservation(t, hotelID, "user-1", "2030-08-01", "2030-08-03", 1))
	if err != nil {
		t.Fatalf("creating reservation: %v", err)
	}

	// Cancelar: devuelve la reserva cancelada y libera las noches
	cancelled, err := repository.CancelReservation(ctx, resID)
	if err != nil {
		t.Fatalf("cancelling reservation: %v", err)
	}
	if cancelled.Status != hotelsDAO.StatusCancelled || cancelled.CancelledAt == nil {
		t.Fatalf("expected cancelled reservation with cancelled_at, got %+v", cancelled)
	}

	// El documento sigue en la colección (soft delete)
	stored, err := repository.GetReservationByID(ctx, resID)
	if err != nil {
		t.Fatalf("cancelled reservation must still exist: %v", err)
	}
	if stored.Status != hotelsDAO.StatusCancelled {
		t.Fatalf("expected stored status cancelled, got %q", stored.Status)
	}

	// Segundo cancel: idempotente, no falla y NO doble-libera
	if _, err := repository.CancelReservation(ctx, resID); err != nil {
		t.Fatalf("second cancel must be idempotent: %v", err)
	}

	inventory := repository.client.Database(config.Database).Collection(config.Collection_inventory)
	var entry hotelsDAO.Inventory
	if err := inventory.FindOne(ctx, bson.M{"hotel_id": hotelID, "date": "2030-08-01"}).Decode(&entry); err != nil {
		t.Fatalf("reading inventory entry: %v", err)
	}
	if entry.Booked != 0 {
		t.Fatalf("expected booked=0 after single release (no double-release), got %d", entry.Booked)
	}

	// El cupo liberado se puede volver a reservar
	if _, err := repository.CreateReservation(ctx, confirmedReservation(t, hotelID, "user-2", "2030-08-01", "2030-08-03", 1)); err != nil {
		t.Fatalf("rebooking after cancel must succeed: %v", err)
	}
}
