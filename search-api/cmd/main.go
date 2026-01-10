package main

import (
	"log"
	"search-api/internal/clients/queues"
	controllers "search-api/internal/controllers/search"
	repositories "search-api/internal/repositories/hotels"
	services "search-api/internal/services/search"
	"search-api/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Solr
	solrRepo := repositories.NewSolr(repositories.SolrConfig{
		Host:       "solr",   // Solr host
		Port:       "8983",   // Solr port
		Collection: "hotels", // Collection name
	})

	// Rabbit
	//Este es el que consume de la cola de rabbit
	eventsQueue := queues.NewRabbit(queues.RabbitConfig{
		Host:      "rabbitmq",
		Port:      "5672",
		Username:  "root",
		Password:  "root",
		QueueName: "hotels-news",
	})

	// Hotels API
	hotelsAPI := repositories.NewHTTP(repositories.HTTPConfig{
		Host: "hotels-api",
		Port: "8081",
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

	router.GET("/search", controller.Search)
	if err := router.Run(":8082"); err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}
