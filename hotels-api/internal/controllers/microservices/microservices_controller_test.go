package microservices

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/config"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func setupRouter(ctrl Controller) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	jwtMiddleware := middlewares.NewJWTMiddleware(config.JWTSecret)
	adminRoutes := r.Group("/admin", jwtMiddleware.Authenticate(), middlewares.AdminOnly())
	{
		adminRoutes.GET("/microservices", ctrl.GetMicroservicesStatus)
	}
	return r
}

func makeJWT(t *testing.T, userType string, userID any) string {
	t.Helper()

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tipo":    userType,
		"user_id": userID,
		"iss":     "users-api",
		"aud":     []string{"users-api", "hotels-api"},
		"iat":     now.Unix(),
		"nbf":     now.Unix(),
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

// statusResponse refleja el shape read-only del panel (C2).
type statusResponse struct {
	Data struct {
		Services []ServiceStatus `json:"services"`
		Summary  struct {
			TotalServices   int `json:"total_services"`
			TotalInstances  int `json:"total_instances"`
			HealthyServices int `json:"healthy_services"`
		} `json:"summary"`
	} `json:"data"`
}

// C2: el health es real — un /readyz que responde 200 aparece "up" y uno que
// responde 503 aparece "down", con latencia medida, no inventada.
func TestAdminGetMicroservicesStatus_RealHealth(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Errorf("probe hit %q, want /readyz", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer broken.Close()

	ctrl := NewController([]ServiceTarget{
		{Name: "users-api", InstanceURLs: []string{healthy.URL, healthy.URL}},
		{Name: "search-api", InstanceURLs: []string{broken.URL}},
	})
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/admin/microservices", nil)
	req.Header.Set("Authorization", authBearer(token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp statusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}

	if got := resp.Data.Summary.TotalServices; got != 2 {
		t.Errorf("total_services: got %d, want 2", got)
	}
	if got := resp.Data.Summary.TotalInstances; got != 3 {
		t.Errorf("total_instances: got %d, want 3", got)
	}
	if got := resp.Data.Summary.HealthyServices; got != 1 {
		t.Errorf("healthy_services: got %d, want 1", got)
	}

	byName := map[string]ServiceStatus{}
	for _, svc := range resp.Data.Services {
		byName[svc.Name] = svc
	}

	users := byName["users-api"]
	if users.Status != "up" {
		t.Errorf("users-api status: got %q, want up", users.Status)
	}
	if !users.LoadBalanced {
		t.Error("users-api with 2 instances must report load_balanced=true")
	}

	search := byName["search-api"]
	if search.Status != "down" {
		t.Errorf("search-api status: got %q, want down", search.Status)
	}
	if len(search.Instances) != 1 || search.Instances[0].Error == "" {
		t.Errorf("down instance must carry the probe error, got %+v", search.Instances)
	}
}

func TestAdminGetMicroservicesStatus_UnreachableInstanceIsDown(t *testing.T) {
	// Puerto sin listener: el probe debe fallar rápido y reportar "down".
	ctrl := NewController([]ServiceTarget{
		{Name: "hotels-api", InstanceURLs: []string{"http://127.0.0.1:1"}},
	})
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/admin/microservices", nil)
	req.Header.Set("Authorization", authBearer(token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp statusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if got := resp.Data.Services[0].Status; got != "down" {
		t.Errorf("unreachable service status: got %q, want down", got)
	}
	if got := resp.Data.Summary.HealthyServices; got != 0 {
		t.Errorf("healthy_services: got %d, want 0", got)
	}
}

func TestAdminGetMicroservicesStatus_ForbiddenForNonAdmin(t *testing.T) {
	ctrl := NewController(nil)
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

func TestParseTargets(t *testing.T) {
	got := ParseTargets("users-api=http://u1:8082,http://u2:8082; search-api=http://s1:8082;;bad;=x")

	if len(got) != 2 {
		t.Fatalf("expected 2 targets, got %d: %+v", len(got), got)
	}
	if got[0].Name != "users-api" || len(got[0].InstanceURLs) != 2 {
		t.Errorf("users-api target malparsed: %+v", got[0])
	}
	if got[1].Name != "search-api" || len(got[1].InstanceURLs) != 1 {
		t.Errorf("search-api target malparsed: %+v", got[1])
	}
}
