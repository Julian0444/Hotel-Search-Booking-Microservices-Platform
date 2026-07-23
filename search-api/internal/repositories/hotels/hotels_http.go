package hotels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
)

const (
	// requestTimeout acota cada request a hotels-api (E4): antes se usaba el
	// http.DefaultClient sin timeout y un upstream colgado congelaba el consumer.
	requestTimeout = 5 * time.Second
	// maxAttempts reintenta errores de conexión y 5xx (hotels-api puede estar
	// arrancando); un 404 NUNCA se reintenta — es una respuesta, no un fallo.
	maxAttempts  = 3
	retryBackoff = 500 * time.Millisecond
)

type HTTPConfig struct {
	Host string
	Port string
}

type HTTP struct {
	baseURL string
	client  *http.Client
}

func NewHTTP(config HTTPConfig) HTTP {
	return HTTP{
		baseURL: fmt.Sprintf("http://%s:%s", config.Host, config.Port),
		// Un único client reutilizado (pool de conexiones) con timeout total
		client: &http.Client{Timeout: requestTimeout},
	}
}

// getJSON hace GET con context, X-Request-ID (O1) y reintentos acotados (E4),
// y decodifica la respuesta en out. Tipifica el 404 como ErrHotelNotFound (RV14).
func (repository HTTP) getJSON(ctx context.Context, url string, out any) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Backoff lineal corto entre reintentos, respetando la cancelación
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt-1) * retryBackoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("error building request (%s): %w", url, err)
		}
		if requestID := utils.RequestIDFromContext(ctx); requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}

		resp, err := repository.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error fetching %s: %w", url, err)
			continue // error de conexión/timeout: reintentar
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusNotFound:
			// 404 tipado (RV14): el recurso no existe — sin reintentos
			return fmt.Errorf("fetching %s: %w", url, hotelsDomain.ErrHotelNotFound)
		case resp.StatusCode >= http.StatusInternalServerError:
			lastErr = fmt.Errorf("fetching %s: received status code %d", url, resp.StatusCode)
			continue // 5xx: transitorio, reintentar
		case resp.StatusCode != http.StatusOK:
			return fmt.Errorf("fetching %s: received status code %d", url, resp.StatusCode)
		}

		if readErr != nil {
			lastErr = fmt.Errorf("error reading response body (%s): %w", url, readErr)
			continue
		}
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("error unmarshaling response (%s): %w", url, err)
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// hotelEnvelope es el envelope estándar {data} de los gets de hotels-api (A5).
type hotelEnvelope struct {
	Data hotelsDomain.Hotel `json:"data"`
}

func (repository HTTP) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	var envelope hotelEnvelope
	url := fmt.Sprintf("%s/api/v1/hotels/%s", repository.baseURL, id)
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

// GetHotels trae una página del catálogo de hotels-api (E3): lo consume el
// backfill/reindex para reconstruir el índice de Solr.
func (repository HTTP) GetHotels(ctx context.Context, limit, offset int) ([]hotelsDomain.Hotel, int, error) {
	var page hotelsPage
	url := fmt.Sprintf("%s/api/v1/hotels?limit=%d&offset=%d", repository.baseURL, limit, offset)
	if err := repository.getJSON(ctx, url, &page); err != nil {
		return nil, 0, fmt.Errorf("error fetching hotels page: %w", err)
	}
	return page.Data, page.Meta.Total, nil
}
