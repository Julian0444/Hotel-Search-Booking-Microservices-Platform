package hotels

import (
	"errors"

	contracts "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/platform-contracts"
)

// Hotel y HotelNew viven en platform-contracts (única fuente de verdad del
// contrato compartido con search-api); los alias mantienen los call-sites
// intactos.
type Hotel = contracts.Hotel

type HotelNew = contracts.HotelNew

// ErrHotelNotFound tipifica "el hotel no existe" (RV14): el controller lo
// mapea a 404 y cualquier otro error de GetHotelByID queda como 500 — el
// consumidor (search-api) usa esa distinción para decidir descartar vs
// reintentar/DLQ un evento.
var ErrHotelNotFound = errors.New("hotel not found")
