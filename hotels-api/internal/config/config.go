package config

import (
	"os"
	"strconv"
	"time"
)

var (
	// MongoDB
	MongoHost                   = getEnv("MONGO_HOST", "localhost")
	MongoPort                   = getEnv("MONGO_PORT", "27017")
	MongoUsername               = getEnv("MONGO_USERNAME", "root")
	MongoPassword               = getEnv("MONGO_PASSWORD", "root")
	MongoDatabase               = getEnv("MONGO_DATABASE", "hotels-api")
	MongoCollectionHotels       = getEnv("MONGO_COLLECTION_HOTELS", "hotels")
	MongoCollectionReservations = getEnv("MONGO_COLLECTION_RESERVATIONS", "reservations")

	// Cache
	CacheMaxSize      = getInt64Env("CACHE_MAX_SIZE", 100000)
	CacheItemsToPrune = getUint32Env("CACHE_ITEMS_TO_PRUNE", 100)
	CacheDuration     = getDurationEnv("CACHE_DURATION", 30*time.Second)

	// RabbitMQ
	RabbitHost      = getEnv("RABBIT_HOST", "localhost")
	RabbitPort      = getEnv("RABBIT_PORT", "5672")
	RabbitUsername  = getEnv("RABBIT_USERNAME", "root")
	RabbitPassword  = getEnv("RABBIT_PASSWORD", "root")
	RabbitQueueName = getEnv("RABBIT_QUEUE_NAME", "hotels-news")

	// JWT - debe coincidir con users-api
	JWTSecret = getEnv("JWT_SECRET", "your-secret-key-change-in-production")

	// Server
	Port = getEnv("PORT", "8081")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getUint32Env(key string, defaultValue uint32) uint32 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseUint(value, 10, 32); err == nil {
			return uint32(intValue)
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
