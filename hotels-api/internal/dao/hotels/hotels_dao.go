package hotels

import "time"

// CheckInTime/CheckOutTime se persisten como string "HH:mm" (RV21), igual que
// en el contrato wire. Los volúmenes de Mongo anteriores las tienen como Date:
// migrarlas con la receta de plans/HANDOFF.md antes de levantar esta versión.
type Hotel struct {
	ID            string   `bson:"_id,omitempty"`
	Name          string   `bson:"name"`
	Description   string   `bson:"description"`
	Address       string   `bson:"address"`
	City          string   `bson:"city"`
	State         string   `bson:"state"`
	Country       string   `bson:"country"`
	Phone         string   `bson:"phone"`
	Email         string   `bson:"email"`
	PricePerNight float64  `bson:"price_per_night"`
	Rating        float64  `bson:"rating"`
	AvaiableRooms int      `bson:"avaiable_rooms"`
	CheckInTime   string   `bson:"check_in_time"`
	CheckOutTime  string   `bson:"check_out_time"`
	Amenities     []string `bson:"amenities"`
	Images        []string `bson:"images"`
}

// Estados del lifecycle de una reserva (DM1). Cancelar es un soft-delete:
// la reserva queda con status=cancelled y sus noches se liberan del inventario.
const (
	StatusConfirmed = "confirmed"
	StatusCancelled = "cancelled"
)

type Reservation struct {
	ID          string     `bson:"_id,omitempty"`
	HotelName   string     `bson:"hotel_name"`
	HotelID     string     `bson:"hotel_id"`
	UserID      string     `bson:"user_id"`
	CheckIn     time.Time  `bson:"check_in"`
	CheckOut    time.Time  `bson:"check_out"`
	Status      string     `bson:"status"`
	NumRooms    int        `bson:"num_rooms"`
	NumGuests   int        `bson:"num_guests"`
	TotalPrice  int64      `bson:"total_price"` // centavos: nunca float para dinero
	Currency    string     `bson:"currency"`
	CreatedAt   time.Time  `bson:"created_at"`
	CancelledAt *time.Time `bson:"cancelled_at,omitempty"`
}

// Inventory es el contador atómico de ocupación por hotel-noche (D1): el par
// {hotel_id, date} tiene índice único y se reclama con findOneAndUpdate.
// Date usa el formato canónico "2006-01-02" (una entrada por noche, checkout
// excluido).
type Inventory struct {
	HotelID  string `bson:"hotel_id"`
	Date     string `bson:"date"`
	Booked   int    `bson:"booked"`
	Capacity int    `bson:"capacity"`
}
