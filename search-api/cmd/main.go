package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/clients/queues"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/config"
	healthControllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/health"
	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/search"
	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/middlewares"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/repositories/hotels"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/services/search"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Logging estructurado JSON (O2): un logger por proceso con el nombre del
	// servicio; los log.* del stdlib quedan puenteados al mismo handler.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("service", "search-api")
	slog.SetDefault(logger)

	slog.Info("starting search-api", "port", config.Port)

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

	// Launch rabbit consumer
	if err := eventsQueue.StartConsumer(service.HandleHotelNew); err != nil {
		slog.Error("error running consumer", "error", err)
		os.Exit(1)
	}

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

	// Routes
	router.GET("/search", controller.Search)

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// (Solr + RabbitMQ); /health queda como alias de /livez por compat.
	healthController := healthControllers.NewController("search-api", map[string]healthControllers.CheckFunc{
		"solr": solrRepo.Ping,
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

	// Run server
	if err := router.Run(":" + config.Port); err != nil {
		slog.Error("error running application", "error", err)
		os.Exit(1)
	}
}
