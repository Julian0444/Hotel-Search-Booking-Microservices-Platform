package users

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	usersRepo "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	usersService "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"

	"github.com/gin-gonic/gin"
)

// Service define las operaciones que el controller necesita.
type Service interface {
	GetAll() ([]usersDomain.User, error)
	GetByID(id int64) (usersDomain.User, error)
	Create(request usersDomain.LoginRequest) (int64, error)
	Delete(id int64) error
	Login(username, password string) (usersDomain.LoginResponse, error)
}

// Controller maneja las peticiones HTTP de usuarios.
type Controller struct {
	service Service
}

// NewController crea una nueva instancia del controller.
func NewController(service Service) Controller {
	return Controller{
		service: service,
	}
}

// GetAll retorna todos los usuarios.
// GET /users
func (c Controller) GetAll(ctx *gin.Context) {
	users, err := c.service.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "error getting users",
		})
		return
	}

	ctx.JSON(http.StatusOK, users)
}

// GetByID retorna un usuario por ID.
// GET /users/:id
func (c Controller) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	user, err := c.service.GetByID(id)
	if err != nil {
		if errors.Is(err, usersRepo.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error getting user"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// Create registra un nuevo usuario.
// POST /users
func (c Controller) Create(ctx *gin.Context) {
	var request usersDomain.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	id, err := c.service.Create(request)
	if err != nil {
		// Errores de validación -> 400
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid tipo") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Duplicado de username -> 409
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			ctx.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error creating user"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"id": id})
}

// Delete elimina un usuario por ID.
// DELETE /users/:id
func (c Controller) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	if err := c.service.Delete(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "error deleting user",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"id": id})
}

// Login autentica un usuario y retorna un JWT.
// POST /login
func (c Controller) Login(ctx *gin.Context) {
	var request usersDomain.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	response, err := c.service.Login(request.Username, request.Password)
	if err != nil {
		if errors.Is(err, usersService.ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error during login"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
