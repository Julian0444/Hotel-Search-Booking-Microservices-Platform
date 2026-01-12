package search_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	controllers "search-api/internal/controllers/search"
	hotelsDomain "search-api/internal/domain/hotels"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockService implementa la interfaz Service del controller para testing.
type mockService struct {
	mock.Mock
}

func (m *mockService) Search(ctx context.Context, query string, offset int, limit int) ([]hotelsDomain.Hotel, error) {
	args := m.Called(ctx, query, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]hotelsDomain.Hotel), args.Error(1)
}

func setupRouter(svc *mockService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	controller := controllers.NewController(svc)

	router.GET("/search", controller.Search)

	return router
}

func TestController_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{
				ID:            "hotel1",
				Name:          "Hotel Paradise",
				Description:   "Luxury hotel",
				City:          "Buenos Aires",
				Country:       "Argentina",
				Rating:        4.5,
				PricePerNight: 150.0,
				AvaiableRooms: 10,
				Amenities:     []string{"wifi", "pool"},
				Images:        []string{"img1.jpg"},
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

		svc.On("Search", mock.Anything, "paradise", 0, 10).Return(mockHotels, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=paradise&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got []hotelsDomain.Hotel
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got, 2)
		assert.Equal(t, "hotel1", got[0].ID)
		assert.Equal(t, "Hotel Paradise", got[0].Name)
		assert.Equal(t, 4.5, got[0].Rating)
		assert.Equal(t, "Hotel Sunset", got[1].Name)

		svc.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "nonexistent", 0, 10).Return([]hotelsDomain.Hotel{}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=nonexistent&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got []hotelsDomain.Hotel
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Empty(t, got)

		svc.AssertExpectations(t)
	})

	t.Run("missing offset -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var got map[string]string
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Contains(t, got["error"], "invalid request")

		svc.AssertNotCalled(t, "Search", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("missing limit -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=0", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var got map[string]string
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Contains(t, got["error"], "invalid request")

		svc.AssertNotCalled(t, "Search", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("invalid offset -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=abc&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var got map[string]string
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Contains(t, got["error"], "invalid request")

		svc.AssertNotCalled(t, "Search", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("invalid limit -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=0&limit=xyz", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var got map[string]string
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Contains(t, got["error"], "invalid request")

		svc.AssertNotCalled(t, "Search", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("service error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Search", mock.Anything, "test", 0, 10).Return(nil, errors.New("solr connection error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var got map[string]string
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Contains(t, got["error"], "error searching hotels")

		svc.AssertExpectations(t)
	})

	t.Run("with pagination", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{ID: "hotel3", Name: "Paginated Hotel"},
		}

		svc.On("Search", mock.Anything, "hotel", 20, 5).Return(mockHotels, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=hotel&offset=20&limit=5", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got []hotelsDomain.Hotel
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got, 1)
		assert.Equal(t, "hotel3", got[0].ID)

		svc.AssertExpectations(t)
	})

	t.Run("empty query", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{ID: "hotel1", Name: "Hotel One"},
			{ID: "hotel2", Name: "Hotel Two"},
		}

		svc.On("Search", mock.Anything, "", 0, 10).Return(mockHotels, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got []hotelsDomain.Hotel
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got, 2)

		svc.AssertExpectations(t)
	})

	t.Run("special characters in query", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		mockHotels := []hotelsDomain.Hotel{
			{ID: "hotel1", Name: "Hotel & Spa"},
		}

		svc.On("Search", mock.Anything, "hotel & spa", 0, 10).Return(mockHotels, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/search?q=hotel+%26+spa&offset=0&limit=10", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got []hotelsDomain.Hotel
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got, 1)
		assert.Equal(t, "Hotel & Spa", got[0].Name)

		svc.AssertExpectations(t)
	})
}
