// hotels-api/middleware/auth.go
package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verifica el método de firma
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.SecretKey), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// Obtiene el tipo de usuario desde los claims
		userType, ok := claims["tipo"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User type not found in token"})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
			return
		}

		if userType != "administrador" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: Administrators only"})
			return
		}

		c.Next()
	}
}

// Valida que el usuario esté autenticado (cualquier tipo de usuario)
func LoggedUserOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Solo verificamos que el usuario tenga un token válido
		_, exists := c.Get("userType")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		c.Next()
	}
}
