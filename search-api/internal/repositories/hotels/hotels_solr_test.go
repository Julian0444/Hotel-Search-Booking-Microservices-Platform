package hotels

import (
	"encoding/json"
	"os"
	"testing"
)

// loadSolrDoc carga el fixture de un documento Solr tal como lo devuelve la
// respuesta de búsqueda (fechas como strings — Solr no devuelve time.Time).
func loadSolrDoc(t *testing.T) map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile("testdata/solr_hotel_doc.json")
	if err != nil {
		t.Fatalf("reading solr doc fixture: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshaling solr doc fixture: %v", err)
	}
	return doc
}

func TestSolrFieldHelpers(t *testing.T) {
	doc := loadSolrDoc(t)

	if got := getStringField(doc, "name"); got != "Hotel Sierras de Córdoba" {
		t.Errorf("getStringField(name): got %q", got)
	}
	if got := getStringField(doc, "nonexistent"); got != "" {
		t.Errorf("getStringField(nonexistent): got %q, want empty", got)
	}
	if got := getFloatField(doc, "price_per_night"); got != 150.5 {
		t.Errorf("getFloatField(price_per_night): got %v, want 150.5", got)
	}
	if got := int(getFloatField(doc, "avaiable_rooms")); got != 20 {
		t.Errorf("getFloatField(avaiable_rooms): got %d, want 20", got)
	}

	// Comportamiento actual (E6, se corrige en el plan 06): Solr devuelve las
	// fechas como string y el type-assert a time.Time nunca matchea, así que
	// getTimeField devuelve el zero value. Este assert documenta el bug; al
	// arreglar E6 hay que invertirlo (esperar la fecha parseada del fixture).
	if got := getTimeField(doc, "check_in_time"); !got.IsZero() {
		t.Errorf("getTimeField(check_in_time): got %v, want zero (comportamiento pre-E6)", got)
	}
}
