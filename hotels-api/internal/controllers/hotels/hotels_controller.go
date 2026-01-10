package hotels

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	hotelsDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/domain/hotels"

	"github.com/gin-gonic/gin"
)

// Estas son las funciones que se encargan de interactuar con el servicio, se encargan de recibir las peticiones y enviar las respuestas (Vienen del service)
type Service interface {
	GetHotelByID(ctx context.Context, id string) (hotelsDomain.Hotel, error)
	Create(ctx context.Context, hotel hotelsDomain.Hotel) (string, error)
	Update(ctx context.Context, hotel hotelsDomain.Hotel) error
	Delete(ctx context.Context, id string) error
	CreateReservation(ctx context.Context, reservation hotelsDomain.Reservation) (string, error)
	GetReservationByID(ctx context.Context, id string) (hotelsDomain.Reservation, error)
	CancelReservation(ctx context.Context, id string) error
	GetReservationsByHotelID(ctx context.Context, hotelID string) ([]hotelsDomain.Reservation, error)
	GetReservationsByUserID(ctx context.Context, userID string) ([]hotelsDomain.Reservation, error)
	// IMPORTANT: el orden semántico es (hotelID, userID) para mantener consistencia con el service/repositories.
	GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string) ([]hotelsDomain.Reservation, error)
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

// Funcion para obtener un hotel por ID (GET)
func (controller Controller) GetHotelByID(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	hotelID := strings.TrimSpace(ctx.Param("hotel_id"))

	// Obtiene el hotel por ID
	hotel, err := controller.service.GetHotelByID(ctx.Request.Context(), hotelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("error getting hotel: %s", err.Error()),
		})
		return
	}

	// Devuelve el hotel encontrado
	ctx.JSON(http.StatusOK, hotel)
}

// Funcion para crear un hotel (POST)
func (controller Controller) Create(ctx *gin.Context) {
	// Le da formato al hotel que viene en el body de la peticiona un DAO
	var hotel hotelsDomain.Hotel
	if err := ctx.ShouldBindJSON(&hotel); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Crea el hotel
	id, err := controller.service.Create(ctx.Request.Context(), hotel)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error creating hotel: %s", err.Error()),
		})
		return
	}

	// Devuelve el ID del hotel creado
	ctx.JSON(http.StatusCreated, gin.H{
		"id": id,
	})
}

// Funcion para actualizar un hotel (PUT)
func (controller Controller) Update(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	id := strings.TrimSpace(ctx.Param("hotel_id"))

	// Le da formato al hotel que viene en el body de la peticiona un DAO
	var hotel hotelsDomain.Hotel
	if err := ctx.ShouldBindJSON(&hotel); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Asigna el ID al hotel
	hotel.ID = id

	// Actualiza el hotel
	if err := controller.service.Update(ctx.Request.Context(), hotel); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error updating hotel: %s", err.Error()),
		})
		return
	}

	// Devuelve el ID del hotel actualizado
	ctx.JSON(http.StatusOK, gin.H{
		"message": id,
	})
}

// Funcion para eliminar un hotel (DELETE)
func (controller Controller) Delete(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	id := strings.TrimSpace(ctx.Param("hotel_id"))

	// Elimina el hotel
	if err := controller.service.Delete(ctx.Request.Context(), id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error deleting hotel: %s", err.Error()),
		})
		return
	}

	// Devuelve el ID del hotel eliminado
	ctx.JSON(http.StatusOK, gin.H{
		"message": id,
	})
}

// Funcion para crear una reserva (POST)
func (controller Controller) CreateReservation(ctx *gin.Context) {
	// Le da formato a la reserva que viene en el body de la peticiona
	var reservation hotelsDomain.Reservation
	if err := ctx.ShouldBindJSON(&reservation); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Obtener el user_id del token JWT
	userIDFromToken, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	userIDString, ok := userIDFromToken.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID format in token",
		})
		return
	}

	// Validar que el usuario solo pueda crear reservas para sí mismo
	if reservation.UserID != userIDString {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Users can only create reservations for themselves",
		})
		return
	}

	// Crea la reserva
	id, err := controller.service.CreateReservation(ctx.Request.Context(), reservation)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error creating reservation: %s", err.Error()),
		})
		return
	}

	// Devuelve el ID de la reserva creada
	ctx.JSON(http.StatusCreated, gin.H{
		"id": id,
	})
}

