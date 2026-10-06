package hotels

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type MongoConfig struct {
	ReplicaSet              string
	Direct                  bool
	Host                    string
	Port                    string
	Username                string
	Password                string
	Database                string
	Collection_hotels       string
	Collection_reservations string
	Collection_inventory    string
	Collection_idempotency  string
}

type Mongo struct {
	client                 *mongo.Client
	database               string
	collection_hotel       string
	collection_reservation string
	collection_inventory   string
	collection_idempotency string
}

const (
	connectionURI = "mongodb://%s:%s"

	// mongoOpTimeout acota cada operación contra Mongo (R2): deriva del ctx
	// del request, así una DB lenta falla en 3s en vez de colgar el handler.
	mongoOpTimeout = 3 * time.Second
	// socketTimeout es la red de contención del driver para sockets colgados
	// (R2); holgado a propósito: el deadline fino lo pone opCtx por operación.
	socketTimeout = 10 * time.Second

	// availabilityMaxConcurrency limita el fan-out del batch de disponibilidad
	// (R4, bulkhead): antes cada ID lanzaba su goroutine sin tope y un body
	// grande multiplicaba queries concurrentes contra Mongo.
	availabilityMaxConcurrency = 8
)

// opCtx deriva el deadline por operación de Mongo (R2).
func opCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, mongoOpTimeout)
}

// Crea una nueva instancia de Mongo
func NewMongo(config MongoConfig) Mongo {
	credentials := options.Credential{
		Username: config.Username,
		Password: config.Password,
	}

	//Crea el contexto (acotado: si Mongo no está, fallar al arranque y no después)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	//Crea la URI de conexion
	uri := fmt.Sprintf(connectionURI, config.Host, config.Port)
	//Crea la configuracion de conexion con pool y timeout de driver (DB3)
	cfg := options.Client().ApplyURI(uri).SetReplicaSet(config.ReplicaSet).SetDirect(config.Direct).
		SetMaxPoolSize(50).
		SetServerSelectionTimeout(5 * time.Second).
		SetSocketTimeout(socketTimeout)

	if config.Username != "" {
		cfg.SetAuth(credentials)
	}
	//Crea la conexion a MongoDB
	client, err := mongo.Connect(ctx, cfg)
	if err != nil {
		log.Panicf("error connecting to mongo DB: %v", err)
	}

	// Ping fail-fast: mongo.Connect es lazy y no valida la conexion
	if err := client.Ping(ctx, nil); err != nil {
		log.Panicf("mongo unreachable: %v", err)
	}

	repository := Mongo{
		client:                 client,
		database:               config.Database,
		collection_hotel:       config.Collection_hotels,
		collection_reservation: config.Collection_reservations,
		collection_inventory:   config.Collection_inventory,
		collection_idempotency: config.Collection_idempotency,
	}

	// Índices secundarios de reservas (DB2)
	if err := repository.EnsureIndexes(ctx); err != nil {
		log.Panicf("error ensuring mongo indexes: %v", err)
	}

	var hello struct {
		SetName string `bson:"setName"`
	}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil || hello.SetName == "" {
		log.Panicf("Mongo replica set required for reservation transactions: set=%q err=%v", hello.SetName, err)
	}
	if err := repository.AuditInventory(ctx); err != nil {
		log.Panicf("existing reservation data requires inspection: %v", err)
	}
	return repository
}

// Ping verifica la conectividad con Mongo (lo usa el /readyz, O3).
func (repository Mongo) Ping(ctx context.Context) error {
	return repository.client.Ping(ctx, nil)
}

// Disconnect cierra el pool de conexiones a Mongo (graceful shutdown, C12).
func (repository Mongo) Disconnect(ctx context.Context) error {
	return repository.client.Disconnect(ctx)
}

