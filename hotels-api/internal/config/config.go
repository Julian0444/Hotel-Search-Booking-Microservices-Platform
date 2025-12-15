package config

import "time"

const (
	MongoHost                   = "mongo"
	MongoPort                   = "27017"
	MongoUsername               = "root"
	MongoPassword               = "root"
	MongoDatabase               = "hotels-api"
	MongoCollectionHotels       = "hotels"
	MongoCollectionReservations = "reservations"

	CacheMaxSize      = 100000
	CacheItemsToPrune = 100
	CacheDuration     = 30 * time.Second

	RabbitHost      = "rabbitmq"
	RabbitPort      = "5672"
	RabbitUsername  = "root"
	RabbitPassword  = "root"
	RabbitQueueName = "hotels-news"

	JWTSecret = "ThisIsAnExampleJWTKey!"
)