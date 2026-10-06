// Package cors es el ÚNICO middleware CORS de la plataforma (C1/CQ4): los
// tres servicios lo montan y el gateway no emite headers CORS en las rutas
// proxiadas (RV27 — duplicar Access-Control-Allow-Origin en gateway y
// servicio hace que el browser bloquee la respuesta por header múltiple).
//
// Config por env CORS_ALLOWED_ORIGINS (lista separada por comas). Reglas:
//   - La auth es por header Bearer (no cookies), así que NUNCA se emite
//     Access-Control-Allow-Credentials: "*" es una configuración válida y el
//     combo inválido "*" + credentials (C1) no puede reaparecer.
//   - Con "*" en la lista se responde el literal "*" — nunca se refleja el
//     Origin entrante (RV27: reflejar cualquier Origin reintroduce el
//     equivalente a credentials-para-todos si algún día se suman cookies).
//   - Con allowlist se responde el Origin solo si está permitido, con
//     Vary: Origin para que ningún cache intermedio mezcle respuestas.
package cors

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// defaultAllowedOrigins cubre el desarrollo local sin configuración: el SPA
// servido por Docker (3000) y el dev server de Vite (5173).
const defaultAllowedOrigins = "http://localhost:3000,http://localhost:5173"

// Middleware construye el handler CORS a partir de CORS_ALLOWED_ORIGINS.
// Responde los preflights OPTIONS con 204 y corta la cadena.
func Middleware() gin.HandlerFunc {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		raw = defaultAllowedOrigins
	}

	wildcard := false
	allowed := make(map[string]bool)
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		switch {
		case origin == "*":
			wildcard = true
		case origin != "":
			allowed[origin] = true
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		switch {
		case wildcard:
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		case allowed[origin]:
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Add("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			// Headers que solo aplican al preflight. Idempotency-Key habilita
			// el POST de reservas de hotels-api (A3); listarlo acá es inocuo
			// para los otros servicios.
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, Idempotency-Key")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