// EnsureIndexes crea los índices de reservas e inventario. Es idempotente:
// CreateMany con la misma spec es un no-op en Mongo. Los nombres de campo son
// los bson tags reales de hotels_dao.go (check_in/check_out, no *_time).
func (repository Mongo) EnsureIndexes(ctx context.Context) error {
	collection := repository.client.Database(repository.database).Collection(repository.collection_reservation)
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "hotel_id", Value: 1}, {Key: "check_in", Value: 1}, {Key: "check_out", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("error creating reservation indexes: %w", err)
	}

	// Índice ÚNICO {hotel_id, date} del inventario: es lo que hace atómico el
	// claim por noche (D1) — dos upserts concurrentes de la misma noche
	// colisionan acá en vez de duplicar el contador.
	inventory := repository.client.Database(repository.database).Collection(repository.collection_inventory)
	_, err = inventory.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "hotel_id", Value: 1}, {Key: "date", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("error creating inventory index: %w", err)
	}

	// Índices de idempotencia (A3): {key, user_id} único hace atómica la
	// detección de replays (el segundo insert colisiona acá), y el índice TTL
	// hace que Mongo borre solo los registros después de 24h.
	idempotency := repository.client.Database(repository.database).Collection(repository.collection_idempotency)
	_, err = idempotency.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32((24 * time.Hour).Seconds())),
		},
	})
	if err != nil {
		return fmt.Errorf("error creating idempotency indexes: %w", err)
	}
	return nil
}

// Obtiene un hotel por su ID de MongoDB
func (repository Mongo) GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error) {

	//Crea el ObjectID de MongoDB a partir del ID para buscar el documento
	// Un ID que ni siquiera es un ObjectID válido no puede existir → not-found
	// tipado (RV14), no un error de infraestructura
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Hotel{}, fmt.Errorf("invalid hotel id %q: %w", id, hotelsDomain.ErrHotelNotFound)
	}

	// Buscar el documento en MongoDB por su ID (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result := repository.client.Database(repository.database).Collection(repository.collection_hotel).FindOne(ctx, bson.M{"_id": objectID})
	if result.Err() != nil {
		// Solo "no existe" es ErrHotelNotFound; cualquier otro fallo (Mongo
		// caído, timeout) se propaga como error real (RV14)
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return hotelsDAO.Hotel{}, fmt.Errorf("hotel %s: %w", id, hotelsDomain.ErrHotelNotFound)
		}
		return hotelsDAO.Hotel{}, fmt.Errorf("error finding document: %w", result.Err())
	}

	// Decodificar el resultado
	var hotelDAO hotelsDAO.Hotel
	if err := result.Decode(&hotelDAO); err != nil {
		return hotelsDAO.Hotel{}, fmt.Errorf("error decoding result: %w", err)
	}
	return hotelDAO, nil
}

// GetHotels lista hoteles paginados con orden estable por _id (E3): lo
// consume el backfill/reindex de search-api vía GET /hotels.
func (repository Mongo) GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDAO.Hotel, error) {
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).Find(ctx, bson.M{}, findPageOptions(limit, offset))
	if err != nil {
		return nil, fmt.Errorf("error finding hotels: %w", err)
	}

	var hotels []hotelsDAO.Hotel
	if err := result.All(ctx, &hotels); err != nil {
		return nil, fmt.Errorf("error decoding hotels: %w", err)
	}
	return hotels, nil
}

// CountHotels devuelve el total del catálogo para el envelope {data, total}
// de GET /hotels (E3).
func (repository Mongo) CountHotels(ctx context.Context) (int64, error) {
	ctx, cancel := opCtx(ctx)
	defer cancel()
	total, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("error counting hotels: %w", err)
	}
	return total, nil
}

// Crea un nuevo hotel en MongoDB
func (repository Mongo) Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	// The server owns identity. Never persist a client-supplied string _id.
	hotel.ID = ""
	// Insertar el documento en MongoDB (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).InsertOne(ctx, hotel)
	if err != nil {
		return "", fmt.Errorf("error creating document: %w", err)
	}

	// Saca el ObjectID del resultado de la insercion
	objectID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("error converting mongo ID to object ID")
	}

	// Regresa el ID del documento insertado
	return objectID.Hex(), nil
}

