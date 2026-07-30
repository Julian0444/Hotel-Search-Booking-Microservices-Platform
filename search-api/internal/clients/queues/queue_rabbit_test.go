package queues

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/streadway/amqp"
)

// fakeAcker registra los acks/nacks para testear la política de E1 sin broker.
type fakeAcker struct {
	acks        int
	nacks       int
	lastRequeue bool
}

func (f *fakeAcker) Ack(tag uint64, multiple bool) error { f.acks++; return nil }
func (f *fakeAcker) Nack(tag uint64, multiple bool, requeue bool) error {
	f.nacks++
	f.lastRequeue = requeue
	return nil
}
func (f *fakeAcker) Reject(tag uint64, requeue bool) error { return nil }

func delivery(acker *fakeAcker, body string, redelivered bool) amqp.Delivery {
	return amqp.Delivery{Acknowledger: acker, Body: []byte(body), Redelivered: redelivered}
}

// E1: mensaje procesado OK → ack
func TestHandleDelivery_AckOnSuccess(t *testing.T) {
	queue := &Rabbit{}
	acker := &fakeAcker{}

	called := false
	queue.handleDelivery(delivery(acker, `{"operation":"CREATE","hotel_id":"h1"}`, false),
		func(_ context.Context, hotelNew hotels.HotelNew) error {
			called = true
			if hotelNew.Operation != "CREATE" || hotelNew.HotelID != "h1" {
				t.Fatalf("unexpected event: %+v", hotelNew)
			}
			return nil
		})

	if !called {
		t.Fatal("handler was not called")
	}
	if acker.acks != 1 || acker.nacks != 0 {
		t.Fatalf("expected 1 ack / 0 nacks, got %d/%d", acker.acks, acker.nacks)
	}
}

// E1: primer fallo → nack con requeue (reintento)
func TestHandleDelivery_RequeueOnFirstFailure(t *testing.T) {
	queue := &Rabbit{}
	acker := &fakeAcker{}

	queue.handleDelivery(delivery(acker, `{"operation":"CREATE","hotel_id":"h1"}`, false),
		func(_ context.Context, _ hotels.HotelNew) error { return errors.New("solr down") })

	if acker.nacks != 1 || !acker.lastRequeue {
		t.Fatalf("expected 1 nack with requeue=true, got nacks=%d requeue=%v", acker.nacks, acker.lastRequeue)
	}
}

// E1: segundo fallo (Redelivered) → nack sin requeue (va a la DLQ)
func TestHandleDelivery_DLQOnRedeliveredFailure(t *testing.T) {
	queue := &Rabbit{}
	acker := &fakeAcker{}

	queue.handleDelivery(delivery(acker, `{"operation":"CREATE","hotel_id":"h1"}`, true),
		func(_ context.Context, _ hotels.HotelNew) error { return errors.New("solr down") })

	if acker.nacks != 1 || acker.lastRequeue {
		t.Fatalf("expected 1 nack with requeue=false, got nacks=%d requeue=%v", acker.nacks, acker.lastRequeue)
	}
}

// E1: mensaje veneno (JSON inválido) → DLQ directo, sin invocar el handler
func TestHandleDelivery_PoisonMessageGoesToDLQ(t *testing.T) {
	queue := &Rabbit{}
	acker := &fakeAcker{}

	queue.handleDelivery(delivery(acker, `not-json`, false),
		func(_ context.Context, _ hotels.HotelNew) error {
			t.Fatal("handler must not run for unparseable messages")
			return nil
		})

	if acker.nacks != 1 || acker.lastRequeue {
		t.Fatalf("expected 1 nack with requeue=false, got nacks=%d requeue=%v", acker.nacks, acker.lastRequeue)
	}
	if acker.acks != 0 {
		t.Fatalf("expected no acks, got %d", acker.acks)
	}
}

// R2: el handler recibe un ctx CON deadline (antes: context.Background sin
// tope — un Solr colgado frenaba el consumer para siempre) y con request_id.
func TestHandleDelivery_ContextHasDeadlineAndRequestID(t *testing.T) {
	queue := &Rabbit{}
	acker := &fakeAcker{}

	queue.handleDelivery(delivery(acker, `{"operation":"CREATE","hotel_id":"h1"}`, false),
		func(ctx context.Context, _ hotels.HotelNew) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("expected per-message deadline in handler context")
			}
			if remaining := time.Until(deadline); remaining <= 0 || remaining > messageTimeout {
				t.Fatalf("deadline out of range: %v remaining", remaining)
			}
			if utils.RequestIDFromContext(ctx) == "" {
				t.Fatal("expected request_id in handler context")
			}
			return nil
		})

	if acker.acks != 1 {
		t.Fatalf("expected 1 ack, got %d", acker.acks)
	}
}

// RV17: un consumer sin loop de consumo activo NO está "conectado"
func TestIsConnected_FalseWithoutActiveConsumer(t *testing.T) {
	queue := &Rabbit{}
	if queue.IsConnected() {
		t.Fatal("expected IsConnected to be false without connection/consumer")
	}
}

func TestCloseIsSafeOnZeroValue(t *testing.T) {
	queue := &Rabbit{}
	queue.Close() // no debe panic
	if !queue.isClosed() {
		t.Fatal("expected closed flag after Close")
	}
}
