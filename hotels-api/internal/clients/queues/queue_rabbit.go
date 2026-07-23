package queues

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/streadway/amqp"
)

const (
	// Configuración de reintentos
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2.0
)

type RabbitConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	QueueName string
	// Cola separada para eventos de reservas (DM5): search-api consume
	// QueueName esperando HotelNew; mezclar tipos rompería su Unmarshal.
	ReservationsQueueName string
}

type RabbitQueue struct {
	config                RabbitConfig
	connection            *amqp.Connection
	channel               *amqp.Channel
	queueName             string
	reservationsQueueName string
	mu                    sync.RWMutex
	connected             bool
}

// NewRabbit crea una nueva instancia de RabbitQueue con reconexión automática
func NewRabbit(config RabbitConfig) *RabbitQueue {
	rq := &RabbitQueue{
		config:                config,
		queueName:             config.QueueName,
		reservationsQueueName: config.ReservationsQueueName,
		connected:             false,
	}

	// Intentar conexión inicial con reintentos
	if err := rq.connectWithRetry(); err != nil {
		slog.Warn("initial RabbitMQ connection failed, will reconnect on next publish", "retries", maxRetries, "error", err)
	}

	return rq
}

// connectWithRetry intenta conectar a RabbitMQ con backoff exponencial
func (rq *RabbitQueue) connectWithRetry() error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("RabbitMQ connection attempt", "attempt", attempt, "max_retries", maxRetries)

		if err := rq.connect(); err != nil {
			lastErr = err
			slog.Warn("RabbitMQ connection attempt failed", "attempt", attempt, "error", err)

			if attempt < maxRetries {
				slog.Info("retrying RabbitMQ connection", "backoff", backoff.String())
				time.Sleep(backoff)

				// Incrementar backoff exponencialmente
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

// connect establece la conexión a RabbitMQ
func (rq *RabbitQueue) connect() error {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	// Cerrar conexiones existentes si las hay
	rq.closeUnsafe()

	// Crear la URL de conexión
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		rq.config.Username,
		rq.config.Password,
		rq.config.Host,
		rq.config.Port,
	)

	// Conectar a RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		rq.connected = false
		return fmt.Errorf("error connecting to RabbitMQ: %w", err)
	}

	// Crear canal
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		rq.connected = false
		return fmt.Errorf("error creating channel: %w", err)
	}

	fail := func(err error) error {
		_ = ch.Close()
		_ = conn.Close()
		rq.connected = false
		return err
	}

	// Topología de hotels-news con dead-lettering (E1): DLX + DLQ + cola
	// principal con x-dead-letter-exchange. IMPORTANTE: espejo EXACTO de la
	// declaración del consumidor (search-api) — args distintos entre productor
	// y consumidor dan 406 PRECONDITION_FAILED al redeclarar (`docker compose
	// down` limpia el broker efímero para poder re-declarar).
	dlx := rq.config.QueueName + "-dlx"
	dlq := rq.config.QueueName + "-dlq"
	if err := ch.ExchangeDeclare(dlx, "direct", true, false, false, false, nil); err != nil {
		return fail(fmt.Errorf("error declaring exchange %s: %w", dlx, err))
	}
	if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return fail(fmt.Errorf("error declaring queue %s: %w", dlq, err))
	}
	// Los mensajes nackeados llegan al DLX con su routing key original (el
	// nombre de la cola, por publicar al default exchange)
	if err := ch.QueueBind(dlq, rq.config.QueueName, dlx, false, nil); err != nil {
		return fail(fmt.Errorf("error binding queue %s: %w", dlq, err))
	}
	if _, err := ch.QueueDeclare(rq.config.QueueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": dlx,
	}); err != nil {
		return fail(fmt.Errorf("error declaring queue %s: %w", rq.config.QueueName, err))
	}

	// La cola de reservas va sin DLX (nadie la consume aún, DM5) y solo si
	// está configurada: los tests unitarios construyen la config sin ella
	if rq.config.ReservationsQueueName != "" {
		if _, err := ch.QueueDeclare(rq.config.ReservationsQueueName, true, false, false, false, nil); err != nil {
			return fail(fmt.Errorf("error declaring queue %s: %w", rq.config.ReservationsQueueName, err))
		}
	}

	rq.connection = conn
	rq.channel = ch
	rq.connected = true

	// Configurar notificación de cierre de conexión
	go rq.handleConnectionClose()

	return nil
}

