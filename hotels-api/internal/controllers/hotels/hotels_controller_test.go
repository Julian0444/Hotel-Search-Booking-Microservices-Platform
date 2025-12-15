package hotels

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/gin-gonic/gin"
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
func (m mockService) GetReservationsByUserAndHotelID(ctx context.Context, userID, hotelID string) ([]hotelsDomain.Reservation, error) {
	if m.getReservationsByUserAndHotelFn != nil {
		return m.getReservationsByUserAndHotelFn(ctx, userID, hotelID)
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
	r := gin.Default()
	// rutas mínimas para tests
	r.GET("/hotels/:hotel_id", ctrl.GetHotelByID)
	r.POST("/hotels", ctrl.Create)
	r.POST("/hotels/availability", ctrl.GetAvailability)
	r.PUT("/hotels/:hotel_id", ctrl.Update)
	r.DELETE("/hotels/:hotel_id", ctrl.Delete)
	r.POST("/reservations", func(c *gin.Context) {
		c.Set("userID", c.GetHeader("X-User-ID"))
		ctrl.CreateReservation(c)
	})
	r.DELETE("/reservations/:id", func(c *gin.Context) {
		c.Set("userID", c.GetHeader("X-User-ID"))
		ctrl.CancelReservation(c)
	})
	r.GET("/hotels/:hotel_id/reservations", ctrl.GetReservationsByHotelID)
	r.GET("/users/:user_id/reservations", ctrl.GetReservationsByUserID)
	r.GET("/users/:user_id/hotels/:hotel_id/reservations", ctrl.GetReservationsByUserAndHotelID)
	return r
}

func TestGetHotelByID_Table(t *testing.T) {
	cases := []struct {
		name     string
		id       string
		svc      mockService
		wantCode int
	}{
		{
			name:     "ok",
			id:       "h1",
			wantCode: http.StatusOK,
			svc: mockService{
				getHotelByIDFn: func(_ context.Context, id string) (hotelsDomain.Hotel, error) {
					return hotelsDomain.Hotel{ID: id, Name: "Hotel Test"}, nil
				},
			},
		},
		{
			name:     "not found",
			id:       "h404",
			wantCode: http.StatusNotFound,
			svc: mockService{
				getHotelByIDFn: func(_ context.Context, id string) (hotelsDomain.Hotel, error) {
					return hotelsDomain.Hotel{}, fmt.Errorf("not found %s", id)
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodGet, "/hotels/"+tc.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestCreateHotel_Table(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		svc      mockService
		wantCode int
	}{
		{
			name:     "created",
			body:     `{"name":"New Hotel"}`,
			wantCode: http.StatusCreated,
			svc: mockService{
				createHotelFn: func(_ context.Context, h hotelsDomain.Hotel) (string, error) {
					return "new-id", nil
				},
			},
		},
		{
			name:     "bad request",
			body:     `{"name":123}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service error",
			body:     `{"name":"Err"}`,
			wantCode: http.StatusInternalServerError,
			svc: mockService{
				createHotelFn: func(_ context.Context, h hotelsDomain.Hotel) (string, error) {
					return "", fmt.Errorf("db error")
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodPost, "/hotels", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestUpdateHotel_Table(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		svc      mockService
		wantCode int
	}{
		{
			name:     "ok",
			body:     `{"name":"Updated"}`,
			wantCode: http.StatusOK,
			svc: mockService{
				updateHotelFn: func(_ context.Context, h hotelsDomain.Hotel) error { return nil },
			},
		},
		{
			name:     "bad request",
			body:     `{"name":123}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service error",
			body:     `{"name":"Err"}`,
			wantCode: http.StatusInternalServerError,
			svc: mockService{
				updateHotelFn: func(_ context.Context, h hotelsDomain.Hotel) error { return fmt.Errorf("fail") },
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodPut, "/hotels/h1", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestDeleteHotel_Table(t *testing.T) {
	cases := []struct {
		name     string
		svc      mockService
		wantCode int
	}{
		{
			name:     "ok",
			wantCode: http.StatusOK,
		},
		{
			name:     "service error",
			wantCode: http.StatusInternalServerError,
			svc: mockService{
				deleteHotelFn: func(_ context.Context, id string) error { return fmt.Errorf("fail") },
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodDelete, "/hotels/h1", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestGetAvailability_Table(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		svc      mockService
		wantCode int
	}{
		{
			name:     "ok",
			body:     `{"hotel_ids":["h1"],"check_in":"2024-01-01","check_out":"2024-01-02"}`,
			wantCode: http.StatusOK,
			svc: mockService{
				getAvailabilityFn: func(_ context.Context, ids []string, ci, co string) (map[string]bool, error) {
					return map[string]bool{"h1": true}, nil
				},
			},
		},
		{
			name:     "bad request",
			body:     `{"hotel_ids":"h1"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service error",
			body:     `{"hotel_ids":["h1"],"check_in":"2024-01-01","check_out":"2024-01-02"}`,
			wantCode: http.StatusInternalServerError,
			svc: mockService{
				getAvailabilityFn: func(_ context.Context, ids []string, ci, co string) (map[string]bool, error) {
					return nil, fmt.Errorf("fail")
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodPost, "/hotels/availability", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestCreateReservation_Table(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		userHdr  string
		svc      mockService
		wantCode int
	}{
		{
			name:     "created",
			body:     `{"hotel_id":"h1","user_id":"u1","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`,
			userHdr:  "u1",
			wantCode: http.StatusCreated,
			svc: mockService{
				createReservationFn: func(_ context.Context, r hotelsDomain.Reservation) (string, error) {
					return "res1", nil
				},
			},
		},
		{
			name:     "forbidden other user",
			body:     `{"hotel_id":"h1","user_id":"u2","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`,
			userHdr:  "u1",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "bad request",
			body:     `{"hotel_id":123}`,
			userHdr:  "u1",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service error",
			body:     `{"hotel_id":"h1","user_id":"u1","check_in":"2024-01-01T00:00:00Z","check_out":"2024-01-02T00:00:00Z"}`,
			userHdr:  "u1",
			wantCode: http.StatusInternalServerError,
			svc: mockService{
				createReservationFn: func(_ context.Context, r hotelsDomain.Reservation) (string, error) {
					return "", fmt.Errorf("fail")
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", tc.userHdr)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestCancelReservation_Table(t *testing.T) {
	cases := []struct {
		name     string
		userHdr  string
		svc      mockService
		wantCode int
	}{
		{
			name:     "ok",
			userHdr:  "u1",
			wantCode: http.StatusOK,
			svc: mockService{
				getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
					return hotelsDomain.Reservation{ID: id, UserID: "u1"}, nil
				},
			},
		},
		{
			name:     "forbidden other user",
			userHdr:  "u2",
			wantCode: http.StatusForbidden,
			svc: mockService{
				getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
					return hotelsDomain.Reservation{ID: id, UserID: "u1"}, nil
				},
			},
		},
		{
			name:     "not found",
			userHdr:  "u1",
			wantCode: http.StatusNotFound,
			svc: mockService{
				getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
					return hotelsDomain.Reservation{}, fmt.Errorf("not found")
				},
			},
		},
		{
			name:     "cancel error",
			userHdr:  "u1",
			wantCode: http.StatusInternalServerError,
			svc: mockService{
				getReservationByIDFn: func(_ context.Context, id string) (hotelsDomain.Reservation, error) {
					return hotelsDomain.Reservation{ID: id, UserID: "u1"}, nil
				},
				cancelReservationFn: func(_ context.Context, id string) error { return fmt.Errorf("fail") },
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodDelete, "/reservations/res1", nil)
			req.Header.Set("X-User-ID", tc.userHdr)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestGetReservations_Table(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		svc      mockService
		wantCode int
	}{
		{
			name:     "by hotel ok",
			path:     "/hotels/h1/reservations",
			wantCode: http.StatusOK,
			svc: mockService{
				getReservationsByHotelIDFn: func(_ context.Context, h string) ([]hotelsDomain.Reservation, error) {
					return []hotelsDomain.Reservation{{ID: "r1"}}, nil
				},
			},
		},
		{
			name:     "by hotel error",
			path:     "/hotels/h1/reservations",
			wantCode: http.StatusNotFound,
			svc: mockService{
				getReservationsByHotelIDFn: func(_ context.Context, h string) ([]hotelsDomain.Reservation, error) {
					return nil, fmt.Errorf("fail")
				},
			},
		},
		{
			name:     "by user ok",
			path:     "/users/u1/reservations",
			wantCode: http.StatusOK,
			svc: mockService{
				getReservationsByUserIDFn: func(_ context.Context, u string) ([]hotelsDomain.Reservation, error) {
					return []hotelsDomain.Reservation{{ID: "r1"}}, nil
				},
			},
		},
		{
			name:     "by user+hotel ok",
			path:     "/users/u1/hotels/h1/reservations",
			wantCode: http.StatusOK,
			svc: mockService{
				getReservationsByUserAndHotelFn: func(_ context.Context, u, h string) ([]hotelsDomain.Reservation, error) {
					return []hotelsDomain.Reservation{{ID: "r1"}}, nil
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := NewController(tc.svc)
			r := setupRouter(ctrl)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}
