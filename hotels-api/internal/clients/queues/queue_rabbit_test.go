package queues

import (
	"context"
	"errors"
	"testing"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
)

func TestDisconnectedPublishDoesNotReconnect(t *testing.T) {
	rq := &RabbitQueue{publishGate: make(chan struct{}, 1)}
	start := time.Now()
	for i := 0; i < 20; i++ {
		if err := rq.Publish(context.Background(), hotelsDomain.HotelNew{}); err == nil {
			t.Fatal("expected unavailable")
		}
	}
	if time.Since(start) > time.Second {
		t.Fatal("disconnected requests must not execute connection backoffs")
	}
	if rq.IsConnected() {
		t.Fatal("unexpected connection")
	}
	rq.Close()
}

func TestPublishBudgetIncludesConcurrentWait(t *testing.T) {
	rq := &RabbitQueue{publishGate: make(chan struct{}, 1)}
	rq.publishGate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := rq.Publish(ctx, hotelsDomain.HotelNew{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
}

func TestMockQueuePublish(t *testing.T) {
	mq := MockQueue{}
	if err := mq.Publish(context.Background(), hotelsDomain.HotelNew{Operation: "CREATE", HotelID: "123"}); err != nil {
		t.Fatal(err)
	}
	msgs := mq.Messages()
	if len(msgs) != 1 || msgs[0].HotelID != "123" {
		t.Fatalf("unexpected messages: %v", msgs)
	}
}
