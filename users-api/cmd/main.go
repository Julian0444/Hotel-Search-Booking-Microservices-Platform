package main

import (
	"log"
	"time"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/config"
	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/controllers/users"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/tokenizers"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// MySQL
	mySQLRepo := repositories.NewMySQL(
		repositories.MySQLConfig{
			Host:     config.MySQLHost,
			Port:     config.MySQLPort,
			Database: config.MySQLDatabase,
			Username: config.MySQLUsername,
			Password: config.MySQLPassword,
		},
	)

	// Cache
	cacheRepo := repositories.NewCache(repositories.CacheConfig{
		TTL: config.CacheDuration,
	})

	// Memcached
	memcachedRepo := repositories.NewMemcached(repositories.MemcachedConfig{
		Host: config.MemcachedHost,
		Port: config.MemcachedPort,
	})

	// Tokenizer
	jwtTokenizer := tokenizers.NewTokenizer(
		tokenizers.JWTConfig{
			Key:      config.JWTKey,
			Duration: config.JWTDuration,
		},
	)

	// Services
	service := services.NewService(mySQLRepo, cacheRepo, memcachedRepo, jwtTokenizer, config.BcryptCost)

	// Handlers
	controller := controllers.NewController(service)

	// Create router
	router := gin.Default()

	// Use CORS middleware
	router.Use(utils.CorsMiddleware())

	// URL mappings
	router.GET("/users", controller.GetAll)
	router.GET("/users/:id", controller.GetByID)
	router.POST("/users", controller.Create)
	router.PUT("/users/:id", controller.Update)
	router.DELETE("/users/:id", controller.Delete)
	router.POST("/login", controller.Login)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "users-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Run application
	if err := router.Run(":" + config.Port); err != nil {
		log.Panicf("Error running application: %v", err)
	}
}
