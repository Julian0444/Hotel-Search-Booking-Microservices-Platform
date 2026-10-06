//go:build integration

package queues

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

// TEST_RABBIT_URL apunta a un broker real de pruebas. Sólo crea y elimina sus
// propias colas con nombre único; nunca purga hotels-news ni datos de la demo.
func TestRabbitConfirmedPublishAndReconnect(t *testing.T) {
	uri := os.Getenv("TEST_RABBIT_URL")
	if uri == "" {
		t.Skip("set TEST_RABBIT_URL for live publisher acceptance/reconnect test")
	}
	parsed, err := url.Parse(uri)
	require.NoError(t, err)
	password, _ := parsed.User.Password()
	name := fmt.Sprintf("publisher-verify-%d", time.Now().UnixNano())
	cfg := RabbitConfig{Host: parsed.Hostname(), Port: parsed.Port(), Username: parsed.User.Username(), Password: password, QueueName: name}
	if cfg.Port == "" {
		cfg.Port = "5672"
	}
	observer, err := amqp.Dial(uri)
	require.NoError(t, err)
	defer func() { _ = observer.Close() }()
	ch, err := observer.Channel()
	require.NoError(t, err)
	defer func() {
		_, _ = ch.QueueDelete(name, false, false, false)
		_, _ = ch.QueueDelete(name+"-dlq", false, false, false)
		_ = ch.ExchangeDelete(name+"-dlx", false, false)
	}()
	publisher := NewRabbit(cfg)
	defer publisher.Close()
	require.Eventually(t, publisher.IsConnected, 10*time.Second, 20*time.Millisecond)
	event := hotelsDomain.HotelNew{Operation: "CREATE", HotelID: "confirmed"}
	require.NoError(t, publisher.Publish(context.Background(), event))
	delivery, ok, err := ch.Get(name, true)
	require.NoError(t, err)
	require.True(t, ok, "confirmed publication must be accepted by broker")
	require.Equal(t, uint8(amqp.Persistent), delivery.DeliveryMode)
	require.JSONEq(t, `{"operation":"CREATE","hotel_id":"confirmed"}`, string(delivery.Body))

	publisher.mu.RLock()
	old := publisher.session
	publisher.mu.RUnlock()
	require.NoError(t, old.socket.Close()) // pérdida real de conexión TCP
	require.Eventually(t, func() bool {
		publisher.mu.RLock()
		defer publisher.mu.RUnlock()
		return publisher.session != nil && publisher.session != old
	}, 10*time.Second, 20*time.Millisecond)
	require.NoError(t, publisher.Publish(context.Background(), hotelsDomain.HotelNew{Operation: "UPDATE", HotelID: "recovered"}))
	delivery, ok, err = ch.Get(name, true)
	require.NoError(t, err)
	require.True(t, ok)
	require.Contains(t, string(delivery.Body), "recovered")
}
