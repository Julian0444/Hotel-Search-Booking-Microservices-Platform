package microservices

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/config"
	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func setupRouter(ctrl Controller) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	jwtMiddleware := middleware.NewJWTMiddleware(config.JWTSecret)
	adminRoutes := r.Group("/admin", jwtMiddleware.Authenticate(), middleware.AdminOnly())
	{
		adminRoutes.GET("/microservices", ctrl.GetMicroservicesStatus)
		adminRoutes.POST("/microservices/scale", ctrl.ScaleService)
		adminRoutes.GET("/microservices/:service_name/logs", ctrl.GetServiceLogs)
		adminRoutes.POST("/microservices/:service_name/restart", ctrl.RestartService)
	}
	return r
}

func makeJWT(t *testing.T, userType string, userID any) string {
	t.Helper()

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tipo":    userType,
		"user_id": userID,
		"iat":     now.Unix(),
		"exp":     now.Add(1 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		t.Fatalf("error signing token: %v", err)
	}
	return signed
}

func authBearer(token string) string {
	return "Bearer " + token
}

func TestAdminGetMicroservicesStatus_OK(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/admin/microservices", nil)
	req.Header.Set("Authorization", authBearer(token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminGetMicroservicesStatus_ForbiddenForNonAdmin(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/admin/microservices", nil)
	req.Header.Set("Authorization", authBearer(token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminScaleService_InvalidService(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(1))
	body := `{"service_name":"unknown","replicas":2}`
	req := httptest.NewRequest(http.MethodPost, "/admin/microservices/scale", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminRestartService_OK(t *testing.T) {
	ctrl := NewController()
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(1))
	req := httptest.NewRequest(http.MethodPost, "/admin/microservices/users-api/restart", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}
