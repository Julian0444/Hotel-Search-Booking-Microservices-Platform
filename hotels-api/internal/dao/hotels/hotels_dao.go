package hotels

import "time"

type Hotel struct {
	ID            string    `bson:"_id,omitempty"`
	Name          string    `bson:"name"`
	Description   string    `bson:"description"`
	Address       string    `bson:"address"`
	City          string    `bson:"city"`
	State         string    `bson:"state"`
	Country       string    `bson:"country"`
	Phone         string    `bson:"phone"`
	Email         string    `bson:"email"`
	PricePerNight float64   `bson:"price_per_night"`
	Rating        float64   `bson:"rating"`
	AvaiableRooms int       `bson:"avaiable_rooms"`
	CheckInTime   time.Time `bson:"check_in_time"`
	CheckOutTime  time.Time `bson:"check_out_time"`
	Amenities     []string  `bson:"amenities"`
	Images        []string  `bson:"images"`
}

type Reservation struct {
	ID        string    `bson:"_id,omitempty"`
	HotelName string    `bson:"hotel_name"`
	HotelID   string    `bson:"hotel_id"`
	UserID    string    `bson:"user_id"`
	CheckIn   time.Time `bson:"check_in"`
	CheckOut  time.Time `bson:"check_out"`
}
