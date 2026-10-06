package hotels

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/apperr"
	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Estas son las funciones que se encargan de interactuar con el servicio, se encargan de recibir las peticiones y enviar las respuestas (Vienen del service)
type Service interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error)
	GetHotels(ctx context.Context, limit, offset int64) ([]hotelsDomain.Hotel, int64, error)
	GetHotelsAfter(ctx context.Context, afterID string, limit int64) ([]hotelsDomain.Hotel, error)
	Create(ctx context.Context, hotel hotelsDomain.Hotel) (string, error)
	Update(ctx context.Context, hotel hotelsDomain.Hotel) error
	Delete(ctx context.Context, id string) error
	CreateReservation(ctx context.Context, reservation hotelsDomain.Reservation) (string, error)
	GetReservationByID(ctx context.Context, id string) (hotelsDomain.Reservation, error)
	CancelReservation(ctx context.Context, id string) error
	GetReservationsByHotelID(ctx context.Context, hotelID string, limit, offset int64) ([]hotelsDomain.Reservation, error)
	GetReservationsByUserID(ctx context.Context, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error)
	// IMPORTANT: el orden semántico es (hotelID, userID) para mantener consistencia con el service/repositories.
	GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, limit, offset int64) ([]hotelsDomain.Reservation, error)
	GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error)
}

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{
		service: service,
	}
}

const (
	defaultPageLimit int64 = 20
	maxPageLimit     int64 = 100
)

// paginationParams parsea ?limit y ?offset con defaults y clamp (max 100),
// la convención única de paginación de la plataforma (A4).
func paginationParams(ctx *gin.Context) (int64, int64) {
	limit, err := strconv.ParseInt(ctx.DefaultQuery("limit", strconv.FormatInt(defaultPageLimit, 10)), 10, 64)
	if err != nil || limit < 1 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	offset, err := strconv.ParseInt(ctx.DefaultQuery("offset", "0"), 10, 64)
	if err != nil || offset < 0 {
		offset = 0
	}

	return limit, offset
}

// userFromToken saca userID y userType del contexto que dejó el middleware
// JWT; si faltan (ruta mal cableada), corta con 401.
func userFromToken(ctx *gin.Context) (userID, userType string, ok bool) {
	userIDAny, exists := ctx.Get("userID")
	if !exists {
		apperr.Abort(ctx, http.StatusUnauthorized, "unauthorized", "user ID not found in token", nil)
		return "", "", false
	}
	userID, okID := userIDAny.(string)
	if !okID {
		apperr.Abort(ctx, http.StatusUnauthorized, "unauthorized", "invalid user ID format in token", nil)
		return "", "", false
	}

	userTypeAny, exists := ctx.Get("userType")
	if !exists {
		apperr.Abort(ctx, http.StatusUnauthorized, "unauthorized", "user type not found in token", nil)
		return "", "", false
	}
	userType, okType := userTypeAny.(string)
	if !okType {
		apperr.Abort(ctx, http.StatusUnauthorized, "unauthorized", "invalid user type format in token", nil)
		return "", "", false
	}

	return userID, userType, true
}

// Funcion para obtener un hotel por ID (GET)
func (controller Controller) GetHotelByID(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	hotelID := strings.TrimSpace(ctx.Param("hotel_id"))

	// Obtiene el hotel por ID
	hotel, err := controller.service.GetHotelByID(ctx.Request.Context(), hotelID)
	if err != nil {
		// Solo "no existe" es 404 (RV14); cualquier otro fallo es un 500 real.
		// search-api decide con este código si descarta el evento (404) o lo
		// reintenta/DLQea (5xx).
		if errors.Is(err, hotelsDomain.ErrHotelNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "hotel_not_found", "hotel not found", err)
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting hotel", err)
		return
	}

	// Devuelve el hotel encontrado
	ctx.JSON(http.StatusOK, gin.H{"data": hotel})
}

// GetHotels lista el catálogo paginado con el envelope estándar (A5): lo
// consumen el backfill/reindex de search-api y el frontend.
func (controller Controller) GetHotels(ctx *gin.Context) {
	limit, offset := paginationParams(ctx)
	if afterID, present := ctx.GetQuery("after_id"); present || ctx.Request.URL.Query().Has("after_id") {
		if afterID != "" {
			if _, err := primitive.ObjectIDFromHex(afterID); err != nil {
				apperr.Abort(ctx, http.StatusBadRequest, "invalid_cursor", "after_id must be a Mongo ObjectID", nil)
				return
			}
		}
		rows, err := controller.service.GetHotelsAfter(ctx.Request.Context(), afterID, limit)
		if err != nil {
			apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting hotels", err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"data": rows, "meta": gin.H{"limit": limit, "after_id": afterID}})
		return
	}
	hotels, total, err := controller.service.GetHotels(ctx.Request.Context(), limit, offset)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting hotels", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": hotels,
		"meta": gin.H{"total": total, "limit": limit, "offset": offset},
	})
}

