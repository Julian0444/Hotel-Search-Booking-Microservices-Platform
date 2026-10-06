package hotels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
)

// newTestHTTP apunta el repositorio HTTP a un httptest.Server.
func newTestHTTP(t *testing.T, server *httptest.Server) HTTP {
	t.Helper()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing test server url: %v", err)
	}
	return NewHTTP(HTTPConfig{Host: parsed.Hostname(), Port: parsed.Port()})
}

// RV14: el 404 de hotels-api se tipifica como ErrHotelNotFound y NO se reintenta
func TestGetHotelByID_404IsTypedAndNotRetried(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	repo := newTestHTTP(t, server)
	_, err := repo.GetHotelByID(context.Background(), "gone")

	if !errors.Is(err, hotelsDomain.ErrHotelNotFound) {
		t.Fatalf("expected ErrHotelNotFound, got %v", err)
	}
	if requests != 1 {
		t.Fatalf("404 must not be retried: got %d requests", requests)
	}
}

// Un intento por llamada; el consumer controla espera/reintentos. Recuperar
// hotels-api funciona inmediatamente, sin esperar una ventana de breaker.
func TestGetHotelByID_TransientFailureAndRecovery(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") != "trace" {
			t.Error("missing request ID")
		}
		if r.Header.Get("Cache-Control") != "no-cache" {
			t.Error("missing fresh-read request")
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, `{"data":{"id":"h1","name":"Recovered"}}`)
	}))
	defer server.Close()
	repo := newTestHTTP(t, server)
	ctx := utils.WithRequestID(context.Background(), "trace")
	if _, err := repo.GetHotelByID(ctx, "h1"); err == nil {
		t.Fatal("expected outage")
	}
	if requests.Load() != 1 {
		t.Fatal("HTTP retries must be owned by consumer")
	}
	hotel, err := repo.GetHotelByID(ctx, "h1")
	if err != nil || hotel.Name != "Recovered" {
		t.Fatalf("recovery: %+v, %v", hotel, err)
	}
}

func TestGetHotelsAfter_ParsesEnvelopeAndUsesCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/hotels" || r.URL.Query().Get("limit") != "50" || r.URL.Query().Get("after_id") != "h1" || r.URL.Query().Has("offset") {
			t.Errorf("unexpected keyset request: %s", r.URL)
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":"h2","name":"Dos"}],"meta":{"total":7}}`)
	}))
	defer server.Close()
	repo := newTestHTTP(t, server)
	hotels, err := repo.GetHotelsAfter(context.Background(), 50, "h1")
	if err != nil || len(hotels) != 1 || hotels[0].Name != "Dos" {
		t.Fatalf("unexpected page: %+v %v", hotels, err)
	}
}

func TestHTTP_Cancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestHTTP(t, server).GetHotelByID(ctx, "h1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

// Guarda el contrato desde el lado consumidor: la respuesta JSON de
// hotels-api (fixture) debe deserializar en el Hotel compartido con los
// campos poblados. Complementa el golden test de platform-contracts.
func TestHotelResponseUnmarshal(t *testing.T) {
	raw, err := os.ReadFile("testdata/hotel_response.json")
	if err != nil {
		t.Fatalf("reading hotel response fixture: %v", err)
	}

	var hotel hotelsDomain.Hotel
	if err := json.Unmarshal(raw, &hotel); err != nil {
		t.Fatalf("unmarshaling hotel response: %v", err)
	}

	if hotel.ID != "68d9f3a2b4c1e5f6a7b8c9d0" {
		t.Errorf("ID: got %q", hotel.ID)
	}
	if hotel.Name != "Hotel Sierras de Córdoba" {
		t.Errorf("Name: got %q", hotel.Name)
	}
	if hotel.PricePerNight != 150.5 {
		t.Errorf("PricePerNight: got %v", hotel.PricePerNight)
	}
	if hotel.AvailableRooms != 20 {
		t.Errorf("AvailableRooms: got %d", hotel.AvailableRooms)
	}
	if len(hotel.Amenities) != 3 {
		t.Errorf("Amenities: got %d, want 3", len(hotel.Amenities))
	}
	// RV21: horas del día como "HH:mm" planas
	if hotel.CheckInTime != "15:00" {
		t.Errorf("CheckInTime: got %q, want 15:00", hotel.CheckInTime)
	}
}

// Benchmark de la serialización de una página de resultados de búsqueda
// (el hot path de GET /search: Gin hace json.Marshal de []Hotel).
func BenchmarkSearchResultsMarshal(b *testing.B) {
	raw, err := os.ReadFile("testdata/hotel_response.json")
	if err != nil {
		b.Fatalf("reading fixture: %v", err)
	}
	var hotel hotelsDomain.Hotel
	if err := json.Unmarshal(raw, &hotel); err != nil {
		b.Fatalf("unmarshaling fixture: %v", err)
	}

	// Página típica: 20 resultados
	results := make([]hotelsDomain.Hotel, 20)
	for i := range results {
		results[i] = hotel
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(results); err != nil {
			b.Fatal(err)
		}
	}
}
