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

	// AutoMigrate de GORM: true para dev pelado (go run); el compose lo apaga
	// porque ahí migra el one-shot de golang-migrate (users-api/migrations).
	AutoMigrate = getBoolEnv("AUTO_MIGRATE", true)

	// Deadline por request: si MySQL se cuelga, fallar en ~5s con 503 en vez de
	// colgar el login en las 3 réplicas (R1).
	RequestTimeout = getDurationEnv("REQUEST_TIMEOUT", 5*time.Second)

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

	// Seed del primer administrador (opcional): si ambos están seteados,
	// se crea al arranque. Es la única vía para obtener un admin.
	SeedAdminUsername = getEnv("ADMIN_USERNAME", "")
	SeedAdminPassword = getEnv("ADMIN_PASSWORD", "")

	// Server
	Port = getEnv("PORT", "8082")

	// Identificador de réplica (users-api-1/2/3 en compose): va como atributo
	// del logger para distinguir instancias detrás del load balancer (O2).
	InstanceID = getEnv("INSTANCE_ID", "")
)

// getEnv obtiene una variable de entorno o retorna el valor por defecto.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv obtiene una variable de entorno como bool o retorna el valor por defecto.
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
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
