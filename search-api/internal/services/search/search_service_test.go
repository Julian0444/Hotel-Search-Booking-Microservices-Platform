package search_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	service "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/services/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (service.Service, *solrMock, *hotelsAPIMock) {
	solrRepo := newSolrMock()
	hotelsAPI := newHotelsAPIMock()

	svc := service.NewService(solrRepo, hotelsAPI)
	return svc, solrRepo, hotelsAPI
}

func TestService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		mockHotels := []hotelsDAO.Hotel{
			{
				ID:             "hotel1",
				Name:           "Hotel Paradise",
				Description:    "Un hotel de lujo",
				City:           "Buenos Aires",
				Country:        "Argentina",
				Rating:         4.5,
				PricePerNight:  150.0,
				AvailableRooms: 10,
				Amenities:      []string{"wifi", "pool"},
				Images:         []string{"img1.jpg"},
			},
			{
				ID:             "hotel2",
				Name:           "Hotel Sunset",
				Description:    "Vista al mar",
				City:           "Cancun",
				Country:        "Mexico",
				Rating:         4.8,
				PricePerNight:  200.0,
				AvailableRooms: 5,
				Amenities:      []string{"wifi", "spa"},
				Images:         []string{"img2.jpg"},
			},
		}

		solrRepo.On("Search", mock.Anything, "paradise", "", 10, 0).Return(mockHotels, 25, nil).Once()

		result, total, err := svc.Search(context.Background(), "paradise", "", 0, 10)

		assert.NoError(t, err)
		// A5/RV22: el total (numFound) atraviesa el service para el meta
		assert.Equal(t, 25, total)
		assert.Len(t, result, 2)
		assert.Equal(t, "hotel1", result[0].ID)
		assert.Equal(t, "Hotel Paradise", result[0].Name)
		assert.Equal(t, 4.5, result[0].Rating)
		assert.Equal(t, "Hotel Sunset", result[1].Name)
		assert.Equal(t, "Mexico", result[1].Country)

		solrRepo.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Search", mock.Anything, "nonexistent", "", 10, 0).Return([]hotelsDAO.Hotel{}, 0, nil).Once()

		result, total, err := svc.Search(context.Background(), "nonexistent", "", 0, 10)

		assert.NoError(t, err)
		assert.Zero(t, total)
		assert.Empty(t, result)

		solrRepo.AssertExpectations(t)
	})

	t.Run("solr error", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Search", mock.Anything, "test", "", 10, 0).Return(nil, 0, errors.New("solr connection error")).Once()

		result, _, err := svc.Search(context.Background(), "test", "", 0, 10)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error searching hotels")

		solrRepo.AssertExpectations(t)
	})

	t.Run("with offset and limit", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		mockHotels := []hotelsDAO.Hotel{
			{ID: "hotel3", Name: "Hotel Paginated", City: "Madrid"},
		}

		solrRepo.On("Search", mock.Anything, "hotel", "", 5, 10).Return(mockHotels, 1, nil).Once()

		result, _, err := svc.Search(context.Background(), "hotel", "", 10, 5)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "hotel3", result[0].ID)

		solrRepo.AssertExpectations(t)
	})

	// plan 13: el sort llega al repositorio tal cual (el controller ya lo
	// whitelisteó; el mapping a Solr es del repo)
	t.Run("sort passes through to the repository", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Search", mock.Anything, "hotel", "price_asc", 10, 0).
			Return([]hotelsDAO.Hotel{}, 0, nil).Once()

		_, _, err := svc.Search(context.Background(), "hotel", "price_asc", 0, 10)

		assert.NoError(t, err)
		solrRepo.AssertExpectations(t)
	})
}

