package hotels

import "time"

type Hotel struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Address string `json:"address"`
	City string `json:"city"`
	State string `json:"state"`
	Country string `json:"country"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	PricePerNight float64 `json:"price_per_night"`
	Rating float64 `json:"rating"`
	AvaiableRooms int `json:"avaiable_rooms"`
	CheckInTime time.Time `json:"check_in_time"`
	CheckOutTime time.Time `json:"check_out_time"`
	Amenities []string `json:"amenities"`
	Images []string `json:"images"`
}

type HotelNew struct {
	Operation string `json:"operation"`
	HotelID   string `json:"hotel_id"`
}
