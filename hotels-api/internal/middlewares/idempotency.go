package middlewares

import (
	"net/http"
	"strings"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/apperr"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/gin-gonic/gin"
)

// Idempotency validates optional request metadata. Persistence and replay
// belong to the reservation transaction, BEFORE the HTTP response is sent.
func Idempotency() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key != "" {
			if len(key) > 200 || strings.TrimSpace(key) != key {
				apperr.Abort(c, http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key must contain 1-200 characters without surrounding whitespace", nil)
				return
			}
			c.Request = c.Request.WithContext(hotelsDomain.WithIdempotency(c.Request.Context(), key))
		}
		c.Next()
	}
}