func TestService_HandleHotelNew_Create(t *testing.T) {
	t.Run("create success", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelDomain := hotelsDomain.Hotel{
			ID:             "hotel1",
			Name:           "New Hotel",
			Description:    "Brand new hotel",
			City:           "Lima",
			Country:        "Peru",
			Rating:         4.0,
			PricePerNight:  100.0,
			AvailableRooms: 20,
			CheckInTime:    "14:00",
			CheckOutTime:   "10:00",
			Amenities:      []string{"wifi"},
			Images:         []string{"new.jpg"},
		}

		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelDomain, nil).Once()
		solrRepo.On("Index", mock.Anything, mock.MatchedBy(func(h hotelsDAO.Hotel) bool {
			return h.ID == "hotel1" && h.Name == "New Hotel"
		})).Return("hotel1", nil).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "CREATE",
			HotelID:   "hotel1",
		}

		// E1: procesado OK → nil (el consumer ackea)
		assert.NoError(t, svc.HandleHotelNew(context.Background(), hotelNew))

		solrRepo.AssertExpectations(t)
		hotelsAPI.AssertExpectations(t)
	})

	t.Run("create - hotels api error", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelsDomain.Hotel{}, errors.New("api error")).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "CREATE",
			HotelID:   "hotel1",
		}

		// E1: fallo transitorio → error (el consumer reintenta/DLQea)
		assert.Error(t, svc.HandleHotelNew(context.Background(), hotelNew))

		// Index no debería ser llamado si falla obtener el hotel
		solrRepo.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
		hotelsAPI.AssertExpectations(t)
	})

	// RV14: hotel inexistente en hotels-api NO es error (se descarta el
	// evento) y se limpia el doc del índice
	t.Run("create - hotel deleted meanwhile (404) is discarded", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		notFound := fmt.Errorf("fetching hotel: %w", hotelsDomain.ErrHotelNotFound)
		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelsDomain.Hotel{}, notFound).Once()
		solrRepo.On("Delete", mock.Anything, "hotel1").Return(nil).Once()

		err := svc.HandleHotelNew(context.Background(), hotelsDomain.HotelNew{
			Operation: "CREATE",
			HotelID:   "hotel1",
		})

		assert.NoError(t, err, "un 404 tipado se descarta, no se reintenta")
		solrRepo.AssertExpectations(t)
		solrRepo.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
	})

	// RV14 bis: si la limpieza del doc falla, el evento SÍ se reintenta
	t.Run("create - 404 with solr delete failure is retried", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		notFound := fmt.Errorf("fetching hotel: %w", hotelsDomain.ErrHotelNotFound)
		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelsDomain.Hotel{}, notFound).Once()
		solrRepo.On("Delete", mock.Anything, "hotel1").Return(errors.New("solr down")).Once()

		err := svc.HandleHotelNew(context.Background(), hotelsDomain.HotelNew{
			Operation: "CREATE",
			HotelID:   "hotel1",
		})

		assert.Error(t, err)
		solrRepo.AssertExpectations(t)
	})

	t.Run("create - solr index error", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelDomain := hotelsDomain.Hotel{
			ID:   "hotel1",
			Name: "Test Hotel",
		}

		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelDomain, nil).Once()
		solrRepo.On("Index", mock.Anything, mock.Anything).Return("", errors.New("solr error")).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "CREATE",
			HotelID:   "hotel1",
		}

		assert.Error(t, svc.HandleHotelNew(context.Background(), hotelNew))

		solrRepo.AssertExpectations(t)
		hotelsAPI.AssertExpectations(t)
	})
}

func TestService_HandleHotelNew_Update(t *testing.T) {
	t.Run("update success", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelDomain := hotelsDomain.Hotel{
			ID:            "hotel1",
			Name:          "Updated Hotel",
			Description:   "Updated description",
			City:          "Santiago",
			Country:       "Chile",
			Rating:        4.7,
			PricePerNight: 180.0,
		}

		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelDomain, nil).Once()
		solrRepo.On("Update", mock.Anything, mock.MatchedBy(func(h hotelsDAO.Hotel) bool {
			return h.ID == "hotel1" && h.Name == "Updated Hotel"
		})).Return(nil).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "UPDATE",
			HotelID:   "hotel1",
		}

		assert.NoError(t, svc.HandleHotelNew(context.Background(), hotelNew))

		solrRepo.AssertExpectations(t)
		hotelsAPI.AssertExpectations(t)
	})

	t.Run("update - solr error", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelDomain := hotelsDomain.Hotel{
			ID:   "hotel1",
			Name: "Test Hotel",
		}

		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelDomain, nil).Once()
		solrRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("solr update error")).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "UPDATE",
			HotelID:   "hotel1",
		}

		assert.Error(t, svc.HandleHotelNew(context.Background(), hotelNew))

		solrRepo.AssertExpectations(t)
		hotelsAPI.AssertExpectations(t)
	})
}

