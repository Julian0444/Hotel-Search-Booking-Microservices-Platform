package hotels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// IdempotencyRecord es el registro persistido de un request idempotente (A3):
// la clave compuesta (key, user_id) tiene índice único y el TTL de created_at
// lo expira a las 24h. Done distingue "en vuelo" de "completado": un replay
// mientras el original sigue en vuelo es 409 request_in_flight.
type IdempotencyRecord struct {
	Key       string    `bson:"key"`
	UserID    string    `bson:"user_id"`
	Done      bool      `bson:"done"`
	Status    int       `bson:"status"`
	Body      []byte    `bson:"body"`
	CreatedAt time.Time `bson:"created_at"`
}

func (repository Mongo) idempotencyCollection() *mongo.Collection {
	return repository.client.Database(repository.database).Collection(repository.collection_idempotency)
}

// ReserveIdempotencyKey intenta registrar (key, userID) como "en vuelo".
// Devuelve created=true si este request es el primero; si la clave ya existía
// (replay o request concurrente), created=false y el registro existente.
func (repository Mongo) ReserveIdempotencyKey(ctx context.Context, key, userID string) (bool, IdempotencyRecord, error) {
	record := IdempotencyRecord{
		Key:       key,
		UserID:    userID,
		Done:      false,
		CreatedAt: time.Now().UTC(),
	}
	_, err := repository.idempotencyCollection().InsertOne(ctx, record)
	if err == nil {
		return true, record, nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return false, IdempotencyRecord{}, fmt.Errorf("error reserving idempotency key: %w", err)
	}

	// La clave ya existe: devolver el registro guardado (replay concurrente:
	// dos requests con la misma key en vuelo → el segundo insert colisiona acá)
	var existing IdempotencyRecord
	err = repository.idempotencyCollection().
		FindOne(ctx, bson.M{"key": key, "user_id": userID}).
		Decode(&existing)
	if err != nil {
		// Carrera con el TTL/release: el doc desapareció entre el insert y el
		// find. Tratarlo como error — el cliente reintenta con la misma key.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, IdempotencyRecord{}, fmt.Errorf("idempotency key vanished between insert and lookup")
		}
		return false, IdempotencyRecord{}, fmt.Errorf("error fetching idempotency record: %w", err)
	}
	return false, existing, nil
}

// CompleteIdempotencyKey guarda status+body de la primera respuesta: los
// replays posteriores devuelven exactamente esto.
func (repository Mongo) CompleteIdempotencyKey(ctx context.Context, key, userID string, status int, body []byte) error {
	_, err := repository.idempotencyCollection().UpdateOne(ctx,
		bson.M{"key": key, "user_id": userID},
		bson.M{"$set": bson.M{"done": true, "status": status, "body": body}})
	if err != nil {
		return fmt.Errorf("error completing idempotency key: %w", err)
	}
	return nil
}

// ReleaseIdempotencyKey borra la reserva de la clave: se usa cuando el request
// original murió con un 5xx — un fallo transitorio del server no debe dejar la
// key quemada 24h impidiendo el reintento legítimo del cliente.
func (repository Mongo) ReleaseIdempotencyKey(ctx context.Context, key, userID string) error {
	_, err := repository.idempotencyCollection().DeleteOne(ctx, bson.M{"key": key, "user_id": userID})
	if err != nil {
		return fmt.Errorf("error releasing idempotency key: %w", err)
	}
	return nil
}
