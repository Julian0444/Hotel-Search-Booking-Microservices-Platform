//go:build integration

// Suite de integración contra un Mongo real (testcontainers).
// Correr con: go test -tags=integration ./...
// Cubre transacciones, resultados ambiguos y coordinación con CRUD administrativo.
package hotels

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
)

// startMongoContainer levanta un mongo:6 (misma imagen que el compose) y
// devuelve la config lista para NewMongo.
func startMongoContainer(t *testing.T) MongoConfig {
	cfg, _ := startMongoContainerWithHandle(t)
	return cfg
}
func startMongoContainerWithHandle(t *testing.T) (MongoConfig, testcontainers.Container) {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:6",
			ExposedPorts: []string{"27017/tcp"},
			Cmd:          []string{"--replSet", "rs0", "--bind_ip_all", "--setParameter", "enableTestCommands=1"},
			WaitingFor:   wait.ForLog("Waiting for connections"),
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

	code, reader, err := container.Exec(ctx, []string{"mongosh", "--quiet", "--eval", `rs.initiate({_id:"rs0",members:[{_id:0,host:"localhost:27017"}]})`})
	if err != nil || code != 0 {
		t.Fatalf("initiating replica set: exit=%d err=%v", code, err)
	}
	if reader != nil {
		_, _ = io.Copy(io.Discard, reader)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		code, reader, err = container.Exec(ctx, []string{"mongosh", "--quiet", "--eval", `quit(db.hello().isWritablePrimary ? 0 : 1)`})
		if reader != nil {
			_, _ = io.Copy(io.Discard, reader)
		}
		if err == nil && code == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("replica set never elected primary: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return MongoConfig{ReplicaSet: "rs0", Direct: true,
		Host:                    host,
		Port:                    port.Port(),
		Username:                "",
		Password:                "",
		Database:                "hotels-api-integration",
		Collection_hotels:       "hotels",
		Collection_reservations: "reservations",
		Collection_inventory:    "reservation_inventory",
		// Sin esto EnsureIndexes (A3, plan 07) panickea con InvalidNamespace
		Collection_idempotency: "idempotency_keys",
	}, container
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

// La suite de solapamiento contra Mongo real (D4).
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
			aliasedID := hotelID
			if n%2 == 0 {
				aliasedID = strings.ToUpper(hotelID)
			}
			_, err := repository.CreateReservation(ctx, confirmedReservation(t, aliasedID, "racer", "2030-08-01", "2030-08-03", 1))
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

// lostCommitDialer drops the first SUCCESSFUL server commit response, after
// Mongo applied it. This is a real transport failure, not a mock write result.
// Other replies pass through unchanged; driver retry must use the same txn.
type lostCommitDialer struct {
	drops   atomic.Int32
	commits atomic.Int32
	delay   time.Duration
}
type lostCommitConn struct {
	net.Conn
	owner         *lostCommitDialer
	commitRequest int32
	pending       []byte
}

func (d *lostCommitDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &lostCommitConn{Conn: c, owner: d}, nil
}
func (c *lostCommitConn) Write(p []byte) (int, error) {
	if len(p) > 21 && binary.LittleEndian.Uint32(p[12:16]) == 2013 && p[20] == 0 {
		var command bson.M
		if bson.Unmarshal(p[21:], &command) == nil && command["commitTransaction"] != nil {
			c.commitRequest = int32(binary.LittleEndian.Uint32(p[4:8]))
			c.owner.commits.Add(1)
		}
	}
	return c.Conn.Write(p)
}
func (c *lostCommitConn) Read(p []byte) (int, error) {
	if len(c.pending) == 0 {
		header := make([]byte, 16)
		if _, err := io.ReadFull(c.Conn, header); err != nil {
			return 0, err
		}
		length := int(binary.LittleEndian.Uint32(header[:4]))
		if length < 16 || length > 64*1024*1024 {
			return 0, fmt.Errorf("invalid wire frame length %d", length)
		}
		frame := make([]byte, length)
		copy(frame, header)
		if _, err := io.ReadFull(c.Conn, frame[16:]); err != nil {
			return 0, err
		}
		responseTo := int32(binary.LittleEndian.Uint32(frame[8:12]))
		if c.commitRequest != 0 && responseTo == c.commitRequest && c.owner.drops.CompareAndSwap(0, 1) {
			var reply bson.M
			if err := bson.Unmarshal(frame[21:], &reply); err != nil {
				return 0, err
			}
			if reply["ok"] != float64(1) {
				return 0, fmt.Errorf("expected successful applied commit, got %v", reply)
			}
			time.Sleep(c.owner.delay)
			_ = c.Conn.Close()
			return 0, io.EOF
		}
		c.pending = frame
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

func TestMongo_TransactionFailureRecoveryAndAdminRaces(t *testing.T) {
	cfg := startMongoContainer(t)
	repo := NewMongo(cfg)
	defer repo.Disconnect(context.Background())
	ctx := context.Background()
	db := repo.client.Database(cfg.Database)
	newHotel := func(capacity int) string {
		t.Helper()
		id, err := repo.Create(ctx, hotelsDAO.Hotel{Name: "Transaction Hotel", AvailableRooms: capacity, PricePerNight: 12.34})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	assertState := func(hotelID string, reservations int64, booked int) {
		t.Helper()
		count, err := db.Collection(cfg.Collection_reservations).CountDocuments(ctx, bson.M{"hotel_id": hotelID})
		if err != nil || count != reservations {
			t.Fatalf("reservations=%d want=%d error=%v", count, reservations, err)
		}
		cursor, err := db.Collection(cfg.Collection_inventory).Find(ctx, bson.M{"hotel_id": hotelID})
		if err != nil {
			t.Fatal(err)
		}
		var entries []hotelsDAO.Inventory
		if err := cursor.All(ctx, &entries); err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Booked != booked || entry.Booked < 0 || entry.Booked > entry.Capacity {
				t.Fatalf("invalid inventory: %+v want booked=%d", entry, booked)
			}
		}
	}
	failCommand := func(command bson.M) {
		t.Helper()
		if err := repo.client.Database("admin").RunCommand(ctx, bson.D{{Key: "configureFailPoint", Value: command["configureFailPoint"]}, {Key: "mode", Value: command["mode"]}, {Key: "data", Value: command["data"]}}).Err(); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("ObjectID aliases share inventory fingerprint and admin lock", func(t *testing.T) {
		hotel := newHotel(1)
		uppercase := strings.ToUpper(hotel)
		request := confirmedReservation(t, hotel, "owner", "2030-08-01", "2030-08-03", 1)
		id, err := repo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "canonical-id"), request)
		if err != nil {
			t.Fatal(err)
		}
		request.HotelID = uppercase
		replay, err := repo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "canonical-id"), request)
		if err != nil || id != replay {
			t.Fatalf("canonical replay %s vs %s err=%v", id, replay, err)
		}
		if _, err := repo.CreateReservation(ctx, request); !errors.Is(err, hotelsDomain.ErrNoAvailability) {
			t.Fatalf("uppercase oversold: %v", err)
		}
		if available, err := repo.IsHotelAvailable(ctx, uppercase, "2030-08-01", "2030-08-03"); err != nil || available {
			t.Fatalf("uppercase availability: %v %v", available, err)
		}
		if err := repo.Update(ctx, hotelsDAO.Hotel{ID: uppercase, AvailableRooms: 0}); !errors.Is(err, hotelsDomain.ErrCapacityConflict) {
			t.Fatalf("uppercase bypassed capacity lock: %v", err)
		}
		if err := repo.Delete(ctx, uppercase); !errors.Is(err, hotelsDomain.ErrHotelHasReservations) {
			t.Fatalf("uppercase orphaned reservation: %v", err)
		}
		rows, err := repo.GetReservationsByHotelID(ctx, uppercase, 20, 0)
		if err != nil || len(rows) != 1 || rows[0].HotelID != hotel {
			t.Fatalf("uppercase history: %+v %v", rows, err)
		}
		count, err := db.Collection(cfg.Collection_inventory).CountDocuments(ctx, bson.M{"hotel_id": uppercase})
		if err != nil || count != 0 {
			t.Fatalf("uppercase inventory should not exist: %d %v", count, err)
		}
		assertState(hotel, 1, 1)
	})
	t.Run("multi-night conflict aborts prior writes", func(t *testing.T) {
		hotel := newHotel(1)
		if _, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "occupant", "2030-08-02", "2030-08-03", 1)); err != nil {
			t.Fatal(err)
		}
		_, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "visitor", "2030-08-01", "2030-08-03", 1))
		if !errors.Is(err, hotelsDomain.ErrNoAvailability) {
			t.Fatalf("want conflict got %v", err)
		}
		count, err := db.Collection(cfg.Collection_inventory).CountDocuments(ctx, bson.M{"hotel_id": hotel, "date": "2030-08-01"})
		if err != nil || count != 0 {
			t.Fatalf("aborted first night exists: %d %v", count, err)
		}
		assertState(hotel, 1, 1)
	})
	t.Run("insert never executed aborts inventory and key", func(t *testing.T) {
		hotel := newHotel(1)
		failCommand(bson.M{"configureFailPoint": "failCommand", "mode": bson.M{"times": 1}, "data": bson.M{"failCommands": bson.A{"insert"}, "errorCode": 2}})
		keyCtx := hotelsDomain.WithIdempotency(ctx, "failed-before-confirming")
		request := confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1)
		if _, err := repo.CreateReservation(keyCtx, request); err == nil {
			t.Fatal("expected injected pre-commit failure")
		}
		assertState(hotel, 0, 0)
		count, err := db.Collection(cfg.Collection_inventory).CountDocuments(ctx, bson.M{"hotel_id": hotel})
		if err != nil || count != 0 {
			t.Fatalf("partial inventory: %d %v", count, err)
		}
		count, err = repo.idempotencyCollection().CountDocuments(ctx, bson.M{"key": "failed-before-confirming"})
		if err != nil || count != 0 {
			t.Fatalf("partial key: %d %v", count, err)
		}
		if _, err := repo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "failed-before-confirming"), request); err != nil {
			t.Fatalf("retry known aborted operation: %v", err)
		}
		assertState(hotel, 1, 1)
	})
	t.Run("applied commit response lost driver retries and restart replays", func(t *testing.T) {
		hotel := newHotel(1)
		dialer := &lostCommitDialer{}
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s/?replicaSet=rs0&directConnection=true", cfg.Host, cfg.Port)).SetDialer(dialer))
		if err != nil {
			t.Fatal(err)
		}
		failingRepo := repo
		failingRepo.client = client
		request := confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1)
		id, err := failingRepo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "lost-response"), request)
		if err != nil {
			t.Fatal(err)
		}
		if dialer.drops.Load() != 1 || dialer.commits.Load() < 2 {
			t.Fatalf("failure not exercised: dropped=%d commits=%d", dialer.drops.Load(), dialer.commits.Load())
		}
		if err := client.Disconnect(ctx); err != nil {
			t.Fatal(err)
		}
		restarted := NewMongo(cfg)
		defer restarted.Disconnect(ctx)
		replayCtx := hotelsDomain.WithIdempotency(ctx, "lost-response")
		replay, err := restarted.CreateReservation(replayCtx, request)
		if err != nil || replay != id || !hotelsDomain.IdempotencyFromContext(replayCtx).Replayed {
			t.Fatalf("replay=%s original=%s error=%v", replay, id, err)
		}
		request.NumGuests++
		if _, err := restarted.CreateReservation(hotelsDomain.WithIdempotency(ctx, "lost-response"), request); !errors.Is(err, hotelsDomain.ErrIdempotencyConflict) {
			t.Fatalf("expected payload conflict: %v", err)
		}
		assertState(hotel, 1, 1)
	})
	t.Run("request timeout after applied commit recovers durable result", func(t *testing.T) {
		hotel := newHotel(1)
		dialer := &lostCommitDialer{delay: 300 * time.Millisecond}
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s/?replicaSet=rs0&directConnection=true", cfg.Host, cfg.Port)).SetDialer(dialer))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Disconnect(ctx)
		if err := client.Ping(ctx, nil); err != nil {
			t.Fatal(err)
		}
		failingRepo := repo
		failingRepo.client = client
		request := confirmedReservation(t, hotel, "timeout-owner", "2030-08-01", "2030-08-03", 1)
		timeoutCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()
		id, err := failingRepo.CreateReservation(hotelsDomain.WithIdempotency(timeoutCtx, "timeout-key"), request)
		if !errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) || dialer.drops.Load() != 1 {
			t.Fatalf("timeout was not exercised: %v drops=%d", timeoutCtx.Err(), dialer.drops.Load())
		}
		if err != nil || id == "" {
			t.Fatalf("durable recovery failed after timeout: %s %v", id, err)
		}
		replay, err := repo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "timeout-key"), request)
		if err != nil || replay != id {
			t.Fatalf("retry duplicated confirmed attempt %s %s %v", id, replay, err)
		}
		assertState(hotel, 1, 1)
	})
	t.Run("applied cancellation commit response lost releases once", func(t *testing.T) {
		hotel := newHotel(1)
		id, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "owner", "2030-08-01", "2030-08-03", 1))
		if err != nil {
			t.Fatal(err)
		}
		dialer := &lostCommitDialer{}
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s/?replicaSet=rs0&directConnection=true", cfg.Host, cfg.Port)).SetDialer(dialer))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Disconnect(ctx)
		failingRepo := repo
		failingRepo.client = client
		cancelled, err := failingRepo.CancelReservation(ctx, id)
		if err != nil || cancelled.Status != hotelsDAO.StatusCancelled {
			t.Fatalf("lost cancellation result: %+v %v", cancelled, err)
		}
		if dialer.drops.Load() != 1 || dialer.commits.Load() < 2 {
			t.Fatalf("failure not exercised: dropped=%d commits=%d", dialer.drops.Load(), dialer.commits.Load())
		}
		if _, err := repo.CancelReservation(ctx, id); err != nil {
			t.Fatal(err)
		}
		assertState(hotel, 1, 0)
	})
	t.Run("concurrent same key has one durable result", func(t *testing.T) {
		hotel := newHotel(20)
		var group sync.WaitGroup
		ids := make(chan string, 20)
		for range 20 {
			group.Add(1)
			go func() {
				defer group.Done()
				id, err := repo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "concurrent-key"), confirmedReservation(t, hotel, "owner", "2030-08-01", "2030-08-03", 1))
				if err != nil {
					t.Error(err)
				}
				ids <- id
			}()
		}
		group.Wait()
		close(ids)
		first := ""
		for id := range ids {
			if first == "" {
				first = id
			}
			if id == "" || id != first {
				t.Errorf("inconsistent id %s vs %s", id, first)
			}
		}
		assertState(hotel, 1, 1)
	})
	t.Run("concurrent cancellation releases exactly once", func(t *testing.T) {
		hotel := newHotel(1)
		id, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1))
		if err != nil {
			t.Fatal(err)
		}
		var group sync.WaitGroup
		for range 20 {
			group.Add(1)
			go func() {
				defer group.Done()
				if _, err := repo.CancelReservation(ctx, id); err != nil {
					t.Error(err)
				}
			}()
		}
		group.Wait()
		assertState(hotel, 1, 0)
		stored, err := repo.GetReservationByID(ctx, id)
		if err != nil || stored.Status != hotelsDAO.StatusCancelled {
			t.Fatalf("cancelled not persisted: %+v %v", stored, err)
		}
	})
	t.Run("cancellation write failure does not partially release", func(t *testing.T) {
		hotel := newHotel(1)
		id, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1))
		if err != nil {
			t.Fatal(err)
		}
		failCommand(bson.M{"configureFailPoint": "failCommand", "mode": bson.M{"times": 1}, "data": bson.M{"failCommands": bson.A{"update"}, "errorCode": 2}})
		if _, err := repo.CancelReservation(ctx, id); err == nil {
			t.Fatal("expected cancellation failure")
		}
		assertState(hotel, 1, 1)
		stored, err := repo.GetReservationByID(ctx, id)
		if err != nil || stored.Status != hotelsDAO.StatusConfirmed {
			t.Fatalf("unexpected state: %+v %v", stored, err)
		}
		if _, err := repo.CancelReservation(ctx, id); err != nil {
			t.Fatal(err)
		}
		assertState(hotel, 1, 0)
	})
	t.Run("failure after first cancellation night rolls back all", func(t *testing.T) {
		hotel := newHotel(1)
		id, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1))
		if err != nil {
			t.Fatal(err)
		}
		var injected atomic.Bool
		monitor := &event.CommandMonitor{Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
			if e.CommandName == "update" && injected.CompareAndSwap(false, true) {
				failCommand(bson.M{"configureFailPoint": "failCommand", "mode": bson.M{"times": 1}, "data": bson.M{"failCommands": bson.A{"update"}, "errorCode": 2}})
			}
		}}
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s/?replicaSet=rs0&directConnection=true", cfg.Host, cfg.Port)).SetMonitor(monitor))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Disconnect(ctx)
		failingRepo := repo
		failingRepo.client = client
		if _, err := failingRepo.CancelReservation(ctx, id); err == nil {
			t.Fatal("expected second-night failure")
		}
		if !injected.Load() {
			t.Fatal("first night was not written")
		}
		assertState(hotel, 1, 1)
		stored, err := repo.GetReservationByID(ctx, id)
		if err != nil || stored.Status != hotelsDAO.StatusConfirmed {
			t.Fatalf("partial cancellation %+v %v", stored, err)
		}
	})
	t.Run("capacity update racing reserve respects serial order", func(t *testing.T) {
		for range 10 {
			hotel := newHotel(1)
			start := make(chan struct{})
			result := make(chan error, 2)
			go func() {
				<-start
				_, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1))
				result <- err
			}()
			go func() {
				<-start
				result <- repo.Update(ctx, hotelsDAO.Hotel{ID: hotel, Name: "Closed", AvailableRooms: 0})
			}()
			close(start)
			a, b := <-result, <-result
			if (a == nil) == (b == nil) {
				t.Fatalf("exactly one operation must win: %v / %v", a, b)
			}
			failure := a
			if failure == nil {
				failure = b
			}
			if !errors.Is(failure, hotelsDomain.ErrCapacityConflict) && !errors.Is(failure, hotelsDomain.ErrNoAvailability) {
				t.Fatal(failure)
			}
			stored, err := repo.GetHotelByID(ctx, hotel)
			if err != nil {
				t.Fatal(err)
			}
			expected := stored.AvailableRooms
			assertState(hotel, int64(expected), expected)
		}
	})
	t.Run("delete racing reserve never orphans history", func(t *testing.T) {
		for range 10 {
			hotel := newHotel(1)
			start := make(chan struct{})
			result := make(chan error, 2)
			go func() {
				<-start
				_, err := repo.CreateReservation(ctx, confirmedReservation(t, hotel, "user", "2030-08-01", "2030-08-03", 1))
				result <- err
			}()
			go func() { <-start; result <- repo.Delete(ctx, hotel) }()
			close(start)
			a, b := <-result, <-result
			if (a == nil) == (b == nil) {
				t.Fatalf("exactly one operation must win: %v / %v", a, b)
			}
			failure := a
			if failure == nil {
				failure = b
			}
			if !errors.Is(failure, hotelsDomain.ErrHotelHasReservations) && !errors.Is(failure, hotelsDomain.ErrHotelNotFound) {
				t.Fatal(failure)
			}
			_, err := repo.GetHotelByID(ctx, hotel)
			if errors.Is(err, hotelsDomain.ErrHotelNotFound) {
				assertState(hotel, 0, 0)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assertState(hotel, 1, 1)
			}
		}
	})
	t.Run("PUT persists zero and clears optional fields", func(t *testing.T) {
		hotel := newHotel(3)
		update := hotelsDAO.Hotel{ID: hotel, Name: "Zero", PricePerNight: 0, Rating: 0, AvailableRooms: 0, Amenities: []string{}, Images: []string{}}
		if err := repo.Update(ctx, update); err != nil {
			t.Fatal(err)
		}
		got, err := repo.GetHotelByID(ctx, hotel)
		if err != nil || got.Name != "Zero" || got.AvailableRooms != 0 || got.PricePerNight != 0 || got.Rating != 0 || len(got.Images) != 0 || len(got.Amenities) != 0 {
			t.Fatalf("PUT was partial: %+v %v", got, err)
		}
	})
}

