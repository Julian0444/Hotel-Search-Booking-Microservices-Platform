package queues

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
	consumerTag    = "search-api-consumer"
	// Dos hops de hasta 5s más espera cancelable por reindex. El intento
	// vence a los 15s; reintentarlo nunca descarta un evento válido.
	messageTimeout = 15 * time.Second
	drainTimeout   = 6 * time.Second
)

type RabbitConfig struct{ Host, Port, Username, Password, QueueName string }

type Rabbit struct {
	config            RabbitConfig
	mu                sync.RWMutex
	connection        *amqp.Connection
	channel           *amqp.Channel
	consuming, closed bool
	loopDone          chan struct{}
	ctx               context.Context
	cancel            context.CancelFunc
	// Opciones internas inyectables para probar tiempos sin esperas largas.
	retryDelay     time.Duration
	attemptTimeout time.Duration
}

// La conexión sucede en el loop: un broker caído no bloquea el HTTP de búsqueda.
func NewRabbit(config RabbitConfig) *Rabbit {
	ctx, cancel := context.WithCancel(context.Background())
	return &Rabbit{config: config, ctx: ctx, cancel: cancel}
}

// connect abre conexión y canal, y declara la topología de la cola.
func (queue *Rabbit) connect() error {
	queue.mu.Lock()
	oldChannel, oldConnection := queue.channel, queue.connection
	queue.channel, queue.connection, queue.consuming = nil, nil, false
	closed := queue.closed
	queue.mu.Unlock()
	if oldChannel != nil {
		_ = oldChannel.Close()
	}
	if oldConnection != nil {
		_ = oldConnection.Close()
	}
	if closed {
		return context.Canceled
	}

	conn, err := amqp.DialConfig(fmt.Sprintf("amqp://%s:%s@%s:%s/",
		queue.config.Username, queue.config.Password, queue.config.Host, queue.config.Port), amqp.Config{Dial: amqp.DefaultDial(5 * time.Second)})
	if err != nil {
		return fmt.Errorf("error getting Rabbit connection: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("error creating Rabbit channel: %w", err)
	}

	fail := func(err error) error {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}

	// Topología con dead-lettering (E1): DLX + DLQ + cola principal con
	// x-dead-letter-exchange. IMPORTANTE: espejo EXACTO de la declaración del
	// productor (hotels-api) — args distintos dan 406 PRECONDITION_FAILED.
	dlx := queue.config.QueueName + "-dlx"
	dlq := queue.config.QueueName + "-dlq"
	if err := ch.ExchangeDeclare(dlx, "direct", true, false, false, false, nil); err != nil {
		return fail(fmt.Errorf("error declaring exchange %s: %w", dlx, err))
	}
	if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return fail(fmt.Errorf("error declaring queue %s: %w", dlq, err))
	}
	if err := ch.QueueBind(dlq, queue.config.QueueName, dlx, false, nil); err != nil {
		return fail(fmt.Errorf("error binding queue %s: %w", dlq, err))
	}
	if _, err := ch.QueueDeclare(queue.config.QueueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": dlx,
	}); err != nil {
		return fail(fmt.Errorf("error declaring queue %s: %w", queue.config.QueueName, err))
	}

	// Prefetch 1: no acumular mensajes sin ack en un consumer que procesa de a uno
	if err := ch.Qos(1, 0, false); err != nil {
		return fail(fmt.Errorf("error setting channel QoS: %w", err))
	}

	queue.mu.Lock()
	if queue.closed {
		queue.mu.Unlock()
		return fail(context.Canceled)
	}
	queue.connection = conn
	queue.channel = ch
	queue.mu.Unlock()
	return nil
}

func (queue *Rabbit) IsConnected() bool {
	queue.mu.RLock()
	defer queue.mu.RUnlock()
	return queue.connection != nil && !queue.connection.IsClosed() && queue.channel != nil && !queue.channel.IsClosed() && queue.consuming
}

func (queue *Rabbit) isClosed() bool {
	queue.mu.RLock()
	defer queue.mu.RUnlock()
	return queue.closed
}

