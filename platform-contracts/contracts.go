// Package contracts es la única fuente de verdad de los tipos que cruzan
// servicios: el payload HTTP de Hotel y el evento HotelNew que viaja por
// RabbitMQ (hotels-api lo produce, search-api lo consume).
//
// hotels-api y search-api consumen estos tipos vía type-alias en sus
// paquetes de dominio, así los call-sites no cambian.
package contracts

// Hotel es la representación wire (JSON) de un hotel.
// NOTA: available_rooms arrastró un typo histórico en su nombre hasta el
// plan 11: C11 lo renombró como cambio coordinado atómico (contrato
// JSON/BSON, Solr, frontend, seeds y goldens en un único cambio; migración
// de datos en hotels-api/seed/rename-available-rooms.js). Breaking dentro
// de /api/v1, aceptado por no haber consumidores externos.
//
// CheckInTime/CheckOutTime son horas del día "HH:mm" (RV21): modelarlas como
// time.Time obligaba a los clientes a mandar un timestamp RFC3339 completo
// para expresar "a partir de las 14:00" (el form admin mandaba "14:00" y
// recibía 400 siempre).
type Hotel struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Address        string   `json:"address"`
	City           string   `json:"city"`
	State          string   `json:"state"`
	Country        string   `json:"country"`
	Phone          string   `json:"phone"`
	Email          string   `json:"email"`
	PricePerNight  float64  `json:"price_per_night"`
	Rating         float64  `json:"rating"`
	AvailableRooms int      `json:"available_rooms"`
	CheckInTime    string   `json:"check_in_time"`  // "HH:mm"
	CheckOutTime   string   `json:"check_out_time"` // "HH:mm"
	Amenities      []string `json:"amenities"`
	Images         []string `json:"images"`
}

// Convención de identidad entre servicios (A7): el `user_id` canónico del
// contrato es SIEMPRE string — es lo que viaja en el claim del JWT y lo que
// usa hotels-api en las reservas. users-api mantiene su PK int64 interna pero
// serializa `id`/`user_id` como string en sus DTOs de respuesta.

// HotelNew es el evento publicado en la cola hotels-news.
// Operation: CREATE | UPDATE | DELETE.
type HotelNew struct {
	Operation string `json:"operation"`
	HotelID   string `json:"hotel_id"`
}
