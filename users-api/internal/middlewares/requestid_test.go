package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRequestIDRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"request_id": c.GetString("request_id")})
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
	assert.Contains(t, recorder.Body.String(), "test-trace-123")
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	router := setupRequestIDRouter()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get("X-Request-ID"))
}
