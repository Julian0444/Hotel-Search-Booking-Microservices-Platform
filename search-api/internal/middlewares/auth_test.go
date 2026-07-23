package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret"

// mintToken firma un token con claims válidos por defecto; mutate permite romperlos por caso.
func mintToken(t *testing.T, secret string, method jwt.SigningMethod, mutate func(jwt.MapClaims)) string {
	t.Helper()

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"tipo":    "cliente",
		"user_id": "7",
		"iss":     "users-api",
		"aud":     []string{"users-api", "hotels-api", "search-api"},
		"iat":     now.Unix(),
		"nbf":     now.Unix(),
		"exp":     now.Add(time.Hour).Unix(),
	}
	if mutate != nil {
		mutate(claims)
	}

	token := jwt.NewWithClaims(method, claims)
	var key any = []byte(secret)
	if method == jwt.SigningMethodNone {
		key = jwt.UnsafeAllowNoneSignatureType
	}
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("error signing token: %v", err)
	}
	return signed
}

func setupAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	m := middleware.NewJWTMiddleware(testSecret)
	// Como POST /reindex en cmd/main.go: autenticado + solo admins
	router.POST("/reindex", m.Authenticate(), middleware.AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })

	return router
}

func doPost(router *gin.Engine, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestAuthenticate_Negative(t *testing.T) {
	router := setupAuthRouter()

	t.Run("missing header -> 401", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", "").Code)
	})

	t.Run("malformed header -> 401", func(t *testing.T) {
		for _, header := range []string{"Bearer", "Token abc", "Bearer not.a.jwt", "garbage"} {
			assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", header).Code, "header: %q", header)
		}
	})

	t.Run("expired token -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["exp"] = time.Now().UTC().Add(-2 * time.Hour).Unix() // vencido más allá del leeway
		})
		assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", "Bearer "+token).Code)
	})

	t.Run("wrong signing key -> 401", func(t *testing.T) {
		token := mintToken(t, "another-secret", jwt.SigningMethodHS256, nil)
		assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", "Bearer "+token).Code)
	})

	t.Run("alg none -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodNone, nil)
		assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", "Bearer "+token).Code)
	})

	t.Run("wrong issuer -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["iss"] = "evil-issuer"
		})
		assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", "Bearer "+token).Code)
	})

	t.Run("wrong audience -> 401", func(t *testing.T) {
		// Un token pre-plan-06 (sin search-api en aud) no habilita /reindex
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["aud"] = []string{"users-api", "hotels-api"}
		})
		assert.Equal(t, http.StatusUnauthorized, doPost(router, "/reindex", "Bearer "+token).Code)
	})
}

func TestAuthenticate_ValidAndRoles(t *testing.T) {
	router := setupAuthRouter()

	t.Run("cliente on admin route -> 403", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, nil)
		assert.Equal(t, http.StatusForbidden, doPost(router, "/reindex", "Bearer "+token).Code)
	})

	t.Run("administrador on admin route -> 200", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["tipo"] = "administrador"
		})
		assert.Equal(t, http.StatusOK, doPost(router, "/reindex", "Bearer "+token).Code)
	})
}
