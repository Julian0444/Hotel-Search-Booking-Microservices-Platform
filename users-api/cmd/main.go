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
	log.Printf("Starting users-api on port %s...", config.Port)

	// MySQL repository (source of truth)
	mySQLRepo := repositories.NewMySQL(
		repositories.MySQLConfig{
			Host:     config.MySQLHost,
			Port:     config.MySQLPort,
			Database: config.MySQLDatabase,
			Username: config.MySQLUsername,
			Password: config.MySQLPassword,
		},
	)

	// Cache L1 (in-process)
	cacheRepo := repositories.NewCache(repositories.CacheConfig{
		TTL: config.CacheDuration,
	})

	// Memcached L2 (distributed)
	memcachedRepo := repositories.NewMemcached(repositories.MemcachedConfig{
		Host: config.MemcachedHost,
		Port: config.MemcachedPort,
	})

	// JWT Tokenizer
	jwtTokenizer := tokenizers.NewTokenizer(
		tokenizers.JWTConfig{
			Key:      config.JWTKey,
			Duration: config.JWTDuration,
		},
	)

	// Service
	service := services.NewService(mySQLRepo, cacheRepo, memcachedRepo, jwtTokenizer, config.BcryptCost)

	// Controller
	controller := controllers.NewController(service)

	// Router
	router := gin.Default()
	router.Use(utils.CorsMiddleware())

	// Routes
	router.GET("/users", controller.GetAll)
	router.GET("/users/:id", controller.GetByID)
	router.POST("/users", controller.Create)
	router.DELETE("/users/:id", controller.Delete)
	router.POST("/login", controller.Login)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "users-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Run server
	if err := router.Run(":" + config.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
