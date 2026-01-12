package config

import (
	"os"
)

var (
	// Solr
	SolrHost       = getEnv("SOLR_HOST", "solr")
	SolrPort       = getEnv("SOLR_PORT", "8983")
	SolrCollection = getEnv("SOLR_COLLECTION", "hotels")

	// RabbitMQ
	RabbitHost      = getEnv("RABBIT_HOST", "rabbitmq")
	RabbitPort      = getEnv("RABBIT_PORT", "5672")
	RabbitUsername  = getEnv("RABBIT_USERNAME", "root")
	RabbitPassword  = getEnv("RABBIT_PASSWORD", "root")
	RabbitQueueName = getEnv("RABBIT_QUEUE_NAME", "hotels-news")

	// Hotels API
	HotelsAPIHost = getEnv("HOTELS_API_HOST", "hotels-api")
	HotelsAPIPort = getEnv("HOTELS_API_PORT", "8081")

	// Server
	Port = getEnv("PORT", "8082")
)

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
