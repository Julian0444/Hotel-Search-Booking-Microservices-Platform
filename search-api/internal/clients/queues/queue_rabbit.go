package queues

import (
	"encoding/json"
	"fmt"
	"log"

	"search-api/internal/domain/hotels"

	"github.com/streadway/amqp"
)

type RabbitConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	QueueName string
}

type Rabbit struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      amqp.Queue
}

// Funcion para crear una nueva conexion a RabbitMQ
func NewRabbit(config RabbitConfig) Rabbit {
	//Dial crea una nueva conexion a RabbitMQ
	connection, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", config.Username, config.Password, config.Host, config.Port))
	if err != nil {
		log.Fatalf("error getting Rabbit connection: %w", err)
	}
	// Channel crea un nuevo canal de comunic
	channel, err := connection.Channel()
	if err != nil {
		log.Fatalf("error creating Rabbit channel: %w", err)
	}
	// QueueDeclare crea una nueva cola en RabbitMQ
	queue, err := channel.QueueDeclare(config.QueueName, true, false, false, false, nil)
	return Rabbit{
		connection: connection,
		channel:    channel,
		queue:      queue,
	}
}

// Inicia el consumidor de la cola de RabbitMQ (El que carga los mensaje ya esta definido en la api de hoteles)
func (queue Rabbit) StartConsumer(handler func(hotels.HotelNew)) error {
	messages, err := queue.channel.Consume(
		queue.queue.Name,
		"",
		true, // Auto-acknowledge messages
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error registering consumer: %w", err)
	}

	//Una goroutine es una funcion que se ejecuta en paralelo con el resto del programa
	go func() {
		//Hace un for para recorrer los mensajes que llegan a la cola
		for msg := range messages {
			var hotelUpdate hotels.HotelNew
			//Unmarshal convierte el json en un objeto de tipo HotelNew
			if err := json.Unmarshal(msg.Body, &hotelUpdate); err != nil {
				log.Printf("error unmarshaling message: %v", err)
				continue
			}
			handler(hotelUpdate)
		}
	}()

	return nil
}

// Cierra la conexion a RabbitMQ
func (queue Rabbit) Close() {
	// Close cierra el canal de comunicacion
	if err := queue.channel.Close(); err != nil {
		log.Printf("error closing Rabbit channel: %v", err)
	}
	// Close cierra la conexion a RabbitMQ
	if err := queue.connection.Close(); err != nil {
		log.Printf("error closing Rabbit connection: %v", err)
	}
}
