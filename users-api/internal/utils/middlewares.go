package utils

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware configura CORS para permitir requests del frontend.
// Por defecto permite localhost:3000 y localhost:5173 (Vite).
// Configurable via CORS_ALLOWED_ORIGINS (comma-separated).
func CorsMiddleware() gin.HandlerFunc {
	allowedOrigins := getEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	origins := strings.Split(allowedOrigins, ",")
	originsMap := make(map[string]bool)
	for _, o := range origins {
		originsMap[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Si el origin está permitido, lo incluimos en la respuesta
		if originsMap[origin] || originsMap["*"] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