func (controller Controller) CancelReservation(ctx *gin.Context) {
	// Valida el ID de la reserva que viene en la URL
	id := strings.TrimSpace(ctx.Param("id"))

	// Obtener el user_id del token JWT
	userIDFromToken, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	userIDString, ok := userIDFromToken.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID format in token",
		})
		return
	}

	// Obtener la reserva para verificar que pertenece al usuario
	reservation, err := controller.service.GetReservationByID(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("reservation not found: %s", err.Error()),
		})
		return
	}

	// Validar que el usuario solo pueda cancelar sus propias reservas
	if reservation.UserID != userIDString {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Users can only cancel their own reservations",
		})
		return
	}

	// Cancela la reserva
	if err := controller.service.CancelReservation(ctx.Request.Context(), id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error canceling reservation: %s", err.Error()),
		})
		return
	}

	// Devuelve el ID de la reserva cancelada
	ctx.JSON(http.StatusOK, gin.H{
		"message": id,
	})
}

func (controller Controller) GetReservationsByHotelID(ctx *gin.Context) {
	// Valida el ID del hotel que viene en la URL
	hotelID := strings.TrimSpace(ctx.Param("hotel_id"))

	// Obtiene las reservas por ID de hotel
	reservations, err := controller.service.GetReservationsByHotelID(ctx.Request.Context(), hotelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("error getting reservations: %s", err.Error()),
		})
		return
	}

	// Devuelve las reservas encontradas
	ctx.JSON(http.StatusOK, reservations)
}

func (controller Controller) GetReservationsByUserID(ctx *gin.Context) {
	// Valida el ID del usuario que viene en la URL
	userID := strings.TrimSpace(ctx.Param("user_id"))

	// Ownership: solo admin puede consultar reservas de otros usuarios
	userTypeAny, exists := ctx.Get("userType")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found in token"})
		return
	}
	userType, ok := userTypeAny.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type format in token"})
		return
	}

	userIDAny, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userIDFromToken, ok := userIDAny.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format in token"})
		return
	}

	if userType != "administrador" && userIDFromToken != userID {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Users can only view their own reservations",
		})
		return
	}

	// Obtiene las reservas por ID de usuario
	reservations, err := controller.service.GetReservationsByUserID(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("error getting reservations: %s", err.Error()),
		})
		return
	}

	// Devuelve las reservas encontradas
	ctx.JSON(http.StatusOK, reservations)
}

func (controller Controller) GetReservationsByUserAndHotelID(ctx *gin.Context) {
	// Valida el ID del usuario que viene en la URL
	userID := strings.TrimSpace(ctx.Param("user_id"))
	hotelID := strings.TrimSpace(ctx.Param("hotel_id"))

	// Ownership: solo admin puede consultar reservas de otros usuarios
	userTypeAny, exists := ctx.Get("userType")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found in token"})
		return
	}
	userType, ok := userTypeAny.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type format in token"})
		return
	}

	userIDAny, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userIDFromToken, ok := userIDAny.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format in token"})
		return
	}

	if userType != "administrador" && userIDFromToken != userID {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Users can only view their own reservations",
		})
		return
	}

	// Obtiene las reservas por ID de usuario y hotel
	reservations, err := controller.service.GetReservationsByUserAndHotelID(ctx.Request.Context(), hotelID, userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("error getting reservations: %s", err.Error()),
		})
		return
	}

	// Devuelve las reservas encontradas
	ctx.JSON(http.StatusOK, reservations)
}

func (controller Controller) GetAvailability(ctx *gin.Context) {
	// Valida los IDs de los hoteles que vienen en el body de la peticion
	var req struct {
		HotelIDs []string `json:"hotel_ids"`
		CheckIn  string   `json:"check_in"`
		CheckOut string   `json:"check_out"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Obtiene la disponibilidad de los hoteles
	availability, err := controller.service.GetAvailability(ctx.Request.Context(), req.HotelIDs, req.CheckIn, req.CheckOut)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error getting availability: %s", err.Error()),
		})
		return
	}

	// Devuelve la disponibilidad de los hoteles
	ctx.JSON(http.StatusOK, availability)
}
