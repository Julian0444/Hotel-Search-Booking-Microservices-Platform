package users

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	usersRepo "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	usersService "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"

	"github.com/gin-gonic/gin"
)

type Service interface {
	GetAll() ([]usersDomain.UserResponse, error)
	GetByID(id int64) (usersDomain.UserResponse, error)
	Create(request usersDomain.UserCreateRequest) (int64, error)
	Update(id int64, request usersDomain.UserUpdateRequest) error
	Delete(id int64) error
	Login(username string, password string) (usersDomain.LoginResponse, error)
}

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{
		service: service,
	}
}

func (controller Controller) GetAll(c *gin.Context) {
	// Invoke service
	users, err := controller.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error getting all users: %s", err.Error()),
		})
		return
	}

	// Send response
	c.JSON(http.StatusOK, users)
}

func (controller Controller) GetByID(c *gin.Context) {
	// Parse user ID from HTTP request
	userID := c.Param("id")
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Invoke service
	user, err := controller.service.GetByID(id)
	if err != nil {
		if errors.Is(err, usersRepo.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send user
	c.JSON(http.StatusOK, user)
}

func (controller Controller) Create(c *gin.Context) {
	// Parse user from HTTP Request
	var request usersDomain.UserCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Invoke service
	id, err := controller.service.Create(request)
	if err != nil {
		// Validaciones del service (required/invalid tipo) -> 400
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid tipo") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error creating user: %s", err.Error())})
		return
	}

	// Send ID
	c.JSON(http.StatusCreated, gin.H{
		"id": id,
	})
}

func (controller Controller) Update(c *gin.Context) {
	// Parse user ID from HTTP request
	userID := c.Param("id")
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Parse updated user data from HTTP request
	var request usersDomain.UserUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Invoke service
	if err := controller.service.Update(id, request); err != nil {
		if errors.Is(err, usersService.ErrNoFieldsToUpdate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, usersRepo.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid tipo") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error updating user: %s", err.Error())})
		return
	}

	// Send response
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (controller Controller) Delete(c *gin.Context) {
	// Parse user ID from HTTP request
	userID := c.Param("id")
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Invoke service
	if err := controller.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error deleting user: %s", err.Error()),
		})
		return
	}

	// Send response
	c.JSON(http.StatusOK, gin.H{
		"id": id,
	})
}

func (controller Controller) Login(c *gin.Context) {
	// Parse user from HTTP request
	var request usersDomain.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Invoke service
	response, err := controller.service.Login(request.Username, request.Password)
	if err != nil {
		if errors.Is(err, usersService.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send login with token
	c.JSON(http.StatusOK, response)
}
