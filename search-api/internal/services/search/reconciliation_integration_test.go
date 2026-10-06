//go:build integration

package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	dao "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	domain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/integrationtest"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/repositories/hotels"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/services/search"
	"github.com/stretchr/testify/require"
)

// Source HTTP controlable: permite situar las escrituras exactamente entre dos
// páginas o entre GET y escritura Solr. Solr y su cursor son reales.
type catalogue struct {
	mu       sync.Mutex
	hotels   map[string]domain.Hotel
	pageHook func(string)
	getHook  func(string)
}

func (c *catalogue) server(t *testing.T) repositories.HTTP {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		if r.URL.Path == "/api/v1/hotels" {
			after := r.URL.Query().Get("after_id")
			var ids []string
			for id := range c.hotels {
				if id > after {
					ids = append(ids, id)
				}
			}
			sort.Strings(ids)
			page := []domain.Hotel{}
			if len(ids) > 0 {
				page = append(page, c.hotels[ids[0]])
			} // fuerza paginación
			hook := c.pageHook
			c.mu.Unlock()
			if hook != nil {
				hook(after)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": page, "meta": map[string]int{"total": 999}})
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/hotels/")
		hotel, ok := c.hotels[id]
		hook := c.getHook
		c.mu.Unlock()
		if hook != nil {
			hook(id)
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": hotel})
	}))
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	return repositories.NewHTTP(repositories.HTTPConfig{Host: u.Hostname(), Port: u.Port()})
}

func TestReconcileRealSolrLostDeleteAndChangingKeysetPages(t *testing.T) {
	repo, _ := integrationtest.NewSolr(t)
	ctx := context.Background()
	for _, h := range []dao.Hotel{{ID: "h1", Name: "Old one"}, {ID: "h2", Name: "Old two"}, {ID: "orphan", Name: "Lost DELETE"}} {
		_, err := repo.Index(ctx, h)
		require.NoError(t, err)
	}
	source := &catalogue{hotels: map[string]domain.Hotel{"h1": {ID: "h1", Name: "One"}, "h2": {ID: "h2", Name: "Two"}}}
	source.pageHook = func(after string) {
		if after == "" {
			source.mu.Lock()
			defer source.mu.Unlock()
			delete(source.hotels, "h1")
			source.hotels["h2"] = domain.Hotel{ID: "h2", Name: "Updated"}
			source.hotels["h3"] = domain.Hotel{ID: "h3", Name: "Created during scan"}
		}
	}
	svc := services.NewService(repo, source.server(t))
	count, err := svc.Backfill(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	ids, err := repo.ListIDs(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"h2", "h3"}, ids)
	docs, total, err := repo.Search(ctx, "Updated", "", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, "h2", docs[0].ID)
}

func TestReconcileRealSolrSerializesEventsAgainstStaleInFlightGET(t *testing.T) {
	for _, operation := range []string{"DELETE", "UPDATE", "CREATE"} {
		t.Run(operation, func(t *testing.T) {
			repo, _ := integrationtest.NewSolr(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			source := &catalogue{hotels: map[string]domain.Hotel{"h1": {ID: "h1", Name: "Old"}}}
			readStarted, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			source.getHook = func(id string) {
				if id == "h1" {
					once.Do(func() {
						close(readStarted)
						select {
						case <-release:
						case <-ctx.Done():
						}
					})
				}
			}
			svc := services.NewService(repo, source.server(t))
			backfillDone := make(chan error, 1)
			go func() { _, err := svc.Backfill(ctx); backfillDone <- err }()
			select {
			case <-readStarted:
			case <-ctx.Done():
				t.Fatal("reconciliation did not reach GET")
			}
			source.mu.Lock()
			id := "h1"
			switch operation {
			case "DELETE":
				delete(source.hotels, "h1")
			case "UPDATE":
				source.hotels["h1"] = domain.Hotel{ID: "h1", Name: "New"}
			case "CREATE":
				id = "h2"
				source.hotels[id] = domain.Hotel{ID: id, Name: "New"}
			}
			source.mu.Unlock()
			eventStarted, eventDone := make(chan struct{}), make(chan error, 1)
			go func() {
				close(eventStarted)
				eventDone <- svc.HandleHotelNew(ctx, domain.HotelNew{Operation: operation, HotelID: id})
			}()
			<-eventStarted
			select {
			case err := <-eventDone:
				t.Fatalf("event overtook in-flight reconciliation: %v", err)
			case <-time.After(30 * time.Millisecond):
			}
			close(release)
			require.NoError(t, <-backfillDone)
			require.NoError(t, <-eventDone)
			ids, err := repo.ListIDs(ctx)
			require.NoError(t, err)
			source.mu.Lock()
			expected := make([]string, 0, len(source.hotels))
			for id := range source.hotels {
				expected = append(expected, id)
			}
			source.mu.Unlock()
			sort.Strings(expected)
			require.Equal(t, expected, ids)
			if operation != "DELETE" {
				docs, total, err := repo.Search(ctx, "New", "", 10, 0)
				require.NoError(t, err)
				require.Equal(t, 1, total)
				require.Equal(t, id, docs[0].ID)
			}
		})
	}
}
