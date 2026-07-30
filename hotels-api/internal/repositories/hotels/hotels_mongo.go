package hotels

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
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

	// Reintentos de la liberación de noches (RV23): un fallo transitorio no
	// puede dejar inventario bloqueado para siempre.
	releaseMaxAttempts  = 3
	releaseRetryBackoff = 200 * time.Millisecond

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
	cfg := options.Client().ApplyURI(uri).SetAuth(credentials).
		SetMaxPoolSize(50).
		SetServerSelectionTimeout(5 * time.Second).
		SetSocketTimeout(socketTimeout)

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

// Actualiza un hotel en MongoDB
func (repository Mongo) Update(ctx context.Context, hotel hotelsDAO.Hotel) error {
	// Convert hotel ID to MongoDB ObjectID (hex inválido = not-found tipado, RV14)
	objectID, err := primitive.ObjectIDFromHex(hotel.ID)
	if err != nil {
		return fmt.Errorf("invalid hotel id %q: %w", hotel.ID, hotelsDomain.ErrHotelNotFound)
	}

	// Crea un mapa con los campos a actualizar
	update := bson.M{}

	// Actualiza solo los campos que no son cero o vacios
	if hotel.Name != "" {
		update["name"] = hotel.Name
	}
	if hotel.Address != "" {
		update["address"] = hotel.Address
	}
	if hotel.Description != "" {
		update["description"] = hotel.Description
	}
	if hotel.City != "" {
		update["city"] = hotel.City
	}
	if hotel.State != "" {
		update["state"] = hotel.State
	}
	if hotel.Country != "" {
		update["country"] = hotel.Country
	}
	if hotel.Phone != "" {
		update["phone"] = hotel.Phone
	}
	if hotel.Email != "" {
		update["email"] = hotel.Email
	}
	if hotel.PricePerNight != 0 { // Asumiendo que 0 es el valor por defecto para PricePerNight
		update["price_per_night"] = hotel.PricePerNight
	}
	if hotel.AvaiableRooms != 0 { // Asumiendo que 0 es el valor por defecto para AvaiableRooms
		update["avaiable_rooms"] = hotel.AvaiableRooms
	}
	if hotel.CheckInTime != "" { // Asumiendo que "" es el valor por defecto para CheckInTime
		update["check_in_time"] = hotel.CheckInTime
	}
	if hotel.CheckOutTime != "" { // Asumiendo que "" es el valor por defecto para CheckOutTime
		update["check_out_time"] = hotel.CheckOutTime
	}
	if hotel.Rating != 0 { // Asumiendo que 0 es el valor por defecto para Rating
		update["rating"] = hotel.Rating
	}
	if len(hotel.Amenities) > 0 { // Asumiendo que un slice vacio es el valor por defecto para Amenities
		update["amenities"] = hotel.Amenities
	}
	if len(hotel.Images) > 0 { // Asumiendo que un slice vacio es el valor por defecto para Images
		update["images"] = hotel.Images
	}

	// Actualiza el documento en MongoDB
	if len(update) == 0 {
		return fmt.Errorf("no fields to update for hotel ID %s", hotel.ID)
	}

	// Saca el objectID del documento y actualiza los campos en MongoDB
	ctx, cancel := opCtx(ctx)
	defer cancel()
	filter := bson.M{"_id": objectID}
	result, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("error updating document: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("hotel %s: %w", hotel.ID, hotelsDomain.ErrHotelNotFound)
	}

	return nil
}

// Elimina un hotel de MongoDB
func (repository Mongo) Delete(ctx context.Context, id string) error {
	// Convert hotel ID to MongoDB ObjectID (hex inválido = not-found tipado, RV14)
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid hotel id %q: %w", id, hotelsDomain.ErrHotelNotFound)
	}

	// Elimina el documento de MongoDB (deadline por operación, R2)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	filter := bson.M{"_id": objectID}
	result, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting document: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("hotel %s: %w", id, hotelsDomain.ErrHotelNotFound)
	}

	return nil
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

// hotelCapacity lee la capacidad actual del hotel (avaiable_rooms).
func (repository Mongo) hotelCapacity(ctx context.Context, hotelID string) (int, error) {
	objectID, err := primitive.ObjectIDFromHex(hotelID)
	if err != nil {
		return 0, fmt.Errorf("error converting hotel ID to object ID: %w", err)
	}

	var hotel struct {
		AvaiableRooms int `bson:"avaiable_rooms"`
	}
	ctx, cancel := opCtx(ctx)
	defer cancel()
	err = repository.client.Database(repository.database).Collection(repository.collection_hotel).
		FindOne(ctx, bson.M{"_id": objectID}, options.FindOne().SetProjection(bson.M{"avaiable_rooms": 1, "_id": 0})).
		Decode(&hotel)
	if err != nil {
		return 0, fmt.Errorf("error finding hotel: %w", err)
	}
	return hotel.AvaiableRooms, nil
}