// Funcion para crear un hotel (POST)
func (controller Controller) Create(ctx *gin.Context) {
	// Le da formato al hotel que viene en el body de la peticiona un DAO
	var hotel hotelsDomain.Hotel
	if err := ctx.ShouldBindJSON(&hotel); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body", "invalid request body", err)
		return
	}
	if err := hotelsDomain.ValidateHotel(hotel); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_hotel", err.Error(), nil)
		return
	}

	// Crea el hotel
	id, err := controller.service.Create(ctx.Request.Context(), hotel)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error creating hotel", err)
		return
	}

	// 201 con Location del recurso creado (A6)
	ctx.Header("Location", "/api/v1/hotels/"+id)
	ctx.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": id}})
}

// Funcion para actualizar un hotel (PUT)
func (controller Controller) Update(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	id := strings.TrimSpace(ctx.Param("hotel_id"))

	// Le da formato al hotel que viene en el body de la peticiona un DAO
	var hotel hotelsDomain.Hotel
	if err := ctx.ShouldBindJSON(&hotel); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body", "invalid request body", err)
		return
	}
	if err := hotelsDomain.ValidateHotel(hotel); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_hotel", err.Error(), nil)
		return
	}

	// Asigna el ID al hotel
	hotel.ID = id

	// Actualiza el hotel
	if err := controller.service.Update(ctx.Request.Context(), hotel); err != nil {
		if errors.Is(err, hotelsDomain.ErrCapacityConflict) {
			apperr.Abort(ctx, http.StatusConflict, "capacity_conflict", err.Error(), nil)
			return
		}
		if errors.Is(err, hotelsDomain.ErrHotelNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "hotel_not_found", "hotel not found", err)
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error updating hotel", err)
		return
	}

	// PUT devuelve la representación actualizada (A6), no un {message:id}
	updated, err := controller.service.GetHotelByID(ctx.Request.Context(), id)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error fetching updated hotel", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": updated})
}

// Funcion para eliminar un hotel (DELETE)
func (controller Controller) Delete(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	id := strings.TrimSpace(ctx.Param("hotel_id"))

	// Elimina el hotel
	if err := controller.service.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, hotelsDomain.ErrHotelHasReservations) {
			apperr.Abort(ctx, http.StatusConflict, "hotel_has_reservations", err.Error(), nil)
			return
		}
		if errors.Is(err, hotelsDomain.ErrHotelNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "hotel_not_found", "hotel not found", err)
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error deleting hotel", err)
		return
	}

	// DELETE exitoso → 204 sin body (A6)
	ctx.Status(http.StatusNoContent)
}

// createReservationRequest es el DTO de creación: fechas como string
// "2006-01-02" (formato canónico, igual que en las respuestas — RV20).
// hotel_name NO se acepta del body — se deriva del hotel en el service.
type createReservationRequest struct {
	HotelID   string `json:"hotel_id" binding:"required"`
	UserID    string `json:"user_id"`
	CheckIn   string `json:"check_in" binding:"required"`
	CheckOut  string `json:"check_out" binding:"required"`
	NumRooms  int    `json:"num_rooms"`
	NumGuests int    `json:"num_guests"`
}

