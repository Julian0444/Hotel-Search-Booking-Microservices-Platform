package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	repositoriesHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"

	"github.com/gin-gonic/gin"
)

// fakeIdempotencyStore reproduce la semántica del store de Mongo en memoria:
// insert único por (key, user_id), complete y release.
type fakeIdempotencyStore struct {
	mu      sync.Mutex
	records map[string]*repositoriesHotels.IdempotencyRecord
	fail    bool
}

func newFakeStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{records: map[string]*repositoriesHotels.IdempotencyRecord{}}
}

func storeKey(key, userID string) string { return key + "|" + userID }

func (s *fakeIdempotencyStore) ReserveIdempotencyKey(_ context.Context, key, userID string) (bool, repositoriesHotels.IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return false, repositoriesHotels.IdempotencyRecord{}, fmt.Errorf("store down")
	}
	if existing, ok := s.records[storeKey(key, userID)]; ok {
		return false, *existing, nil
	}
	record := &repositoriesHotels.IdempotencyRecord{Key: key, UserID: userID}
	s.records[storeKey(key, userID)] = record
	return true, *record, nil
}

func (s *fakeIdempotencyStore) CompleteIdempotencyKey(_ context.Context, key, userID string, status int, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.records[storeKey(key, userID)]
	record.Done = true
	record.Status = status
	record.Body = body
	return nil
}

func (s *fakeIdempotencyStore) ReleaseIdempotencyKey(_ context.Context, key, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, storeKey(key, userID))
	return nil
}

// setupIdempotencyRouter arma un POST protegido por el middleware; el handler
// cuenta ejecuciones reales para distinguir replay de re-ejecución.
func setupIdempotencyRouter(store IdempotencyStore, handlerStatus int) (*gin.Engine, *int) {
	gin.SetMode(gin.TestMode)
	executions := 0
	r := gin.New()
	r.POST("/reservations", func(c *gin.Context) {
		// Simula el userID que deja el middleware JWT
		c.Set("userID", "1")
		c.Next()
	}, Idempotency(store), func(c *gin.Context) {
		executions++
		if handlerStatus >= http.StatusInternalServerError {
			c.JSON(handlerStatus, gin.H{"error": gin.H{"code": "internal"}})
			return
		}
		c.JSON(handlerStatus, gin.H{"data": gin.H{"id": fmt.Sprintf("res-%d", executions)}})
	})
	return r, &executions
}

func postWithKey(r *gin.Engine, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// A3: misma key dos veces → misma respuesta, el handler corre UNA sola vez
func TestIdempotency_ReplayReturnsStoredResponse(t *testing.T) {
	r, executions := setupIdempotencyRouter(newFakeStore(), http.StatusCreated)

	first := postWithKey(r, "key-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("first: code=%d body=%s", first.Code, first.Body.String())
	}

	second := postWithKey(r, "key-1")
	if second.Code != http.StatusCreated {
		t.Fatalf("replay: code=%d body=%s", second.Code, second.Body.String())
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("replay must return the stored body:\n first: %s\n second: %s", first.Body.String(), second.Body.String())
	}
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected Idempotency-Replayed header on replay")
	}
	if *executions != 1 {
		t.Fatalf("handler must run exactly once, ran %d times", *executions)
	}
}

// A3: sin header el middleware es un no-op (opt-in del cliente)
func TestIdempotency_NoHeaderIsPassthrough(t *testing.T) {
	r, executions := setupIdempotencyRouter(newFakeStore(), http.StatusCreated)

	postWithKey(r, "")
	postWithKey(r, "")
	if *executions != 2 {
		t.Fatalf("without header each request must execute, ran %d times", *executions)
	}
}

// A3: replay con el original todavía en vuelo → 409 request_in_flight
func TestIdempotency_InFlightIs409(t *testing.T) {
	store := newFakeStore()
	// Reservar la key a mano sin completarla: simula el request original en vuelo
	if created, _, err := store.ReserveIdempotencyKey(context.Background(), "key-inflight", "1"); err != nil || !created {
		t.Fatalf("error seeding in-flight key: created=%v err=%v", created, err)
	}

	r, executions := setupIdempotencyRouter(store, http.StatusCreated)
	w := postWithKey(r, "key-inflight")

	if w.Code != http.StatusConflict {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusConflict, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"request_in_flight"`) {
		t.Fatalf("expected request_in_flight code, got: %s", w.Body.String())
	}
	if *executions != 0 {
		t.Fatalf("handler must not run on in-flight conflict, ran %d times", *executions)
	}
}

// A3: un 5xx del handler libera la key — el reintento del cliente re-ejecuta
func TestIdempotency_ServerErrorReleasesKey(t *testing.T) {
	store := newFakeStore()
	r, executions := setupIdempotencyRouter(store, http.StatusInternalServerError)

	first := postWithKey(r, "key-5xx")
	if first.Code != http.StatusInternalServerError {
		t.Fatalf("first: code=%d", first.Code)
	}

	second := postWithKey(r, "key-5xx")
	if second.Code != http.StatusInternalServerError {
		t.Fatalf("retry: code=%d", second.Code)
	}
	if *executions != 2 {
		t.Fatalf("after a 5xx the retry must re-execute, ran %d times", *executions)
	}
}

// A3: los 4xx SÍ se persisten — un replay del mismo request inválido no
// re-ejecuta el dominio
func TestIdempotency_ClientErrorIsStored(t *testing.T) {
	r, executions := setupIdempotencyRouter(newFakeStore(), http.StatusConflict)

	postWithKey(r, "key-409")
	replay := postWithKey(r, "key-409")

	if replay.Code != http.StatusConflict {
		t.Fatalf("replay: code=%d", replay.Code)
	}
	if *executions != 1 {
		t.Fatalf("4xx replay must not re-execute, ran %d times", *executions)
	}
}

// A3: si el store no responde, el request falla con 500 (no se puede
// garantizar la semántica de idempotencia sin el registro)
func TestIdempotency_StoreErrorIs500(t *testing.T) {
	store := newFakeStore()
	store.fail = true
	r, executions := setupIdempotencyRouter(store, http.StatusCreated)

	w := postWithKey(r, "key-err")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
	if *executions != 0 {
		t.Fatalf("handler must not run when the store fails, ran %d times", *executions)
	}
}
