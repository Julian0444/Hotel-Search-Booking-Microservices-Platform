//go:build integration

// Package integrationtest crea recursos aislados, nunca vacía el core de demo.
package integrationtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/repositories/hotels"
	"github.com/google/uuid"
)

func NewSolr(t *testing.T) (repositories.Solr, string) {
	t.Helper()
	base := strings.TrimRight(os.Getenv("TEST_SOLR_URL"), "/")
	if base == "" {
		t.Skip("TEST_SOLR_URL is required for real Solr integration")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	name := "verify_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	client := &http.Client{Timeout: 20 * time.Second}
	admin := func(values url.Values) {
		t.Helper()
		resp, err := client.Post(base+"/solr/admin/cores?"+values.Encode(), "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Solr admin %v: status %d %s", values, resp.StatusCode, body)
		}
	}
	configSet := os.Getenv("TEST_SOLR_CONFIGSET")
	if configSet == "" {
		configSet = "hotels"
	}
	admin(url.Values{"action": {"CREATE"}, "name": {name}, "configSet": {configSet}, "wt": {"json"}})
	t.Cleanup(func() {
		admin(url.Values{"action": {"UNLOAD"}, "core": {name}, "deleteIndex": {"true"}, "deleteDataDir": {"true"}, "deleteInstanceDir": {"true"}, "wt": {"json"}})
	})
	repo := repositories.NewSolr(repositories.SolrConfig{Host: parsed.Hostname(), Port: parsed.Port(), Collection: name})
	if err := repo.Ping(context.Background()); err != nil {
		t.Fatal(fmt.Errorf("new core: %w", err))
	}
	return repo, base
}