// Funcion para crear una reserva (POST)
func (controller Controller) CreateReservation(ctx *gin.Context) {
	// Le da formato a la reserva que viene en el body de la peticion
	var req createReservationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body", "invalid request body", err)
		return
	}

	// Parseo explícito de fechas canónicas (el service revalida con errores
	// tipados; acá se corta temprano con mensajes más específicos)
	checkIn, err := time.Parse(hotelsDomain.DateFormat, req.CheckIn)
	if err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_reservation",
			"check_in must be a date in YYYY-MM-DD format", nil)
		return
	}
	checkOut, err := time.Parse(hotelsDomain.DateFormat, req.CheckOut)
	if err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_reservation",
			"check_out must be a date in YYYY-MM-DD format", nil)
		return
	}
	if !checkOut.After(checkIn) {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_reservation",
			"check_out must be after check_in", nil)
		return
	}
	if req.NumRooms < 0 || req.NumGuests < 0 {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_reservation",
			"num_rooms and num_guests must be positive", nil)
		return
	}

	// Obtener el user_id del token JWT
	userIDString, _, ok := userFromToken(ctx)
	if !ok {
		return
	}

	// Validar que el usuario solo pueda crear reservas para sí mismo
	// (si el body no trae user_id, se toma el del token)
	if req.UserID != "" && req.UserID != userIDString {
		apperr.Abort(ctx, http.StatusForbidden, "forbidden",
			"users can only create reservations for themselves", nil)
		return
	}

	// Crea la reserva
	id, err := controller.service.CreateReservation(ctx.Request.Context(), hotelsDomain.Reservation{
		HotelID:   req.HotelID,
		UserID:    userIDString,
		CheckIn:   req.CheckIn,
		CheckOut:  req.CheckOut,
		NumRooms:  req.NumRooms,
		NumGuests: req.NumGuests,
	})
	if err != nil {
		switch {
		case errors.Is(err, hotelsDomain.ErrIdempotencyConflict):
			apperr.Abort(ctx, http.StatusConflict, "idempotency_conflict", hotelsDomain.ErrIdempotencyConflict.Error(), nil)
		case errors.Is(err, hotelsDomain.ErrLegacyIdempotency):
			apperr.Abort(ctx, http.StatusConflict, "idempotency_migration_required", hotelsDomain.ErrLegacyIdempotency.Error(), nil)
		// Reservar sobre un hotel que no existe → 404, no 500 (RV19)
		case errors.Is(err, hotelsDomain.ErrHotelNotFound):
			apperr.Abort(ctx, http.StatusNotFound, "hotel_not_found", "hotel not found", err)
		// Sin cupo en el rango pedido → 409 Conflict (D1)
		case errors.Is(err, hotelsDomain.ErrNoAvailability):
			apperr.Abort(ctx, http.StatusConflict, "no_availability",
				hotelsDomain.ErrNoAvailability.Error(), err)
		// Validación del service → 400, no 500 (RV19). El texto es nuestro
		// (sentinel wrappeado), no filtra internals.
		case errors.Is(err, hotelsDomain.ErrInvalidReservation):
			apperr.Abort(ctx, http.StatusBadRequest, "invalid_reservation", err.Error(), nil)
		default:
			apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error creating reservation", err)
		}
		return
	}

	// 201 con Location del recurso creado (A6)
	if operation := hotelsDomain.IdempotencyFromContext(ctx.Request.Context()); operation != nil && operation.Replayed {
		ctx.Header("Idempotency-Replayed", "true")
	}
	ctx.Header("Location", "/api/v1/reservations/"+id)
	ctx.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": id}})
}

// GetReservationByID devuelve una reserva puntual (el recurso al que apunta el
// Location del create). Solo el dueño o un admin pueden verla.
func (controller Controller) GetReservationByID(ctx *gin.Context) {
	id := strings.TrimSpace(ctx.Param("id"))

	userIDString, userType, ok := userFromToken(ctx)
	if !ok {
		return
	}

	reservation, err := controller.service.GetReservationByID(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, hotelsDomain.ErrReservationNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "reservation_not_found", "reservation not found", err)
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting reservation", err)
		return
	}

	if userType != "administrador" && reservation.UserID != userIDString {
		apperr.Abort(ctx, http.StatusForbidden, "forbidden",
			"users can only view their own reservations", nil)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": reservation})
}

func (controller Controller) CancelReservation(ctx *gin.Context) {
	// Valida el ID de la reserva que viene en la URL
	id := strings.TrimSpace(ctx.Param("id"))

	// Obtener el user_id del token JWT
	userIDString, _, ok := userFromToken(ctx)
	if !ok {
		return
	}

	// Obtener la reserva para verificar que pertenece al usuario
	reservation, err := controller.service.GetReservationByID(ctx.Request.Context(), id)
	if err != nil {
		// Solo "no existe" es 404; un fallo de infraestructura es 500 (A6)
		if errors.Is(err, hotelsDomain.ErrReservationNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "reservation_not_found", "reservation not found", err)
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting reservation", err)
		return
	}

	// Validar que el usuario solo pueda cancelar sus propias reservas
	if reservation.UserID != userIDString {
		apperr.Abort(ctx, http.StatusForbidden, "forbidden",
			"users can only cancel their own reservations", nil)
		return
	}

	// Cancela la reserva
	if err := controller.service.CancelReservation(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, hotelsDomain.ErrReservationNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "reservation_not_found", "reservation not found", err)
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error canceling reservation", err)
		return
	}

	// DELETE exitoso → 204 sin body (A6)
	ctx.Status(http.StatusNoContent)
}

