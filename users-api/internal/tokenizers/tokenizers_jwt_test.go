package tokenizers_test

import (
	"testing"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/tokenizers"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWT_GenerateToken_RoundTrip(t *testing.T) {
	tokenizer := tokenizers.NewTokenizer(tokenizers.JWTConfig{
		Key:      "test-key",
		Duration: time.Hour,
	})

	signed, err := tokenizer.GenerateToken("mallory", 42, "cliente")
	assert.NoError(t, err)
	assert.NotEmpty(t, signed)

	// El token debe validar con las mismas restricciones que aplican los middlewares.
	token, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-key"), nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer("users-api"),
		jwt.WithAudience("users-api"),
	)
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, "mallory", claims["username"])
	assert.Equal(t, float64(42), claims["user_id"]) // números JSON -> float64
	assert.Equal(t, "cliente", claims["tipo"])
	assert.Equal(t, "users-api", claims["iss"])

	aud, err := claims.GetAudience()
	assert.NoError(t, err)
	assert.Contains(t, aud, "users-api")
	assert.Contains(t, aud, "hotels-api")

	exp, err := claims.GetExpirationTime()
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), exp.Time, time.Minute)

	nbf, err := claims.GetNotBefore()
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now(), nbf.Time, time.Minute)
}

func TestJWT_GenerateToken_Validation(t *testing.T) {
	t.Run("empty tipo defaults to cliente", func(t *testing.T) {
		tokenizer := tokenizers.NewTokenizer(tokenizers.JWTConfig{Key: "test-key", Duration: time.Hour})

		signed, err := tokenizer.GenerateToken("user", 1, "")
		assert.NoError(t, err)

		token, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
			return []byte("test-key"), nil
		})
		assert.NoError(t, err)
		claims := token.Claims.(jwt.MapClaims)
		assert.Equal(t, "cliente", claims["tipo"])
	})

	t.Run("missing key -> error", func(t *testing.T) {
		tokenizer := tokenizers.NewTokenizer(tokenizers.JWTConfig{Key: "", Duration: time.Hour})

		_, err := tokenizer.GenerateToken("user", 1, "cliente")
		assert.Error(t, err)
	})

	t.Run("nonpositive duration -> error", func(t *testing.T) {
		tokenizer := tokenizers.NewTokenizer(tokenizers.JWTConfig{Key: "test-key", Duration: 0})

		_, err := tokenizer.GenerateToken("user", 1, "cliente")
		assert.Error(t, err)
	})
}
