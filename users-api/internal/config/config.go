package config

import (
	"os"
	"strconv"
	"time"
)

var (
	// MySQL
	MySQLHost     = getEnv("MYSQL_HOST", "localhost")
	MySQLPort     = getEnv("MYSQL_PORT", "3306")
	MySQLDatabase = getEnv("MYSQL_DATABASE", "users_db")
	MySQLUsername = getEnv("MYSQL_USERNAME", "root")
	MySQLPassword = getEnv("MYSQL_PASSWORD", "root")

	// Cache L1 (in-process)
	CacheDuration = getDurationEnv("CACHE_DURATION", 30*time.Second)

	// Memcached L2
	MemcachedHost = getEnv("MEMCACHED_HOST", "localhost")
	MemcachedPort = getEnv("MEMCACHED_PORT", "11211")

	// JWT - debe coincidir con hotels-api para validar tokens
	JWTKey      = getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	JWTDuration = getDurationEnv("JWT_DURATION", 24*time.Hour)

	// Bcrypt
	BcryptCost = getIntEnv("BCRYPT_COST", 10)

	// Server
	Port = getEnv("PORT", "8082")
)

// getEnv obtiene una variable de entorno o retorna el valor por defecto.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv obtiene una variable de entorno como int o retorna el valor por defecto.
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv obtiene una variable de entorno como duration o retorna el valor por defecto.
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
