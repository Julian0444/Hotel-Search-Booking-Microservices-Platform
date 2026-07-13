package hotels

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
)

func newTestCache() Cache {
	return NewCache(CacheConfig{MaxSize: 1000, ItemsToPrune: 10, Duration: time.Minute})
}

// RV1: una escritura individual NUNCA crea listas agregadas parciales — las
// tres listas del par hotel/usuario deben ser MISS para que la próxima
// lectura vaya a Mongo por la lista completa.
func TestCacheCreateReservation_DoesNotCreatePartialLists(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	r := hotelsDAO.Reservation{ID: "r1", HotelID: "h1", UserID: "u1", Status: hotelsDAO.StatusConfirmed}
	if _, err := cache.CreateReservation(ctx, r); err != nil {
		t.Fatalf("creating reservation: %v", err)
	}

	if _, err := cache.GetReservationsByHotelID(ctx, "h1", 20, 0); err == nil {
		t.Error("hotel list must be a MISS after an individual write (RV1)")
	}
	if _, err := cache.GetReservationsByUserID(ctx, "u1", 20, 0); err == nil {
		t.Error("user list must be a MISS after an individual write (RV1)")
	}
	if _, err := cache.GetReservationsByUserAndHotelID(ctx, "h1", "u1", 20, 0); err == nil {
		t.Error("user+hotel list must be a MISS after an individual write (RV1)")
	}

	// La copia individual sí queda cacheada
	if _, err := cache.GetReservationByID(ctx, "r1"); err != nil {
		t.Errorf("individual reservation must be cached: %v", err)
	}
}

// RV1 bis: una escritura invalida las listas completas ya cacheadas — el
// getter debe dar MISS (repoblar desde Mongo), nunca servir la lista vieja.
func TestCacheCreateReservation_InvalidatesStoredLists(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	r1 := hotelsDAO.Reservation{ID: "r1", HotelID: "h1", UserID: "u1", Status: hotelsDAO.StatusConfirmed}
	r2 := hotelsDAO.Reservation{ID: "r2", HotelID: "h2", UserID: "u1", Status: hotelsDAO.StatusConfirmed}
	cache.SetReservationsByUserID(ctx, "u1", []hotelsDAO.Reservation{r1, r2})

	if list, err := cache.GetReservationsByUserID(ctx, "u1", 20, 0); err != nil || len(list) != 2 {
		t.Fatalf("expected stored full list of 2, got %d items, err=%v", len(list), err)
	}

	r3 := hotelsDAO.Reservation{ID: "r3", HotelID: "h1", UserID: "u1", Status: hotelsDAO.StatusConfirmed}
	if _, err := cache.CreateReservation(ctx, r3); err != nil {
		t.Fatalf("creating reservation: %v", err)
	}

	if _, err := cache.GetReservationsByUserID(ctx, "u1", 20, 0); err == nil {
		t.Error("user list must be invalidated after a new reservation (RV1)")
	}
}

// RV3 (vía el path del service): setear la copia cancelada con
// CreateReservation invalida las listas aunque la key individual no estuviera.
func TestCacheCancelledCopy_InvalidatesListsOnMiss(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	confirmed := hotelsDAO.Reservation{ID: "r1", HotelID: "h1", UserID: "u1", Status: hotelsDAO.StatusConfirmed}
	cache.SetReservationsByHotelID(ctx, "h1", []hotelsDAO.Reservation{confirmed})

	// La key individual se evicta (LRU simulada); la lista sigue viva
	cache.client.Delete("reservation:r1")

	// El service setea la copia cancelada que devolvió Mongo
	now := time.Now().UTC()
	cancelled := confirmed
	cancelled.Status = hotelsDAO.StatusCancelled
	cancelled.CancelledAt = &now
	if _, err := cache.CreateReservation(ctx, cancelled); err != nil {
		t.Fatalf("setting cancelled copy: %v", err)
	}

	// La lista con la copia confirmed NO puede sobrevivir (RV3)
	if _, err := cache.GetReservationsByHotelID(ctx, "h1", 20, 0); err == nil {
		t.Error("hotel list with stale confirmed copy must be invalidated (RV3)")
	}
	got, err := cache.GetReservationByID(ctx, "r1")
	if err != nil {
		t.Fatalf("cancelled copy must be cached: %v", err)
	}
	if got.Status != hotelsDAO.StatusCancelled {
		t.Errorf("expected cancelled status, got %q", got.Status)
	}
}

// RV2: el setter guarda una COPIA — mutar el slice del llamador después no
// afecta lo cacheado.
func TestCacheStoreList_CopiesInput(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	input := []hotelsDAO.Reservation{
		{ID: "r1", HotelID: "h1", UserID: "u1", Status: hotelsDAO.StatusConfirmed},
		{ID: "r2", HotelID: "h1", UserID: "u2", Status: hotelsDAO.StatusConfirmed},
	}
	cache.SetReservationsByHotelID(ctx, "h1", input)

	// El llamador muta su slice tras el set
	input[0].Status = hotelsDAO.StatusCancelled

	list, err := cache.GetReservationsByHotelID(ctx, "h1", 20, 0)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected cached list of 2, got %d items, err=%v", len(list), err)
	}
	if list[0].Status != hotelsDAO.StatusConfirmed {
		t.Error("cached list must be a copy of the caller's slice (RV2)")
	}
}

// RV2: escrituras y lecturas concurrentes sobre el mismo hotel corren limpias
// bajo -race (la versión anterior mutaba in-place slices compartidos con los
// lectores; este test la hacía fallar).
func TestCacheReservationLists_ConcurrentReadWrite(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	const hotelID = "race-hotel"
	base := make([]hotelsDAO.Reservation, 5)
	for i := range base {
		base[i] = hotelsDAO.Reservation{
			ID: fmt.Sprintf("base-%d", i), HotelID: hotelID, UserID: "reader", Status: hotelsDAO.StatusConfirmed,
		}
	}
	cache.SetReservationsByHotelID(ctx, hotelID, base)

	var wg sync.WaitGroup
	for worker := 0; worker < 9; worker++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				switch n % 3 {
				case 0: // re-publica la lista completa
					cache.SetReservationsByHotelID(ctx, hotelID, base)
				case 1: // escritura individual: invalida
					_, _ = cache.CreateReservation(ctx, hotelsDAO.Reservation{
						ID: fmt.Sprintf("w%d-%d", n, i), HotelID: hotelID, UserID: "writer", Status: hotelsDAO.StatusConfirmed,
					})
				default: // lector: recorre la página devuelta
					if list, err := cache.GetReservationsByHotelID(ctx, hotelID, 3, 0); err == nil {
						for _, r := range list {
							_ = r.Status
						}
					}
				}
			}
		}(worker)
	}
	wg.Wait()
}
