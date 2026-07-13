package queues

import (
	"sync"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

// MockQueue almacena mensajes publicados en memoria para inspección en tests.
type MockQueue struct {
	mu                  sync.Mutex
	messages            []hotelsDomain.HotelNew
	reservationMessages []hotelsDomain.ReservationNew
}

func NewMock() MockQueue {
	return MockQueue{
		messages:            make([]hotelsDomain.HotelNew, 0),
		reservationMessages: make([]hotelsDomain.ReservationNew, 0),
	}
}

func (mq *MockQueue) Publish(hotelNew hotelsDomain.HotelNew) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.messages = append(mq.messages, hotelNew)
	return nil
}

func (mq *MockQueue) PublishReservation(reservationNew hotelsDomain.ReservationNew) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.reservationMessages = append(mq.reservationMessages, reservationNew)
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

// ReservationMessages devuelve una copia de los eventos de reserva publicados.
func (mq *MockQueue) ReservationMessages() []hotelsDomain.ReservationNew {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	cp := make([]hotelsDomain.ReservationNew, len(mq.reservationMessages))
	copy(cp, mq.reservationMessages)
	return cp
}
