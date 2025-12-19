package users_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func (m *mockService) GetAll() ([]usersDomain.UserResponse, error) {
	args := m.Called()
	if err := args.Error(1); err != nil {
		return nil, err
	}
	return args.Get(0).([]usersDomain.UserResponse), nil
}

func (m *mockService) GetByID(id int64) (usersDomain.UserResponse, error) {
	args := m.Called(id)
	if err := args.Error(1); err != nil {
		return usersDomain.UserResponse{}, err
	}
	return args.Get(0).(usersDomain.UserResponse), nil
}

func (m *mockService) Create(request usersDomain.UserCreateRequest) (int64, error) {
	args := m.Called(request)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockService) Update(id int64, request usersDomain.UserUpdateRequest) error {
	args := m.Called(id, request)
	return args.Error(0)
}

func (m *mockService) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockService) Login(username string, password string) (usersDomain.LoginResponse, error) {
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
	router.PUT("/users/:id", controller.Update)
	router.DELETE("/users/:id", controller.Delete)
	router.POST("/login", controller.Login)

	return router
}

func TestUsersController_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetAll").Return([]usersDomain.UserResponse{
			{ID: 1, Username: "u1", Tipo: "cliente"},
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var got []usersDomain.UserResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Len(t, got, 1)
		assert.Equal(t, int64(1), got[0].ID)

		svc.AssertExpectations(t)
	})

	t.Run("service error -> 500", func(t *testing.T) {
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

func TestUsersController_GetByID(t *testing.T) {
	t.Run("bad id -> 400", func(t *testing.T) {
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

		svc.On("GetByID", int64(1)).Return(usersDomain.UserResponse{}, usersRepo.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("service error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetByID", int64(1)).Return(usersDomain.UserResponse{}, errors.New("db error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("GetByID", int64(1)).Return(usersDomain.UserResponse{ID: 1, Username: "u1", Tipo: "cliente"}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var got usersDomain.UserResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, "u1", got.Username)
		assert.Equal(t, "cliente", got.Tipo)
		svc.AssertExpectations(t)
	})
}

func TestUsersController_Create(t *testing.T) {
	t.Run("bad body -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("service validation -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"u","password":"p","tipo":"hacker"}`
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		svc.On("Create", usersDomain.UserCreateRequest{Username: "u", Password: "p", Tipo: "hacker"}).
			Return(int64(0), fmt.Errorf("invalid tipo: hacker")).
			Once()

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("service error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"u","password":"p"}`
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		svc.On("Create", usersDomain.UserCreateRequest{Username: "u", Password: "p", Tipo: ""}).
			Return(int64(0), errors.New("db error")).
			Once()

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success -> 201", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		body := `{"username":"u","password":"p"}`
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		svc.On("Create", usersDomain.UserCreateRequest{Username: "u", Password: "p", Tipo: ""}).
			Return(int64(123), nil).
			Once()

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		var got map[string]int64
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, int64(123), got["id"])
		svc.AssertExpectations(t)
	})
}

func TestUsersController_Update(t *testing.T) {
	t.Run("bad id -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPut, "/users/abc", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("bad body -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("no fields -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Update", int64(1), mock.Anything).Return(usersService.ErrNoFieldsToUpdate).Once()

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("not found -> 404", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Update", int64(1), mock.Anything).Return(usersRepo.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{"username":"new"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success -> 200", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.
			On("Update", int64(1), mock.MatchedBy(func(r usersDomain.UserUpdateRequest) bool {
				return r.Username != nil && *r.Username == "new" && r.Password == nil && r.Tipo == nil
			})).
			Return(nil).
			Once()

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{"username":"new"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var got map[string]int64
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, int64(1), got["id"])
		svc.AssertExpectations(t)
	})
}

func TestUsersController_Delete(t *testing.T) {
	t.Run("bad id -> 400", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodDelete, "/users/abc", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "Delete", mock.Anything)
	})

	t.Run("service error -> 500", func(t *testing.T) {
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

func TestUsersController_Login(t *testing.T) {
	t.Run("bad body -> 400", func(t *testing.T) {
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

		svc.On("Login", "u", "p").Return(usersDomain.LoginResponse{}, usersService.ErrInvalidCredentials).Once()

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"u","password":"p"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("token error -> 500", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		svc.On("Login", "u", "p").Return(usersDomain.LoginResponse{}, errors.New("token error")).Once()

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"u","password":"p"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("success -> 200", func(t *testing.T) {
		svc := &mockService{}
		router := setupRouter(svc)

		resp := usersDomain.LoginResponse{UserID: 1, Username: "u", Token: "t", Tipo: "cliente"}
		svc.On("Login", "u", "p").Return(resp, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"u","password":"p"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var got usersDomain.LoginResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
		assert.Equal(t, resp, got)
		svc.AssertExpectations(t)
	})
}