// transaction usa los reintentos del driver para TransientTransactionError y
// UnknownTransactionCommitResult. Nunca compensa un commit ambiguo. El ID y la
// clave de reserva viven fuera del callback y no hay efectos externos en él.
func (repository Mongo) transaction(ctx context.Context, fn func(mongo.SessionContext) (interface{}, error)) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	session, err := repository.client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(context.Background())
	return session.WithTransaction(ctx, fn, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
}

// lockHotel coordina TODAS las escrituras de capacidad, reservas y borrado.
// Una lectura snapshot por sí sola no evitaría write skew entre documentos.
func (repository Mongo) lockHotel(ctx mongo.SessionContext, id string) (hotelsDAO.Hotel, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Hotel{}, hotelsDomain.ErrHotelNotFound
	}
	var hotel hotelsDAO.Hotel
	err = repository.client.Database(repository.database).Collection(repository.collection_hotel).
		FindOneAndUpdate(ctx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"booking_version": 1}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&hotel)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return hotel, hotelsDomain.ErrHotelNotFound
	}
	return hotel, err
}

// PUT reemplaza todos los campos públicos, incluidos cero y listas vacías.
// Se conservan _id y booking_version internos. No se destruye historial.
func (repository Mongo) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	canonicalID, err := canonicalHotelID(hotel.ID)
	if err != nil {
		return err
	}
	hotel.ID = canonicalID
	_, err = repository.transaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if _, err := repository.lockHotel(sc, hotel.ID); err != nil {
			return nil, err
		}
		inventory := repository.client.Database(repository.database).Collection(repository.collection_inventory)
		count, err := inventory.CountDocuments(sc, bson.M{"hotel_id": hotel.ID, "booked": bson.M{"$gt": hotel.AvailableRooms}})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, hotelsDomain.ErrCapacityConflict
		}
		raw, err := bson.Marshal(hotel)
		if err != nil {
			return nil, err
		}
		var fields bson.M
		if err := bson.Unmarshal(raw, &fields); err != nil {
			return nil, err
		}
		delete(fields, "_id")
		id, _ := primitive.ObjectIDFromHex(hotel.ID)
		if _, err = repository.client.Database(repository.database).Collection(repository.collection_hotel).UpdateOne(sc, bson.M{"_id": id}, bson.M{"$set": fields}); err != nil {
			return nil, err
		}
		_, err = inventory.UpdateMany(sc, bson.M{"hotel_id": hotel.ID}, bson.M{"$set": bson.M{"capacity": hotel.AvailableRooms}})
		return nil, err
	})
	return err
}

// Borrar un hotel con historial se rechaza; las reservas nunca desaparecen
// detrás de un DELETE administrativo. Hotel e inventario se borran juntos.
func (repository Mongo) Delete(ctx context.Context, id string) error {
	canonicalID, err := canonicalHotelID(id)
	if err != nil {
		return err
	}
	id = canonicalID
	_, err = repository.transaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if _, err := repository.lockHotel(sc, id); err != nil {
			return nil, err
		}
		db := repository.client.Database(repository.database)
		count, err := db.Collection(repository.collection_reservation).CountDocuments(sc, bson.M{"hotel_id": id})
		if err != nil {
			return nil, err
		}
		if count != 0 {
			return nil, hotelsDomain.ErrHotelHasReservations
		}
		objectID, _ := primitive.ObjectIDFromHex(id)
		if _, err = db.Collection(repository.collection_hotel).DeleteOne(sc, bson.M{"_id": objectID}); err != nil {
			return nil, err
		}
		_, err = db.Collection(repository.collection_inventory).DeleteMany(sc, bson.M{"hotel_id": id})
		return nil, err
	})
	return err
}

