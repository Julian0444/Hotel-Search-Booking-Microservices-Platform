package queues

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

const (
	// Configuración de reintentos de conexión (mismo patrón que el productor
	// de hotels-api)
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2.0

	// consumerTag identifica el Consume registrado: Close() lo cancela para
	// drenar el mensaje en vuelo antes de cerrar la conexión (C12).
	consumerTag = "search-api-consumer"
	// messageTimeout acota el procesamiento de cada mensaje (R2): antes el
	// handler corría sobre context.Background() sin deadline. Holgado a
	// propósito: contiene los reintentos del cliente HTTP (E4, hasta ~16s) —
	// un tope menor los truncaría y mandaría eventos a la DLQ durante
	// reinicios cortos de hotels-api.
	messageTimeout = 30 * time.Second
	// drainTimeout es cuánto espera Close() a que el loop termine el mensaje
	// en vuelo antes de cerrar igual (C12).
	drainTimeout = 5 * time.Second
)

type RabbitConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	QueueName string
}

// Rabbit es el consumidor de hotels-news con reconexión automática (E5):
// si la conexión o el canal AMQP mueren, el loop del consumer se re-arma solo
// (backoff exponencial) y vuelve a registrar el Consume.
type Rabbit struct {
	config RabbitConfig

	mu         sync.RWMutex
	connection *amqp.Connection
	channel    *amqp.Channel
	// consuming refleja "consumer vivo" (RV17): hay un Consume registrado y su
	// canal de deliveries sigue abierto. Si solo muere el channel AMQP, la
	// conexión puede seguir abierta pero el servicio NO está procesando eventos.
	consuming bool
	closed    bool
	// loopDone se cierra cuando consumeLoop termina: Close() lo espera para
	// no cortar un mensaje a mitad de procesamiento (C12).
	loopDone chan struct{}
}

// NewRabbit crea el consumidor. La conexión inicial es best-effort con
// reintentos: si RabbitMQ todavía no está listo NO se aborta el arranque
// (antes: log.Fatalf) — el loop del consumer sigue reintentando.
func NewRabbit(config RabbitConfig) *Rabbit {
	rabbit := &Rabbit{config: config}
	if err := rabbit.connectWithRetry(); err != nil {
		slog.Warn("initial RabbitMQ connection failed, consumer loop will keep retrying", "error", err)
	}
	return rabbit
}

// connectWithRetry intenta conectar con backoff exponencial (espejo de hotels-api).
func (queue *Rabbit) connectWithRetry() error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("RabbitMQ connection attempt", "attempt", attempt, "max_retries", maxRetries)

		if err := queue.connect(); err != nil {
			lastErr = err
			slog.Warn("RabbitMQ connection attempt failed", "attempt", attempt, "error", err)

			if attempt < maxRetries {
				slog.Info("retrying RabbitMQ connection", "backoff", backoff.String())
				time.Sleep(backoff)

				backoff = time.Duration(float64(backoff) * backoffFactor)
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		} else {
			slog.Info("connected to RabbitMQ", "attempt", attempt)
			return nil
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, lastErr)
}

// connect abre conexión y canal, y declara la topología de la cola.
func (queue *Rabbit) connect() error {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.closeUnsafe()

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/",
		queue.config.Username, queue.config.Password, queue.config.Host, queue.config.Port))
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

	queue.connection = conn
	queue.channel = ch
	return nil
}

// connectionAlive es el chequeo interno de conexión (sin exigir un consumer
// activo — se usa justo antes de registrar el Consume).
func (queue *Rabbit) connectionAlive() bool {
	queue.mu.RLock()
	defer queue.mu.RUnlock()
	return queue.connection != nil && !queue.connection.IsClosed() && queue.channel != nil
}

// IsConnected informa si el CONSUMER está vivo (lo usa el /readyz, O3/RV17):
// conexión abierta Y loop de consumo activo. Si el canal AMQP muere en
// silencio, deliveries se cierra, consuming pasa a false y el readyz degrada
// aunque la conexión TCP siga abierta.
func (queue *Rabbit) IsConnected() bool {
	queue.mu.RLock()
	defer queue.mu.RUnlock()
	return queue.connection != nil && !queue.connection.IsClosed() && queue.consuming
}

func (queue *Rabbit) setConsuming(value bool) {
	queue.mu.Lock()
	queue.consuming = value
	queue.mu.Unlock()
}

func (queue *Rabbit) isClosed() bool {
	queue.mu.RLock()
	defer queue.mu.RUnlock()
	return queue.closed
}

