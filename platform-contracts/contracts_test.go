package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// Contract test productor↔consumidor del evento HotelNew (cola hotels-news).
// El golden fija el wire format: si un tag JSON cambia, este test falla en CI
// antes de que el cambio rompa RabbitMQ en runtime. El golden solo se actualiza
// deliberadamente como parte de un cambio de contrato coordinado (ver C11).
func TestHotelNewWireFormat(t *testing.T) {
	golden, err := os.ReadFile("testdata/hotel_new.golden.json")
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	golden = bytes.TrimSpace(golden)

	// (a) Lado productor: serializar debe producir exactamente el golden.
	event := HotelNew{Operation: "CREATE", HotelID: "abc-123"}
	got, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshaling HotelNew: %v", err)
	}
	if !bytes.Equal(got, golden) {
		t.Errorf("wire format drifted:\n got:    %s\n golden: %s", got, golden)
	}

	// (b) Lado consumidor: deserializar el golden debe poblar todos los campos.
	var decoded HotelNew
	if err := json.Unmarshal(golden, &decoded); err != nil {
		t.Fatalf("unmarshaling golden: %v", err)
	}
	if decoded.Operation != "CREATE" {
		t.Errorf("Operation: got %q, want %q", decoded.Operation, "CREATE")
	}
	if decoded.HotelID != "abc-123" {
		t.Errorf("HotelID: got %q, want %q", decoded.HotelID, "abc-123")
	}
}
