package middlewares

import (
	"bytes"
	"context"
	"net/http"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/apperr"
	repositoriesHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"

	"github.com/gin-gonic/gin"
)

// IdempotencyStore es lo que el middleware necesita del repositorio (lo
// implementa Mongo; los tests usan un fake).
type IdempotencyStore interface {
	ReserveIdempotencyKey(ctx context.Context, key, userID string) (bool, repositoriesHotels.IdempotencyRecord, error)
	CompleteIdempotencyKey(ctx context.Context, key, userID string, status int, body []byte) error
	ReleaseIdempotencyKey(ctx context.Context, key, userID string) error
}

// bodyRecorder duplica lo que el handler escribe: la respuesta sale al cliente
// normalmente y una copia queda para persistir en el store.
type bodyRecorder struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *bodyRecorder) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *bodyRecorder) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Idempotency implementa el header `Idempotency-Key` en POSTs no seguros (A3).
// La clave es (Idempotency-Key, userID) — NUNCA el X-Request-ID, que es de
// tracing y cambia por request. Semántica:
//   - sin header → comportamiento normal (el header es opt-in del cliente);
//   - primera vez → se ejecuta el handler y se persiste status+body;
//   - replay completado → se devuelve la respuesta guardada sin re-ejecutar
//     (header `Idempotency-Replayed: true`);
//   - replay con el original aún en vuelo → 409 request_in_flight;
//   - el original terminó en 5xx → la key se libera para permitir reintento.
//
// Compone con el inventario del plan 04: esto dedupea reintentos del MISMO
// cliente/key; el claim atómico evita overbooking entre usuarios distintos.
// Debe registrarse DESPUÉS de Authenticate() (necesita el userID del token).
func Idempotency(store IdempotencyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}
		userID := c.GetString("userID")

		created, existing, err := store.ReserveIdempotencyKey(c.Request.Context(), key, userID)
		if err != nil {
			apperr.Abort(c, http.StatusInternalServerError, "internal",
				"error processing idempotency key", err)
			return
		}

		if !created {
			if !existing.Done {
				apperr.Abort(c, http.StatusConflict, "request_in_flight",
					"a request with this Idempotency-Key is still being processed", nil)
				return
			}
			// Replay: misma respuesta que la primera vez, sin tocar el dominio
			c.Header("Idempotency-Replayed", "true")
			c.Data(existing.Status, "application/json; charset=utf-8", existing.Body)
			c.Abort()
			return
		}

		// Primera ejecución: capturar la respuesta para los replays.
		// WithoutCancel para persistir/liberar aunque el cliente se desconecte
		// a mitad de la escritura.
		recorder := &bodyRecorder{ResponseWriter: c.Writer}
		c.Writer = recorder
		c.Next()

		storeCtx := context.WithoutCancel(c.Request.Context())
		status := c.Writer.Status()
		if status >= http.StatusInternalServerError {
			// Fallo del server: liberar la key para no bloquear el reintento
			if err := store.ReleaseIdempotencyKey(storeCtx, key, userID); err != nil {
				_ = c.Error(err)
			}
			return
		}
		if err := store.CompleteIdempotencyKey(storeCtx, key, userID, status, recorder.body.Bytes()); err != nil {
			_ = c.Error(err)
		}
	}
}