// StartConsumer lanza el loop del consumidor en background. El handler recibe
// un context con un request_id generado por mensaje (O1) y devuelve error si
// el evento debe reintentarse (E1): 1er fallo → requeue, 2º → DLQ.
func (queue *Rabbit) StartConsumer(handler func(context.Context, hotels.HotelNew) error) {
	queue.mu.Lock()
	queue.loopDone = make(chan struct{})
	queue.mu.Unlock()

	go func() {
		defer close(queue.loopDone)
		queue.consumeLoop(handler)
	}()
}

// consumeLoop mantiene un Consume registrado para siempre: si el canal de
// deliveries se cierra (conexión/canal caído), reconecta con backoff y vuelve
// a consumir (E5). Solo termina con Close().
func (queue *Rabbit) consumeLoop(handler func(context.Context, hotels.HotelNew) error) {
	for {
		if queue.isClosed() {
			return
		}

		if !queue.connectionAlive() {
			if err := queue.connectWithRetry(); err != nil {
				// connectWithRetry ya durmió ~1 min de backoff: loguear y
				// seguir intentando — un consumer no tiene "próximo publish"
				// que dispare la reconexión, así que nunca se rinde.
				slog.Error("RabbitMQ still unreachable, consumer keeps retrying", "error", err)
				continue
			}
		}

		queue.mu.RLock()
		channel := queue.channel
		queue.mu.RUnlock()

		messages, err := channel.Consume(
			queue.config.QueueName,
			consumerTag, // tag conocido: Close() lo cancela para drenar (C12)
			false,       // autoAck=false: ack manual tras procesar OK (E1)
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			slog.Error("error registering consumer, reconnecting", "error", err)
			queue.setConsuming(false)
			if err := queue.connectWithRetry(); err != nil {
				slog.Error("RabbitMQ reconnection failed, retrying", "error", err)
			}
			continue
		}

		queue.setConsuming(true)
		slog.Info("rabbitmq consumer started", "queue", queue.config.QueueName)

		for msg := range messages {
			queue.handleDelivery(msg, handler)
		}

		// El canal de deliveries se cerró: shutdown ordenado o conexión caída
		queue.setConsuming(false)
		if queue.isClosed() {
			return
		}
		slog.Warn("rabbitmq deliveries channel closed, reconnecting")
	}
}

// handleDelivery procesa un mensaje con la política de acks de E1:
//   - unmarshal inválido → DLQ directo (mensaje veneno, requeue jamás lo arregla)
//   - handler falla la 1ª vez → requeue (transitorio: Solr/hotels-api caídos)
//   - handler falla la 2ª vez (Redelivered) → DLQ para inspección/replay
//   - OK → ack
func (queue *Rabbit) handleDelivery(msg amqp.Delivery, handler func(context.Context, hotels.HotelNew) error) {
	var hotelUpdate hotels.HotelNew
	if err := json.Unmarshal(msg.Body, &hotelUpdate); err != nil {
		slog.Error("error unmarshaling message, sending to DLQ", "error", err)
		_ = msg.Nack(false, false)
		return
	}

	// Deadline por mensaje (R2): un Solr/hotels-api colgado no puede frenar el
	// consumer para siempre — al vencer, el error sigue la política retry/DLQ.
	ctx, cancel := context.WithTimeout(utils.WithRequestID(context.Background(), uuid.NewString()), messageTimeout)
	defer cancel()
	if err := handler(ctx, hotelUpdate); err != nil {
		if msg.Redelivered {
			slog.Error("handler failed on redelivery, sending to DLQ",
				"operation", hotelUpdate.Operation, "hotel_id", hotelUpdate.HotelID, "error", err)
			_ = msg.Nack(false, false)
		} else {
			slog.Warn("handler failed, requeueing once",
				"operation", hotelUpdate.Operation, "hotel_id", hotelUpdate.HotelID, "error", err)
			_ = msg.Nack(false, true)
		}
		return
	}

	_ = msg.Ack(false)
}

// closeUnsafe cierra conexión y canal (llamar con mu tomado).
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

// Close termina el consumer y cierra la conexión a RabbitMQ. Drenaje (C12):
// primero cancela el Consume — el broker deja de entregar, el canal de
// deliveries se cierra después del mensaje en vuelo y consumeLoop termina
// solo — y recién entonces cierra canal y conexión. Cortar sin drenar no
// pierde mensajes (manual ack ⇒ re-entrega) pero obliga al retrabajo.
func (queue *Rabbit) Close() {
	queue.mu.Lock()
	queue.closed = true
	channel := queue.channel
	consuming := queue.consuming
	loopDone := queue.loopDone
	queue.mu.Unlock()

	if channel != nil && consuming {
		if err := channel.Cancel(consumerTag, false); err != nil {
			slog.Warn("error cancelling consumer", "error", err)
		}
	}
	if loopDone != nil {
		select {
		case <-loopDone:
		case <-time.After(drainTimeout):
			slog.Warn("consumer did not drain in time, closing anyway")
		}
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.closeUnsafe()
}
