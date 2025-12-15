package queues

import (
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

func TestRabbitQueuePublishWithoutChannel(t *testing.T) {
	rq := RabbitQueue{} // channel nil simula fallo de conexión
	err := rq.Publish(hotelsDomain.HotelNew{Operation: "CREATE", HotelID: "1"})
	if err == nil {
		t.Fatalf("expected error when channel is nil")
	}
}

func TestRabbitQueueCloseIsSafe(t *testing.T) {
	rq := RabbitQueue{} // channel/connection nil
	// No debe panic
	rq.Close()
}

func TestMockQueuePublish(t *testing.T) {
	mq := MockQueue{}
	if err := mq.Publish(hotelsDomain.HotelNew{Operation: "TEST", HotelID: "123"}); err != nil {
		t.Fatalf("mock publish should not error: %v", err)
	}

	msgs := mq.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message stored, got %d", len(msgs))
	}
	if msgs[0].Operation != "TEST" || msgs[0].HotelID != "123" {
		t.Fatalf("unexpected message stored: %+v", msgs[0])
	}
}