func TestMongo_InventoryAuditAndLegacyMigration(t *testing.T) {
	cfg, container := startMongoContainerWithHandle(t)
	repo := NewMongo(cfg)
	defer repo.Disconnect(context.Background())
	ctx := context.Background()
	hotel, err := repo.Create(ctx, hotelsDAO.Hotel{Name: "Migration", AvailableRooms: 1})
	if err != nil {
		t.Fatal(err)
	}
	reservation := confirmedReservation(t, hotel, "legacy-user", "2030-08-01", "2030-08-03", 1)
	id, err := repo.CreateReservation(ctx, reservation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.idempotencyCollection().InsertOne(ctx, bson.M{"key": "legacy-key", "user_id": "legacy-user", "done": true, "status": 201, "body": []byte(`{"data":{"id":"` + id + `"}}`), "created_at": time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AuditInventory(ctx); !errors.Is(err, hotelsDomain.ErrLegacyIdempotency) {
		t.Fatalf("legacy not detected: %v", err)
	}
	if err := container.CopyFileToContainer(ctx, "../../../seed/audit-reservations.js", "/audit-reservations.js", 0444); err != nil {
		t.Fatal(err)
	}
	run := func(apply bool) int {
		t.Helper()
		flag := "0"
		if apply {
			flag = "1"
		}
		code, output, err := container.Exec(ctx, []string{"mongosh", "--quiet", "--eval", fmt.Sprintf(`process.env.MONGO_DATABASE=%q;process.env.AUDIT_APPLY=%q;load('/audit-reservations.js')`, cfg.Database, flag)})
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(output)
		t.Logf("migration apply=%v exit=%d: %s", apply, code, data)
		return code
	}
	if code := run(false); code != 0 {
		t.Fatalf("dry run failed: %d", code)
	}
	if err := repo.AuditInventory(ctx); !errors.Is(err, hotelsDomain.ErrLegacyIdempotency) {
		t.Fatal("dry run mutated key")
	}
	for range 2 {
		if code := run(true); code != 0 {
			t.Fatalf("idempotent migration failed: %d", code)
		}
	}
	replay, err := repo.CreateReservation(hotelsDomain.WithIdempotency(ctx, "legacy-key"), reservation)
	if err != nil || replay != id {
		t.Fatalf("migrated replay not recovered: %s %v", replay, err)
	}
	if err := repo.AuditInventory(ctx); err != nil {
		t.Fatal(err)
	}
	inventory := repo.client.Database(cfg.Database).Collection(cfg.Collection_inventory)
	for _, bad := range []int{0, -1, 2} {
		if _, err := inventory.UpdateOne(ctx, bson.M{"hotel_id": hotel, "date": "2030-08-01"}, bson.M{"$set": bson.M{"booked": bad}}); err != nil {
			t.Fatal(err)
		}
		if err := repo.AuditInventory(ctx); !errors.Is(err, hotelsDomain.ErrInventoryInconsistent) {
			t.Fatalf("bad counter %d not detected: %v", bad, err)
		}
		if code := run(true); code == 0 {
			t.Fatal("migration accepted corrupt inventory")
		}
		var unchanged hotelsDAO.Inventory
		if err := inventory.FindOne(ctx, bson.M{"hotel_id": hotel, "date": "2030-08-01"}).Decode(&unchanged); err != nil || unchanged.Booked != bad {
			t.Fatalf("migration arbitrarily repaired inventory %+v %v", unchanged, err)
		}
	}
	if _, err := inventory.DeleteOne(ctx, bson.M{"hotel_id": hotel, "date": "2030-08-01"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AuditInventory(ctx); !errors.Is(err, hotelsDomain.ErrInventoryInconsistent) {
		t.Fatalf("missing counter not detected: %v", err)
	}
}