// claimNight reclama atómicamente `rooms` habitaciones para una noche vía
// findOneAndUpdate con upsert sobre el índice único {hotel_id, date} (D1).
// Devuelve (false, nil) si la noche no tiene cupo.
func (repository Mongo) claimNight(ctx context.Context, collection *mongo.Collection, hotelID, day string, rooms, capacity int) (bool, error) {
	// Deadline por claim (R2): un claim colgado por timeout es ambiguo (pudo o
	// no haber impactado) — acotarlo achica esa ventana; ver releaseNights.
	ctx, cancel := opCtx(ctx)
	defer cancel()
	filter := bson.M{"hotel_id": hotelID, "date": day, "booked": bson.M{"$lte": capacity - rooms}}
	// hotel_id/date NO van en $setOnInsert: ya están en el filtro de igualdad
	// y duplicarlos da error "conflict" en el upsert.
	update := bson.M{"$inc": bson.M{"booked": rooms}, "$setOnInsert": bson.M{"capacity": capacity}}

	// SetReturnDocument(After): el default Before devuelve ErrNoDocuments en un
	// upsert-insert exitoso y parecería fallo.
	err := collection.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Err()
	if err == nil {
		return true, nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return false, fmt.Errorf("error claiming night %s: %w", day, err)
	}

	// Carrera del upsert: otro request insertó el doc entre el no-match del
	// filtro y nuestro insert. Reintentar UNA sola vez SIN upsert — el doc ya
	// existe y el filtro decide: matchea → claim ok; no matchea → noche llena.
	err = collection.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, fmt.Errorf("error claiming night %s (retry): %w", day, err)
}

// releaseNights libera habitaciones reclamadas (compensación del claim o
// cancelación). Usa un contexto sin cancelación: si el request original murió
// a mitad de camino, la compensación tiene que correr igual.
// Cada noche se reintenta con backoff (RV23): un único intento best-effort
// dejaba inventario bloqueado para siempre ante un fallo transitorio — y la
// idempotencia del segundo DELETE impide re-liberar después. Si aun así una
// noche falla, queda el log estructurado para reconciliación manual: el sesgo
// es deliberadamente conservador — liberar de más habilitaría overbooking,
// que acá es peor que una noche bloqueada.
func (repository Mongo) releaseNights(ctx context.Context, collection *mongo.Collection, hotelID string, nights []string, rooms int) {
	base := context.WithoutCancel(ctx)
	for _, night := range nights {
		if err := repository.releaseNight(base, collection, hotelID, night, rooms); err != nil {
			slog.Error("error releasing inventory night after retries; requires manual reconciliation",
				"hotel_id", hotelID, "date", night, "rooms", rooms, "error", err)
		}
	}
}

// releaseNight decrementa el contador de una noche con reintentos acotados y
// deadline propio por intento (R2/RV23).
func (repository Mongo) releaseNight(ctx context.Context, collection *mongo.Collection, hotelID, night string, rooms int) error {
	var lastErr error
	for attempt := 1; attempt <= releaseMaxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt-1) * releaseRetryBackoff)
		}
		attemptCtx, cancel := opCtx(ctx)
		_, err := collection.UpdateOne(attemptCtx,
			bson.M{"hotel_id": hotelID, "date": night},
			bson.M{"$inc": bson.M{"booked": -rooms}})
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