// nightsBetween devuelve las noches ocupadas por el rango en formato canónico
// "2006-01-02": incluye check-in, excluye check-out (D4).
func nightsBetween(checkIn, checkOut time.Time) []string {
	var nights []string
	for current := normalizeDate(checkIn); current.Before(normalizeDate(checkOut)); current = current.AddDate(0, 0, 1) {
		nights = append(nights, current.Format("2006-01-02"))
	}
	return nights
}

// hotelCapacity lee la capacidad actual del hotel (available_rooms).
func (repository Mongo) hotelCapacity(ctx context.Context, hotelID string) (int, error) {
	objectID, err := primitive.ObjectIDFromHex(hotelID)
	if err != nil {
		return 0, fmt.Errorf("error converting hotel ID to object ID: %w", err)
	}

	var hotel struct {
		AvailableRooms int `bson:"available_rooms"`
	}
	ctx, cancel := opCtx(ctx)
	defer cancel()
	err = repository.client.Database(repository.database).Collection(repository.collection_hotel).
		FindOne(ctx, bson.M{"_id": objectID}, options.FindOne().SetProjection(bson.M{"available_rooms": 1, "_id": 0})).
		Decode(&hotel)
	if err != nil {
		return 0, fmt.Errorf("error finding hotel: %w", err)
	}
	return hotel.AvailableRooms, nil
}

// CreateReservation confirma reserva, noches e identidad del intento en una
// transacción. Un replay se resuelve antes de revalidar capacidad/precio/fecha.
func (repository Mongo) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	canonicalID, err := canonicalHotelID(reservation.HotelID)
	if err != nil {
		return "", err
	}
	reservation.HotelID = canonicalID
	operation := hotelsDomain.IdempotencyFromContext(ctx)
	fingerprint := reservationFingerprint(reservation)
	id := primitive.NewObjectID()
	result, err := repository.transaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if operation != nil && operation.Key != "" {
			existing, err := repository.lookupIdempotency(sc, operation.Key, reservation.UserID, fingerprint)
			if err != nil {
				return nil, err
			}
			if existing != "" {
				operation.Replayed = true
				return existing, nil
			}
		}
		hotel, err := repository.lockHotel(sc, reservation.HotelID)
		if err != nil {
			return nil, err
		}
		if math.IsNaN(hotel.PricePerNight) || math.IsInf(hotel.PricePerNight, 0) || hotel.PricePerNight < 0 || hotel.PricePerNight > 1000000 {
			return nil, hotelsDomain.ErrInvalidHotel
		}
		if reservation.NumRooms < 1 || reservation.NumRooms > hotel.AvailableRooms {
			return nil, hotelsDomain.ErrNoAvailability
		}
		nights := nightsBetween(reservation.CheckIn, reservation.CheckOut)
		if len(nights) == 0 || len(nights) > 366 || reservation.NumGuests < 1 || reservation.CheckIn.Before(normalizeDate(time.Now().UTC())) {
			return nil, hotelsDomain.ErrInvalidReservation
		}
		inventory := repository.client.Database(repository.database).Collection(repository.collection_inventory)
		for _, night := range nights {
			filter := bson.M{"hotel_id": reservation.HotelID, "date": night}
			var entry hotelsDAO.Inventory
			err := inventory.FindOne(sc, filter).Decode(&entry)
			if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
				return nil, err
			}
			if entry.Booked < 0 {
				return nil, hotelsDomain.ErrInventoryInconsistent
			}
			if entry.Booked+reservation.NumRooms > hotel.AvailableRooms {
				return nil, hotelsDomain.ErrNoAvailability
			}
			if _, err := inventory.UpdateOne(sc, filter, bson.M{"$inc": bson.M{"booked": reservation.NumRooms}, "$set": bson.M{"capacity": hotel.AvailableRooms}}, options.Update().SetUpsert(true)); err != nil {
				return nil, err
			}
		}
		reservation.HotelName = hotel.Name
		reservation.TotalPrice = int64(math.Round(hotel.PricePerNight*100)) * int64(len(nights)) * int64(reservation.NumRooms)
		reservation.Currency = "USD"
		reservation.Status = hotelsDAO.StatusConfirmed
		reservation.CreatedAt = time.Now().UTC()
		raw, err := bson.Marshal(reservation)
		if err != nil {
			return nil, err
		}
		var doc bson.M
		if err := bson.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		doc["_id"] = id
		if _, err = repository.client.Database(repository.database).Collection(repository.collection_reservation).InsertOne(sc, doc); err != nil {
			return nil, err
		}
		if operation != nil && operation.Key != "" {
			_, err = repository.idempotencyCollection().InsertOne(sc, IdempotencyRecord{Key: operation.Key, UserID: reservation.UserID, Fingerprint: fingerprint, ReservationID: id.Hex(), CreatedAt: time.Now().UTC()})
			if err != nil {
				return nil, err
			}
		}
		return id.Hex(), nil
	})
	if err != nil {
		// Una key concurrente sobre otro hotel puede chocar en el índice único.
		// También recuperamos commits aplicados cuya respuesta se perdió al vencer
		// el request: sólo se devuelve éxito si existe la identidad durable.
		if operation != nil && operation.Key != "" {
			recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), mongoOpTimeout)
			defer cancel()
			existing, lookupErr := repository.lookupIdempotency(recoveryCtx, operation.Key, reservation.UserID, fingerprint)
			if lookupErr == nil && existing != "" {
				operation.Replayed = true
				return existing, nil
			}
			if errors.Is(lookupErr, hotelsDomain.ErrIdempotencyConflict) {
				return "", lookupErr
			}
		}
		return "", err
	}
	return result.(string), nil
}

