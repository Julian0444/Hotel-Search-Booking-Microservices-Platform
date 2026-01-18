package queues

import (
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

func TestRabbitQueuePublishWithoutChannel(t *testing.T) {
	// Crear un RabbitQueue sin conexión (simula fallo de conexión)
	rq := &RabbitQueue{
		config: RabbitConfig{
			Host:      "invalid-host",
			Port:      "5672",
			Username:  "guest",
			Password:  "guest",
			QueueName: "test-queue",
		},
		connected: false,
		channel:   nil,
	}

	// Debe fallar porque no hay conexión y el host es inválido
	err := rq.Publish(hotelsDomain.HotelNew{Operation: "CREATE", HotelID: "1"})
	if err == nil {
		t.Fatalf("expected error when channel is nil and cannot reconnect")
	}
}

func TestRabbitQueueCloseIsSafe(t *testing.T) {
	rq := &RabbitQueue{} // channel/connection nil
	// No debe panic
	rq.Close()
}

func TestRabbitQueueIsConnected(t *testing.T) {
	rq := &RabbitQueue{
		connected: false,
		channel:   nil,
	}

	if rq.IsConnected() {
		t.Fatal("expected IsConnected to return false when not connected")
	}
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
