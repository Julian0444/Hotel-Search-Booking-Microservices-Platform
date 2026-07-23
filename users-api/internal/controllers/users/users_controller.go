package users

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/apperr"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	usersRepo "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	usersService "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"

	"github.com/gin-gonic/gin"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// Service define las operaciones que el controller necesita.
type Service interface {
	GetAll(ctx context.Context, limit, offset int) ([]usersDomain.User, int64, error)
	GetByID(ctx context.Context, id int64) (usersDomain.User, error)
	Create(ctx context.Context, request usersDomain.LoginRequest) (int64, error)
	Delete(ctx context.Context, id int64) error
	Login(ctx context.Context, username, password string) (usersDomain.LoginResponse, error)
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

// abortIfTimedOut mapea un deadline vencido a 503 (fallar rápido, no colgar — R1).
func abortIfTimedOut(ctx *gin.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		apperr.Abort(ctx, http.StatusServiceUnavailable, "timeout", "request timed out", err)
		return true
	}
	return false
}

// paginationParams parsea ?limit y ?offset con defaults y clamp (max 100).
func paginationParams(ctx *gin.Context) (int, int) {
	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	if err != nil || limit < 1 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	offset, err := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	return limit, offset
}

// GetAll retorna una página de usuarios con el envelope estándar (A5): el
// total viaja en meta y reemplaza al interino X-Total-Count del plan 03.
// GET /api/v1/users?limit=20&offset=0
func (c Controller) GetAll(ctx *gin.Context) {
	limit, offset := paginationParams(ctx)

	users, total, err := c.service.GetAll(ctx.Request.Context(), limit, offset)
	if err != nil {
		if abortIfTimedOut(ctx, err) {
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting users", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": users,
		"meta": gin.H{"total": total, "limit": limit, "offset": offset},
	})
}

// GetByID retorna un usuario por ID.
// GET /users/:id
func (c Controller) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_id", "invalid user id", nil)
		return
	}

	user, err := c.service.GetByID(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, usersRepo.ErrUserNotFound) {
			apperr.Abort(ctx, http.StatusNotFound, "user_not_found", "user not found", err)
			return
		}
		if abortIfTimedOut(ctx, err) {
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error getting user", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": user})
}

// Create registra un nuevo usuario.
// POST /users
func (c Controller) Create(ctx *gin.Context) {
	var request usersDomain.RegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body",
			"invalid request body: username (3-50 chars) and password (8-72 chars) are required", err)
		return
	}

	// El registro público nunca elige rol: siempre cliente. Admins solo vía seed.
	id, err := c.service.Create(ctx.Request.Context(), usersDomain.LoginRequest{
		Username: request.Username,
		Password: request.Password,
		Tipo:     "cliente",
	})
	if err != nil {
		// Errores de validación -> 400 (el texto es nuestro, no filtra internals)
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid tipo") || strings.Contains(err.Error(), "too long") {
			apperr.Abort(ctx, http.StatusBadRequest, "invalid_body", err.Error(), nil)
			return
		}
		// Duplicado de username -> 409
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			apperr.Abort(ctx, http.StatusConflict, "username_taken", "username already exists", err)
			return
		}
		if abortIfTimedOut(ctx, err) {
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error creating user", err)
		return
	}

	// 201 con Location del recurso creado (A6); el id sale como string (A7)
	ctx.Header("Location", "/api/v1/users/"+strconv.FormatInt(id, 10))
	ctx.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": strconv.FormatInt(id, 10)}})
}

// Delete elimina un usuario por ID.
// DELETE /users/:id
func (c Controller) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_id", "invalid user id", nil)
		return
	}

	if err := c.service.Delete(ctx.Request.Context(), id); err != nil {
		if abortIfTimedOut(ctx, err) {
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error deleting user", err)
		return
	}

	// DELETE exitoso → 204 sin body (A6)
	ctx.Status(http.StatusNoContent)
}

// Login autentica un usuario y retorna un JWT.
// POST /login
func (c Controller) Login(ctx *gin.Context) {
	var request usersDomain.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		apperr.Abort(ctx, http.StatusBadRequest, "invalid_body", "invalid request body", err)
		return
	}

	response, err := c.service.Login(ctx.Request.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, usersService.ErrInvalidCredentials) {
			apperr.Abort(ctx, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", nil)
			return
		}
		if abortIfTimedOut(ctx, err) {
			return
		}
		apperr.Abort(ctx, http.StatusInternalServerError, "internal", "error during login", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": response})
}
