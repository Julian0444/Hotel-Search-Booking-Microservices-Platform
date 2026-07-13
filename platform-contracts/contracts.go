// Package contracts es la única fuente de verdad de los tipos que cruzan
// servicios: el payload HTTP de Hotel y el evento HotelNew que viaja por
// RabbitMQ (hotels-api lo produce, search-api lo consume).
//
// hotels-api y search-api consumen estos tipos vía type-alias en sus
// paquetes de dominio, así los call-sites no cambian.
package contracts

import "time"

// Hotel es la representación wire (JSON) de un hotel.
// NOTA: el tag "avaiable_rooms" (typo) es deliberado — está propagado a
// Mongo, Solr y el frontend, y se renombra como cambio coordinado atómico
// en C11 (plan 11). No corregirlo acá de forma aislada.
type Hotel struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Country       string    `json:"country"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	PricePerNight float64   `json:"price_per_night"`
	Rating        float64   `json:"rating"`
	AvaiableRooms int       `json:"avaiable_rooms"`
	CheckInTime   time.Time `json:"check_in_time"`
	CheckOutTime  time.Time `json:"check_out_time"`
	Amenities     []string  `json:"amenities"`
	Images        []string  `json:"images"`
}

// HotelNew es el evento publicado en la cola hotels-news.
// Operation: CREATE | UPDATE | DELETE.
type HotelNew struct {
	Operation string `json:"operation"`
	HotelID   string `json:"hotel_id"`
}
