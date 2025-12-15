package queues

import (
	"sync"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

// MockQueue almacena mensajes publicados en memoria para inspección en tests.
type MockQueue struct {
	mu       sync.Mutex
	messages []hotelsDomain.HotelNew
}

func NewMock() MockQueue {
	return MockQueue{
		messages: make([]hotelsDomain.HotelNew, 0),
	}
}

func (mq *MockQueue) Publish(hotelNew hotelsDomain.HotelNew) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.messages = append(mq.messages, hotelNew)
	return nil
}

// Messages devuelve una copia de los mensajes publicados (para asserts en tests).
func (mq *MockQueue) Messages() []hotelsDomain.HotelNew {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	cp := make([]hotelsDomain.HotelNew, len(mq.messages))
	copy(cp, mq.messages)
	return cp
}
