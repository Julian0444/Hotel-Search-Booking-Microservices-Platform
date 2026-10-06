package hotels

import (
	"errors"

	contracts "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/platform-contracts"
)

// Hotel y HotelNew viven en platform-contracts (única fuente de verdad del
// contrato compartido con hotels-api); los alias mantienen los call-sites
// intactos.
type Hotel = contracts.Hotel

type HotelNew = contracts.HotelNew

// ErrHotelNotFound tipifica el 404 de hotels-api (RV14): para el consumer un
// hotel inexistente se descarta (reintentar jamás lo resolvería); cualquier
// otro fallo del fetch se reintenta con demora sin descartar el evento válido.
var ErrHotelNotFound = errors.New("hotel not found in hotels-api")

// ErrInvalidEvent identifica mensajes que ningún reintento puede corregir.
var ErrInvalidEvent = errors.New("invalid hotel event")