func TestService_HandleHotelNew_Delete(t *testing.T) {
	t.Run("delete success", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Delete", mock.Anything, "hotel1").Return(nil).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "DELETE",
			HotelID:   "hotel1",
		}

		assert.NoError(t, svc.HandleHotelNew(context.Background(), hotelNew))

		solrRepo.AssertExpectations(t)
	})

	t.Run("delete - solr error", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Delete", mock.Anything, "hotel1").Return(errors.New("solr delete error")).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "DELETE",
			HotelID:   "hotel1",
		}

		assert.Error(t, svc.HandleHotelNew(context.Background(), hotelNew))

		solrRepo.AssertExpectations(t)
	})
}

func TestService_HandleHotelNew_UnknownOperation(t *testing.T) {
	t.Run("unknown operation", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "UNKNOWN",
			HotelID:   "hotel1",
		}

		// Mensaje no procesable → error (el consumer lo termina mandando a la
		// DLQ en vez de perderlo en silencio)
		assert.Error(t, svc.HandleHotelNew(context.Background(), hotelNew))

		// No debería llamar a ningún método del repositorio
		solrRepo.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
		solrRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		solrRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		hotelsAPI.AssertNotCalled(t, "GetHotelByID", mock.Anything, mock.Anything)
	})
}

// E3: el backfill pagina GET /hotels e indexa todo el catálogo
func TestService_Backfill(t *testing.T) {
	t.Run("indexes all pages", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		// 3 hoteles en 2 páginas (pageSize=50: el mock devuelve páginas cortas
		// con total=3 para forzar la segunda vuelta)
		page1 := []hotelsDomain.Hotel{{ID: "h1", Name: "Uno"}, {ID: "h2", Name: "Dos"}}
		page2 := []hotelsDomain.Hotel{{ID: "h3", Name: "Tres"}}
		hotelsAPI.On("GetHotels", mock.Anything, 50, 0).Return(page1, 3, nil).Once()
		hotelsAPI.On("GetHotels", mock.Anything, 50, 2).Return(page2, 3, nil).Once()
		for _, id := range []string{"h1", "h2", "h3"} {
			id := id
			solrRepo.On("Index", mock.Anything, mock.MatchedBy(func(h hotelsDAO.Hotel) bool {
				return h.ID == id
			})).Return(id, nil).Once()
		}

		indexed, err := svc.Backfill(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 3, indexed)
		solrRepo.AssertExpectations(t)
		hotelsAPI.AssertExpectations(t)
	})

	t.Run("empty catalog", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelsAPI.On("GetHotels", mock.Anything, 50, 0).Return([]hotelsDomain.Hotel{}, 0, nil).Once()

		indexed, err := svc.Backfill(context.Background())

		assert.NoError(t, err)
		assert.Zero(t, indexed)
		solrRepo.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
	})

	t.Run("fetch error propagates", func(t *testing.T) {
		svc, _, hotelsAPI := newTestService()

		hotelsAPI.On("GetHotels", mock.Anything, 50, 0).Return(nil, 0, errors.New("hotels-api down")).Once()

		_, err := svc.Backfill(context.Background())
		assert.Error(t, err)
	})

	t.Run("index error propagates", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelsAPI.On("GetHotels", mock.Anything, 50, 0).
			Return([]hotelsDomain.Hotel{{ID: "h1"}}, 1, nil).Once()
		solrRepo.On("Index", mock.Anything, mock.Anything).Return("", errors.New("solr down")).Once()

		_, err := svc.Backfill(context.Background())
		assert.Error(t, err)
	})
}
