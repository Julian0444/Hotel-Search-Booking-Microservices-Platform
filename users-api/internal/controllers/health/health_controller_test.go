package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter(checks map[string]CheckFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	controller := NewController("test-service", checks)
	router := gin.New()
	router.GET("/livez", controller.Livez)
	router.GET("/readyz", controller.Readyz)
	router.GET("/health", controller.Livez)
	return router
}

func doRequest(router *gin.Engine, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestLivez_OKWithoutTouchingDependencies(t *testing.T) {
	// Un check que falla no afecta a /livez: solo importa que el proceso viva
	router := setupRouter(map[string]CheckFunc{
		"broken": func(_ context.Context) error { return errors.New("down") },
	})

	for _, path := range []string{"/livez", "/health"} {
		response := doRequest(router, path)
		assert.Equal(t, http.StatusOK, response.Code, path)

		var body map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, "ok", body["status"])
		assert.Equal(t, "test-service", body["service"])
	}
}

func TestReadyz_AllChecksOK(t *testing.T) {
	router := setupRouter(map[string]CheckFunc{
		"db":    func(_ context.Context) error { return nil },
		"queue": func(_ context.Context) error { return nil },
	})

	response := doRequest(router, "/readyz")
	assert.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Status)
	assert.Equal(t, map[string]string{"db": "ok", "queue": "ok"}, body.Checks)
}

func TestReadyz_DependencyDown(t *testing.T) {
	router := setupRouter(map[string]CheckFunc{
		"db":    func(_ context.Context) error { return nil },
		"queue": func(_ context.Context) error { return errors.New("connection refused") },
	})

	response := doRequest(router, "/readyz")
	assert.Equal(t, http.StatusServiceUnavailable, response.Code)

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "degraded", body.Status)
	assert.Equal(t, map[string]string{"db": "ok", "queue": "down"}, body.Checks)
}

func TestReadyzOptionalDependencyDownKeepsServing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewController("test", map[string]CheckFunc{
		"database": func(context.Context) error { return nil },
	}, map[string]CheckFunc{
		"memcached": func(context.Context) error { return errors.New("offline") },
	})
	router := gin.New()
	router.GET("/readyz", controller.Readyz)
	response := doRequest(router, "/readyz")
	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Status string
		Checks map[string]string
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "degraded", body.Status)
	assert.Equal(t, "down", body.Checks["memcached"])
	assert.Equal(t, "ok", body.Checks["database"])
}
