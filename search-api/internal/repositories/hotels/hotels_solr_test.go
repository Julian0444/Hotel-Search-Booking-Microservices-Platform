package hotels

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stevenferrer/solr-go"
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
	if got := int(getFloatField(doc, "available_rooms")); got != 20 {
		t.Errorf("getFloatField(available_rooms): got %d, want 20", got)
	}

	// RV21: check_in_time/check_out_time son strings "HH:mm" planos — salen
	// por getStringField (el getTimeField de E6 quedó obsoleto con el cambio
	// de contrato del plan 07).
	if got := getStringField(doc, "check_in_time"); got != "15:00" {
		t.Errorf("getStringField(check_in_time): got %q, want 15:00", got)
	}
	if got := getStringField(doc, "check_out_time"); got != "10:00" {
		t.Errorf("getStringField(check_out_time): got %q, want 10:00", got)
	}
}

// E2: la query del usuario nunca se interpola — viaja como parámetro $qq con
// los metacaracteres Lucene escapados, y rows/start (limit/offset) son reales.
func TestBuildSearchQuery(t *testing.T) {
	t.Run("multi-word query goes as dereferenced param", func(t *testing.T) {
		qm := buildSearchQuery("hotel spa", "", 10, 20).BuildQuery()

		if got := qm["query"]; got != "{!edismax qf='name description' v=$qq}" {
			t.Errorf("query: got %v", got)
		}
		params, ok := qm["params"].(solr.M)
		if !ok {
			t.Fatalf("params: unexpected type %T", qm["params"])
		}
		if got := params["qq"]; got != "hotel spa" {
			t.Errorf("qq: got %q, want %q (los espacios no se escapan)", got, "hotel spa")
		}
		if got := qm["limit"]; got != 10 {
			t.Errorf("limit: got %v, want 10", got)
		}
		if got := qm["offset"]; got != 20 {
			t.Errorf("offset: got %v, want 20", got)
		}
	})

	t.Run("empty query is match-all", func(t *testing.T) {
		qm := buildSearchQuery("   ", "", 10, 0).BuildQuery()

		if got := qm["query"]; got != "*:*" {
			t.Errorf("query: got %v, want *:*", got)
		}
		if _, hasParams := qm["params"]; hasParams {
			t.Error("match-all must not carry user params")
		}
	})

	t.Run("lucene metacharacters are escaped", func(t *testing.T) {
		qm := buildSearchQuery(`"(malicious:*"`, "", 10, 0).BuildQuery()

		params := qm["params"].(solr.M)
		want := `\"\(malicious\:\*\"`
		if got := params["qq"]; got != want {
			t.Errorf("qq: got %q, want %q", got, want)
		}
	})

	// plan 13 (F13-03): el sort whitelisteado se mapea a cláusulas fijas de
	// Solr con `id asc` de desempate (paginación estable); relevance/desconocido
	// no agregan cláusula (score de edismax)
	t.Run("sort maps to fixed solr clauses", func(t *testing.T) {
		cases := []struct{ sort, want string }{
			{"price_asc", "price_per_night asc, id asc"},
			{"price_desc", "price_per_night desc, id asc"},
			{"rating_desc", "rating desc, id asc"},
		}
		for _, c := range cases {
			qm := buildSearchQuery("spa", c.sort, 10, 0).BuildQuery()
			if got := qm["sort"]; got != c.want {
				t.Errorf("sort=%s: got %v, want %q", c.sort, got, c.want)
			}
		}
	})

	t.Run("relevance and unknown sorts add no clause", func(t *testing.T) {
		for _, sort := range []string{"", "relevance", "drop table"} {
			qm := buildSearchQuery("spa", sort, 10, 0).BuildQuery()
			if _, hasSort := qm["sort"]; hasSort {
				t.Errorf("sort=%q: unexpected sort clause %v", sort, qm["sort"])
			}
		}
	})

	t.Run("sort also applies to match-all", func(t *testing.T) {
		qm := buildSearchQuery("", "price_asc", 10, 0).BuildQuery()
		if got := qm["sort"]; got != "price_per_night asc, id asc" {
			t.Errorf("sort: got %v", got)
		}
	})
}

func TestEscapeSolrQuery(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain words", "plain words"},
		{"a&&b||c", `a\&\&b\|\|c`},
		{`path/to\thing`, `path\/to\\thing`},
		{"wild*card?", `wild\*card\?`},
		{"boost^2 fuzzy~ -not +must", `boost\^2 fuzzy\~ \-not \+must`},
		{"[a TO b] {c}", `\[a TO b\] \{c\}`},
	}
	for _, c := range cases {
		if got := escapeSolrQuery(c.in); got != c.want {
			t.Errorf("escapeSolrQuery(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}
