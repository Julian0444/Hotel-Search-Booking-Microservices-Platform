package middleware

import (
	"log/slog"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Endpoints de infraestructura que no se loguean: los pollean los
// healthchecks de docker cada pocos segundos y ensuciarían el log JSON.
var quietPaths = map[string]bool{
	"/health": true,
	"/livez":  true,
	"/readyz": true,
}

// RequestID propaga el X-Request-ID que inyecta nginx (o genera uno si el
// request llegó directo), lo devuelve en la respuesta y emite un access-log
// estructurado con el id para correlacionar la request entre el gateway y el
// servicio (O1).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		// También en el context.Context del request (RV16): el path HTTP
		// (p.ej. POST /reindex → fetch a hotels-api) propaga el id como
		// header X-Request-ID igual que el consumer.
		c.Request = c.Request.WithContext(utils.WithRequestID(c.Request.Context(), id))
		c.Writer.Header().Set("X-Request-ID", id)

		start := time.Now()
		c.Next()

		if quietPaths[c.Request.URL.Path] {
			return
		}
		// Las causas que apperr adjuntó con c.Error salen acá, correlacionadas
		// con el request_id — el body del cliente nunca las lleva (A1).
		attrs := []any{
			"request_id", id,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}
		slog.Info("request", attrs...)
	}
}