// Funcion para obtener una reserva por ID en MongoDB
func (repository Mongo) GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	// Convert reservation ID to MongoDB ObjectID
	// Un ID que ni siquiera es un ObjectID válido no puede existir → not-found
	// tipado, no un error de infraestructura (mismo criterio que RV14)
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Reservation{}, fmt.Errorf("invalid reservation id %q: %w", id, hotelsDomain.ErrReservationNotFound)
	}

	// Buscar el documento en MongoDB por su ID (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	var reservation hotelsDAO.Reservation
	filter := bson.M{"_id": objectID}
	err = repository.client.Database(repository.database).Collection(repository.collection_reservation).FindOne(ctx, filter).Decode(&reservation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return hotelsDAO.Reservation{}, fmt.Errorf("reservation %s: %w", id, hotelsDomain.ErrReservationNotFound)
		}
		return hotelsDAO.Reservation{}, fmt.Errorf("error finding reservation: %w", err)
	}

	// Asignar el ID como string para el objeto de retorno
	reservation.ID = id

	return reservation, nil
}

// CancelReservation cambia estado y libera todas las noches en la misma
// transacción. Repetirla sólo lee cancelled; no vuelve a decrementar.
func (repository Mongo) CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Reservation{}, hotelsDomain.ErrReservationNotFound
	}
	result, err := repository.transaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		db := repository.client.Database(repository.database)
		var reservation hotelsDAO.Reservation
		if err := db.Collection(repository.collection_reservation).FindOne(sc, bson.M{"_id": objectID}).Decode(&reservation); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, hotelsDomain.ErrReservationNotFound
			}
			return nil, err
		}
		if reservation.Status == hotelsDAO.StatusCancelled {
			return reservation, nil
		}
		if reservation.Status != hotelsDAO.StatusConfirmed || reservation.NumRooms < 1 {
			return nil, hotelsDomain.ErrInventoryInconsistent
		}
		if _, err := repository.lockHotel(sc, reservation.HotelID); err != nil {
			return nil, err
		}
		for _, night := range nightsBetween(reservation.CheckIn, reservation.CheckOut) {
			changed, err := db.Collection(repository.collection_inventory).UpdateOne(sc, bson.M{"hotel_id": reservation.HotelID, "date": night, "booked": bson.M{"$gte": reservation.NumRooms}}, bson.M{"$inc": bson.M{"booked": -reservation.NumRooms}})
			if err != nil {
				return nil, err
			}
			if changed.MatchedCount != 1 {
				return nil, hotelsDomain.ErrInventoryInconsistent
			}
		}
		now := time.Now().UTC()
		_, err := db.Collection(repository.collection_reservation).UpdateOne(sc, bson.M{"_id": objectID}, bson.M{"$set": bson.M{"status": hotelsDAO.StatusCancelled, "cancelled_at": now}})
		reservation.Status = hotelsDAO.StatusCancelled
		reservation.CancelledAt = &now
		return reservation, err
	})
	if err != nil {
		return hotelsDAO.Reservation{}, err
	}
	return result.(hotelsDAO.Reservation), nil
}

