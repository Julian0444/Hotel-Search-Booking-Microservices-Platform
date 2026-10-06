package hotels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
)

// El consumer controla los reintentos y su demora. Cada intento HTTP tiene
// un único presupuesto, sin breaker ni backoffs anidados.
const requestTimeout = 5 * time.Second

type HTTPConfig struct{ Host, Port string }

type HTTP struct {
	baseURL string
	client  *http.Client
}

func NewHTTP(config HTTPConfig) HTTP {
	return HTTP{baseURL: fmt.Sprintf("http://%s:%s", config.Host, config.Port), client: &http.Client{Timeout: requestTimeout}}
}

func (repository HTTP) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	if id := utils.RequestIDFromContext(ctx); id != "" {
		req.Header.Set("X-Request-ID", id)
	}
	resp, err := repository.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching hotels-api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return hotelsDomain.ErrHotelNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hotels-api returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding hotels-api: %w", err)
	}
	return nil
}

// hotelEnvelope es el envelope estándar {data} de los gets de hotels-api (A5).
type hotelEnvelope struct {
	Data hotelsDomain.Hotel `json:"data"`
}

func (repository HTTP) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	var envelope hotelEnvelope
	url := fmt.Sprintf("%s/api/v1/hotels/%s", repository.baseURL, url.PathEscape(id))
	if err := repository.getJSON(ctx, url, &envelope); err != nil {
		return hotelsDomain.Hotel{}, fmt.Errorf("error fetching hotel (%s): %w", id, err)
	}
	return envelope.Data, nil
}

// hotelsPage es el envelope estándar {data, meta} de GET /api/v1/hotels (A5).
type hotelsPage struct {
	Data []hotelsDomain.Hotel `json:"data"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

// GetHotelsAfter recorre el catálogo con keyset: un DELETE anterior al cursor
// no desplaza la página siguiente. El total no es un snapshot y no se usa.
func (repository HTTP) GetHotelsAfter(ctx context.Context, limit int, after string) ([]hotelsDomain.Hotel, error) {
	var page hotelsPage
	url := fmt.Sprintf("%s/api/v1/hotels?limit=%d&after_id=%s", repository.baseURL, limit, url.QueryEscape(after))
	if err := repository.getJSON(ctx, url, &page); err != nil {
		return nil, fmt.Errorf("fetching hotels page: %w", err)
	}
	return page.Data, nil
}
