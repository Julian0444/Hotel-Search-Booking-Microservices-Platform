package middlewares_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/middlewares"

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
		"aud":     []string{"users-api", "hotels-api"},
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

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	m := middlewares.NewJWTMiddleware(testSecret)
	auth := router.Group("/", m.Authenticate())
	auth.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"userID":   c.GetString("userID"),
			"userType": c.GetString("userType"),
		})
	})
	auth.GET("/admin", middlewares.AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	auth.GET("/users/:id", middlewares.OwnerOrAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })

	return router
}

func doGet(router *gin.Engine, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestAuthenticate_Negative(t *testing.T) {
	router := setupRouter()

	t.Run("missing header -> 401", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", "").Code)
	})

	t.Run("malformed header -> 401", func(t *testing.T) {
		for _, header := range []string{"Bearer", "Token abc", "Bearer not.a.jwt", "garbage"} {
			assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", header).Code, "header: %q", header)
		}
	})

	t.Run("expired token -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["exp"] = time.Now().UTC().Add(-2 * time.Hour).Unix() // vencido más allá del leeway
		})
		assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", "Bearer "+token).Code)
	})

	t.Run("wrong signing key -> 401", func(t *testing.T) {
		token := mintToken(t, "another-secret", jwt.SigningMethodHS256, nil)
		assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", "Bearer "+token).Code)
	})

	t.Run("alg none -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodNone, nil)
		assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", "Bearer "+token).Code)
	})

	t.Run("wrong issuer -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["iss"] = "evil-issuer"
		})
		assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", "Bearer "+token).Code)
	})

	t.Run("wrong audience -> 401", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["aud"] = []string{"hotels-api"} // no incluye la audiencia de este servicio
		})
		assert.Equal(t, http.StatusUnauthorized, doGet(router, "/me", "Bearer "+token).Code)
	})
}

func TestAuthenticate_Valid(t *testing.T) {
	router := setupRouter()

	token := mintToken(t, testSecret, jwt.SigningMethodHS256, nil)
	rr := doGet(router, "/me", "Bearer "+token)

	assert.Equal(t, http.StatusOK, rr.Code)

	var got map[string]string
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	assert.Equal(t, "7", got["userID"])
	assert.Equal(t, "cliente", got["userType"])
}

func TestAdminOnly(t *testing.T) {
	router := setupRouter()

	t.Run("cliente -> 403", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, nil)
		assert.Equal(t, http.StatusForbidden, doGet(router, "/admin", "Bearer "+token).Code)
	})

	t.Run("administrador -> 200", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["tipo"] = "administrador"
		})
		assert.Equal(t, http.StatusOK, doGet(router, "/admin", "Bearer "+token).Code)
	})
}

func TestOwnerOrAdmin(t *testing.T) {
	router := setupRouter()

	t.Run("owner -> 200", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, nil) // user_id "7"
		assert.Equal(t, http.StatusOK, doGet(router, "/users/7", "Bearer "+token).Code)
	})

	t.Run("numeric user_id claim matches -> 200", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["user_id"] = 7 // número JSON -> float64 -> "7" en el contexto
		})
		assert.Equal(t, http.StatusOK, doGet(router, "/users/7", "Bearer "+token).Code)
	})

	t.Run("other user -> 403", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, nil)
		assert.Equal(t, http.StatusForbidden, doGet(router, "/users/8", "Bearer "+token).Code)
	})

	t.Run("admin accessing other user -> 200", func(t *testing.T) {
		token := mintToken(t, testSecret, jwt.SigningMethodHS256, func(c jwt.MapClaims) {
			c["tipo"] = "administrador"
		})
		assert.Equal(t, http.StatusOK, doGet(router, "/users/8", "Bearer "+token).Code)
	})
}
