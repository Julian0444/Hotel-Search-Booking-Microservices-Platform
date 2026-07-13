package hotels

import (
	"encoding/json"
	"os"
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
)

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
