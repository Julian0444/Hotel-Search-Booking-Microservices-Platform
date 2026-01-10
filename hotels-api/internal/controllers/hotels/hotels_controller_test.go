package hotels

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/config"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// mockService implementa la interfaz Service con funciones configurables.
type mockService struct {
	getHotelByIDFn                  func(context.Context, string) (hotelsDomain.Hotel, error)
	createHotelFn                   func(context.Context, hotelsDomain.Hotel) (string, error)
	updateHotelFn                   func(context.Context, hotelsDomain.Hotel) error
	deleteHotelFn                   func(context.Context, string) error
	createReservationFn             func(context.Context, hotelsDomain.Reservation) (string, error)
	getReservationByIDFn            func(context.Context, string) (hotelsDomain.Reservation, error)
	cancelReservationFn             func(context.Context, string) error
	getReservationsByHotelIDFn      func(context.Context, string) ([]hotelsDomain.Reservation, error)
	getReservationsByUserIDFn       func(context.Context, string) ([]hotelsDomain.Reservation, error)
	getReservationsByUserAndHotelFn func(context.Context, string, string) ([]hotelsDomain.Reservation, error)
	getAvailabilityFn               func(context.Context, []string, string, string) (map[string]bool, error)
}

func (m mockService) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	if m.getHotelByIDFn != nil {
		return m.getHotelByIDFn(ctx, id)
	}
	return hotelsDomain.Hotel{}, nil
}
func (m mockService) Create(ctx context.Context, hotel hotelsDomain.Hotel) (string, error) {
	if m.createHotelFn != nil {
		return m.createHotelFn(ctx, hotel)
	}
	return "", nil
}
func (m mockService) Update(ctx context.Context, hotel hotelsDomain.Hotel) error {
	if m.updateHotelFn != nil {
		return m.updateHotelFn(ctx, hotel)
	}
	return nil
}
func (m mockService) Delete(ctx context.Context, id string) error {
	if m.deleteHotelFn != nil {
		return m.deleteHotelFn(ctx, id)
	}
	return nil
}
func (m mockService) CreateReservation(ctx context.Context, reservation hotelsDomain.Reservation) (string, error) {
	if m.createReservationFn != nil {
		return m.createReservationFn(ctx, reservation)
	}
	return "", nil
}
func (m mockService) GetReservationByID(ctx context.Context, id string) (hotelsDomain.Reservation, error) {
	if m.getReservationByIDFn != nil {
		return m.getReservationByIDFn(ctx, id)
	}
	return hotelsDomain.Reservation{}, nil
}
func (m mockService) CancelReservation(ctx context.Context, id string) error {
	if m.cancelReservationFn != nil {
		return m.cancelReservationFn(ctx, id)
	}
	return nil
}
func (m mockService) GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByHotelIDFn != nil {
		return m.getReservationsByHotelIDFn(ctx, hotelID)
	}
	return nil, nil
}
func (m mockService) GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByUserIDFn != nil {
		return m.getReservationsByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m mockService) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByUserAndHotelFn != nil {
		return m.getReservationsByUserAndHotelFn(ctx, hotelID, userID)
	}
	return nil, nil
}
func (m mockService) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	if m.getAvailabilityFn != nil {
		return m.getAvailabilityFn(ctx, hotelIDs, checkIn, checkOut)
	}
	return nil, nil
}

