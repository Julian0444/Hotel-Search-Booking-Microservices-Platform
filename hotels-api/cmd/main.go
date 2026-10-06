package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/clients/queues"
	controllersHealth "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/health"
	controllersHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/hotels"
	controllersMicroservices "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/controllers/microservices"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/middlewares"
	repositoriesHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/repositories/hotels"
	servicesHotels "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/services/hotels"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/config"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/platform-contracts/cors"

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
		ReplicaSet:              config.MongoReplicaSet,
		Port:                    config.MongoPort,
		Username:                config.MongoUsername,
		Password:                config.MongoPassword,
		Database:                config.MongoDatabase,
		Collection_hotels:       config.MongoCollectionHotels,
		Collection_reservations: config.MongoCollectionReservations,
		Collection_inventory:    config.MongoCollectionInventory,
		Collection_idempotency:  config.MongoCollectionIdempotency,
	})

	eventsQueue := queues.NewRabbit(queues.RabbitConfig{
		Host:      config.RabbitHost,
		Port:      config.RabbitPort,
		Username:  config.RabbitUsername,
		Password:  config.RabbitPassword,
		QueueName: config.RabbitQueueName,
	})

	// Configuración de Servicios
	hotelsService := servicesHotels.NewService(hotelsRepo, eventsQueue)

	// Configuración de Controladores
	hotelsController := controllersHotels.NewController(hotelsService)
	microservicesController := controllersMicroservices.NewController(
		controllersMicroservices.ParseTargets(config.MicroservicesTargets))

	// Configuración de middlewares
	jwtMiddleware := middlewares.NewJWTMiddleware(config.JWTSecret)

	// Gin en release salvo override explícito (O4): GIN_MODE=debug lo
	// restaura para desarrollo local.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Configuración del servidor HTTP: gin.New en vez de gin.Default — el
	// access-log lo emite el middleware RequestID en JSON vía slog (O1/O2).
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())

	// CORS compartido de la plataforma (C1/CQ4): allowlist por env, sin
	// credentials — reemplaza el combo inválido "*" + AllowCredentials:true
	// que el browser rechaza. El gateway ya no duplica estos headers (RV27).
	// Idempotency-Key va en los Allow-Headers (POST de reservas, A3).
	router.Use(cors.Middleware())

	// Rutas versionadas bajo /api/v1 (A2); health/livez/readyz quedan fuera.
	// RequireJSON: la API es JSON-only, Accept incompatible → 406 (A8).
	v1 := router.Group("/api/v1", middlewares.RequireJSON())

	v1.GET("/hotels", hotelsController.GetHotels) // listado paginado (E3: lo consume el backfill de search-api)
	v1.GET("/hotels/:hotel_id", hotelsController.GetHotelByID)
	v1.POST("/hotels/availability", hotelsController.GetAvailability)

	// Las reservas de un hotel exponen user_id de todos los huéspedes (PII):
	// solo admins (C3). Misma ruta, ahora autenticada.
	v1.GET("/hotels/:hotel_id/reservations",
		jwtMiddleware.Authenticate(), middlewares.AdminOnly(), hotelsController.GetReservationsByHotelID)

	// Rutas protegidas para usuarios autenticados
	userRoutes := v1.Group("/", jwtMiddleware.Authenticate(), middlewares.LoggedUserOnly())
	{
		// POST de reservas con Idempotency-Key (A3): mismo key+user → misma
		// respuesta, una sola reserva
		userRoutes.POST("/reservations", middlewares.Idempotency(), hotelsController.CreateReservation)
		userRoutes.GET("/reservations/:id", hotelsController.GetReservationByID)
		userRoutes.DELETE("/reservations/:id", hotelsController.CancelReservation)
		userRoutes.GET("/users/:user_id/reservations", hotelsController.GetReservationsByUserID)
		userRoutes.GET("/users/:user_id/hotels/:hotel_id/reservations", hotelsController.GetReservationsByUserAndHotelID)
	}

	// Rutas protegidas para administradores
	adminRoutes := v1.Group("/admin", jwtMiddleware.Authenticate(), middlewares.AdminOnly())
	{
		// Gestión de hoteles (solo admins)
		adminRoutes.POST("/hotels", hotelsController.Create)
		adminRoutes.PUT("/hotels/:hotel_id", hotelsController.Update)
		adminRoutes.DELETE("/hotels/:hotel_id", hotelsController.Delete)

		// Panel de microservicios (solo admins): READ-ONLY con health real
		// vía /readyz (C2) — las acciones mock de scale/restart/logs se
		// eliminaron: un panel de solo estado no simula operaciones de escritura.
		adminRoutes.GET("/microservices", microservicesController.GetMicroservicesStatus)
	}

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// Mongo es obligatorio; RabbitMQ es secundario y se informa sin retirar reservas.
	healthController := controllersHealth.NewController("hotels-api", map[string]controllersHealth.CheckFunc{
		"mongo": hotelsRepo.Ping,
	}, map[string]controllersHealth.CheckFunc{
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

	// Graceful shutdown (C12): el server corre en una goroutine y main espera
	// SIGINT/SIGTERM. Al recibir la señal se drenan los requests en vuelo
	// (hasta 10s) y recién entonces se cierran RabbitMQ y Mongo — es lo que
	// permite rolling restarts sin 502 (compose hoy, k8s en el plan 09).
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("server shutdown incomplete", "error", err)
	}
	eventsQueue.Close()
	if err := hotelsRepo.Disconnect(shutdownCtx); err != nil {
		slog.Warn("error disconnecting from mongo", "error", err)
	}
	slog.Info("shutdown complete")
}
