package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/clients/queues"
	controllersHealth "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/health"
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
	// Logging estructurado JSON (O2): un logger por proceso con el nombre del
	// servicio; los log.* del stdlib quedan puenteados al mismo handler.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("service", "hotels-api")
	slog.SetDefault(logger)

	slog.Info("starting hotels-api", "port", config.Port)

	// Nunca arrancar con el secreto placeholder: permitiría forjar tokens admin.
	if config.JWTSecret == "" || config.JWTSecret == "your-secret-key-change-in-production" {
		slog.Error("JWT_SECRET must be set to a non-default value")
		os.Exit(1)
	}

	// Configuración de Repositorios
	hotelsRepo := repositoriesHotels.NewMongo(repositoriesHotels.MongoConfig{
		Host:                    config.MongoHost,
		Port:                    config.MongoPort,
		Username:                config.MongoUsername,
		Password:                config.MongoPassword,
		Database:                config.MongoDatabase,
		Collection_hotels:       config.MongoCollectionHotels,
		Collection_reservations: config.MongoCollectionReservations,
		Collection_inventory:    config.MongoCollectionInventory,
		Collection_idempotency:  config.MongoCollectionIdempotency,
	})

	cacheRepo := repositoriesHotels.NewCache(repositoriesHotels.CacheConfig{
		MaxSize:      config.CacheMaxSize,
		ItemsToPrune: config.CacheItemsToPrune,
		Duration:     config.CacheDuration,
	})

	eventsQueue := queues.NewRabbit(queues.RabbitConfig{
		Host:                  config.RabbitHost,
		Port:                  config.RabbitPort,
		Username:              config.RabbitUsername,
		Password:              config.RabbitPassword,
		QueueName:             config.RabbitQueueName,
		ReservationsQueueName: config.RabbitReservationsQueueName,
	})

	// Configuración de Servicios
	hotelsService := servicesHotels.NewService(hotelsRepo, cacheRepo, eventsQueue)

	// Configuración de Controladores
	hotelsController := controllersHotels.NewController(hotelsService)
	microservicesController := controllersMicroservices.NewController()

	// Configuración de middlewares
	jwtMiddleware := middleware.NewJWTMiddleware(config.JWTSecret)

	// Gin en release salvo override explícito (O4): GIN_MODE=debug lo
	// restaura para desarrollo local.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Configuración del servidor HTTP: gin.New en vez de gin.Default — el
	// access-log lo emite el middleware RequestID en JSON vía slog (O1/O2).
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())

	// Configuración de CORS (Idempotency-Key habilitado para el POST de reservas, A3)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Idempotency-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rutas versionadas bajo /api/v1 (A2); health/livez/readyz quedan fuera.
	// RequireJSON: la API es JSON-only, Accept incompatible → 406 (A8).
	v1 := router.Group("/api/v1", middleware.RequireJSON())

	v1.GET("/hotels", hotelsController.GetHotels) // listado paginado (E3: lo consume el backfill de search-api)
	v1.GET("/hotels/:hotel_id", hotelsController.GetHotelByID)
	v1.GET("/hotels/:hotel_id/reservations", hotelsController.GetReservationsByHotelID)
	v1.POST("/hotels/availability", hotelsController.GetAvailability)

	// Rutas protegidas para usuarios autenticados
	userRoutes := v1.Group("/", jwtMiddleware.Authenticate(), middleware.LoggedUserOnly())
	{
		// POST de reservas con Idempotency-Key (A3): mismo key+user → misma
		// respuesta, una sola reserva
		userRoutes.POST("/reservations", middleware.Idempotency(hotelsRepo), hotelsController.CreateReservation)
		userRoutes.GET("/reservations/:id", hotelsController.GetReservationByID)
		userRoutes.DELETE("/reservations/:id", hotelsController.CancelReservation)
		userRoutes.GET("/users/:user_id/reservations", hotelsController.GetReservationsByUserID)
		userRoutes.GET("/users/:user_id/hotels/:hotel_id/reservations", hotelsController.GetReservationsByUserAndHotelID)
	}

	// Rutas protegidas para administradores
	adminRoutes := v1.Group("/admin", jwtMiddleware.Authenticate(), middleware.AdminOnly())
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

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// (Mongo + RabbitMQ); /health queda como alias de /livez por compat.
	healthController := controllersHealth.NewController("hotels-api", map[string]controllersHealth.CheckFunc{
		"mongo": hotelsRepo.Ping,
		"rabbitmq": func(_ context.Context) error {
			if !eventsQueue.IsConnected() {
				return errors.New("rabbitmq not connected")
			}
			return nil
		},
	})
	router.GET("/livez", healthController.Livez)
	router.GET("/readyz", healthController.Readyz)
	router.GET("/health", healthController.Livez)

	// Ejecutar el servidor
	if err := router.Run(":" + config.Port); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