// findPageOptions arma la paginación a nivel DB (DB5) con orden estable por _id.
func findPageOptions(limit, offset int64) *options.FindOptions {
	return options.Find().
		SetSort(bson.D{{Key: "_id", Value: 1}}).
		SetLimit(limit).
		SetSkip(offset)
}

// Funcion para encontrar todas las reservas de un usuario en MongoDB
func (repository Mongo) GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	// Buscar el documento en MongoDB por su ID (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).Find(ctx, bson.M{"user_id": userID}, findPageOptions(limit, offset))
	if err != nil {
		return nil, fmt.Errorf("error finding document: %w", err)
	}

	// Decodificar el resultado
	var reservations []hotelsDAO.Reservation
	if err := result.All(ctx, &reservations); err != nil {
		return nil, fmt.Errorf("error decoding result: %w", err)
	}
	return reservations, nil
}

// Funcion para encontrar las reservas de un usuario en un hotel en MongoDB
func (repository Mongo) GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	canonicalID, err := canonicalHotelID(hotelID)
	if err != nil {
		return nil, err
	}
	hotelID = canonicalID
	// Buscar el documento en MongoDB por su ID (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).Find(ctx, bson.M{"hotel_id": hotelID}, findPageOptions(limit, offset))
	if err != nil {
		return nil, fmt.Errorf("error finding document: %w", err)
	}

	// Decodificar el resultado
	var reservations []hotelsDAO.Reservation
	if err := result.All(ctx, &reservations); err != nil {
		return nil, fmt.Errorf("error decoding result: %w", err)
	}
	return reservations, nil
}

// Funcion para encontrar las reservas de un usuario en un hotel en MongoDB
func (repository Mongo) GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string, limit, offset int64) ([]hotelsDAO.Reservation, error) {
	canonicalID, err := canonicalHotelID(hotelID)
	if err != nil {
		return nil, err
	}
	hotelID = canonicalID
	// Buscar el documento en MongoDB por su ID (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).Find(ctx, bson.M{"hotel_id": hotelID, "user_id": userID}, findPageOptions(limit, offset))
	if err != nil {
		return nil, fmt.Errorf("error finding document: %w", err)
	}

	// Decodificar el resultado
	var reservations []hotelsDAO.Reservation
	if err := result.All(ctx, &reservations); err != nil {
		return nil, fmt.Errorf("error decoding result: %w", err)
	}
	return reservations, nil
}

