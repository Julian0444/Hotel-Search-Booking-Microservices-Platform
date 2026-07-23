// Package apperr define el envelope de error estándar de la API (A1):
//
//	{"error": {"code": "...", "message": "...", "trace_id": "..."}}
//
// El code es estable (los clientes pueden hacer switch sobre él), el message
// es apto para humanos y la causa real va SOLO al log estructurado — nunca al
// body: antes ~18 handlers filtraban texto crudo de Mongo/Solr al cliente.
//
// El paquete está copiado verbatim en cada módulo (internal/ no cruza
// módulos); si el contrato compartido crece puede migrar a platform-contracts.
package apperr

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// Abort corta el request con el envelope estándar. cause (opcional) se loguea
// con el request_id para correlacionar; el body solo lleva code/message.
func Abort(c *gin.Context, status int, code, message string, cause error) {
	traceID := c.GetString("request_id")
	if cause != nil {
		_ = c.Error(cause)
		slog.Error("request failed",
			"request_id", traceID,
			"status", status,
			"code", code,
			"error", cause,
		)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{
		"code":     code,
		"message":  message,
		"trace_id": traceID,
	}})
}
