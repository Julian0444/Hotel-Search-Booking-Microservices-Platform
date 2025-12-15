package main

import (
	"log"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/clients/queues"
	controllersHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/hotels"
	controllersMicroservices "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/microservices"
	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/middlewares"
	repositoriesHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"
	servicesHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/services"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Configuración de Repositorios
	hotelsRepo := repositoriesHotels.NewMongo(repositoriesHotels.MongoConfig{
		Host:                    config.MongoHost,
		Port:                    config.MongoPort,
		Username:                config.MongoUsername,
		Password:                config.MongoPassword,
		Database:                config.MongoDatabase,
		Collection_hotels:       config.MongoCollectionHotels,
		Collection_reservations: config.MongoCollectionReservations,
	})

	cacheRepo := repositoriesHotels.NewCache(repositoriesHotels.CacheConfig{
		MaxSize:      config.CacheMaxSize,
		ItemsToPrune: config.CacheItemsToPrune,
		Duration:     config.CacheDuration,
	})

	eventsQueue := queues.NewRabbit(queues.RabbitConfig{
		Host:      config.RabbitHost,
		Port:      config.RabbitPort,
		Username:  config.RabbitUsername,
		Password:  config.RabbitPassword,
		QueueName: config.RabbitQueueName,
	})

	// Configuración de Servicios
	hotelsService := servicesHotels.NewService(hotelsRepo, cacheRepo, eventsQueue)

	// Configuración de Controladores
	hotelsController := controllersHotels.NewController(hotelsService)
	microservicesController := controllersMicroservices.NewController()

	// Configuración de middlewares
	jwtMiddleware := middleware.NewJWTMiddleware(config.JWTSecret)

	// Configuración del servidor HTTP
	router := gin.Default()

	// Configuración de CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Configuración de rutas
	router.GET("/hotels/:hotel_id", hotelsController.GetHotelByID)
	router.GET("/hotels/:hotel_id/reservations", hotelsController.GetReservationsByHotelID)
	router.GET("/users/:user_id/hotels/:hotel_id/reservations", hotelsController.GetReservationsByUserAndHotelID)
	router.POST("/hotels/availability", hotelsController.GetAvailability)

	// Rutas protegidas para usuarios autenticados
	userRoutes := router.Group("/", jwtMiddleware.Authenticate(), middleware.LoggedUserOnly())
	{
		userRoutes.POST("/reservations", hotelsController.CreateReservation)
		userRoutes.DELETE("/reservations/:id", hotelsController.CancelReservation)
		userRoutes.GET("/users/:user_id/reservations", hotelsController.GetReservationsByUserID)
	}

	// Rutas protegidas para administradores
	adminRoutes := router.Group("/admin", jwtMiddleware.Authenticate(), middleware.AdminOnly())
	{
		// Gestión de hoteles (solo admins)
		adminRoutes.POST("/hotels", hotelsController.Create)
		adminRoutes.PUT("/hotels/:hotel_id", hotelsController.Update)
		adminRoutes.DELETE("/hotels/:hotel_id", hotelsController.Delete)

		// Gestión de microservicios (solo admins)
		adminRoutes.GET("/microservices", microservicesController.GetMicroservicesStatus)
		adminRoutes.POST("/microservices/scale", microservicesController.ScaleService)
		adminRoutes.GET("/microservices/:service_name/logs", microservicesController.GetServiceLogs)
		adminRoutes.POST("/microservices/:service_name/restart", microservicesController.RestartService)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "hotels-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Ejecutar el servidor
	if err := router.Run(":8081"); err != nil {
		log.Fatal("Error al ejecutar el servidor:", err)
	}
}