// GetAvailability verifica la disponibilidad de múltiples hoteles de forma
// concurrente, con límite de concurrencia (R4, bulkhead) y respuesta parcial
// (RV24): un hotel que falla se reporta available=false — sesgo conservador,
// nunca ofrecer lo que no se pudo verificar — en vez de tumbar el batch
// entero con 500. Sin errgroup a propósito: su cancelación en cascada es
// exactamente el comportamiento que se quiere evitar acá.
func (repository Mongo) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	type result struct {
		hotelID   string
		available bool
	}

	results := make(chan result, len(hotelIDs))
	semaphore := make(chan struct{}, availabilityMaxConcurrency)

	for _, id := range hotelIDs {
		go func(hotelID string) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			available, err := repository.IsHotelAvailable(ctx, hotelID, checkIn, checkOut)
			if err != nil {
				slog.Warn("availability check failed, reporting hotel as unavailable",
					"hotel_id", hotelID, "error", err)
				available = false
			}
			results <- result{hotelID: hotelID, available: available}
		}(id)
	}

	// Recolectar resultados (IDs duplicados en el body colapsan en el mapa)
	availability := make(map[string]bool, len(hotelIDs))
	for range hotelIDs {
		r := <-results
		availability[r.hotelID] = r.available
	}

	return availability, nil
}

// IsHotelAvailable verifica la disponibilidad leyendo los contadores del
// inventario (D4): modelo canónico por noche, checkout excluido. Una noche sin
// documento de inventario es una noche libre. Compara contra la capacidad
// actual del hotel — la misma que usa el filtro del claim.
func (repository Mongo) IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error) {
	canonicalID, err := canonicalHotelID(hotelID)
	if err != nil {
		return false, err
	}
	hotelID = canonicalID
	// Convertir las fechas
	checkInTime, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return false, fmt.Errorf("error parsing check-in date: %w", err)
	}
	checkOutTime, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return false, fmt.Errorf("error parsing check-out date: %w", err)
	}
	if !checkOutTime.After(checkInTime) {
		return false, fmt.Errorf("check-out date must be after check-in date")
	}

	// Capacidad actual del hotel (también valida que exista)
	capacity, err := repository.hotelCapacity(ctx, hotelID)
	if err != nil {
		return false, err
	}
	if capacity < 1 {
		return false, nil
	}

	// Una sola query por todo el rango de noches (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	nights := nightsBetween(checkInTime, checkOutTime)
	cursor, err := repository.client.Database(repository.database).Collection(repository.collection_inventory).
		Find(ctx, bson.M{"hotel_id": hotelID, "date": bson.M{"$in": nights}})
	if err != nil {
		return false, fmt.Errorf("error finding inventory: %w", err)
	}

	var entries []hotelsDAO.Inventory
	if err := cursor.All(ctx, &entries); err != nil {
		return false, fmt.Errorf("error decoding inventory: %w", err)
	}

	// Disponible si TODAS las noches del rango tienen cupo
	for _, entry := range entries {
		if entry.Booked < 0 {
			return false, hotelsDomain.ErrInventoryInconsistent
		}
		if entry.Booked >= capacity {
			return false, nil
		}
	}
	return true, nil
}

// GetHotelsAfter enumera sin offsets inestables ante borrados concurrentes.
func (repository Mongo) GetHotelsAfter(ctx context.Context, afterID string, limit int64) ([]hotelsDAO.Hotel, error) {
	ctx, cancel := opCtx(ctx)
	defer cancel()
	filter := bson.M{}
	if afterID != "" {
		id, err := primitive.ObjectIDFromHex(afterID)
		if err != nil {
			return nil, err
		}
		filter["_id"] = bson.M{"$gt": id}
	}
	cursor, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).Find(ctx, filter, findPageOptions(limit, 0))
	if err != nil {
		return nil, err
	}
	var result []hotelsDAO.Hotel
	err = cursor.All(ctx, &result)
	return result, err
}

func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ObjectIDs accept hexadecimal aliases, but foreign keys must have one
// spelling or capacity and idempotency could split into distinct inventories.
func canonicalHotelID(id string) (string, error) {
	parsed, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return "", hotelsDomain.ErrHotelNotFound
	}
	return parsed.Hex(), nil
}
