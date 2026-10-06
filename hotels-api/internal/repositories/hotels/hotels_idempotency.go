package hotels

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Retained for at least 24h (Mongo TTL cleanup is asynchronous). The contract
// guarantees replay during that window, not indefinite deduplication.
// There is no persisted in-flight state: the key commits with its reservation.
type IdempotencyRecord struct {
	Key           string    `bson:"key"`
	UserID        string    `bson:"user_id"`
	Fingerprint   string    `bson:"payload_hash"`
	ReservationID string    `bson:"reservation_id"`
	CreatedAt     time.Time `bson:"created_at"`
}

func reservationFingerprint(r hotelsDAO.Reservation) string {
	payload := struct {
		HotelID   string `json:"hotel_id"`
		UserID    string `json:"user_id"`
		CheckIn   string `json:"check_in"`
		CheckOut  string `json:"check_out"`
		NumRooms  int    `json:"num_rooms"`
		NumGuests int    `json:"num_guests"`
	}{r.HotelID, r.UserID, r.CheckIn.Format(hotelsDomain.DateFormat), r.CheckOut.Format(hotelsDomain.DateFormat), r.NumRooms, r.NumGuests}
	data, _ := json.Marshal(payload)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
func (repository Mongo) idempotencyCollection() *mongo.Collection {
	return repository.client.Database(repository.database).Collection(repository.collection_idempotency)
}
func (repository Mongo) lookupIdempotency(ctx context.Context, key, userID, fingerprint string) (string, error) {
	var existing IdempotencyRecord
	err := repository.idempotencyCollection().FindOne(ctx, bson.M{"key": key, "user_id": userID}).Decode(&existing)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if existing.Fingerprint == "" || existing.ReservationID == "" {
		return "", hotelsDomain.ErrLegacyIdempotency
	}
	if existing.Fingerprint != fingerprint {
		return "", hotelsDomain.ErrIdempotencyConflict
	}
	return existing.ReservationID, nil
}