// handleConnectionClose maneja el cierre inesperado de la conexión
func (rq *RabbitQueue) handleConnectionClose() {
	rq.mu.RLock()
	if rq.connection == nil {
		rq.mu.RUnlock()
		return
	}
	closeChan := rq.connection.NotifyClose(make(chan *amqp.Error, 1))
	rq.mu.RUnlock()

	// Esperar a que se cierre la conexión
	closeErr := <-closeChan
	if closeErr != nil {
		slog.Warn("RabbitMQ connection closed unexpectedly", "error", closeErr)

		rq.mu.Lock()
		rq.connected = false
		rq.mu.Unlock()

		// Intentar reconectar automáticamente
		slog.Info("attempting to reconnect to RabbitMQ")
		if err := rq.connectWithRetry(); err != nil {
			slog.Error("failed to reconnect to RabbitMQ", "error", err)
		}
	}
}

// IsConnected verifica si hay una conexión activa
func (rq *RabbitQueue) IsConnected() bool {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	return rq.connected && rq.channel != nil
}

// ensureConnection asegura que haya una conexión activa, reconectando si es necesario
func (rq *RabbitQueue) ensureConnection() error {
	if rq.IsConnected() {
		return nil
	}

	slog.Info("RabbitMQ not connected, attempting to reconnect")
	return rq.connectWithRetry()
}

// Publish publica un evento de hotel en la cola hotels-news con reintentos
func (rq *RabbitQueue) Publish(hotelNew hotelsDomain.HotelNew) error {
	body, err := json.Marshal(hotelNew)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}
	return rq.publish(rq.queueName, body)
}

// PublishReservation publica un evento de reserva en la cola reservations-news
// (separada de hotels-news: search-api espera HotelNew ahí)
func (rq *RabbitQueue) PublishReservation(reservationNew hotelsDomain.ReservationNew) error {
	if rq.reservationsQueueName == "" {
		return fmt.Errorf("reservations queue name not configured")
	}
	body, err := json.Marshal(reservationNew)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}
	return rq.publish(rq.reservationsQueueName, body)
}

// publish publica un mensaje en la cola indicada con reintentos
func (rq *RabbitQueue) publish(queueName string, body []byte) error {
	// Asegurar conexión antes de publicar
	if err := rq.ensureConnection(); err != nil {
		return fmt.Errorf("RabbitMQ connection unavailable: %w", err)
	}

	// Intentar publicar con reintentos
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		rq.mu.RLock()
		channel := rq.channel
		rq.mu.RUnlock()

		if channel == nil {
			// Intentar reconectar
			if err := rq.ensureConnection(); err != nil {
				lastErr = err
				continue
			}
			rq.mu.RLock()
			channel = rq.channel
			rq.mu.RUnlock()
		}

		// Publicar el mensaje
		err := channel.Publish(
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         body,
			})

		if err == nil {
			return nil
		}

		lastErr = err
		slog.Warn("publish attempt failed", "attempt", attempt, "error", err)

		// Marcar como desconectado y reintentar
		rq.mu.Lock()
		rq.connected = false
		rq.mu.Unlock()

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}

	return fmt.Errorf("error publishing message after retries: %w", lastErr)
}

// closeUnsafe cierra las conexiones sin bloqueo (debe llamarse con mu bloqueado)
func (rq *RabbitQueue) closeUnsafe() {
	if rq.channel != nil {
		_ = rq.channel.Close()
		rq.channel = nil
	}
	if rq.connection != nil {
		_ = rq.connection.Close()
		rq.connection = nil
	}
	rq.connected = false
}

// Close cierra las conexiones de forma segura
func (rq *RabbitQueue) Close() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	rq.closeUnsafe()
	slog.Info("RabbitMQ connection closed")
}
