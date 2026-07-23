// search-api/middleware/auth.go — copia del middleware JWT de hotels-api con
// la audiencia propia de este servicio (protege POST /reindex, plan 06).
package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/apperr"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// tokenIssuer es quien emite los tokens de la plataforma (users-api).
	tokenIssuer = "users-api"
	// tokenAudience es la audiencia que este servicio exige en los tokens que acepta.
	tokenAudience = "search-api"
)

type JWTMiddleware struct {
	SecretKey string
}

func NewJWTMiddleware(secretKey string) JWTMiddleware {
	return JWTMiddleware{SecretKey: secretKey}
}

func (m JWTMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "authorization header missing", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "authorization header format must be Bearer {token}", nil)
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verifica el método de firma
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.SecretKey), nil
		},
			jwt.WithIssuer(tokenIssuer),
			jwt.WithAudience(tokenAudience),
			jwt.WithLeeway(30*time.Second),
		)

		if err != nil || !token.Valid {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "invalid token", err)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "invalid token claims", nil)
			return
		}

		// Obtiene el tipo de usuario desde los claims
		userType, ok := claims["tipo"].(string)
		if !ok {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "user type not found in token", nil)
			return
		}

		// Obtiene el ID de usuario desde los claims
		var userID string
		if userIDFloat, ok := claims["user_id"].(float64); ok {
			userID = strconv.FormatInt(int64(userIDFloat), 10)
		} else if userIDInt, ok := claims["user_id"].(int64); ok {
			userID = strconv.FormatInt(userIDInt, 10)
		} else if userIDString, ok := claims["user_id"].(string); ok {
			userID = userIDString
		} else {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "user ID not found in token", nil)
			return
		}

		// Almacena el tipo de usuario y user_id en el contexto para usarlo posteriormente
		c.Set("userType", userType)
		c.Set("userID", userID)

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("userType")
		if !exists {
			apperr.Abort(c, http.StatusUnauthorized, "unauthorized", "user type not found", nil)
			return
		}

		if userType != "administrador" {
			apperr.Abort(c, http.StatusForbidden, "forbidden", "administrators only", nil)
			return
		}

		c.Next()
	}
}
