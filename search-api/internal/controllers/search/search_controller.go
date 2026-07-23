package search

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/apperr"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"

	"github.com/gin-gonic/gin"
)

type Service interface {
	Search(ctx context.Context, query string, offset int, limit int) ([]hotelsDomain.Hotel, int, error)
	Backfill(ctx context.Context) (int, error)
}

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{
		service: service,
	}
}

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// paginationParams parsea ?limit y ?offset con defaults y clamp (RV15), el
// mismo patrón que hotels-api: ausente/inválido/negativo cae al default en
// vez de 400, y limit se topea en 100 (post-E2 un rows negativo era 500 y uno
// gigante un DoS contra Solr).
func paginationParams(c *gin.Context) (int, int) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	if err != nil || limit < 1 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	return limit, offset
}

// Funcion para buscar hoteles en Solr. Responde el envelope estándar (A5) con
// el total real del índice en meta (RV22: el frontend calculaba las páginas
// sobre la página actual y la paginación quedaba rota por construcción).
func (controller Controller) Search(c *gin.Context) {
	// Saca el query de la URL; limit/offset clampeados (RV15)
	query := c.Query("q")
	limit, offset := paginationParams(c)

	// Llama a la funcion de busqueda de hoteles del servicio
	hotels, total, err := controller.service.Search(c.Request.Context(), query, offset, limit)
	if err != nil {
		apperr.Abort(c, http.StatusInternalServerError, "internal", "error searching hotels", err)
		return
	}

	// Devuelve los hoteles encontrados
	c.JSON(http.StatusOK, gin.H{
		"data": hotels,
		"meta": gin.H{"total": total, "limit": limit, "offset": offset},
	})
}

// Reindex dispara el backfill completo on-demand (E3). Solo admins: la ruta
// va detrás del middleware JWT + AdminOnly en cmd/main.go. Es sincrónico a
// propósito — el catálogo de la demo es chico y el 200 confirma el índice
// reconstruido.
func (controller Controller) Reindex(c *gin.Context) {
	indexed, err := controller.service.Backfill(c.Request.Context())
	if err != nil {
		apperr.Abort(c, http.StatusInternalServerError, "internal", "error reindexing hotels", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"indexed": indexed}})
}
