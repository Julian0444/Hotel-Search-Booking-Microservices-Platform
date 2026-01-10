package hotels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"search-api/dao/hotels"
	"time"

	"github.com/stevenferrer/solr-go"
)

type SolrConfig struct {
	Host       string // Solr host
	Port       string // Solr port
	Collection string // Solr collection name
}

type Solr struct {
	Client     *solr.JSONClient
	Collection string
}

// Funcion para crear una nueva conexion a Solr
func NewSolr(config SolrConfig) Solr {
	// Construimos la URL base para la conexion a Solr
	baseURL := fmt.Sprintf("http://%s:%s", config.Host, config.Port)
	// Creamos un nuevo cliente JSON para Solr
	client := solr.NewJSONClient(baseURL)

	// Devuelve una nueva instancia de Solr
	return Solr{
		Client:     client,
		Collection: config.Collection,
	}
}

// Index crea un nuevo documento de hotel en la coleccion de Solr
func (searchEngine Solr) Index(ctx context.Context, hotel hotels.Hotel) (string, error) {
	// Prepara el documento para Solr
	doc := map[string]interface{}{
		"id":        hotel.ID,
		"name":      hotel.Name,
		"description": hotel.Description,
		"address":   hotel.Address,
		"city":      hotel.City,
		"state":     hotel.State,
		"country":   hotel.Country,
		"phone":     hotel.Phone,
		"email":     hotel.Email,
		"price_per_night": hotel.PricePerNight,
		"avaiable_rooms": hotel.AvaiableRooms,
		"check_in_time": hotel.CheckInTime,
		"check_out_time": hotel.CheckOutTime,
		"rating":    hotel.Rating,
		"amenities": hotel.Amenities,
		"images":    hotel.Images,
	}

	// Prepara el request de indexacion
	indexRequest := map[string]interface{}{
		"add": []interface{}{doc}, // Usa "add" con una lista de documentos para indexar varios a la vez 
	}

	// Indexa el documento en Solr (Lo pasa a JSON)
	body, err := json.Marshal(indexRequest)
	if err != nil {
		return "", fmt.Errorf("error marshaling hotel document: %w", err)
	}

	// Manda el request de indexacion usando el metodo Update
	resp, err := searchEngine.Client.Update(ctx, searchEngine.Collection, solr.JSON, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error indexing hotel: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("failed to index hotel: %v", resp.Error)
	}

	// Commitea los cambios
	if err := searchEngine.Client.Commit(ctx, searchEngine.Collection); err != nil {
		return "", fmt.Errorf("error committing changes to Solr: %w", err)
	}

	// Si todo sale bien, devuelve el ID del hotel
	return hotel.ID, nil
}

// Funcion para actualizar un documento de hotel en la coleccion de Solr
func (searchEngine Solr) Update(ctx context.Context, hotel hotels.Hotel) error {
	// Prepara el documento para Solr
	doc := map[string]interface{}{
		"id":        hotel.ID,
		"name":      hotel.Name,
		"description": hotel.Description,
		"address":   hotel.Address,
		"city":      hotel.City,
		"state":     hotel.State,
		"country":   hotel.Country,
		"phone":     hotel.Phone,
		"email":     hotel.Email,
		"price_per_night": hotel.PricePerNight,
		"rating":    hotel.Rating,
		"avaiable_rooms": hotel.AvaiableRooms,
		"check_in_time": hotel.CheckInTime,
		"check_out_time": hotel.CheckOutTime,
		"amenities": hotel.Amenities,
		"images":    hotel.Images,
	}

	// Prepara el request de actualizacion
	updateRequest := map[string]interface{}{
		"add": []interface{}{doc}, // Use "add" with a list of documents
	}

	// Hace el update del documento en Solr (Lo pasa a JSON)
	body, err := json.Marshal(updateRequest)
	if err != nil {
		return fmt.Errorf("error marshaling hotel document: %w", err)
	}

	// Ejecuta el request de actualizacion usando el metodo Update
	resp, err := searchEngine.Client.Update(ctx, searchEngine.Collection, solr.JSON, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error updating hotel: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("failed to update hotel: %v", resp.Error)
	}

	// Hace commit de los cambios 
	if err := searchEngine.Client.Commit(ctx, searchEngine.Collection); err != nil {
		return fmt.Errorf("error committing changes to Solr: %w", err)
	}

	return nil
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

	// Ejecuta el request de borrado usando el metodo Update
	//La diferencia entre el metodo Update y Delete es que el metodo Update permite borrar varios documentos a la vez
	resp, err := searchEngine.Client.Update(ctx, searchEngine.Collection, solr.JSON, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error deleting hotel: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("failed to index hotel: %v", resp.Error)
	}

	// Hace commit de los cambios
	if err := searchEngine.Client.Commit(ctx, searchEngine.Collection); err != nil {
		return fmt.Errorf("error committing changes to Solr: %w", err)
	}

	return nil
}


// Funcion para buscar hoteles en Solr
func (searchEngine Solr) Search(ctx context.Context, query string, limit int, offset int) ([]hotels.Hotel, error) {
	// Construye la query de busqueda
	solrQuery := fmt.Sprintf("q=(name:%s OR description:%s)&rows=%d&start=%d", query, query, limit, offset)

	// Ejecuta la query en Solr
	resp, err := searchEngine.Client.Query(ctx, searchEngine.Collection, solr.NewQuery(solrQuery))
	if err != nil {
		return nil, fmt.Errorf("error executing search query: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("failed to execute search query: %v", resp.Error)
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
			ID:        getStringField(doc, "id"),
			Name:      getStringField(doc, "name"),
			Description: getStringField(doc, "description"),
			Address:   getStringField(doc, "address"),
			City:      getStringField(doc, "city"),
			State:     getStringField(doc, "state"),
			Country:   getStringField(doc, "country"),
			Phone:     getStringField(doc, "phone"),
			Email:     getStringField(doc, "email"),
			PricePerNight: getFloatField(doc, "price_per_night"),
			AvaiableRooms: int(getFloatField(doc, "avaiable_rooms")),
			CheckInTime: getTimeField(doc, "check_in_time"),
			CheckOutTime: getTimeField(doc, "check_out_time"),
			Rating:    getFloatField(doc, "rating"),
			Amenities: amenities,
			Images: images,
		}
		// Agrega el hotel a la lista
		hotelsList = append(hotelsList, hotel)
	}

	// Devuelve la lista de hoteles
	return hotelsList, nil
}


// Funcion auxiliar para obtener campos de tipo time de un documento
func getTimeField(doc map[string]interface{}, field string) time.Time {
	if val, ok := doc[field].(time.Time); ok {
		return val
	}
	return time.Time{}
}


// Funcion auxiliar para obtener campos de tipo string de un documento
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
