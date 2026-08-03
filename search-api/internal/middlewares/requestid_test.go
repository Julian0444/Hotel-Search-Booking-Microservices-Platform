package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRequestIDRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"request_id": c.GetString("request_id"),
			// RV16: el id también viaja en el context.Context del request
			// (es el que propagan los clientes HTTP como X-Request-ID)
			"ctx_request_id": utils.RequestIDFromContext(c.Request.Context()),
		})
	})
	return router
}

func TestRequestID_PropagatesIncomingHeader(t *testing.T) {
	router := setupRequestIDRouter()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("X-Request-ID", "test-trace-123")
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "test-trace-123", recorder.Header().Get("X-Request-ID"))
	assert.Contains(t, recorder.Body.String(), `"request_id":"test-trace-123"`)
	assert.Contains(t, recorder.Body.String(), `"ctx_request_id":"test-trace-123"`)
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	router := setupRequestIDRouter()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get("X-Request-ID"))
}
