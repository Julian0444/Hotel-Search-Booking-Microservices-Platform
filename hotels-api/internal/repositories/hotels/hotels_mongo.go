package hotels

import (
	"context"
	"fmt"
	"log"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"

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
}

type Mongo struct {
	client                 *mongo.Client
	database               string
	collection_hotel       string
	collection_reservation string
}

const (
	connectionURI = "mongodb://%s:%s"
)

// Crea una nueva instancia de Mongo
func NewMongo(config MongoConfig) Mongo {
	credentials := options.Credential{
		Username: config.Username,
		Password: config.Password,
	}

	//Crea el contexto
	ctx := context.Background()
	//Crea la URI de conexion
	uri := fmt.Sprintf(connectionURI, config.Host, config.Port)
	//Crea la configuracion de conexion
	cfg := options.Client().ApplyURI(uri).SetAuth(credentials)

	//Crea la conexion a MongoDB
	client, err := mongo.Connect(ctx, cfg)
	if err != nil {
		log.Panicf("error connecting to mongo DB: %v", err)
	}

	return Mongo{
		client:                 client,
		database:               config.Database,
		collection_hotel:       config.Collection_hotels,
		collection_reservation: config.Collection_reservations,
	}
}

// Obtiene un hotel por su ID de MongoDB
func (repository Mongo) GetHotelByID(ctx context.Context, id string) (hotelsDAO.Hotel, error) {

	//Crea el ObjectID de MongoDB a partir del ID para buscar el documento
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Hotel{}, fmt.Errorf("error converting id to mongo ID: %w", err)
	}

	// Buscar el documento en MongoDB por su ID
	result := repository.client.Database(repository.database).Collection(repository.collection_hotel).FindOne(ctx, bson.M{"_id": objectID})
	if result.Err() != nil {
		return hotelsDAO.Hotel{}, fmt.Errorf("error finding document: %w", result.Err())
	}

	// Decodificar el resultado
	var hotelDAO hotelsDAO.Hotel
	if err := result.Decode(&hotelDAO); err != nil {
		return hotelsDAO.Hotel{}, fmt.Errorf("error decoding result: %w", err)
	}
	return hotelDAO, nil
}

// Crea un nuevo hotel en MongoDB
func (repository Mongo) Create(ctx context.Context, hotel hotelsDAO.Hotel) (string, error) {
	// Insertar el documento en MongoDB
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
	// Convert hotel ID to MongoDB ObjectID
	objectID, err := primitive.ObjectIDFromHex(hotel.ID)
	if err != nil {
		return fmt.Errorf("error converting id to mongo ID: %w", err)
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
	if !hotel.CheckInTime.IsZero() { // Asumiendo que una fecha cero es el valor por defecto para CheckInTime
		update["check_in_time"] = hotel.CheckInTime
	}
	if !hotel.CheckOutTime.IsZero() { // Asumiendo que una fecha cero es el valor por defecto para CheckOutTime
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
	filter := bson.M{"_id": objectID}
	result, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("error updating document: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with ID %s", hotel.ID)
	}

	return nil
}

// Elimina un hotel de MongoDB
func (repository Mongo) Delete(ctx context.Context, id string) error {
	// Convert hotel ID to MongoDB ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("error converting id to mongo ID: %w", err)
	}

	// Elimina el documento de MongoDB
	filter := bson.M{"_id": objectID}
	result, err := repository.client.Database(repository.database).Collection(repository.collection_hotel).DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting document: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("no document found with ID %s", id)
	}

	return nil
}

// Funcion para crear una reserva en MongoDB
func (repository Mongo) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	// Insertar el documento en MongoDB
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).InsertOne(ctx, reservation)
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

// Funcion para obtener una reserva por ID en MongoDB
func (repository Mongo) GetReservationByID(ctx context.Context, id string) (hotelsDAO.Reservation, error) {
	// Convert reservation ID to MongoDB ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return hotelsDAO.Reservation{}, fmt.Errorf("error converting id to mongo ID: %w", err)
	}

	// Buscar el documento en MongoDB por su ID
	var reservation hotelsDAO.Reservation
	filter := bson.M{"_id": objectID}
	err = repository.client.Database(repository.database).Collection(repository.collection_reservation).FindOne(ctx, filter).Decode(&reservation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return hotelsDAO.Reservation{}, fmt.Errorf("reservation not found with ID %s", id)
		}
		return hotelsDAO.Reservation{}, fmt.Errorf("error finding reservation: %w", err)
	}

	// Asignar el ID como string para el objeto de retorno
	reservation.ID = id

	return reservation, nil
}

// Funcion para cancelar una reserva en MongoDB
func (repository Mongo) CancelReservation(ctx context.Context, id string) error {
	// Convert reservation ID to MongoDB ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("error converting id to mongo ID: %w", err)
	}

	// Elimina el documento de MongoDB
	filter := bson.M{"_id": objectID}
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting document: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("no document found with ID %s", id)
	}

	return nil
}

// Funcion para encontrar todas las reservas de un usuario en MongoDB
func (repository Mongo) GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDAO.Reservation, error) {
	// Buscar el documento en MongoDB por su ID
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).Find(ctx, bson.M{"user_id": userID})
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
func (repository Mongo) GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDAO.Reservation, error) {
	// Buscar el documento en MongoDB por su ID
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).Find(ctx, bson.M{"hotel_id": hotelID})
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
func (repository Mongo) GetReservationsByUserAndHotelID(ctx context.Context, hotelID string, userID string) ([]hotelsDAO.Reservation, error) {
	// Buscar el documento en MongoDB por su ID
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).Find(ctx, bson.M{"hotel_id": hotelID, "user_id": userID})
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

