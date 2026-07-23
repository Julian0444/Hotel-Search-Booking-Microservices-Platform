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
	getHotelsFn                     func(context.Context, int64, int64) ([]hotelsDomain.Hotel, int64, error)
	createHotelFn                   func(context.Context, hotelsDomain.Hotel) (string, error)
	updateHotelFn                   func(context.Context, hotelsDomain.Hotel) error
	deleteHotelFn                   func(context.Context, string) error
	createReservationFn             func(context.Context, hotelsDomain.Reservation) (string, error)
	getReservationByIDFn            func(context.Context, string) (hotelsDomain.Reservation, error)
	cancelReservationFn             func(context.Context, string) error
	getReservationsByHotelIDFn      func(context.Context, string, int64, int64) ([]hotelsDomain.Reservation, error)
	getReservationsByUserIDFn       func(context.Context, string, int64, int64) ([]hotelsDomain.Reservation, error)
	getReservationsByUserAndHotelFn func(context.Context, string, string, int64, int64) ([]hotelsDomain.Reservation, error)
	getAvailabilityFn               func(context.Context, []string, string, string) (map[string]bool, error)
}

func (m mockService) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	if m.getHotelByIDFn != nil {
		return m.getHotelByIDFn(ctx, id)
	}
	return hotelsDomain.Hotel{}, nil
}
func (m mockService) GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDomain.Hotel, int64, error) {
	if m.getHotelsFn != nil {
		return m.getHotelsFn(ctx, limit, offset)
	}
	return nil, 0, nil
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
func (m mockService) GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByHotelIDFn != nil {
		return m.getReservationsByHotelIDFn(ctx, hotelID, limit, offset)
	}
	return nil, nil
}
func (m mockService) GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByUserIDFn != nil {
		return m.getReservationsByUserIDFn(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m mockService) GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByUserAndHotelFn != nil {
		return m.getReservationsByUserAndHotelFn(ctx, hotelID, userID, limit, offset)
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

	// RequireJSON global como en cmd/main.go (A8); acá sin el prefijo /api/v1
	// para que los paths de los tests queden cortos
	r.Use(middleware.RequireJSON())

	// Rutas públicas (como en cmd/main.go)
	r.GET("/hotels", ctrl.GetHotels)
	r.GET("/hotels/:hotel_id", ctrl.GetHotelByID)
	r.GET("/hotels/:hotel_id/reservations", ctrl.GetReservationsByHotelID)
	r.POST("/hotels/availability", ctrl.GetAvailability)

	// Rutas protegidas (usuarios autenticados)
	userRoutes := r.Group("/", jwtMiddleware.Authenticate(), middleware.LoggedUserOnly())
	{
		userRoutes.POST("/reservations", ctrl.CreateReservation)
		userRoutes.GET("/reservations/:id", ctrl.GetReservationByID)
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

// RV14: solo el sentinel ErrHotelNotFound (aunque venga envuelto) es 404
func TestGetHotelByID_NotFound(t *testing.T) {
	svc := mockService{
		getHotelByIDFn: func(_ context.Context, id string) (hotelsDomain.Hotel, error) {
			return hotelsDomain.Hotel{}, fmt.Errorf("error getting hotel from repository: hotel %s: %w", id, hotelsDomain.ErrHotelNotFound)
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

// RV14: un error de infraestructura ya NO se disfraza de 404
func TestGetHotelByID_InfraErrorIs500(t *testing.T) {
	svc := mockService{
		getHotelByIDFn: func(_ context.Context, _ string) (hotelsDomain.Hotel, error) {
			return hotelsDomain.Hotel{}, fmt.Errorf("mongo: server selection timeout")
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/hotels/h1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

// E3: GET /hotels devuelve el envelope {data, total} y clampa la paginación
func TestGetHotels_OK(t *testing.T) {
	svc := mockService{
		getHotelsFn: func(_ context.Context, limit, offset int64) ([]hotelsDomain.Hotel, int64, error) {
			// limit=9999 debe llegar clampeado a 100; offset=-3 a 0
			if limit != 100 || offset != 0 {
				t.Fatalf("expected clamped limit=100 offset=0, got %d/%d", limit, offset)
			}
			return []hotelsDomain.Hotel{{ID: "h1", Name: "Hotel Uno"}, {ID: "h2", Name: "Hotel Dos"}}, 5, nil
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/hotels?limit=9999&offset=-3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"total":5`) {
		t.Fatalf("expected total in body, got: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"name":"Hotel Dos"`) {
		t.Fatalf("expected hotels in data, got: %s", w.Body.String())
	}
}

func TestGetHotels_ServiceErrorIs500(t *testing.T) {
	svc := mockService{
		getHotelsFn: func(_ context.Context, _, _ int64) ([]hotelsDomain.Hotel, int64, error) {
			return nil, 0, fmt.Errorf("mongo down")
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/hotels", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusInternalServerError, w.Body.String())
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
	// A6: 201 con Location del recurso creado
	if got := w.Header().Get("Location"); got != "/api/v1/hotels/new-id" {
		t.Fatalf("expected Location header, got %q", got)
	}
}

// A6: PUT devuelve la representación actualizada, no {message:id}
func TestAdminUpdateHotel_ReturnsUpdatedRepresentation(t *testing.T) {
	svc := mockService{
		updateHotelFn: func(_ context.Context, h hotelsDomain.Hotel) error {
			if h.ID != "h1" {
				t.Fatalf("expected id=h1, got %q", h.ID)
			}
			return nil
		},
		getHotelByIDFn: func(_ context.Context, id string) (hotelsDomain.Hotel, error) {
			return hotelsDomain.Hotel{ID: id, Name: "Renamed", CheckInTime: "14:00"}, nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(999))
	req := httptest.NewRequest(http.MethodPut, "/admin/hotels/h1", strings.NewReader(`{"name":"Renamed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"name":"Renamed"`) {
		t.Fatalf("expected updated representation in body, got: %s", w.Body.String())
	}
}

// RV21: el contrato exige "HH:mm" en check_in_time/check_out_time
func TestAdminCreateHotel_BadTimeOfDay(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(999))
	body := `{"name":"Bad Times","check_in_time":"2024-01-01T14:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/hotels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// A6: DELETE exitoso responde 204 sin body
func TestAdminDeleteHotel_NoContent(t *testing.T) {
	svc := mockService{
		deleteHotelFn: func(_ context.Context, id string) error {
			if id != "h1" {
				t.Fatalf("expected id=h1, got %q", id)
			}
			return nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(999))
	req := httptest.NewRequest(http.MethodDelete, "/admin/hotels/h1", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body on 204, got: %s", w.Body.String())
	}
}

// A6: borrar un hotel inexistente es 404, no 500
func TestAdminDeleteHotel_NotFound(t *testing.T) {
	svc := mockService{
		deleteHotelFn: func(_ context.Context, id string) error {
			return fmt.Errorf("error deleting hotel from main repository: hotel %s: %w", id, hotelsDomain.ErrHotelNotFound)
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "administrador", int64(999))
	req := httptest.NewRequest(http.MethodDelete, "/admin/hotels/missing", nil)
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusNotFound, w.Body.String())
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
		getReservationsByUserIDFn: func(_ context.Context, userID string, _, _ int64) ([]hotelsDomain.Reservation, error) {
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
		getReservationsByUserAndHotelFn: func(_ context.Context, hotelID, userID string, _, _ int64) ([]hotelsDomain.Reservation, error) {
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

	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-01","check_out":"2024-01-02"}`
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
	body := `{"hotel_id":"h1","user_id":"2","check_in":"2024-01-01","check_out":"2024-01-02"}`
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
			if r.CheckIn != "2024-01-01" || r.CheckOut != "2024-01-02" {
				t.Fatalf("unexpected dates: %v - %v", r.CheckIn, r.CheckOut)
			}
			if r.NumRooms != 2 || r.NumGuests != 4 {
				t.Fatalf("expected num_rooms=2 num_guests=4, got %d/%d", r.NumRooms, r.NumGuests)
			}
			return "res1", nil
		},
	}

	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-01","check_out":"2024-01-02","num_rooms":2,"num_guests":4}`
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

// El DTO exige fechas canónicas YYYY-MM-DD: un datetime RFC3339 es 400
func TestCreateReservation_BadDateFormat(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateReservation_CheckOutNotAfterCheckIn(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-02","check_out":"2024-01-02"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// D1: ErrNoAvailability del service se mapea a 409 Conflict
func TestCreateReservation_ConflictWhenNoAvailability(t *testing.T) {
	svc := mockService{
		createReservationFn: func(_ context.Context, _ hotelsDomain.Reservation) (string, error) {
			return "", fmt.Errorf("error creating reservation in main repository: %w", hotelsDomain.ErrNoAvailability)
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","user_id":"1","check_in":"2024-01-01","check_out":"2024-01-02"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusConflict, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "no availability") {
		t.Fatalf("expected availability error in body, got: %s", w.Body.String())
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

	// A6: DELETE exitoso responde 204 sin body
	if w.Code != http.StatusNoContent {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

// RV19: una validación del service (sentinel ErrInvalidReservation) es 400
func TestCreateReservation_ServiceValidationIs400(t *testing.T) {
	svc := mockService{
		createReservationFn: func(_ context.Context, _ hotelsDomain.Reservation) (string, error) {
			return "", fmt.Errorf("check-in date must not be in the past: %w", hotelsDomain.ErrInvalidReservation)
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"h1","check_in":"2024-01-01","check_out":"2024-01-02"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"invalid_reservation"`) {
		t.Fatalf("expected invalid_reservation code, got: %s", w.Body.String())
	}
}

// RV19: reservar sobre un hotel inexistente es 404, no 500
func TestCreateReservation_UnknownHotelIs404(t *testing.T) {
	svc := mockService{
		createReservationFn: func(_ context.Context, _ hotelsDomain.Reservation) (string, error) {
			return "", fmt.Errorf("error getting hotel for reservation: %w", hotelsDomain.ErrHotelNotFound)
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	token := makeJWT(t, "cliente", int64(1))
	body := `{"hotel_id":"missing","check_in":"2024-01-01","check_out":"2024-01-02"}`
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authBearer(token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"hotel_not_found"`) {
		t.Fatalf("expected hotel_not_found code, got: %s", w.Body.String())
	}
}

// GET /reservations/:id: dueño la ve; otro usuario recibe 403
func TestGetReservationByID_OwnershipEnforced(t *testing.T) {
	svc := mockService{
		getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
			return hotelsDomain.Reservation{ID: id, UserID: "1", CheckIn: "2026-08-01", CheckOut: "2026-08-03"}, nil
		},
	}
	ctrl := NewController(svc)
	r := setupRouter(ctrl)

	// dueño → 200 con fechas date-only (RV20)
	token := makeJWT(t, "cliente", int64(1))
	req := httptest.NewRequest(http.MethodGet, "/reservations/res1", nil)
	req.Header.Set("Authorization", authBearer(token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("owner: code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"check_in":"2026-08-01"`) {
		t.Fatalf("expected date-only check_in, got: %s", w.Body.String())
	}

	// otro cliente → 403
	otherToken := makeJWT(t, "cliente", int64(2))
	req = httptest.NewRequest(http.MethodGet, "/reservations/res1", nil)
	req.Header.Set("Authorization", authBearer(otherToken))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("other user: code=%d want=%d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}

	// admin → 200
	adminToken := makeJWT(t, "administrador", int64(99))
	req = httptest.NewRequest(http.MethodGet, "/reservations/res1", nil)
	req.Header.Set("Authorization", authBearer(adminToken))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin: code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// A8: Accept incompatible con JSON responde 406 con el envelope
func TestRequireJSON_NotAcceptable(t *testing.T) {
	ctrl := NewController(mockService{})
	r := setupRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/hotels/h1", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotAcceptable {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusNotAcceptable, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"not_acceptable"`) {
		t.Fatalf("expected not_acceptable code, got: %s", w.Body.String())
	}

	// Accept que sí admite JSON pasa
	req = httptest.NewRequest(http.MethodGet, "/hotels/h1", nil)
	req.Header.Set("Accept", "text/html, application/json;q=0.9")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}
