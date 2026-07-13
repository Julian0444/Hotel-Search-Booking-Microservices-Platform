package hotels

import (
	contracts "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/platform-contracts"
)

// Hotel y HotelNew viven en platform-contracts (única fuente de verdad del
// contrato compartido con search-api); los alias mantienen los call-sites
// intactos.
type Hotel = contracts.Hotel

type HotelNew = contracts.HotelNew
