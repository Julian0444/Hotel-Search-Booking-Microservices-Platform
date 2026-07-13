package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestTimeout envuelve el context del request con un deadline (R1): si una
// dependencia (MySQL) se cuelga, el request falla rápido en vez de bloquear la
// réplica para siempre. Los controllers mapean DeadlineExceeded a 503.
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
