package microservices

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(ctrl Controller) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/microservices/status", ctrl.GetMicroservicesStatus)
	r.POST("/microservices/scale", ctrl.ScaleService)
	r.POST("/microservices/:service_name/restart", ctrl.RestartService)
	return r
}

func TestGetMicroservicesStatus_OK(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/microservices/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestScaleService_InvalidService(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	body := `{"service_name":"unknown","replicas":2}`
	req := httptest.NewRequest(http.MethodPost, "/microservices/scale", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRestartService_OK(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/microservices/users-api/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