// Funcion para eliminar todas las reservas de un hotel
func (repository Mongo) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	// Eliminar todas las reservas que pertenezcan al hotel especificado
	result, err := repository.client.Database(repository.database).Collection(repository.collection_reservation).DeleteMany(ctx, bson.M{"hotel_id": hotelID})
	if err != nil {
		return fmt.Errorf("error deleting reservations for hotel %s: %w", hotelID, err)
	}

	// Log para debugging
	fmt.Printf("Deleted %d reservations for hotel %s\n", result.DeletedCount, hotelID)

	return nil
}

// Funcion para calcular la dispinibilidad de multiples hoteles de forma concurrente utilizando goroutines
// GetAvailability verifica la disponibilidad de múltiples hoteles de forma concurrente
func (repository Mongo) GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error) {
	type result struct {
		hotelID   string
		available bool
		err       error
	}

	results := make(chan result, len(hotelIDs))

	// Crear un WaitGroup para esperar a que todas las goroutines terminen
	for _, id := range hotelIDs {
		go func(hotelID string) {
			available, err := repository.IsHotelAvailable(ctx, hotelID, checkIn, checkOut)
			results <- result{
				hotelID:   hotelID,
				available: available,
				err:       err,
			}
		}(id)
	}

	// Recolectar resultados
	availability := make(map[string]bool)
	for i := 0; i < len(hotelIDs); i++ {
		r := <-results
		if r.err != nil {
			return nil, fmt.Errorf("error checking availability for hotel %s: %w", r.hotelID, r.err)
		}
		availability[r.hotelID] = r.available
	}

	return availability, nil
}

// IsHotelAvailable verifica la disponibilidad de un hotel para un rango de fechas
func (repository Mongo) IsHotelAvailable(ctx context.Context, hotelID, checkIn, checkOut string) (bool, error) {
	// Convertir el ID del hotel a ObjectID
	objectID, err := primitive.ObjectIDFromHex(hotelID)
	if err != nil {
		return false, fmt.Errorf("error converting hotel ID to object ID: %w", err)
	}

	// Convertir las fechas
	checkInTime, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return false, fmt.Errorf("error parsing check-in date: %w", err)
	}
	checkOutTime, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return false, fmt.Errorf("error parsing check-out date: %w", err)
	}

	// Primero obtener el hotel y su capacidad en una sola consulta
	var av struct {
		AvailableRooms int32 `bson:"avaiable_rooms"`
	}
	err = repository.client.Database(repository.database).Collection(repository.collection_hotel).
		FindOne(ctx, bson.M{"_id": objectID}, options.FindOne().SetProjection(bson.M{"avaiable_rooms": 1, "_id": 0})).
		Decode(&av)
	if err != nil {
		return false, fmt.Errorf("error finding hotel: %w", err)
	}

	// Lista para almacenar los días
	var days []string

	// Iterar desde checkIn hasta checkOut
	for current := checkInTime; !current.After(checkOutTime); current = current.AddDate(0, 0, 1) {
		days = append(days, current.Format("2006-01-02"))
	}

	var maxreservations int = -1
	for _, day := range days {
		time, err := time.Parse("2006-01-02", day)
		if err != nil {
			return false, fmt.Errorf("error parsing date: %w", err)
		}
		pipeline := []bson.M{
			{
				"$match": bson.M{
					"hotel_id":  hotelID,
					"check_in":  bson.M{"$lte": time},
					"check_out": bson.M{"$gt": time},
				},
			},
			{
				"$group": bson.M{
					"_id":              nil,
					"reservas_activas": bson.M{"$sum": 1},
				},
			},
			{
				"$project": bson.M{
					"_id":              0,
					"reservas_activas": 1,
				},
			},
			{
				"$unionWith": bson.M{
					"coll": nil, // Esto es necesario para indicar que estamos agregando un documento manual
					"pipeline": []bson.M{
						{
							"$documents": []bson.M{
								{"reservas_activas": 0},
							},
						},
					},
				},
			},
			{
				"$group": bson.M{
					"_id":              nil,
					"reservas_activas": bson.M{"$max": "$reservas_activas"},
				},
			},
			{
				"$project": bson.M{
					"_id":              0,
					"reservas_activas": 1,
				},
			},
		}

		// Ejecutar el pipeline
		cursor, err := repository.client.Database(repository.database).
			Collection(repository.collection_reservation).
			Aggregate(ctx, pipeline)
		if err != nil {
			return false, fmt.Errorf("error aggregating reservations: %w", err)
		}
		defer cursor.Close(ctx)

		// Obtener el resultado
		var result struct {
			ReservasActivas int `bson:"reservas_activas"`
		}
		if cursor.Next(ctx) {
			if err := cursor.Decode(&result); err != nil {
				return false, fmt.Errorf("error decoding result: %w", err)
			}
		} else {
			return false, fmt.Errorf("no results found")
		}

		if result.ReservasActivas > maxreservations {
			maxreservations = result.ReservasActivas
		}
	}
	// Verificar disponibilidad
	return maxreservations < int(av.AvailableRooms), nil
}
