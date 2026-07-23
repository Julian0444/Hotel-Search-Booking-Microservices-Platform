package middleware

import (
	"net/http"
	"strings"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/apperr"

	"github.com/gin-gonic/gin"
)

// RequireJSON implementa la decisión de content negotiation (A8): la API es
// JSON-only. Si el cliente manda un Accept que no admite application/json,
// se responde 406 con el envelope estándar en vez de fingir que se puede
// negociar otra representación. Sin header Accept se asume JSON.
func RequireJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		accept := c.GetHeader("Accept")
		if accept == "" || acceptsJSON(accept) {
			c.Next()
			return
		}
		apperr.Abort(c, http.StatusNotAcceptable, "not_acceptable",
			"this API only produces application/json", nil)
	}
}

// acceptsJSON acepta application/json, application/* o */* en cualquier
// posición de la lista del header (los q-values se ignoran: alcanza con que
// JSON sea aceptable, no hace falta rankearlo).
func acceptsJSON(accept string) bool {
	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch strings.ToLower(mediaType) {
		case "application/json", "application/*", "*/*":
			return true
		}
	}
	return false
}
