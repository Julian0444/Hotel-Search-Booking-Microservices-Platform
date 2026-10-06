package tokenizers

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Key      string
	Duration time.Duration
}

type JWT struct {
	config JWTConfig
}

func NewTokenizer(config JWTConfig) JWT {
	return JWT{config: config}
}

// GenerateToken genera un JWT compatible con `hotels-api` (claims `user_id` y `tipo`).
// Usa claims estándar `iat`/`exp`/`nbf`, y scopea el token con `iss`/`aud`:
// lo emite users-api y lo aceptan users-api, hotels-api y search-api (cada
// uno valida su propia audiencia; search-api la usa para POST /reindex).
func (tokenizer JWT) GenerateToken(username string, userID int64, tipo string) (string, error) {
	if tokenizer.config.Key == "" {
		return "", fmt.Errorf("jwt key is required")
	}
	if tokenizer.config.Duration <= 0 {
		return "", fmt.Errorf("jwt duration must be positive")
	}
	if tipo == "" {
		tipo = "cliente"
	}

	now := time.Now().UTC()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"user_id":  userID,
		"tipo":     tipo,
		"iss":      "users-api",
		"aud":      []string{"users-api", "hotels-api", "search-api"},
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"exp":      now.Add(tokenizer.config.Duration).Unix(),
	})

	value, err := token.SignedString([]byte(tokenizer.config.Key))
	if err != nil {
		return "", fmt.Errorf("error generating JWT token: %w", err)
	}

	return value, nil
}
