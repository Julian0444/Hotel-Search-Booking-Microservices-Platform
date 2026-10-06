package queues

import (
	"context"
	"sync"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

// MockQueue almacena eventos de catálogo para tests.
type MockQueue struct {
	mu       sync.Mutex
	messages []hotelsDomain.HotelNew
}

func NewMock() MockQueue { return MockQueue{} }
func (mq *MockQueue) Publish(_ context.Context, event hotelsDomain.HotelNew) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.messages = append(mq.messages, event)
	return nil
}
func (mq *MockQueue) Messages() []hotelsDomain.HotelNew {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return append([]hotelsDomain.HotelNew(nil), mq.messages...)
}