// Funcion para crear una reserva en MongoDB: primero reclama el inventario de
// cada noche de forma atómica (D1) y recién entonces inserta la reserva. Si
// algo falla a mitad de camino, libera lo ya reclamado (compensación).
func (repository Mongo) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	capacity, err := repository.hotelCapacity(ctx, reservation.HotelID)
	if err != nil {
		return "", err
	}
	// Guard defensivo: sin él, un pedido de más habitaciones que la capacidad
	// sobre una noche sin doc de inventario insertaría booked > capacity.
	if reservation.NumRooms < 1 || reservation.NumRooms > capacity {
		return "", hotelsDomain.ErrNoAvailability
	}

	inventory := repository.client.Database(repository.database).Collection(repository.collection_inventory)
	nights := nightsBetween(reservation.CheckIn, reservation.CheckOut)

	// Reclamar noche por noche; ante la primera que falle, liberar las ya
	// reclamadas y devolver el sentinel (el controller lo mapea a 409).
	var claimed []string
	for _, night := range nights {
		ok, err := repository.claimNight(ctx, inventory, reservation.HotelID, night, reservation.NumRooms, capacity)
		if err != nil {
			repository.releaseNights(ctx, inventory, reservation.HotelID, claimed, reservation.NumRooms)
			return "", err
		}
		if !ok {
			repository.releaseNights(ctx, inventory, reservation.HotelID, claimed, reservation.NumRooms)
			return "", hotelsDomain.ErrNoAvailability
		}
		claimed = append(claimed, night)
	}

	// Con el inventario asegurado, insertar el documento de la reserva
	// (deadline propio, R2: el ctx del método sigue sin acotar a propósito —
	// los claims ya se acotaron de a uno)
	insertCtx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).InsertOne(insertCtx, reservation)
	if err != nil {
		repository.releaseNights(ctx, inventory, reservation.HotelID, claimed, reservation.NumRooms)
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

// CancelReservation es un soft-delete idempotente (DM1): marca la reserva
// como cancelled y libera sus noches del inventario. El filtro por
// status=confirmed hace la operación idempotente — un segundo cancel no
// matchea y NO doble-libera. Devuelve la reserva cancelada (el service la usa
// para la caché y el evento).
func (repository Mongo) CancelReservation(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	// Convert reservation ID to MongoDB ObjectID (hex inválido = not-found tipado)
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Reservation{}, fmt.Errorf("invalid reservation id %q: %w", id, hotelsDomain.ErrReservationNotFound)
	}

	collection := repository.client.Database(repository.database).Collection(repository.collection_reservation)
	now := time.Now().UTC()

	// FindOneAndUpdate devuelve la pre-imagen (default Before): de ahí salen
	// las noches y habitaciones a liberar. Deadline por operación (R2) — la
	// liberación posterior corre sobre el ctx sin acotar (releaseNights pone
	// el suyo por intento).
	var reservation hotelsDAO.Reservation
	filter := bson.M{"_id": objectID, "status": hotelsDAO.StatusConfirmed}
	update := bson.M{"$set": bson.M{"status": hotelsDAO.StatusCancelled, "cancelled_at": now}}
	cancelCtx, cancelOp := opCtx(ctx)
	defer cancelOp()
	err = collection.FindOneAndUpdate(cancelCtx, filter, update).Decode(&reservation)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Idempotencia: si ya estaba cancelada, devolverla sin re-liberar
		existing, getErr := repository.GetReservationByID(ctx, id)
		if getErr != nil {
			return hotelsDAO.Reservation{}, fmt.Errorf("reservation %s: %w", id, hotelsDomain.ErrReservationNotFound)
		}
		if existing.Status == hotelsDAO.StatusCancelled {
			return existing, nil
		}
		return hotelsDAO.Reservation{}, fmt.Errorf("reservation %s is not cancellable (status %q)", id, existing.Status)
	}
	if err != nil {
		return hotelsDAO.Reservation{}, fmt.Errorf("error cancelling reservation: %w", err)
	}
	reservation.ID = id

	// Liberar las noches de la reserva del inventario
	inventory := repository.client.Database(repository.database).Collection(repository.collection_inventory)
	repository.releaseNights(ctx, inventory, reservation.HotelID, nightsBetween(reservation.CheckIn, reservation.CheckOut), reservation.NumRooms)

	// Reflejar el estado post-cancelación en el valor devuelto
	reservation.Status = hotelsDAO.StatusCancelled
	reservation.CancelledAt = &now

	return reservation, nil
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

// Funcion para eliminar todas las reservas de un hotel (y su inventario:
// sin esto, borrar un hotel dejaría contadores huérfanos)
func (repository Mongo) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	// Eliminar todas las reservas que pertenezcan al hotel especificado
	// (deadline por operación, R2 — cubre los dos DeleteMany)
	ctx, cancel := opCtx(ctx)
	defer cancel()
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).DeleteMany(ctx, bson.M{"hotel_id": hotelID})
	if err != nil {
		return fmt.Errorf("error deleting reservations for hotel %s: %w", hotelID, err)
	}

	if _, err := repository.client.Database(repository.database).Collection(repository.collection_inventory).DeleteMany(ctx, bson.M{"hotel_id": hotelID}); err != nil {
		return fmt.Errorf("error deleting inventory for hotel %s: %w", hotelID, err)
	}

	// Log para debugging
	fmt.Printf("Deleted %d reservations for hotel %s\n", result.DeletedCount, hotelID)

	return nil
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
		if entry.Booked >= capacity {
			return false, nil
		}
	}
	return true, nil
}
