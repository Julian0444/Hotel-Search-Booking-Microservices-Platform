package users_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	controllers "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/controllers/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	usersRepo "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	usersService "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) GetAll() ([]usersDomain.User, error) {
	args := m.Called()
	if err := args.Error(1); err != nil {
		return nil, err
	}
	return args.Get(0).([]usersDomain.User), nil
}

func (m *mockService) GetByID(id int64) (usersDomain.User, error) {
	args := m.Called(id)
	if err := args.Error(1); err != nil {
		return usersDomain.User{}, err
	}
	return args.Get(0).(usersDomain.User), nil
}

func (m *mockService) Create(request usersDomain.LoginRequest) (int64, error) {
	args := m.Called(request)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockService) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockService) Login(username, password string) (usersDomain.LoginResponse, error) {
	args := m.Called(username, password)
	if err := args.Error(1); err != nil {
		return usersDomain.LoginResponse{}, err
	}
	return args.Get(0).(usersDomain.LoginResponse), nil
}

func setupRouter(svc *mockService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	controller := controllers.NewController(svc)

	router.GET("/users", controller.GetAll)
	router.GET("/users/:id", controller.GetByID)
	router.POST("/users", controller.Create)
	router.DELETE("/users/:id", controller.Delete)
	router.POST("/login", controller.Login)

	return router
}

func TestController_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetAll").Return([]usersDomain.User{
			{ID: 1, Username: "user1", Tipo: "cliente"},
			{ID: 2, Username: "admin", Tipo: "administrador"},
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got []usersDomain.User
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got, 2)
		assert.Equal(t, int64(1), got[0].ID)
		assert.Equal(t, "administrador", got[1].Tipo)

		svc.AssertExpectations(t)
	})

	t.Run("error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetAll").Return(nil, errors.New("db error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		svc.AssertExpectations(t)
	})
}

func TestController_GetByID(t *testing.T) {
	t.Run("invalid id -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "GetByID", mock.Anything)
	})

	t.Run("not found -> 404", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetByID", int64(999)).Return(usersDomain.User{}, usersRepo.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetByID", int64(1)).Return(usersDomain.User{
			ID: 1, Username: "user1", Tipo: "cliente",
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got usersDomain.User
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, "user1", got.Username)

		svc.AssertExpectations(t)
	})
}

func TestController_Create(t *testing.T) {
	t.Run("invalid body -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("validation error -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"u","password":"p","tipo":"hacker"}`
		svc.On("Create", usersDomain.LoginRequest{
			Username: "u", Password: "p", Tipo: "hacker",
		}).Return(int64(0), errors.New("invalid tipo: hacker")).Once()

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success -> 201", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"newuser","password":"pass123"}`
		svc.On("Create", usersDomain.LoginRequest{
			Username: "newuser", Password: "pass123",
		}).Return(int64(42), nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var got map[string]int64
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, int64(42), got["id"])

		svc.AssertExpectations(t)
	})

	t.Run("create admin -> 201", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"admin","password":"secret","tipo":"administrador"}`
		svc.On("Create", usersDomain.LoginRequest{
			Username: "admin", Password: "secret", Tipo: "administrador",
		}).Return(int64(1), nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		svc.AssertExpectations(t)
	})
}

func TestController_Delete(t *testing.T) {
	t.Run("invalid id -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodDelete, "/users/abc", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Delete", mock.Anything)
	})

	t.Run("error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Delete", int64(1)).Return(errors.New("db error")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success -> 200", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Delete", int64(1)).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got map[string]int64
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, int64(1), got["id"])

		svc.AssertExpectations(t)
	})
}

func TestController_Login(t *testing.T) {
	t.Run("invalid body -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
	})

	t.Run("invalid credentials -> 401", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"user","password":"wrong"}`
		svc.On("Login", "user", "wrong").Return(usersDomain.LoginResponse{}, usersService.ErrInvalidCredentials).Once()

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success -> 200", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"user","password":"correct"}`
		expected := usersDomain.LoginResponse{
			UserID:   1,
			Username: "user",
			Token:    "jwt.token.here",
			Tipo:     "cliente",
		}
		svc.On("Login", "user", "correct").Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got usersDomain.LoginResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, expected, got)

		svc.AssertExpectations(t)
	})

	t.Run("admin login -> 200", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"admin","password":"secret"}`
		expected := usersDomain.LoginResponse{
			UserID:   2,
			Username: "admin",
			Token:    "admin.jwt.token",
			Tipo:     "administrador",
		}
		svc.On("Login", "admin", "secret").Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var got usersDomain.LoginResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, "administrador", got.Tipo)

		svc.AssertExpectations(t)
	})
}
