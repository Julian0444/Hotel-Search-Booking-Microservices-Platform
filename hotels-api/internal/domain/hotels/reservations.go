package hotels

import "time"

type Reservation struct {
	ID       string    `json:"id"`
	HotelID  string    `json:"hotel_id"`
	HotelName string   `json:"hotel_name"`
	UserID   string    `json:"user_id"`
	CheckIn  time.Time `json:"check_in"`
	CheckOut time.Time `json:"check_out"`
}

type ReservationNew struct {
	Operation string `json:"operation"`
	ReservationID   string `json:"reservation_id"`
}