func (controller Controller) GetReservationsByHotelID(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	hotelID := strings.TrimSpace(ctx.Param("hotel_id"))

	// Obtiene las reservas por ID de hotel (paginadas)
	limit, offset := paginationParams(ctx)
	reservations, err := controller.service.GetReservationsByHotelID(ctx.Request.Context(), hotelID, limit, offset)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting reservations", err)
		return
	}

	// Devuelve las reservas encontradas
	ctx.JSON(http.StatusOK, gin.H{
		"data": reservations,
		"meta": gin.H{"limit": limit, "offset": offset},
	})
}

func (controller Controller) GetReservationsByUserID(ctx *gin.Context) {
	// Valida el ID del usuario que viene en la URL
	userID := strings.TrimSpace(ctx.Param("user_id"))

	// Ownership: solo admin puede consultar reservas de otros usuarios
	userIDFromToken, userType, ok := userFromToken(ctx)
	if !ok {
		return
	}

	if userType != "administrador" && userIDFromToken != userID {
		apperr.Abort(ctx, http.StatusForbidden, "forbidden",
			"users can only view their own reservations", nil)
		return
	}

	// Obtiene las reservas por ID de usuario (paginadas)
	limit, offset := paginationParams(ctx)
	reservations, err := controller.service.GetReservationsByUserID(ctx.Request.Context(), userID, limit, offset)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting reservations", err)
		return
	}

	// Devuelve las reservas encontradas
	ctx.JSON(http.StatusOK, gin.H{
		"data": reservations,
		"meta": gin.H{"limit": limit, "offset": offset},
	})
}

func (controller Controller) GetReservationsByUserAndHotelID(ctx *gin.Context) {
	// Valida el ID del usuario que viene en la URL
	userID := strings.TrimSpace(ctx.Param("user_id"))
	hotelID := strings.TrimSpace(ctx.Param("hotel_id"))

	// Ownership: solo admin puede consultar reservas de otros usuarios
	userIDFromToken, userType, ok := userFromToken(ctx)
	if !ok {
		return
	}

	if userType != "administrador" && userIDFromToken != userID {
		apperr.Abort(ctx, http.StatusForbidden, "forbidden",
			"users can only view their own reservations", nil)
		return
	}

	// Obtiene las reservas por ID de usuario y hotel (paginadas)
	limit, offset := paginationParams(ctx)
	reservations, err := controller.service.GetReservationsByUserAndHotelID(ctx.Request.Context(), hotelID, userID, limit, offset)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting reservations", err)
		return
	}

	// Devuelve las reservas encontradas
	ctx.JSON(http.StatusOK, gin.H{
		"data": reservations,
		"meta": gin.H{"limit": limit, "offset": offset},
	})
}

func (controller Controller) GetAvailability(ctx *gin.Context) {
	// Valida los IDs de los hoteles que vienen en el body de la peticion
	var req struct {
		HotelIDs []string `json:"hotel_ids"`
		CheckIn  string   `json:"check_in"`
		CheckOut string   `json:"check_out"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body", "invalid request body", err)
		return
	}

	// Fechas mal formadas son un 400 del cliente, no un 500 (RV19)
	checkIn, err := time.Parse(hotelsDomain.DateFormat, req.CheckIn)
	if err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body",
			"check_in must be a date in YYYY-MM-DD format", nil)
		return
	}
	checkOut, err := time.Parse(hotelsDomain.DateFormat, req.CheckOut)
	if err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body",
			"check_out must be a date in YYYY-MM-DD format", nil)
		return
	}
	if !checkOut.After(checkIn) {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body",
			"check_out must be after check_in", nil)
		return
	}

	// Obtiene la disponibilidad de los hoteles
	availability, err := controller.service.GetAvailability(ctx.Request.Context(), req.HotelIDs, req.CheckIn, req.CheckOut)
	if err != nil {
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting availability", err)
		return
	}

	// Devuelve la disponibilidad de los hoteles
	ctx.JSON(http.StatusOK, gin.H{"data": availability})
}