func setupRouter(ctrl Controller) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	jwtMiddleware := middleware.NewJWTMiddleware(config.JWTSecret)

	// Rutas públicas (como en cmd/main.go)
	r.GET("/hotels/:hotel_id", ctrl.GetHotelByID)
	r.GET("/hotels/:hotel_id/reservations", ctrl.GetReservationsByHotelID)
	r.POST("/hotels/availability", ctrl.GetAvailability)

	// Rutas protegidas (usuarios autenticados)
	userRoutes := r.Group("/", jwtMiddleware.Authenticate(), middleware.LoggedUserOnly())
	{
		userRoutes.POST("/reservations", ctrl.CreateReservation)
		userRoutes.DELETE("/reservations/:id", ctrl.CancelReservation)
		userRoutes.GET("/users/:user_id/reservations", ctrl.GetReservationsByUserID)
		userRoutes.GET("/users/:user_id/hotels/:hotel_id/reservations", ctrl.GetReservationsByUserAndHotelID)
	}

	// Rutas protegidas (admins)
	adminRoutes := r.Group("/admin", jwtMiddleware.Authenticate(), middleware.AdminOnly())
	{
		adminRoutes.POST("/hotels", ctrl.Create)
		adminRoutes.PUT("/hotels/:hotel_id", ctrl.Update)
		adminRoutes.DELETE("/hotels/:hotel_id", ctrl.Delete)
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

func TestGetHotelByID_OK(t *testing.T) {
	svc := mockService{
		getHotelByIDFn: func(_ context.Context, id string) (hotelsDomain.Hotel, error) {
			if id != "h1" {
				t.Fatalf("expected id=h1, got %s", id)
			}
			return hotelsDomain.Hotel{ID: id, Name: "Hotel Test"}, nil
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/hotels/h1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"name":"Hotel Test"`) {
		t.Fatalf("expected body to contain hotel name, got: %s", w.Body.String())
	}
}

func TestGetHotelByID_NotFound(t *testing.T) {
	svc := mockService{
		getHotelByIDFn: func(_ context.Context, id string) (hotelsDomain.Hotel, error) {
			return hotelsDomain.Hotel{}, fmt.Errorf("not found %s", id)
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/hotels/h404", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestGetAvailability_OK(t *testing.T) {
	svc := mockService{
		getAvailabilityFn: func(_ context.Context, ids []string, ci, co string) (map[string]bool, error) {
			if len(ids) != 1 || ids[0] != "h1" {
				t.Fatalf("expected hotel_ids=[h1], got %v", ids)
			}
			if ci != "2024-01-01" || co != "2024-01-02" {
				t.Fatalf("unexpected dates: %s - %s", ci, co)
			}
			return map[string]bool{"h1": true}, nil
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	body := `{"hotel_ids":["h1"],"check_in":"2024-01-01","check_out":"2024-01-02"}`
	req := httptest.NewRequest(http.MethodPost, "/hotels/availability", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"h1":true`) {
		t.Fatalf("expected availability in body, got: %s", w.Body.String())
	}
}

func TestGetAvailability_BadRequest(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/hotels/availability", strings.NewReader(`{"hotel_ids":"h1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAdminCreateHotel_ForbiddenForNonAdmin(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodPost, "/admin/hotels", strings.NewReader(`{"name":"New Hotel"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestAdminCreateHotel_Created(t *testing.T) {
	svc := mockService{
		createHotelFn: func(_ context.Context, h hotelsDomain.Hotel) (string, error) {
			if h.Name != "New Hotel" {
				t.Fatalf("expected hotel name 'New Hotel', got %q", h.Name)
			}
			return "new-id", nil
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(999))
	req := httptest.NewRequest(http.MethodPost, "/admin/hotels", strings.NewReader(`{"name":"New Hotel"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"id":"new-id"`) {
		t.Fatalf("expected id in body, got: %s", w.Body.String())
	}
}

func TestGetReservationsByUserID_ForbiddenWhenUserMismatch(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	// token user_id = 1, pero URL pide user_id = 2
	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/users/2/reservations", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestGetReservationsByUserID_OK(t *testing.T) {
	svc := mockService{
		getReservationsByUserIDFn: func(_ context.Context, userID string) ([]hotelsDomain.Reservation, error) {
			if userID != "1" {
				t.Fatalf("expected userID=1, got %s", userID)
			}
			return []hotelsDomain.Reservation{{ID: "r1"}}, nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/users/1/reservations", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"id":"r1"`) {
		t.Fatalf("expected reservation id in body, got: %s", w.Body.String())
	}
}

func TestGetReservationsByUserAndHotelID_UnauthorizedWithoutToken(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/users/1/hotels/h1/reservations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestGetReservationsByUserAndHotelID_OK(t *testing.T) {
	svc := mockService{
		getReservationsByUserAndHotelFn: func(_ context.Context, hotelID, userID string) ([]hotelsDomain.Reservation, error) {
			if hotelID != "h1" {
				t.Fatalf("expected hotelID=h1, got %s", hotelID)
			}
			if userID != "1" {
				t.Fatalf("expected userID=1, got %s", userID)
			}
			return []hotelsDomain.Reservation{{ID: "r1"}}, nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/users/1/hotels/h1/reservations", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"id":"r1"`) {
		t.Fatalf("expected reservation id in body, got: %s", w.Body.String())
	}
}

func TestCreateReservation_UnauthorizedWithoutToken(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestCreateReservation_ForbiddenWhenUserMismatch(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","user_id":"2","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestCreateReservation_Created(t *testing.T) {
	svc := mockService{
		createReservationFn: func(_ context.Context, r hotelsDomain.Reservation) (string, error) {
			if r.UserID != "1" {
				t.Fatalf("expected reservation user_id=1, got %q", r.UserID)
			}
			if r.HotelID != "h1" {
				t.Fatalf("expected reservation hotel_id=h1, got %q", r.HotelID)
			}
			return "res1", nil
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"id":"res1"`) {
		t.Fatalf("expected id in body, got: %s", w.Body.String())
	}
}

func TestCancelReservation_ForbiddenWhenNotOwner(t *testing.T) {
	svc := mockService{
		getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
			return hotelsDomain.Reservation{ID: id, UserID: "2"}, nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodDelete, "/reservations/res1", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestCancelReservation_OK(t *testing.T) {
	svc := mockService{
		getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
			return hotelsDomain.Reservation{ID: id, UserID: "1"}, nil
		},
		cancelReservationFn: func(_ context.Context, id string) error {
			if id != "res1" {
				t.Fatalf("expected id=res1, got %s", id)
			}
			return nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodDelete, "/reservations/res1", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}
