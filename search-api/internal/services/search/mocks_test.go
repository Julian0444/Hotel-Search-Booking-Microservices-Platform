package search_test

// Mocks de testify SOLO en archivos _test (C13): antes vivían como código de
// producción en repositories/hotels y metían testify en el binario de
// search-api.

import (
	"context"

	"github.com/stretchr/testify/mock"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
)

// solrMock implementa la interfaz Repository (Solr) del service.
type solrMock struct {
	mock.Mock
}

func newSolrMock() *solrMock {
	return &solrMock{}
}

func (m *solrMock) Index(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	args := m.Called(ctx, hotel)
	return args.String(0), args.Error(1)
}

func (m *solrMock) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	args := m.Called(ctx, hotel)
	return args.Error(0)
}

func (m *solrMock) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *solrMock) Search(ctx context.Context, query string, sort string, limit int, offset int) ([]hotelsDAO.Hotel, int, error) {
	args := m.Called(ctx, query, sort, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]hotelsDAO.Hotel), args.Int(1), args.Error(2)
}

// hotelsAPIMock implementa la interfaz ExternalRepository (Hotels API) del service.
type hotelsAPIMock struct {
	mock.Mock
}

func newHotelsAPIMock() *hotelsAPIMock {
	return &hotelsAPIMock{}
}

func (m *hotelsAPIMock) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(hotelsDomain.Hotel), args.Error(1)
}

func (m *hotelsAPIMock) GetHotels(ctx context.Context, limit, offset int) ([]hotelsDomain.Hotel, int, error) {
	args := m.Called(ctx, limit, offset)
	var hotels []hotelsDomain.Hotel
	if args.Get(0) != nil {
		hotels = args.Get(0).([]hotelsDomain.Hotel)
	}
	return hotels, args.Int(1), args.Error(2)
}
