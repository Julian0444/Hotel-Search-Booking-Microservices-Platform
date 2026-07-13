package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	config "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/config"
	healthControllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/controllers/health"
	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/controllers/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	middleware "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/middlewares"
	repositories "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	services "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/tokenizers"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/utils"

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
	jwtMiddleware := middleware.NewJWTMiddleware(config.JWTKey)

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
	router.Use(middleware.RequestID())
	router.Use(utils.CorsMiddleware())
	// Deadline por request: DeadlineExceeded se mapea a 503 en los controllers (R1)
	router.Use(middleware.RequestTimeout(config.RequestTimeout))

	// Rutas públicas
	router.POST("/users", controller.Create)
	router.POST("/login", controller.Login)

	// Rutas protegidas: listar solo admin; ver/borrar solo el dueño o admin
	authRoutes := router.Group("/", jwtMiddleware.Authenticate())
	{
		authRoutes.GET("/users", middleware.AdminOnly(), controller.GetAll)
		authRoutes.GET("/users/:id", middleware.OwnerOrAdmin(), controller.GetByID)
		authRoutes.DELETE("/users/:id", middleware.OwnerOrAdmin(), controller.Delete)
	}

	// Health endpoints (O3): /livez barato, /readyz pinguea las deps propias
	// (MySQL + Memcached); /health queda como alias de /livez por compat.
	healthController := healthControllers.NewController("users-api", map[string]healthControllers.CheckFunc{
		"mysql":     mySQLRepo.Ping,
		"memcached": memcachedRepo.Ping,
	})
	router.GET("/livez", healthController.Livez)
	router.GET("/readyz", healthController.Readyz)
	router.GET("/health", healthController.Livez)

	// Run server
	if err := router.Run(":" + config.Port); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
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
	case strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate"):
		slog.Info("seed: admin user already exists", "username", config.SeedAdminUsername)
	default:
		slog.Warn("seed: admin creation failed", "error", err)
	}
}
