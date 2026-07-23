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

// E4: los 5xx se reintentan (acotado) y el request lleva el X-Request-ID del ctx
func TestGetHotelByID_RetriesOn5xxAndPropagatesRequestID(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.Header.Get("X-Request-ID"); got != "trace-e4" {
			t.Errorf("expected X-Request-ID header, got %q", got)
		}
		if requests < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Envelope estándar {data} de los gets (A5)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": hotelsDomain.Hotel{ID: "h1", Name: "Recovered"}})
	}))
	defer server.Close()

	repo := newTestHTTP(t, server)
	hotel, err := repo.GetHotelByID(utils.WithRequestID(context.Background(), "trace-e4"), "h1")
	if err != nil {
		t.Fatalf("expected recovery on third attempt, got %v", err)
	}
	if hotel.Name != "Recovered" || requests != 3 {
		t.Fatalf("expected 3 attempts and the recovered hotel, got %d attempts, %+v", requests, hotel)
	}
}

// E4: si todos los intentos fallan, el error sale con el conteo de intentos
func TestGetHotelByID_FailsAfterMaxAttempts(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	repo := newTestHTTP(t, server)
	_, err := repo.GetHotelByID(context.Background(), "h1")

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if requests != maxAttempts {
		t.Fatalf("expected %d attempts, got %d", maxAttempts, requests)
	}
}

// E3/A5: GetHotels pega a la ruta versionada y parsea el envelope {data, meta}
func TestGetHotels_ParsesEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/hotels" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "50" || r.URL.Query().Get("offset") != "10" {
			t.Errorf("unexpected pagination: %s", r.URL.RawQuery)
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":"h1","name":"Uno"},{"id":"h2","name":"Dos"}],"meta":{"total":7,"limit":50,"offset":10}}`)
	}))
	defer server.Close()

	repo := newTestHTTP(t, server)
	hotels, total, err := repo.GetHotels(context.Background(), 50, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 7 || len(hotels) != 2 || hotels[1].Name != "Dos" {
		t.Fatalf("unexpected page: total=%d hotels=%+v", total, hotels)
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
	if hotel.AvaiableRooms != 20 {
		t.Errorf("AvaiableRooms: got %d", hotel.AvaiableRooms)
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
