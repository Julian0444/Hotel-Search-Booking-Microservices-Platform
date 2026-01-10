package users

import (
	"context"
	"fmt"
	"net/http"
	hotelsDomain "search-api/domain/hotels"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Service interface {
	Search(ctx context.Context, query string, offset int, limit int) ([]hotelsDomain.Hotel, error)
}

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{
		service: service,
	}
}


// Funcion para buscar hoteles en Solr
func (controller Controller) Search(c *gin.Context) {
	// Saca el query de la URL
	query := c.Query("q")

	// Saca el offset de la URL
	offset, err := strconv.Atoi(c.Query("offset"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err),
		})
		return
	}

	// Saca el limit de la URL
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err),
		})
		return
	}

	// Llama a la funcion de busqueda de hoteles del servicio
	hotels, err := controller.service.Search(c.Request.Context(), query, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error searching hotels: %s", err.Error()),
		})
		return
	}

	// Devuelve los hoteles encontrados
	c.JSON(http.StatusOK, hotels)
}
