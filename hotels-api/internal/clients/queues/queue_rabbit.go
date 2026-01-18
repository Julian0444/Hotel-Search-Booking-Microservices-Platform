package queues

import (
	"encoding/json"
	"fmt"
	"log"
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
}

type RabbitQueue struct {
	config     RabbitConfig
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
	mu         sync.RWMutex
	connected  bool
}

// NewRabbit crea una nueva instancia de RabbitQueue con reconexión automática
func NewRabbit(config RabbitConfig) *RabbitQueue {
	rq := &RabbitQueue{
		config:    config,
		queueName: config.QueueName,
		connected: false,
	}

	// Intentar conexión inicial con reintentos
	if err := rq.connectWithRetry(); err != nil {
		log.Printf("Warning: Initial RabbitMQ connection failed after %d retries: %v", maxRetries, err)
		log.Printf("RabbitMQ will attempt to reconnect on next publish")
	}

	return rq
}

// connectWithRetry intenta conectar a RabbitMQ con backoff exponencial
func (rq *RabbitQueue) connectWithRetry() error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("RabbitMQ connection attempt %d/%d...", attempt, maxRetries)

		if err := rq.connect(); err != nil {
			lastErr = err
			log.Printf("Connection attempt %d failed: %v", attempt, err)

			if attempt < maxRetries {
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)

				// Incrementar backoff exponencialmente
				backoff = time.Duration(float64(backoff) * backoffFactor)
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		} else {
			log.Printf("Successfully connected to RabbitMQ on attempt %d", attempt)
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
		conn.Close()
		rq.connected = false
		return fmt.Errorf("error creating channel: %w", err)
	}

	// Declarar la cola
	_, err = ch.QueueDeclare(
		rq.config.QueueName, // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		rq.connected = false
		return fmt.Errorf("error declaring queue: %w", err)
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
		log.Printf("RabbitMQ connection closed unexpectedly: %v", closeErr)

		rq.mu.Lock()
		rq.connected = false
		rq.mu.Unlock()

		// Intentar reconectar automáticamente
		log.Printf("Attempting to reconnect to RabbitMQ...")
		if err := rq.connectWithRetry(); err != nil {
			log.Printf("Failed to reconnect to RabbitMQ: %v", err)
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

	log.Printf("RabbitMQ not connected, attempting to reconnect...")
	return rq.connectWithRetry()
}

// Publish publica un mensaje en la cola con reintentos
func (rq *RabbitQueue) Publish(hotelNew hotelsDomain.HotelNew) error {
	// Asegurar conexión antes de publicar
	if err := rq.ensureConnection(); err != nil {
		return fmt.Errorf("RabbitMQ connection unavailable: %w", err)
	}

	// Convertir el mensaje a JSON
	body, err := json.Marshal(hotelNew)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
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
		err = channel.Publish(
			"",           // exchange
			rq.queueName, // routing key
			false,        // mandatory
			false,        // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         body,
			})

		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Publish attempt %d failed: %v", attempt, err)

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
		rq.channel.Close()
		rq.channel = nil
	}
	if rq.connection != nil {
		rq.connection.Close()
		rq.connection = nil
	}
	rq.connected = false
}

// Close cierra las conexiones de forma segura
func (rq *RabbitQueue) Close() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	rq.closeUnsafe()
	log.Printf("RabbitMQ connection closed")
}
