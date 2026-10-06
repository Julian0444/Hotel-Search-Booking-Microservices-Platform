//go:build integration

package queues

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	domain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/integrationtest"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/repositories/hotels"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/services/search"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestRealRabbitRetainsTransientFailuresAndRecoversWithoutReindex(t *testing.T) {
	rabbitURL := os.Getenv("TEST_RABBIT_URL")
	if rabbitURL == "" {
		t.Skip("TEST_RABBIT_URL required")
	}
	broker, err := url.Parse(rabbitURL)
	require.NoError(t, err)
	for _, dependency := range []string{"hotels-api", "solr"} {
		t.Run(dependency, func(t *testing.T) {
			actualSolr, solrURL := integrationtest.NewSolr(t)
			var down atomic.Bool
			down.Store(true)
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if dependency == "hotels-api" && down.Load() {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": domain.Hotel{ID: "000000000000000000000001", Name: "Recovered hotel", City: "Córdoba"}})
			}))
			defer source.Close()
			target, _ := url.Parse(solrURL)
			proxy := httputil.NewSingleHostReverseProxy(target)
			solrProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if dependency == "solr" && down.Load() {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				proxy.ServeHTTP(w, r)
			}))
			defer solrProxy.Close()
			upstream, _ := url.Parse(source.URL)
			solrUpstream, _ := url.Parse(solrProxy.URL)
			svc := services.NewService(repositories.NewSolr(repositories.SolrConfig{Host: solrUpstream.Hostname(), Port: solrUpstream.Port(), Collection: actualSolr.Collection}),
				repositories.NewHTTP(repositories.HTTPConfig{Host: upstream.Hostname(), Port: upstream.Port()}))
			password, _ := broker.User.Password()
			name := "verify-" + uuid.NewString()
			queue := NewRabbit(RabbitConfig{Host: broker.Hostname(), Port: broker.Port(), Username: broker.User.Username(), Password: password, QueueName: name})
			queue.retryDelay = 10 * time.Millisecond
			queue.attemptTimeout = time.Second
			defer queue.Close()
			var attempts, successes atomic.Int32
			queue.StartConsumer(func(ctx context.Context, event domain.HotelNew) error {
				attempts.Add(1)
				err := svc.HandleHotelNew(ctx, event)
				if err == nil {
					successes.Add(1)
				}
				return err
			})
			require.Eventually(t, queue.IsConnected, 10*time.Second, 10*time.Millisecond)
			connection, err := amqp.Dial(rabbitURL)
			require.NoError(t, err)
			defer connection.Close()
			ch, err := connection.Channel()
			require.NoError(t, err)
			defer ch.Close()
			defer func() {
				_, _ = ch.QueueDelete(name, false, false, false)
				_, _ = ch.QueueDelete(name+"-dlq", false, false, false)
				_ = ch.ExchangeDelete(name+"-dlx", false, false)
			}()
			require.NoError(t, ch.Confirm(false))
			publish := func(body string) {
				confirmation, err := ch.PublishWithDeferredConfirmWithContext(context.Background(), "", name, true, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", Body: []byte(body)})
				require.NoError(t, err)
				require.True(t, confirmation.Wait())
			}
			start := time.Now()
			publish(`{"operation":"CREATE","hotel_id":"000000000000000000000001"}`)
			require.Eventually(t, func() bool { return attempts.Load() >= 4 }, 5*time.Second, 5*time.Millisecond)
			require.GreaterOrEqual(t, time.Since(start), 70*time.Millisecond, "consumer must wait between retries")
			dlq, err := ch.QueueInspect(name + "-dlq")
			require.NoError(t, err)
			require.Zero(t, dlq.Messages, "valid events must not become dead letters")
			require.Zero(t, successes.Load())
			down.Store(false)
			require.Eventually(t, func() bool { return successes.Load() == 1 }, 5*time.Second, 10*time.Millisecond)
			ids, err := actualSolr.ListIDs(context.Background())
			require.NoError(t, err)
			require.Equal(t, []string{"000000000000000000000001"}, ids)
			// Un fallo AMQP real obliga a registrar de nuevo el consumer. El evento
			// siguiente continúa sin reiniciar search-api ni llamar /reindex.
			queue.mu.RLock()
			consumerConnection := queue.connection
			queue.mu.RUnlock()
			require.NoError(t, consumerConnection.Close())
			publish(`{"operation":"UPDATE","hotel_id":"000000000000000000000001"}`)
			require.Eventually(t, func() bool { return successes.Load() == 2 }, 5*time.Second, 10*time.Millisecond)
			publish(`{"operation":"INVALID","hotel_id":"000000000000000000000001"}`)
			require.Eventually(t, func() bool { q, err := ch.QueueInspect(name + "-dlq"); return err == nil && q.Messages == 1 }, 5*time.Second, 10*time.Millisecond)
			require.Equal(t, int32(2), successes.Load())
		})
	}
}
