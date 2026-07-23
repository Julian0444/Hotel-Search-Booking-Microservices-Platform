package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/clients/queues"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/config"
	healthControllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/health"
	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/search"
	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/middlewares"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/repositories/hotels"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/services/search"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	// Logging estructurado JSON (O2): un logger por proceso con el nombre del
	// servicio; los log.* del stdlib quedan puenteados al mismo handler.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("service", "search-api")
	slog.SetDefault(logger)

	slog.Info("starting search-api", "port", config.Port)

	// Nunca arrancar con el secreto placeholder: POST /reindex es solo-admin
	// y valida JWTs con este secreto (mismo fail-fast que users/hotels-api).
	if config.JWTSecret == "" || config.JWTSecret == "your-secret-key-change-in-production" {
		slog.Error("JWT_SECRET must be set to a non-default value")
		os.Exit(1)
	}

	// Solr
	solrRepo := repositories.NewSolr(repositories.SolrConfig{
		Host:       config.SolrHost,
		Port:       config.SolrPort,
		Collection: config.SolrCollection,
	})

	// Rabbit - consume de la cola de RabbitMQ
	eventsQueue := queues.NewRabbit(queues.RabbitConfig{
		Host:      config.RabbitHost,
		Port:      config.RabbitPort,
		Username:  config.RabbitUsername,
		Password:  config.RabbitPassword,
		QueueName: config.RabbitQueueName,
	})

	// Hotels API
	hotelsAPI := repositories.NewHTTP(repositories.HTTPConfig{
		Host: config.HotelsAPIHost,
		Port: config.HotelsAPIPort,
	})

	// Services
	service := services.NewService(solrRepo, hotelsAPI)

	// Controllers
	controller := controllers.NewController(service)

	// Launch rabbit consumer: corre en background con reconexión propia (E5)
	// — ya no aborta el proceso si RabbitMQ tarda en estar listo.
	eventsQueue.StartConsumer(service.HandleHotelNew)

	// Backfill del índice al arranque (E3): Solr es un índice derivado y se
	// reconstruye desde hotels-api (fuente de verdad). En un arranque en frío
	// hotels-api puede tardar: reintentos con backoff antes de rendirse
	// (queda POST /reindex como rescate manual).
	go func() {
		backoff := 2 * time.Second
		for attempt := 1; attempt <= 5; attempt++ {
			ctx := utils.WithRequestID(context.Background(), uuid.NewString())
			indexed, err := service.Backfill(ctx)
			if err == nil {
				slog.Info("startup backfill completed", "hotels_indexed", indexed, "attempt", attempt)
				return
			}
			slog.Warn("startup backfill attempt failed", "attempt", attempt, "error", err)
			time.Sleep(backoff)
			backoff *= 2
		}
		slog.Error("startup backfill failed after retries; index may be stale until POST /reindex")
	}()

	// Gin en release salvo override explícito (O4): GIN_MODE=debug lo
	// restaura para desarrollo local.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router: gin.New en vez de gin.Default — el access-log lo emite
	// el middleware RequestID en JSON vía slog (O1/O2).
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())

	// Use CORS middleware
	router.Use(utils.CorsMiddleware())

	// Middleware JWT (audiencia search-api) para las rutas de administración
	jwtMiddleware := middleware.NewJWTMiddleware(config.JWTSecret)

	// Rutas versionadas bajo /api/v1 (A2); health/livez/readyz quedan fuera.
	// RequireJSON: la API es JSON-only, Accept incompatible → 406 (A8).
	v1 := router.Group("/api/v1", middleware.RequireJSON())
	v1.GET("/search", controller.Search)
	// Reindex on-demand (E3): mismo backfill del arranque, solo admins
	v1.POST("/reindex", jwtMiddleware.Authenticate(), middleware.AdminOnly(), controller.Reindex)

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// (Solr + RabbitMQ); /health queda como alias de /livez por compat.
	healthController := healthControllers.NewController("search-api", map[string]healthControllers.CheckFunc{
		"solr": solrRepo.Ping,
		"rabbitmq": func(_ context.Context) error {
			// IsConnected ahora exige el consumer vivo, no solo la conexión (RV17)
			if !eventsQueue.IsConnected() {
				return errors.New("rabbitmq consumer not running")
			}
			return nil
		},
	})
	router.GET("/livez", healthController.Livez)
	router.GET("/readyz", healthController.Readyz)
	router.GET("/health", healthController.Livez)

	// Run server
	if err := router.Run(":" + config.Port); err != nil {
		slog.Error("error running application", "error", err)
		os.Exit(1)
	}
}