func (queue *Rabbit) context() context.Context {
	if queue.ctx != nil {
		return queue.ctx
	}
	return context.Background()
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (queue *Rabbit) StartConsumer(handler func(context.Context, hotels.HotelNew) error) {
	queue.mu.Lock()
	if queue.closed || queue.loopDone != nil {
		queue.mu.Unlock()
		return
	}
	queue.loopDone = make(chan struct{})
	done := queue.loopDone
	queue.mu.Unlock()
	go func() { defer close(done); queue.consumeLoop(handler) }()
}

func (queue *Rabbit) consumeLoop(handler func(context.Context, hotels.HotelNew) error) {
	backoff := initialBackoff
	for !queue.isClosed() {
		err := queue.connect()
		if err != nil {
			slog.Warn("RabbitMQ connection failed; retrying", "error", err, "delay", backoff)
			if !wait(queue.context(), backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		queue.mu.RLock()
		channel := queue.channel
		queue.mu.RUnlock()
		// Exclusivo: un segundo consumer accidental falla, no comparte eventos.
		messages, err := channel.Consume(queue.config.QueueName, consumerTag, false, true, false, false, nil)
		if err != nil {
			slog.Warn("registering consumer failed", "error", err)
			if !wait(queue.context(), backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		backoff = initialBackoff
		queue.mu.Lock()
		queue.consuming = true
		queue.mu.Unlock()
		for msg := range messages {
			queue.handleDelivery(msg, handler)
			if queue.isClosed() {
				break
			}
		}
		queue.mu.Lock()
		queue.consuming = false
		queue.mu.Unlock()
	}
}

func validHotelID(id string) bool {
	if len(id) != 24 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// Mantiene el mensaje sin ack mientras una dependencia falla, con prefetch=1.
// Reintentos indefinidos 1,2,4,8,16,30s; un reinicio conserva el mensaje en
// RabbitMQ. Solo datos inválidos van a DLQ. No hay requeue ocupado ni cola extra.
func (queue *Rabbit) handleDelivery(msg amqp.Delivery, handler func(context.Context, hotels.HotelNew) error) {
	var event hotels.HotelNew
	if err := json.Unmarshal(msg.Body, &event); err != nil || !validHotelID(event.HotelID) ||
		(event.Operation != "CREATE" && event.Operation != "UPDATE" && event.Operation != "DELETE") {
		slog.Warn("invalid hotel event, sending to DLQ")
		_ = msg.Nack(false, false)
		return
	}
	delay := queue.retryDelay
	if delay <= 0 {
		delay = initialBackoff
	}
	timeout := queue.attemptTimeout
	if timeout <= 0 {
		timeout = messageTimeout
	}
	requestCtx := utils.WithRequestID(queue.context(), uuid.NewString())
	for {
		if queue.isClosed() || requestCtx.Err() != nil {
			_ = msg.Nack(false, true)
			return
		}
		queue.mu.RLock()
		channel := queue.channel
		queue.mu.RUnlock()
		if channel != nil && channel.IsClosed() {
			return
		} // broker reentrega al reconectar
		ctx, cancel := context.WithTimeout(requestCtx, timeout)
		err := handler(ctx, event)
		cancel()
		if err == nil {
			if err := msg.Ack(false); err != nil {
				slog.Warn("ack failed; broker may redeliver", "error", err)
			}
			return
		}
		if errors.Is(err, hotels.ErrInvalidEvent) {
			_ = msg.Nack(false, false)
			return
		}
		slog.Warn("hotel event temporarily failed; retaining unacked delivery", "hotel_id", event.HotelID, "error", err, "delay", delay)
		if !wait(requestCtx, delay) {
			_ = msg.Nack(false, true)
			return
		}
		delay = min(delay*2, maxBackoff)
	}
}

// closeUnsafe requiere mu tomado.
func (queue *Rabbit) closeUnsafe() {
	if queue.channel != nil {
		_ = queue.channel.Close()
		queue.channel = nil
	}
	if queue.connection != nil {
		_ = queue.connection.Close()
		queue.connection = nil
	}
	queue.consuming = false
}

func (queue *Rabbit) Close() {
	queue.mu.Lock()
	queue.closed = true
	if queue.cancel != nil {
		queue.cancel()
	}
	channel, done := queue.channel, queue.loopDone
	queue.mu.Unlock()
	if channel != nil {
		_ = channel.Cancel(consumerTag, false)
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(drainTimeout):
			slog.Warn("consumer shutdown timed out")
		}
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.closeUnsafe()
}
