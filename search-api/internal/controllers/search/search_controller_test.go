package search_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/search"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockService implementa la interfaz Service del controller para testing.
type mockService struct {
	mock.Mock
}

func (m *mockService) Search(ctx context.Context, query string, offset int, limit int) ([]hotelsDomain.Hotel, int, error) {
	args := m.Called(ctx, query, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]hotelsDomain.Hotel), args.Int(1), args.Error(2)
}

func (m *mockService) Backfill(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// searchEnvelope es el envelope estándar de las listas (A5).
type searchEnvelope struct {
	Data []hotelsDomain.Hotel `json:"data"`
	Meta struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func setupRouter(svc *mockService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	controller := controllers.NewController(svc)

	router.GET("/search", controller.Search)
	// Sin middleware JWT acá: los 401/403 de /reindex se cubren en
	// middlewares/auth_test.go (la ruta real se arma en cmd/main.go)
	router.POST("/reindex", controller.Reindex)

	return router
}

func TestController_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{
				ID:             "hotel1",
				Name:           "Hotel Paradise",
				Description:    "Luxury hotel",
				City:           "Buenos Aires",
				Country:        "Argentina",
				Rating:         4.5,
				PricePerNight:  150.0,
				AvailableRooms: 10,
				Amenities:      []string{"wifi", "pool"},
				Images:         []string{"img1.jpg"},
			},
			{
				ID:            "hotel2",
				Name:          "Hotel Sunset",
				Description:   "Beach hotel",
				City:          "Cancun",
				Country:       "Mexico",
				Rating:        4.8,
				PricePerNight: 200.0,
			},
		}

		svc.On("Search", mock.Anything, "paradise", 0, 10).Return(mockHotels, len(mockHotels), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=paradise&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got searchEnvelope
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got.Data, 2)
		assert.Equal(t, "hotel1", got.Data[0].ID)
		assert.Equal(t, "Hotel Paradise", got.Data[0].Name)
		assert.Equal(t, 4.5, got.Data[0].Rating)
		assert.Equal(t, "Hotel Sunset", got.Data[1].Name)
		// A5/RV22: el meta lleva el total real del índice
		assert.Equal(t, 2, got.Meta.Total)
		assert.Equal(t, 10, got.Meta.Limit)

		svc.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "nonexistent", 0, 10).Return([]hotelsDomain.Hotel{}, 0, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=nonexistent&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got searchEnvelope
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Empty(t, got.Data)

		svc.AssertExpectations(t)
	})

	// RV15: paginación clampeada — ausente/inválido cae a defaults (antes: 400)
	t.Run("missing pagination -> defaults", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "test", 0, 20).Return([]hotelsDomain.Hotel{}, 0, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("non-numeric pagination -> defaults", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "test", 0, 20).Return([]hotelsDomain.Hotel{}, 0, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=abc&limit=xyz", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("negative values -> clamped to defaults", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		// offset -5 → 0; limit -1 → 20 (post-E2 un rows negativo era 500)
		svc.On("Search", mock.Anything, "test", 0, 20).Return([]hotelsDomain.Hotel{}, 0, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=-5&limit=-1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("huge limit -> capped at 100", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "test", 0, 100).Return([]hotelsDomain.Hotel{}, 0, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=0&limit=99999", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("service error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "test", 0, 10).Return(nil, 0, errors.New("solr connection error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		// A1: envelope de error estándar sin internals de Solr en el body
		var got struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, "internal", got.Error.Code)
		assert.NotContains(t, rr.Body.String(), "solr connection error")

		svc.AssertExpectations(t)
	})

	t.Run("with pagination", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{ID: "hotel3", Name: "Paginated Hotel"},
		}

		svc.On("Search", mock.Anything, "hotel", 20, 5).Return(mockHotels, len(mockHotels), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=hotel&offset=20&limit=5", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got searchEnvelope
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got.Data, 1)
		assert.Equal(t, "hotel3", got.Data[0].ID)

		svc.AssertExpectations(t)
	})

	t.Run("empty query", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{ID: "hotel1", Name: "Hotel One"},
			{ID: "hotel2", Name: "Hotel Two"},
		}

		svc.On("Search", mock.Anything, "", 0, 10).Return(mockHotels, len(mockHotels), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got searchEnvelope
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got.Data, 2)

		svc.AssertExpectations(t)
	})

	t.Run("special characters in query", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{ID: "hotel1", Name: "Hotel & Spa"},
		}

		svc.On("Search", mock.Anything, "hotel & spa", 0, 10).Return(mockHotels, len(mockHotels), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=hotel+%26+spa&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got searchEnvelope
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got.Data, 1)
		assert.Equal(t, "Hotel & Spa", got.Data[0].Name)

		svc.AssertExpectations(t)
	})
}

// E3: POST /reindex dispara el backfill y devuelve el conteo
func TestController_Reindex(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Backfill", mock.Anything).Return(5, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/reindex", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got struct {
			Data map[string]int `json:"data"`
		}
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, 5, got.Data["indexed"])

		svc.AssertExpectations(t)
	})

	t.Run("backfill error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Backfill", mock.Anything).Return(0, errors.New("hotels-api unreachable")).Once()

		req := httptest.NewRequest(http.MethodPost, "/reindex", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var got struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, "internal", got.Error.Code)
		assert.NotContains(t, rr.Body.String(), "hotels-api unreachable")

		svc.AssertExpectations(t)
	})
}
