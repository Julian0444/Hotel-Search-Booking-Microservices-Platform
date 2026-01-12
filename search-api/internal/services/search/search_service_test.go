package search_test

import (
	"context"
	"errors"
	"testing"
	"time"

	hotelsDAO "search-api/internal/dao/hotels"
	hotelsDomain "search-api/internal/domain/hotels"
	hotelsRepo "search-api/internal/repositories/hotels"
	service "search-api/internal/services/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (service.Service, *hotelsRepo.Mock, *hotelsRepo.ExternalMock) {
	solrRepo := hotelsRepo.NewMock()
	hotelsAPI := hotelsRepo.NewExternalMock()

	svc := service.NewService(solrRepo, hotelsAPI)
	return svc, solrRepo, hotelsAPI
}

func TestService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		mockHotels := []hotelsDAO.Hotel{
			{
				ID:            "hotel1",
				Name:          "Hotel Paradise",
				Description:   "Un hotel de lujo",
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
				Description:   "Vista al mar",
				City:          "Cancun",
				Country:       "Mexico",
				Rating:        4.8,
				PricePerNight: 200.0,
				AvaiableRooms: 5,
				Amenities:     []string{"wifi", "spa"},
				Images:        []string{"img2.jpg"},
			},
		}

		solrRepo.On("Search", mock.Anything, "paradise", 10, 0).Return(mockHotels, nil).Once()

		result, err := svc.Search(context.Background(), "paradise", 0, 10)

		assert.NoError(t, err)
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

		solrRepo.On("Search", mock.Anything, "nonexistent", 10, 0).Return([]hotelsDAO.Hotel{}, nil).Once()

		result, err := svc.Search(context.Background(), "nonexistent", 0, 10)

		assert.NoError(t, err)
		assert.Empty(t, result)

		solrRepo.AssertExpectations(t)
	})

	t.Run("solr error", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Search", mock.Anything, "test", 10, 0).Return(nil, errors.New("solr connection error")).Once()

		result, err := svc.Search(context.Background(), "test", 0, 10)

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

		solrRepo.On("Search", mock.Anything, "hotel", 5, 10).Return(mockHotels, nil).Once()

		result, err := svc.Search(context.Background(), "hotel", 10, 5)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "hotel3", result[0].ID)

		solrRepo.AssertExpectations(t)
	})
}

func TestService_HandleHotelNew_Create(t *testing.T) {
	t.Run("create success", func(t *testing.T) {
		svc, solrRepo, hotelsAPI := newTestService()

		hotelDomain := hotelsDomain.Hotel{
			ID:            "hotel1",
			Name:          "New Hotel",
			Description:   "Brand new hotel",
			City:          "Lima",
			Country:       "Peru",
			Rating:        4.0,
			PricePerNight: 100.0,
			AvaiableRooms: 20,
			CheckInTime:   time.Now(),
			CheckOutTime:  time.Now().Add(24 * time.Hour),
			Amenities:     []string{"wifi"},
			Images:        []string{"new.jpg"},
		}

		hotelsAPI.On("GetHotelByID", mock.Anything, "hotel1").Return(hotelDomain, nil).Once()
		solrRepo.On("Index", mock.Anything, mock.MatchedBy(func(h hotelsDAO.Hotel) bool {
			return h.ID == "hotel1" && h.Name == "New Hotel"
		})).Return("hotel1", nil).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "CREATE",
			HotelID:   "hotel1",
		}

		// HandleHotelNew no devuelve error, solo imprime logs
		svc.HandleHotelNew(hotelNew)

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

		svc.HandleHotelNew(hotelNew)

		// Index no debería ser llamado si falla obtener el hotel
		solrRepo.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
		hotelsAPI.AssertExpectations(t)
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

		svc.HandleHotelNew(hotelNew)

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

		svc.HandleHotelNew(hotelNew)

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

		svc.HandleHotelNew(hotelNew)

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

		svc.HandleHotelNew(hotelNew)

		solrRepo.AssertExpectations(t)
	})

	t.Run("delete - solr error", func(t *testing.T) {
		svc, solrRepo, _ := newTestService()

		solrRepo.On("Delete", mock.Anything, "hotel1").Return(errors.New("solr delete error")).Once()

		hotelNew := hotelsDomain.HotelNew{
			Operation: "DELETE",
			HotelID:   "hotel1",
		}

		svc.HandleHotelNew(hotelNew)

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

		svc.HandleHotelNew(hotelNew)

		// No debería llamar a ningún método del repositorio
		solrRepo.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
		solrRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		solrRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		hotelsAPI.AssertNotCalled(t, "GetHotelByID", mock.Anything, mock.Anything)
	})
}
