//go:build integration

package hotels_test

import (
	"context"
	"fmt"
	"testing"

	dao "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/integrationtest"
	"github.com/stretchr/testify/require"
)

func TestSolrCanonicalDestinationsAccentsPaginationAndOriginalText(t *testing.T) {
	repo, _ := integrationtest.NewSolr(t)
	ctx := context.Background()
	// Mismos nombre/destino/precio que hotels-api/seed/mongo-init.js.
	fixture := []dao.Hotel{
		{ID: "1", Name: "Hotel Sierras de Córdoba", City: "Córdoba", Country: "Argentina", PricePerNight: 95},
		{ID: "2", Name: "Palermo Soho Suites", City: "Buenos Aires", Country: "Argentina", PricePerNight: 140},
		{ID: "3", Name: "Posada del Vino", City: "Mendoza", Country: "Argentina", PricePerNight: 110},
		{ID: "4", Name: "Refugio del Lago", City: "Bariloche", Country: "Argentina", PricePerNight: 180},
		{ID: "5", Name: "Hostal de la Quebrada", City: "Salta", Country: "Argentina", PricePerNight: 70},
	}
	for _, h := range fixture {
		_, err := repo.Index(ctx, h)
		require.NoError(t, err)
	}
	ids, err := repo.ListIDs(ctx)
	require.NoError(t, err)
	require.Len(t, ids, 5)
	for _, tc := range []struct{ query, name string }{
		{"Mendoza", "Posada del Vino"}, {"Bariloche", "Refugio del Lago"}, {"Salta", "Hostal de la Quebrada"}, {"Buenos Aires", "Palermo Soho Suites"},
		{"Córdoba", "Hotel Sierras de Córdoba"}, {"cordoba", "Hotel Sierras de Córdoba"}, {"CORDOBA", "Hotel Sierras de Córdoba"}, {"sierras", "Hotel Sierras de Córdoba"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			docs, total, err := repo.Search(ctx, tc.query, "", 20, 0)
			require.NoError(t, err)
			require.Equal(t, 1, total)
			require.Equal(t, tc.name, docs[0].Name)
		})
	}
	docs, total, err := repo.Search(ctx, "Argentina", "price_asc", 2, 2)
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Len(t, docs, 2)
	require.Equal(t, "Posada del Vino", docs[0].Name)
	require.Equal(t, "Palermo Soho Suites", docs[1].Name)
	for _, query := range []string{`*: *`, `" OR *:*`, `{!lucene}*:*`, `Argentina&sort=price_desc`} {
		_, _, err := repo.Search(ctx, query, "", 20, 0)
		require.NoError(t, err, fmt.Sprintf("literal query %q", query))
	}
}
