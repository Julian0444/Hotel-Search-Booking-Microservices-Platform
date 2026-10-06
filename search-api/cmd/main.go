package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/clients/queues"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/config"
	healthControllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/health"
	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/controllers/search"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/middlewares"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/repositories/hotels"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/services/search"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/platform-contracts/cors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	reindexOnly := flag.Bool("reindex-once", false, "reconcile once without HTTP or RabbitMQ consumer; requires sole writer")
	flag.Parse()
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
	if *reindexOnly {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		count, err := service.Backfill(ctx)
		if err != nil {
			slog.Error("reconciliation failed", "error", err)
			os.Exit(1)
		}
		slog.Info("reconciliation finished", "hotels_indexed", count)
		return
	}

	// Controllers
	controller := controllers.NewController(service)

	// Launch rabbit consumer: corre en background con reconexión propia (E5)
	// — ya no aborta el proceso si RabbitMQ tarda en estar listo.
	eventsQueue.StartConsumer(service.HandleHotelNew)

	// La misma exclusión cubre startup, reconciliación manual y eventos.
	// La pasada periódica recupera publicaciones perdidas sin intervención.
	lifecycle, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	reconcileDone := make(chan struct{})
	go func() {
		defer close(reconcileDone)
		for lifecycle.Err() == nil {
			ctx, cancel := context.WithTimeout(utils.WithRequestID(lifecycle, uuid.NewString()), 2*time.Minute)
			indexed, err := service.Backfill(ctx)
			cancel()
			delay := time.Minute
			if err != nil {
				slog.Warn("catalogue reconciliation failed; will retry", "error", err)
				delay = 5 * time.Second
			} else {
				slog.Info("catalogue reconciled", "hotels_indexed", indexed)
			}
			select {
			case <-lifecycle.Done():
				return
			case <-time.After(delay):
			}
		}
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
	router.Use(middlewares.RequestID())

	// CORS compartido de la plataforma (C1/CQ4): allowlist por env, sin
	// credentials — reemplaza el "*" + credentials hardcodeado. El gateway ya
	// no duplica estos headers (RV27).
	router.Use(cors.Middleware())

	// Middleware JWT (audiencia search-api) para las rutas de administración
	jwtMiddleware := middlewares.NewJWTMiddleware(config.JWTSecret)

	// Rutas versionadas bajo /api/v1 (A2); health/livez/readyz quedan fuera.
	// RequireJSON: la API es JSON-only, Accept incompatible → 406 (A8).
	v1 := router.Group("/api/v1", middlewares.RequireJSON())
	v1.GET("/search", controller.Search)
	// Reindex on-demand (E3): mismo backfill del arranque, solo admins
	v1.POST("/reindex", jwtMiddleware.Authenticate(), middlewares.AdminOnly(), controller.Reindex)

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// (Solr requerido, RabbitMQ informativo); /health es alias de /livez.
	healthController := healthControllers.NewController("search-api", map[string]healthControllers.CheckFunc{
		"solr": solrRepo.Ping,
	}, map[string]healthControllers.CheckFunc{
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

	// Graceful shutdown (C12): el server corre en una goroutine y main espera
	// SIGINT/SIGTERM. Al recibir la señal se drenan los requests en vuelo
	// (hasta 10s) y después Close() cancela el Consume y espera el mensaje en
	// vuelo antes de cerrar RabbitMQ — con manual ack (E1) un corte a mitad se
	// re-entregaría, pero salir limpio evita el retrabajo.
	srv := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-lifecycle.Done()

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("server shutdown incomplete", "error", err)
	}
	eventsQueue.Close()
	<-reconcileDone
	slog.Info("shutdown complete")
}
