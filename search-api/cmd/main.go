package main

import (
	"log"
	"time"

	"search-api/internal/clients/queues"
	"search-api/internal/config"
	controllers "search-api/internal/controllers/search"
	repositories "search-api/internal/repositories/hotels"
	services "search-api/internal/services/search"
	"search-api/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Printf("Starting search-api on port %s...", config.Port)

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

	// Controllers
	controller := controllers.NewController(service)

	// Launch rabbit consumer
	if err := eventsQueue.StartConsumer(service.HandleHotelNew); err != nil {
		log.Fatalf("Error running consumer: %v", err)
	}

	// Create router
	router := gin.Default()

	// Use CORS middleware
	router.Use(utils.CorsMiddleware())

	// Routes
	router.GET("/search", controller.Search)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "search-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Run server
	if err := router.Run(":" + config.Port); err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}
