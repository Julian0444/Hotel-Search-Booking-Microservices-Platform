package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"github.com/gin-gonic/gin"
)

// Persistence, conflicts and replay are exercised against the real replica
// set by repository integration tests. Middleware only validates transport.
func TestIdempotencyMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		key    string
		status int
	}{{"", 201}, {"key-1", 201}, {strings.Repeat("k", 201), 400}, {" key ", 400}} {
		t.Run(tc.key, func(t *testing.T) {
			called := false
			r := gin.New()
			r.POST("/", Idempotency(), func(c *gin.Context) {
				called = true
				operation := hotelsDomain.IdempotencyFromContext(c.Request.Context())
				if tc.key == "" {
					if operation != nil {
						t.Fatal("no header should not create an operation")
					}
				} else if operation == nil || operation.Key != tc.key {
					t.Fatal("missing key metadata")
				}
				c.Status(http.StatusCreated)
			})
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Idempotency-Key", tc.key)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != tc.status || called != (tc.status == 201) {
				t.Fatalf("status=%d called=%v", response.Code, called)
			}
		})
	}
}
