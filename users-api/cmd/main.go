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

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/config"
	healthControllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/controllers/health"
	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/controllers/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/middlewares"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/tokenizers"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/platform-contracts/cors"

	"github.com/gin-gonic/gin"
)

func main() {
	// Logging estructurado JSON (O2): un logger por proceso con el nombre del
	// servicio y la réplica; los log.* del stdlib quedan puenteados al mismo handler.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("service", "users-api")
	if config.InstanceID != "" {
		logger = logger.With("instance", config.InstanceID)
	}
	slog.SetDefault(logger)

	slog.Info("starting users-api", "port", config.Port)

	// Nunca arrancar con el secreto placeholder: permitiría forjar tokens admin.
	if config.JWTKey == "" || config.JWTKey == "your-secret-key-change-in-production" {
		slog.Error("JWT_SECRET must be set to a non-default value")
		os.Exit(1)
	}

	// MySQL repository (source of truth)
	mySQLRepo := repositories.NewMySQL(
		repositories.MySQLConfig{
			Host:        config.MySQLHost,
			Port:        config.MySQLPort,
			Database:    config.MySQLDatabase,
			Username:    config.MySQLUsername,
			Password:    config.MySQLPassword,
			AutoMigrate: config.AutoMigrate,
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

	// JWT middleware (verify) — mismo contrato de claims que hotels-api
	jwtMiddleware := middlewares.NewJWTMiddleware(config.JWTKey)

	// Seed del primer admin (idempotente entre réplicas)
	seedAdmin(service)

	// Gin en release salvo override explícito (O4): GIN_MODE=debug lo
	// restaura para desarrollo local.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Router: gin.New en vez de gin.Default — el access-log lo emite el
	// middleware RequestID en JSON vía slog (O1/O2).
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())
	// CORS compartido de la plataforma (C1/CQ4): allowlist por env, sin
	// credentials. El gateway ya no duplica estos headers (RV27).
	router.Use(cors.Middleware())
	// Deadline por request: DeadlineExceeded se mapea a 503 en los controllers (R1)
	router.Use(middlewares.RequestTimeout(config.RequestTimeout))

	// Rutas versionadas bajo /api/v1 (A2); health/livez/readyz quedan fuera.
	// RequireJSON: la API es JSON-only, Accept incompatible → 406 (A8).
	v1 := router.Group("/api/v1", middlewares.RequireJSON())

	// Rutas públicas
	v1.POST("/users", controller.Create)
	v1.POST("/login", controller.Login)

	// Rutas protegidas: listar solo admin; ver/borrar solo el dueño o admin
	authRoutes := v1.Group("/", jwtMiddleware.Authenticate())
	{
		authRoutes.GET("/users", middlewares.AdminOnly(), controller.GetAll)
		authRoutes.GET("/users/:id", middlewares.OwnerOrAdmin(), controller.GetByID)
		authRoutes.DELETE("/users/:id", middlewares.OwnerOrAdmin(), controller.Delete)
	}

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// MySQL obligatorio; Memcached opcional con fallback a DB.
	healthController := healthControllers.NewController("users-api", map[string]healthControllers.CheckFunc{
		"mysql": mySQLRepo.Ping,
	}, map[string]healthControllers.CheckFunc{
		"memcached": memcachedRepo.Ping,
	})
	router.GET("/livez", healthController.Livez)
	router.GET("/readyz", healthController.Readyz)
	router.GET("/health", healthController.Livez)

	// Graceful shutdown (C12): el server corre en una goroutine y main espera
	// SIGINT/SIGTERM; con 3 réplicas detrás de nginx, drenar los requests en
	// vuelo (hasta 10s) es lo que hace transparente un rolling restart.
	// Memcached no requiere cierre (cliente sin estado de conexión dedicada).
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
	if err := mySQLRepo.Close(); err != nil {
		slog.Warn("error closing mysql pool", "error", err)
	}
	slog.Info("shutdown complete")
}

// seedAdmin crea el primer administrador si ADMIN_USERNAME/ADMIN_PASSWORD están seteados.
// Es idempotente: las 3 réplicas lo ejecutan y el índice único de username hace
// fallar a las que llegan tarde (el duplicado se ignora).
func seedAdmin(service services.Service) {
	if config.SeedAdminUsername == "" || config.SeedAdminPassword == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := service.Create(ctx, usersDomain.LoginRequest{
		Username: config.SeedAdminUsername,
		Password: config.SeedAdminPassword,
		Tipo:     "administrador",
	})
	switch {
	case err == nil:
		slog.Info("seed: admin user created", "username", config.SeedAdminUsername)
	case errors.Is(err, usersDomain.ErrUsernameTaken):
		// Idempotencia entre réplicas via sentinel tipado (C6)
		slog.Info("seed: admin user already exists", "username", config.SeedAdminUsername)
	default:
		slog.Warn("seed: admin creation failed", "error", err)
	}
}
