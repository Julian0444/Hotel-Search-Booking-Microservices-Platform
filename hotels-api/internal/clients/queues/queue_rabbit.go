package queues

import (
	"encoding/json"
	"fmt"
	"log"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/streadway/amqp"
)

type RabbitConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	QueueName string
}

type RabbitQueue struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
}

func NewRabbit(config RabbitConfig) RabbitQueue {
	// Crear la URL de conexión
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", config.Username, config.Password, config.Host, config.Port)

	// Conectar a RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("Error connecting to RabbitMQ: %v", err)
		// Retornar una implementación mock en caso de error
		return RabbitQueue{}
	}

	// Crear canal
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Error creating channel: %v", err)
		return RabbitQueue{}
	}

	// Declarar la cola
	_, err = ch.QueueDeclare(
		config.QueueName, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Printf("Error declaring queue: %v", err)
		return RabbitQueue{}
	}

	return RabbitQueue{
		connection: conn,
		channel:    ch,
		queueName:  config.QueueName,
	}
}

func (rq RabbitQueue) Publish(hotelNew hotelsDomain.HotelNew) error {
	if rq.channel == nil {
		// Si no hay canal, retornar error
		return fmt.Errorf("RabbitMQ channel not available")
	}

	// Convertir el mensaje a JSON (implementación básica)
	body, err := json.Marshal(hotelNew)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	// Publicar el mensaje
	err = rq.channel.Publish(
		"",           // exchange
		rq.queueName, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})

	if err != nil {
		return fmt.Errorf("error publishing message: %w", err)
	}

	return nil
}

func (rq RabbitQueue) Close() {
	if rq.channel != nil {
		rq.channel.Close()
	}
	if rq.connection != nil {
		rq.connection.Close()
	}
}
