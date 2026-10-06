//go:build integration

package hotels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/hotels"
	domain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/middlewares"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/services/hotels"
	"github.com/gin-gonic/gin"
)

type failedHotelPublisher struct{ calls int }

func (p *failedHotelPublisher) Publish(context.Context, domain.HotelNew) error {
	p.calls++
	return errors.New("broker unavailable")
}

// Controllers and service run against a real replica set. Publishing failures
// are injected at the broker boundary; root queue integration tests exercise
// the actual broker separately.
func TestMongo_HTTPPersistenceAvailabilityAndReplay(t *testing.T) {
	cfg := startMongoContainer(t)
	repo := NewMongo(cfg)
	defer repo.Disconnect(context.Background())
	queue := &failedHotelPublisher{}
	newRouter := func(repository Mongo) *gin.Engine {
		controller := controllers.NewController(services.NewService(repository, queue))
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) { c.Set("userID", "client-1"); c.Set("userType", "cliente"); c.Next() })
		router.POST("/hotels", controller.Create)
		router.PUT("/hotels/:hotel_id", controller.Update)
		router.DELETE("/hotels/:hotel_id", controller.Delete)
		router.GET("/hotels/:hotel_id", controller.GetHotelByID)
		router.POST("/availability", controller.GetAvailability)
		router.POST("/reservations", middlewares.Idempotency(), controller.CreateReservation)
		router.GET("/reservations/:id", controller.GetReservationByID)
		router.DELETE("/reservations/:id", controller.CancelReservation)
		return router
	}
	router := newRouter(repo)
	request := func(method, url, body, key string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("%s %s: status=%d want=%d body=%s", method, url, response.Code, status, response.Body.String())
		}
		return response
	}
	hotelPayload := `{"id":"client-supplied-invalid-id","name":"Hotel HTTP","city":"Córdoba","country":"Argentina","address":"Main 1","available_rooms":1,"price_per_night":12.34,"rating":4,"check_in_time":"15:00","check_out_time":"11:00","amenities":["Wifi"],"images":["https://example.com/room.jpg"]}`
	readID := func(response *httptest.ResponseRecorder) string {
		t.Helper()
		var decoded struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Data.ID == "" {
			t.Fatal("empty ID")
		}
		return decoded.Data.ID
	}
	id := readID(request(http.MethodPost, "/hotels", hotelPayload, "", 201))
	if queue.calls != 1 {
		t.Fatal("create did not try publish")
	}
	stored, err := repo.GetHotelByID(context.Background(), id)
	if err != nil || stored.Name != "Hotel HTTP" {
		t.Fatalf("create response inconsistent with Mongo: %+v %v", stored, err)
	}
	// Detail-first is the actual guest order; it must not populate any alternate
	// source of availability. Repeat detail reads to cover formerly hot cache.
	for range 2 {
		request(http.MethodGet, "/hotels/"+id, "", "", 200)
	}
	availabilityBody := fmt.Sprintf(`{"hotel_ids":[%q],"check_in":"2030-08-01","check_out":"2030-08-03"}`, id)
	availability := request(http.MethodPost, "/availability", availabilityBody, "", 200)
	if !strings.Contains(availability.Body.String(), `"`+id+`":true`) {
		t.Fatal(availability.Body.String())
	}
	reservationBody := fmt.Sprintf(`{"hotel_id":%q,"check_in":"2030-08-01","check_out":"2030-08-03","num_rooms":1,"num_guests":2}`, id)
	first := request(http.MethodPost, "/reservations", reservationBody, "attempt-1", 201)
	reservationID := readID(first)
	if queue.calls != 1 {
		t.Fatal("booking called the broken publisher")
	}
	request(http.MethodGet, "/hotels/"+id, "", "", 200)
	availability = request(http.MethodPost, "/availability", availabilityBody, "", 200)
	if !strings.Contains(availability.Body.String(), `"`+id+`":false`) {
		t.Fatal("full hotel announced available: " + availability.Body.String())
	}
	freshRepo := NewMongo(cfg)
	defer freshRepo.Disconnect(context.Background())
	router = newRouter(freshRepo)
	replay := request(http.MethodPost, "/reservations", reservationBody, "attempt-1", 201)
	if replay.Body.String() != first.Body.String() || replay.Header().Get("Idempotency-Replayed") != "true" || replay.Header().Get("Location") != first.Header().Get("Location") {
		t.Fatalf("restart replay differs: %s", replay.Body.String())
	}
	request(http.MethodPost, "/reservations", strings.Replace(reservationBody, `"num_guests":2`, `"num_guests":3`, 1), "attempt-1", 409)
	request(http.MethodPost, "/reservations", reservationBody, "another-attempt", 409)
	request(http.MethodPut, "/hotels/"+id, strings.Replace(hotelPayload, `"available_rooms":1`, `"available_rooms":0`, 1), "", 409)
	request(http.MethodDelete, "/hotels/"+id, "", "", 409)
	request(http.MethodDelete, "/reservations/"+reservationID, "", "", 204)
	request(http.MethodDelete, "/reservations/"+reservationID, "", "", 204)
	if queue.calls != 1 {
		t.Fatal("cancellation called publisher")
	}
	replay = request(http.MethodPost, "/reservations", reservationBody, "attempt-1", 201)
	if replay.Body.String() != first.Body.String() {
		t.Fatal("cancelled result changed original booking response")
	}
	availability = request(http.MethodPost, "/availability", availabilityBody, "", 200)
	if !strings.Contains(availability.Body.String(), `"`+id+`":true`) {
		t.Fatal("cancellation did not free availability")
	}
	// Full PUT persists meaningful zeros and cleared optionals while publishing
	// fails. The next normal GET returns Mongo's actual replacement values.
	replacement := `{"name":"Zero Hotel","city":"Córdoba","country":"Argentina","address":"Main 1","available_rooms":0,"price_per_night":0,"rating":0,"check_in_time":"15:00","check_out_time":"11:00","description":"","amenities":[],"images":[]}`
	request(http.MethodPut, "/hotels/"+id, replacement, "", 200)
	reread := request(http.MethodGet, "/hotels/"+id, "", "", 200)
	for _, field := range []string{`"available_rooms":0`, `"price_per_night":0`, `"rating":0`, `"amenities":[]`, `"images":[]`} {
		if !strings.Contains(reread.Body.String(), field) {
			t.Fatalf("missing %s: %s", field, reread.Body.String())
		}
	}
	// A separate unreserved hotel can be deleted even while publish fails.
	deletable := readID(request(http.MethodPost, "/hotels", hotelPayload, "", 201))
	request(http.MethodDelete, "/hotels/"+deletable, "", "", 204)
	if _, err := repo.GetHotelByID(context.Background(), deletable); !errors.Is(err, domain.ErrHotelNotFound) {
		t.Fatalf("delete response inconsistent with Mongo: %v", err)
	}
	before, _ := repo.CountHotels(context.Background())
	calls := queue.calls
	for _, bad := range []string{`{}`, strings.Replace(hotelPayload, `"Hotel HTTP"`, `"  "`, 1), strings.Replace(hotelPayload, `"price_per_night":12.34`, `"price_per_night":-1`, 1), strings.Replace(hotelPayload, `"rating":4`, `"rating":6`, 1), strings.Replace(hotelPayload, `"available_rooms":1`, `"available_rooms":-1`, 1)} {
		request(http.MethodPost, "/hotels", bad, "", 400)
		request(http.MethodPut, "/hotels/"+id, bad, "", 400)
	}
	after, _ := repo.CountHotels(context.Background())
	if before != after || queue.calls != calls {
		t.Fatal("invalid hotel wrote or published")
	}
	current, _ := repo.GetHotelByID(context.Background(), id)
	if current.Name != "Zero Hotel" {
		t.Fatal("invalid PUT mutated Mongo")
	}
	if err := repo.AuditInventory(context.Background()); err != nil {
		t.Fatal(err)
	}
}
