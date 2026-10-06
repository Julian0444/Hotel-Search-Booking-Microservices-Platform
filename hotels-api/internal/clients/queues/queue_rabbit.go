package queues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sync"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	amqp "github.com/rabbitmq/amqp091-go"
)

const publishTimeout = 2 * time.Second

type RabbitConfig struct {
	Host, Port, Username, Password, QueueName string
}

type rabbitSession struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	socket     net.Conn
	returns    <-chan amqp.Return
}

// RabbitQueue tiene un único reconector de fondo. Un request nunca conecta ni
// duerme un backoff: publica con confirmación en <=2s o informa indisponibilidad.
// Mongo y RabbitMQ no son atómicos; el caller conserva el éxito de Mongo y la
// reconciliación periódica recupera el índice cuando esta publicación falla.
type RabbitQueue struct {
	config      RabbitConfig
	mu          sync.RWMutex
	session     *rabbitSession
	publishGate chan struct{}
	cancel      context.CancelFunc
	done        chan struct{}
}

func NewRabbit(config RabbitConfig) *RabbitQueue {
	ctx, cancel := context.WithCancel(context.Background())
	rq := &RabbitQueue{config: config, publishGate: make(chan struct{}, 1), cancel: cancel, done: make(chan struct{})}
	go rq.reconnect(ctx)
	return rq
}

func (rq *RabbitQueue) reconnect(ctx context.Context) {
	defer close(rq.done)
	backoff := time.Second
	for ctx.Err() == nil {
		session, err := rq.connect(ctx)
		if err == nil {
			connClosed := session.connection.NotifyClose(make(chan *amqp.Error, 1))
			channelClosed := session.channel.NotifyClose(make(chan *amqp.Error, 1))
			rq.mu.Lock()
			rq.session = session
			rq.mu.Unlock()
			backoff = time.Second
			slog.Info("RabbitMQ publisher connected")
			select {
			case <-ctx.Done():
			case <-connClosed:
			case <-channelClosed:
			}
			rq.mu.Lock()
			rq.session = nil
			rq.mu.Unlock()
			_ = session.socket.Close()
		} else if ctx.Err() == nil {
			slog.Warn("RabbitMQ publisher unavailable; background retry", "error", err)
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
}

func (rq *RabbitQueue) connect(ctx context.Context) (*rabbitSession, error) {
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	session := &rabbitSession{}
	uri := url.URL{Scheme: "amqp", Host: net.JoinHostPort(rq.config.Host, rq.config.Port), User: url.UserPassword(rq.config.Username, rq.config.Password), Path: "/"}
	conn, err := amqp.DialConfig(uri.String(), amqp.Config{
		Heartbeat: 5 * time.Second,
		Dial: func(network, addr string) (net.Conn, error) {
			socket, err := (&net.Dialer{}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			session.socket = socket
			deadline, _ := ctx.Deadline()
			if err := socket.SetDeadline(deadline); err != nil {
				_ = socket.Close()
				return nil, err
			}
			return socket, nil
		},
	})
	if err != nil {
		if session.socket != nil {
			_ = session.socket.Close()
		}
		return nil, fmt.Errorf("connect broker: %w", err)
	}
	session.connection = conn
	// AMQP clears handshake deadlines; bound channel/topology RPCs too.
	stop := context.AfterFunc(ctx, func() { _ = session.socket.Close() })
	defer stop()
	ch, err := conn.Channel()
	if err != nil {
		_ = session.socket.Close()
		return nil, err
	}
	session.channel = ch
	session.returns = ch.NotifyReturn(make(chan amqp.Return, 1))
	if err = declareHotelQueue(ch, rq.config.QueueName); err == nil {
		err = ch.Confirm(false)
	}
	if err != nil {
		_ = session.socket.Close()
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		_ = session.socket.Close()
		return nil, err
	}
	return session, nil
}

// Espejo de la topología del consumer. Durable + Persistent + confirms y el
// volumen estable del broker protegen aceptación; no cierran la ventana Mongo.
func declareHotelQueue(ch *amqp.Channel, name string) error {
	dlx, dlq := name+"-dlx", name+"-dlq"
	if err := ch.ExchangeDeclare(dlx, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(dlq, name, dlx, false, nil); err != nil {
		return err
	}
	_, err := ch.QueueDeclare(name, true, false, false, false, amqp.Table{"x-dead-letter-exchange": dlx})
	return err
}

func (rq *RabbitQueue) IsConnected() bool {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	return rq.session != nil && !rq.session.connection.IsClosed() && !rq.session.channel.IsClosed()
}

func (rq *RabbitQueue) Publish(ctx context.Context, event hotelsDomain.HotelNew) error {
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	// El presupuesto incluye la espera por otro publicador concurrente.
	select {
	case rq.publishGate <- struct{}{}:
		defer func() { <-rq.publishGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	rq.mu.RLock()
	session := rq.session
	rq.mu.RUnlock()
	if session == nil {
		return errors.New("RabbitMQ publisher unavailable")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	// amqp091-go no cancela el I/O iniciado por PublishWith...Context. Cerrar el
	// socket al vencer el contexto limita también escrituras bloqueadas, no sólo
	// la espera del confirm. El reconector lo reemplaza en segundo plano.
	stop := context.AfterFunc(ctx, func() { _ = session.socket.Close() })
	defer stop()
	confirmation, err := session.channel.PublishWithDeferredConfirmWithContext(ctx, "", rq.config.QueueName, true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: body,
	})
	if err != nil {
		_ = session.socket.Close()
		return err
	}
	ack, err := confirmation.WaitContext(ctx)
	if err != nil {
		_ = session.socket.Close()
		return fmt.Errorf("broker acceptance unknown: %w", err)
	}
	select {
	case returned := <-session.returns:
		return fmt.Errorf("broker could not route publication: %s", returned.ReplyText)
	default:
	}
	if !ack {
		return errors.New("broker rejected publication")
	}
	return nil
}

func (rq *RabbitQueue) Close() {
	if rq.cancel == nil {
		return
	}
	rq.cancel()
	<-rq.done
}
