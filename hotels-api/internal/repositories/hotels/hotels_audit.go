package hotels

import (
	"context"
	"fmt"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AuditInventory validates a snapshot at startup. Legacy inconsistencies are
// reported, never repaired by guessing counters or deleting local data.
// Run seed/audit-reservations.js before upgrading an existing database.
func (repository Mongo) AuditInventory(ctx context.Context) error {
	_, err := repository.transaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		db := repository.client.Database(repository.database)
		hotelsCursor, err := db.Collection(repository.collection_hotel).Find(sc, bson.M{})
		if err != nil {
			return nil, err
		}
		var hotels []hotelsDAO.Hotel
		if err := hotelsCursor.All(sc, &hotels); err != nil {
			return nil, err
		}
		capacities := make(map[string]int, len(hotels))
		for _, hotel := range hotels {
			capacities[hotel.ID] = hotel.AvailableRooms
		}
		reservationsCursor, err := db.Collection(repository.collection_reservation).Find(sc, bson.M{})
		if err != nil {
			return nil, err
		}
		var reservations []hotelsDAO.Reservation
		if err := reservationsCursor.All(sc, &reservations); err != nil {
			return nil, err
		}
		expected := map[string]int{}
		for _, reservation := range reservations {
			if _, exists := capacities[reservation.HotelID]; !exists {
				return nil, fmt.Errorf("reservation %s refers to missing hotel: %w", reservation.ID, hotelsDomain.ErrInventoryInconsistent)
			}
			if reservation.Status == hotelsDAO.StatusCancelled {
				continue
			}
			if reservation.Status != hotelsDAO.StatusConfirmed || reservation.NumRooms < 1 || !reservation.CheckOut.After(reservation.CheckIn) {
				return nil, fmt.Errorf("reservation %s has invalid persisted state: %w", reservation.ID, hotelsDomain.ErrInventoryInconsistent)
			}
			for _, night := range nightsBetween(reservation.CheckIn, reservation.CheckOut) {
				expected[reservation.HotelID+"/"+night] += reservation.NumRooms
			}
		}
		inventoryCursor, err := db.Collection(repository.collection_inventory).Find(sc, bson.M{})
		if err != nil {
			return nil, err
		}
		var inventory []hotelsDAO.Inventory
		if err := inventoryCursor.All(sc, &inventory); err != nil {
			return nil, err
		}
		for _, entry := range inventory {
			key := entry.HotelID + "/" + entry.Date
			capacity, exists := capacities[entry.HotelID]
			if !exists || entry.Booked < 0 || entry.Booked != expected[key] || entry.Booked > capacity {
				return nil, fmt.Errorf("inventory %s has booked=%d expected=%d capacity=%d: %w", key, entry.Booked, expected[key], capacity, hotelsDomain.ErrInventoryInconsistent)
			}
			delete(expected, key)
		}
		for key, count := range expected {
			if count > 0 {
				return nil, fmt.Errorf("missing inventory %s expected=%d: %w", key, count, hotelsDomain.ErrInventoryInconsistent)
			}
		}
		count, err := db.Collection(repository.collection_idempotency).CountDocuments(sc, bson.M{"$or": bson.A{bson.M{"payload_hash": bson.M{"$exists": false}}, bson.M{"reservation_id": bson.M{"$exists": false}}}})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%d old idempotency records: %w", count, hotelsDomain.ErrLegacyIdempotency)
		}
		return nil, nil
	})
	return err
}
