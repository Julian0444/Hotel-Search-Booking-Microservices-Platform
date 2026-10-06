package config

import "os"

var (
	// MongoDB
	MongoReplicaSet             = getEnv("MONGO_REPLICA_SET", "rs0")
	MongoHost                   = getEnv("MONGO_HOST", "localhost")
	MongoPort                   = getEnv("MONGO_PORT", "27017")
	MongoUsername               = getEnv("MONGO_USERNAME", "root")
	MongoPassword               = getEnv("MONGO_PASSWORD", "root")
	MongoDatabase               = getEnv("MONGO_DATABASE", "hotels-api")
	MongoCollectionHotels       = getEnv("MONGO_COLLECTION_HOTELS", "hotels")
	MongoCollectionReservations = getEnv("MONGO_COLLECTION_RESERVATIONS", "reservations")
	MongoCollectionInventory    = getEnv("MONGO_COLLECTION_INVENTORY", "reservation_inventory")
	// Registros de Idempotency-Key (A3): índice único {key, user_id} + TTL 24h
	MongoCollectionIdempotency = getEnv("MONGO_COLLECTION_IDEMPOTENCY", "idempotency_keys")

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

	// Panel admin de microservicios (C2): a qué /readyz le pega cada probe.
	// Formato: "svc=url1,url2;svc2=url3". El default refleja los nombres DNS
	// de docker-compose; en k8s se sobreescribe por env (Services propios).
	MicroservicesTargets = getEnv("MICROSERVICES_TARGETS",
		"users-api=http://users-api-1:8082,http://users-api-2:8082,http://users-api-3:8082;"+
			"hotels-api=http://127.0.0.1:8081;"+
			"search-api=http://search-api:8082")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
