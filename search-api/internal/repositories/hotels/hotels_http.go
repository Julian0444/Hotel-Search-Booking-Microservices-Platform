package hotels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/sony/gobreaker/v2"
)

const (
	// requestTimeout acota cada request a hotels-api (E4): antes se usaba el
	// http.DefaultClient sin timeout y un upstream colgado congelaba el consumer.
	requestTimeout = 5 * time.Second
	// maxAttempts reintenta errores de conexión y 5xx (hotels-api puede estar
	// arrancando); un 404 NUNCA se reintenta — es una respuesta, no un fallo.
	maxAttempts  = 3
	retryBackoff = 500 * time.Millisecond
	// retryJitter desincroniza los reintentos (R5): sin él, réplicas o
	// mensajes concurrentes martillan al upstream en la misma cadencia.
	retryJitter = 250 * time.Millisecond
	// breakerOpenTimeout: cuánto queda abierto el breaker antes de probar en
	// half-open (R5).
	breakerOpenTimeout = 30 * time.Second
)

type HTTPConfig struct {
	Host string
	Port string
}

// httpResult es lo que viaja por el circuit breaker: status + body ya leído
// (el error del Execute queda reservado para fallos REALES del upstream).
type httpResult struct {
	status int
	body   []byte
}

type HTTP struct {
	baseURL string
	client  *http.Client
	breaker *gobreaker.CircuitBreaker[httpResult]
}

func NewHTTP(config HTTPConfig) HTTP {
	return HTTP{
		baseURL: fmt.Sprintf("http://%s:%s", config.Host, config.Port),
		// Un único client reutilizado (pool de conexiones) con timeout total
		client:  &http.Client{Timeout: requestTimeout},
		breaker: newHotelsBreaker(breakerOpenTimeout),
	}
}

// newHotelsBreaker arma el circuit breaker del hop search→hotels (R5), el
// único hop HTTP inter-servicio: abre tras 5 fallos consecutivos (deja de
// martillar a un upstream caído), tras openTimeout pasa a half-open y deja
// pasar hasta 3 requests de prueba antes de volver a cerrar.
func newHotelsBreaker(openTimeout time.Duration) *gobreaker.CircuitBreaker[httpResult] {
	return gobreaker.NewCircuitBreaker[httpResult](gobreaker.Settings{
		Name:        "hotels-api",
		MaxRequests: 3,
		Timeout:     openTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state change",
				"breaker", name, "from", from.String(), "to", to.String())
		},
	})
}

// getJSON hace GET con context, X-Request-ID (O1) y reintentos acotados (E4)
// con jitter tras el circuit breaker (R5), y decodifica la respuesta en out.
// Tipifica el 404 como ErrHotelNotFound (RV14).
func (repository HTTP) getJSON(ctx context.Context, url string, out any) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Backoff lineal corto + jitter entre reintentos, respetando la
			// cancelación
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt-1)*retryBackoff + rand.N(retryJitter)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("error building request (%s): %w", url, err)
		}
		if requestID := utils.RequestIDFromContext(ctx); requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}

		// El breaker (R5) envuelve cada intento: cuentan como fallo suyo los
		// errores de conexión, la lectura del body y los 5xx — un 404 (o
		// cualquier 4xx) es una respuesta válida del upstream, no una falla.
		result, err := repository.breaker.Execute(func() (httpResult, error) {
			resp, err := repository.client.Do(req)
			if err != nil {
				return httpResult{}, fmt.Errorf("error fetching %s: %w", url, err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return httpResult{}, fmt.Errorf("error reading response body (%s): %w", url, err)
			}
			if resp.StatusCode >= http.StatusInternalServerError {
				return httpResult{}, fmt.Errorf("fetching %s: received status code %d", url, resp.StatusCode)
			}
			return httpResult{status: resp.StatusCode, body: body}, nil
		})
		if err != nil {
			// Breaker abierto: hotels-api viene fallando — fail-fast sin
			// reintentos que lo martillen (half-open prueba solo tras el
			// timeout del breaker)
			if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
				return fmt.Errorf("hotels-api circuit breaker open (%s): %w", url, err)
			}
			lastErr = err
			continue // conexión/5xx: transitorio, reintentar
		}

		switch {
		case result.status == http.StatusNotFound:
			// 404 tipado (RV14): el recurso no existe — sin reintentos
			return fmt.Errorf("fetching %s: %w", url, hotelsDomain.ErrHotelNotFound)
		case result.status != http.StatusOK:
			return fmt.Errorf("fetching %s: received status code %d", url, result.status)
		}

		if err := json.Unmarshal(result.body, out); err != nil {
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
