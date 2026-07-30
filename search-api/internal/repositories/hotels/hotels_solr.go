package hotels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/dao/hotels"

	"github.com/stevenferrer/solr-go"
)

// solrOpTimeout acota cada llamada a Solr (R2): ni el consumer ni el backfill
// pueden quedar colgados en un índice que no responde — con deadline, un
// mensaje falla rápido y sigue la política retry/DLQ (E1).
const solrOpTimeout = 5 * time.Second

// solrOpCtx deriva el deadline por operación de Solr (R2).
func solrOpCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, solrOpTimeout)
}

type SolrConfig struct {
	Host       string // Solr host
	Port       string // Solr port
	Collection string // Solr collection name
}

type Solr struct {
	Client     *solr.JSONClient
	Collection string
	baseURL    string
	httpClient *http.Client
}

// Funcion para crear una nueva conexion a Solr
func NewSolr(config SolrConfig) Solr {
	// Construimos la URL base para la conexion a Solr
	baseURL := fmt.Sprintf("http://%s:%s", config.Host, config.Port)
	// Cliente HTTP con timeout propio (R2): red de contención por si alguna
	// llamada llega sin deadline en el ctx; lo comparten solr-go y el Ping.
	httpClient := &http.Client{Timeout: solrOpTimeout}
	// Creamos un nuevo cliente JSON para Solr
	client := solr.NewJSONClient(baseURL).
		WithRequestSender(solr.NewDefaultRequestSender().WithHTTPClient(httpClient))

	// Devuelve una nueva instancia de Solr
	return Solr{
		Client:     client,
		Collection: config.Collection,
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// Ping golpea el admin/ping del core (lo usa el /readyz, O3). solr-go no
// expone ping, así que va directo por HTTP.
func (searchEngine Solr) Ping(ctx context.Context) error {
	ctx, cancel := solrOpCtx(ctx)
	defer cancel()
	url := fmt.Sprintf("%s/solr/%s/admin/ping", searchEngine.baseURL, searchEngine.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error building solr ping request: %w", err)
	}
	resp, err := searchEngine.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error pinging solr: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("solr ping returned status %d", resp.StatusCode)
	}
	return nil
}

// hotelToDoc arma el documento Solr con los nombres de campo del schema.
func hotelToDoc(hotel hotels.Hotel) map[string]interface{} {
	return map[string]interface{}{
		"id":              hotel.ID,
		"name":            hotel.Name,
		"description":     hotel.Description,
		"address":         hotel.Address,
		"city":            hotel.City,
		"state":           hotel.State,
		"country":         hotel.Country,
		"phone":           hotel.Phone,
		"email":           hotel.Email,
		"price_per_night": hotel.PricePerNight,
		"avaiable_rooms":  hotel.AvaiableRooms,
		"check_in_time":   hotel.CheckInTime,
		"check_out_time":  hotel.CheckOutTime,
		"rating":          hotel.Rating,
		"amenities":       hotel.Amenities,
		"images":          hotel.Images,
	}
}

// addDocument indexa (o pisa, por uniqueKey id) un documento de hotel.
// Sin Commit() explícito (DB6): el autoCommit/autoSoftCommit del core
// (solrconfig.xml) se encarga — un hard-commit por documento era lo más caro
// que se le podía pedir a Solr.
func (searchEngine Solr) addDocument(ctx context.Context, hotel hotels.Hotel) error {
	// "add" con lista de documentos (permite indexar varios a la vez)
	indexRequest := map[string]interface{}{
		"add": []interface{}{hotelToDoc(hotel)},
	}

	body, err := json.Marshal(indexRequest)
	if err != nil {
		return fmt.Errorf("error marshaling hotel document: %w", err)
	}

	// Deadline por operación (R2)
	ctx, cancel := solrOpCtx(ctx)
	defer cancel()
	resp, err := searchEngine.Client.Update(ctx, searchEngine.Collection, solr.JSON, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error indexing hotel: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("failed to index hotel: %v", resp.Error)
	}
	return nil
}

// Index crea un nuevo documento de hotel en la coleccion de Solr
func (searchEngine Solr) Index(ctx context.Context, hotel hotels.Hotel) (string, error) {
	if err := searchEngine.addDocument(ctx, hotel); err != nil {
		return "", err
	}
	return hotel.ID, nil
}

// Update actualiza un documento de hotel (mismo "add": id es la uniqueKey)
func (searchEngine Solr) Update(ctx context.Context, hotel hotels.Hotel) error {
	return searchEngine.addDocument(ctx, hotel)
}

func (searchEngine Solr) Delete(ctx context.Context, id string) error {
	// Papara el documento a borrar, con el ID del hotel a borrar
	docToDelete := map[string]interface{}{
		"delete": map[string]interface{}{
			"id": id,
		},
	}

	// Convierte el documento a JSON
	body, err := json.Marshal(docToDelete)
	if err != nil {
		return fmt.Errorf("error marshaling hotel document: %w", err)
	}

	// Ejecuta el request de borrado usando el metodo Update (deadline R2)
	ctx, cancel := solrOpCtx(ctx)
	defer cancel()
	resp, err := searchEngine.Client.Update(ctx, searchEngine.Collection, solr.JSON, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error deleting hotel: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("failed to delete hotel: %v", resp.Error)
	}

	return nil
}

// solrMetaChars son los metacaracteres de la sintaxis Lucene/edismax
// (incluye & y | sueltos: cubren los operadores && y ||).
const solrMetaChars = `\+-&|!(){}[]^"~*?:/`

// escapeSolrQuery escapa los metacaracteres del input del usuario para que
// siempre sea un conjunto de términos literales (E2).
func escapeSolrQuery(input string) string {
	var escaped strings.Builder
	escaped.Grow(len(input) * 2)
	for _, r := range input {
		if strings.ContainsRune(solrMetaChars, r) {
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(r)
	}
	return escaped.String()
}

// buildSearchQuery arma el request JSON de búsqueda (E2): edismax sobre
// name/description con el input del usuario como parámetro dereferenciado
// ($qq) — nunca interpolado en la query — y limit/offset reales (rows/start:
// antes la paginación se ignoraba). Query vacía = match-all (antes: 500 de
// sintaxis).
func buildSearchQuery(query string, limit int, offset int) *solr.Query {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return solr.NewQuery("*:*").Limit(limit).Offset(offset)
	}
	return solr.NewQuery("{!edismax qf='name description' v=$qq}").
		Params(solr.M{"qq": escapeSolrQuery(trimmed)}).
		Limit(limit).
		Offset(offset)
}

// Funcion para buscar hoteles en Solr. Devuelve además el total de matches
// (numFound) para el meta del envelope (A5/RV22): la página sola no alcanza
// para que el cliente calcule cuántas páginas hay.
func (searchEngine Solr) Search(ctx context.Context, query string, limit int, offset int) ([]hotels.Hotel, int, error) {
	// Ejecuta la query en Solr (construcción segura E2, deadline R2)
	ctx, cancel := solrOpCtx(ctx)
	defer cancel()
	resp, err := searchEngine.Client.Query(ctx, searchEngine.Collection, buildSearchQuery(query, limit, offset))
	if err != nil {
		return nil, 0, fmt.Errorf("error executing search query: %w", err)
	}
	if resp.Error != nil {
		return nil, 0, fmt.Errorf("failed to execute search query: %v", resp.Error)
	}

	// Itera sobre los documentos de la respuesta y los convierte en hoteles
	var hotelsList []hotels.Hotel
	for _, doc := range resp.Response.Documents {
		// Crea un slice de strings para los amenities
		var amenities []string
		var images []string

		// Extrae los amenities del documento y los agrega al slice
		if amenitiesData, ok := doc["amenities"].([]interface{}); ok {
			for _, amenity := range amenitiesData {
				if amenityStr, ok := amenity.(string); ok {
					amenities = append(amenities, amenityStr)
				}
			}
		}
		if imagesData, ok := doc["images"].([]interface{}); ok {
			for _, image := range imagesData {
				if imageStr, ok := image.(string); ok {
					images = append(images, imageStr)
				}
			}
		}

		// Lo convierte en un objeto de tipo Hotel y lo agrega a la lista
		hotel := hotels.Hotel{
			ID:            getStringField(doc, "id"),
			Name:          getStringField(doc, "name"),
			Description:   getStringField(doc, "description"),
			Address:       getStringField(doc, "address"),
			City:          getStringField(doc, "city"),
			State:         getStringField(doc, "state"),
			Country:       getStringField(doc, "country"),
			Phone:         getStringField(doc, "phone"),
			Email:         getStringField(doc, "email"),
			PricePerNight: getFloatField(doc, "price_per_night"),
			AvaiableRooms: int(getFloatField(doc, "avaiable_rooms")),
			CheckInTime:   getStringField(doc, "check_in_time"),
			CheckOutTime:  getStringField(doc, "check_out_time"),
			Rating:        getFloatField(doc, "rating"),
			Amenities:     amenities,
			Images:        images,
		}
		// Agrega el hotel a la lista
		hotelsList = append(hotelsList, hotel)
	}

	// Devuelve la lista de hoteles y el total de matches del índice
	return hotelsList, resp.Response.NumFound, nil
}

// Funcion auxiliar para obtener campos de tipo string de un documento.
// (check_in_time/check_out_time también salen por acá desde RV21: son "HH:mm"
// planos; el getTimeField que parseaba pdate/RFC3339 —E6— quedó obsoleto.)
func getStringField(doc map[string]interface{}, field string) string {
	if val, ok := doc[field].(string); ok {
		return val
	}
	if val, ok := doc[field].([]interface{}); ok && len(val) > 0 {
		if strVal, ok := val[0].(string); ok {
			return strVal
		}
	}
	return ""
}

// Funcion auxiliar para obtener campos de tipo float de un documento
func getFloatField(doc map[string]interface{}, field string) float64 {
	if val, ok := doc[field].(float64); ok {
		return val
	}
	if val, ok := doc[field].([]interface{}); ok && len(val) > 0 {
		if floatVal, ok := val[0].(float64); ok {
			return floatVal
		}
	}
	// Devuelve 0.0 si no se encuentra el campo
	return 0.0
}
