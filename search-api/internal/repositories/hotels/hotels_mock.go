package hotels

import (
	"context"

	"github.com/stretchr/testify/mock"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
)

// Mock implementa la interfaz Repository (Solr) para testing.
type Mock struct {
	mock.Mock
}

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) Index(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	args := m.Called(ctx, hotel)
	return args.String(0), args.Error(1)
}

func (m *Mock) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	args := m.Called(ctx, hotel)
	return args.Error(0)
}

func (m *Mock) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *Mock) Search(ctx context.Context, query string, limit int, offset int) ([]hotelsDAO.Hotel, int, error) {
	args := m.Called(ctx, query, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]hotelsDAO.Hotel), args.Int(1), args.Error(2)
}

// ExternalMock implementa la interfaz ExternalRepository (Hotels API) para testing.
type ExternalMock struct {
	mock.Mock
}

func NewExternalMock() *ExternalMock {
	return &ExternalMock{}
}

func (m *ExternalMock) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(hotelsDomain.Hotel), args.Error(1)
}

func (m *ExternalMock) GetHotels(ctx context.Context, limit, offset int) ([]hotelsDomain.Hotel, int, error) {
	args := m.Called(ctx, limit, offset)
	var hotels []hotelsDomain.Hotel
	if args.Get(0) != nil {
		hotels = args.Get(0).([]hotelsDomain.Hotel)
	}
	return hotels, args.Int(1), args.Error(2)
}
