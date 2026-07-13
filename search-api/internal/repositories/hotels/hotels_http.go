package hotels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/domain/hotels"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api/internal/utils"
)

type HTTPConfig struct {
	Host string
	Port string
}

type HTTP struct {
	baseURL func(hotelID string) string
}

func NewHTTP(config HTTPConfig) HTTP {
	return HTTP{
		//Aca creamos la funcion para que vaya a buscar el hotel por id
		baseURL: func(hotelID string) string {
			return fmt.Sprintf("http://%s:%s/hotels/%s", config.Host, config.Port, hotelID)
		},
	}
}

func (repository HTTP) GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error) {
	// NewRequestWithContext: propaga cancelación y el X-Request-ID del mensaje
	// consumido para correlacionar el hop search→hotels (O1). El client con
	// timeout completo lo agrega el plan 06 (E4).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repository.baseURL(id), nil)
	if err != nil {
		return hotelsDomain.Hotel{}, fmt.Errorf("error building request for hotel (%s): %w", id, err)
	}
	if requestID := utils.RequestIDFromContext(ctx); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return hotelsDomain.Hotel{}, fmt.Errorf("error fetching hotel (%s): %w", id, err)
	}
	// Defer hace que se ejecute la funcion Close() cuando la funcion GetHotelByID termine
	//La parte de body.Close() es para cerrar el body de la respuesta
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return hotelsDomain.Hotel{}, fmt.Errorf("failed to fetch hotel (%s): received status code %d", id, resp.StatusCode)
	}

	// Lee el body de la respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return hotelsDomain.Hotel{}, fmt.Errorf("error reading response body for hotel (%s): %w", id, err)
	}

	// Unmarshal the hotel details into the hotel struct
	var hotel hotelsDomain.Hotel
	if err := json.Unmarshal(body, &hotel); err != nil {
		return hotelsDomain.Hotel{}, fmt.Errorf("error unmarshaling hotel data (%s): %w", id, err)
	}

	return hotel, nil
}
