package hotels

// CheckInTime/CheckOutTime son horas del día "HH:mm" (RV21), igual que en el
// contrato wire y en el schema de Solr (fieldType string).
type Hotel struct {
	ID             string   `bson:"_id,omitempty"`
	Name           string   `bson:"name"`
	Description    string   `bson:"description"`
	Address        string   `bson:"address"`
	City           string   `bson:"city"`
	State          string   `bson:"state"`
	Country        string   `bson:"country"`
	Phone          string   `bson:"phone"`
	Email          string   `bson:"email"`
	PricePerNight  float64  `bson:"price_per_night"`
	Rating         float64  `bson:"rating"`
	AvailableRooms int      `bson:"available_rooms"`
	CheckInTime    string   `bson:"check_in_time"`
	CheckOutTime   string   `bson:"check_out_time"`
	Amenities      []string `bson:"amenities"`
	Images         []string `bson:"images"`
}
