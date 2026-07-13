package users_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"
	usersRepo "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/repositories/users"
	service "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/services/users"
	"github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/tokenizers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func newTestService() (service.Service, *usersRepo.Mock, *usersRepo.Mock, *usersRepo.Mock, *tokenizers.Mock) {
	mainRepo := usersRepo.NewMock()
	cacheRepo := usersRepo.NewMock()
	memcachedRepo := usersRepo.NewMock()
	tokenizer := tokenizers.NewMock()

	svc := service.NewService(mainRepo, cacheRepo, memcachedRepo, tokenizer, bcrypt.MinCost)
	return svc, mainRepo, cacheRepo, memcachedRepo, tokenizer
}

func TestService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mainRepo, _, _, _ := newTestService()

		mockUsers := []usersDAO.User{
			{ID: 1, Username: "user1", Password: "hash1", Tipo: "cliente"},
			{ID: 2, Username: "admin", Password: "hash2", Tipo: "administrador"},
		}
		mainRepo.On("GetAll", mock.Anything, 20, 0).Return(mockUsers, nil).Once()
		mainRepo.On("CountAll", mock.Anything).Return(int64(2), nil).Once()

		result, total, err := svc.GetAll(context.Background(), 20, 0)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(1), result[0].ID)
		assert.Equal(t, "user1", result[0].Username)
		assert.Equal(t, "cliente", result[0].Tipo)
		assert.Equal(t, "administrador", result[1].Tipo)

		mainRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		svc, mainRepo, _, _, _ := newTestService()

		mainRepo.On("GetAll", mock.Anything, 20, 0).Return(nil, errors.New("db error")).Once()

		result, _, err := svc.GetAll(context.Background(), 20, 0)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error getting all users")

		mainRepo.AssertExpectations(t)
	})
}

func TestService_GetByID(t *testing.T) {
	t.Run("cache hit L1", func(t *testing.T) {
		svc, _, cacheRepo, memRepo, _ := newTestService()

		mockUser := usersDAO.User{ID: 1, Username: "user1", Password: "hash", Tipo: "cliente"}
		cacheRepo.On("GetByID", mock.Anything, int64(1)).Return(mockUser, nil).Once()

		result, err := svc.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, "user1", result.Username)
		assert.Equal(t, "cliente", result.Tipo)

		memRepo.AssertNotCalled(t, "GetByID", mock.Anything)
	})

	t.Run("cache miss L1, hit L2", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mockUser := usersDAO.User{ID: 1, Username: "user1", Password: "hash", Tipo: "cliente"}
		cacheRepo.On("GetByID", mock.Anything, int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByID", mock.Anything, int64(1)).Return(mockUser, nil).Once()
		cacheRepo.On("Create", mock.Anything, mockUser).Return(int64(1), nil).Once()

		result, err := svc.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, "user1", result.Username)

		mainRepo.AssertNotCalled(t, "GetByID", mock.Anything)
	})

	t.Run("cache miss L1+L2, hit DB", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mockUser := usersDAO.User{ID: 1, Username: "user1", Password: "hash", Tipo: "cliente"}
		cacheRepo.On("GetByID", mock.Anything, int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByID", mock.Anything, int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByID", mock.Anything, int64(1)).Return(mockUser, nil).Once()
		cacheRepo.On("Create", mock.Anything, mockUser).Return(int64(1), nil).Once()
		memRepo.On("Create", mock.Anything, mockUser).Return(int64(1), nil).Once()

		result, err := svc.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.ID)
		assert.Equal(t, "cliente", result.Tipo)
	})

	t.Run("DB error", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		cacheRepo.On("GetByID", mock.Anything, int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByID", mock.Anything, int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByID", mock.Anything, int64(1)).Return(usersDAO.User{}, errors.New("not found")).Once()

		result, err := svc.GetByID(context.Background(), 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error getting user by ID")
		assert.Equal(t, usersDomain.User{}, result)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("success with default tipo", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		var created usersDAO.User
		mainRepo.
			On("Create", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { created = args.Get(1).(usersDAO.User) }).
			Return(int64(1), nil).
			Once()

		cacheRepo.On("Create", mock.Anything, mock.MatchedBy(func(u usersDAO.User) bool {
			return u.ID == 1 && u.Username == "newuser" && u.Tipo == "cliente"
		})).Return(int64(1), nil).Once()

		memRepo.On("Create", mock.Anything, mock.MatchedBy(func(u usersDAO.User) bool {
			return u.ID == 1 && u.Username == "newuser" && u.Tipo == "cliente"
		})).Return(int64(1), nil).Once()

		id, err := svc.Create(context.Background(), usersDomain.LoginRequest{Username: "newuser", Password: "password"})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
		assert.Equal(t, "newuser", created.Username)
		assert.Equal(t, "cliente", created.Tipo)
		assert.NotEqual(t, "password", created.Password) // hashed
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.Password), []byte("password")))
	})

	t.Run("success with admin tipo", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Create", mock.Anything, mock.MatchedBy(func(u usersDAO.User) bool {
			return u.Tipo == "administrador"
		})).Return(int64(2), nil).Once()
		cacheRepo.On("Create", mock.Anything, mock.Anything).Return(int64(2), nil).Once()
		memRepo.On("Create", mock.Anything, mock.Anything).Return(int64(2), nil).Once()

		id, err := svc.Create(context.Background(), usersDomain.LoginRequest{
			Username: "admin",
			Password: "password",
			Tipo:     "administrador",
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(2), id)
	})

	t.Run("invalid tipo", func(t *testing.T) {
		svc, mainRepo, _, _, _ := newTestService()

		id, err := svc.Create(context.Background(), usersDomain.LoginRequest{Username: "u", Password: "p", Tipo: "hacker"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tipo")
		assert.Equal(t, int64(0), id)

		mainRepo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("empty username", func(t *testing.T) {
		svc, mainRepo, _, _, _ := newTestService()

		id, err := svc.Create(context.Background(), usersDomain.LoginRequest{Username: "", Password: "p"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username is required")
		assert.Equal(t, int64(0), id)

		mainRepo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("password too long for bcrypt", func(t *testing.T) {
		svc, mainRepo, _, _, _ := newTestService()

		longPassword := strings.Repeat("a", 73) // bcrypt opera sobre 72 bytes
		id, err := svc.Create(context.Background(), usersDomain.LoginRequest{Username: "u", Password: longPassword})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
		assert.Equal(t, int64(0), id)

		mainRepo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("DB error", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Create", mock.Anything, mock.Anything).Return(int64(0), errors.New("db error")).Once()

		id, err := svc.Create(context.Background(), usersDomain.LoginRequest{Username: "u", Password: "p"})

		assert.Error(t, err)
		assert.Equal(t, int64(0), id)

		cacheRepo.AssertNotCalled(t, "Create", mock.Anything)
		memRepo.AssertNotCalled(t, "Create", mock.Anything)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Delete", mock.Anything, int64(1)).Return(nil).Once()
		cacheRepo.On("Delete", mock.Anything, int64(1)).Return(nil).Once()
		memRepo.On("Delete", mock.Anything, int64(1)).Return(nil).Once()

		err := svc.Delete(context.Background(), 1)

		assert.NoError(t, err)
		mainRepo.AssertExpectations(t)
		cacheRepo.AssertExpectations(t)
		memRepo.AssertExpectations(t)
	})

	t.Run("DB error", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Delete", mock.Anything, int64(1)).Return(errors.New("db error")).Once()

		err := svc.Delete(context.Background(), 1)

		assert.Error(t, err)
		cacheRepo.AssertNotCalled(t, "Delete", mock.Anything)
		memRepo.AssertNotCalled(t, "Delete", mock.Anything)
	})
}

func TestService_Login(t *testing.T) {
	t.Run("success from cache", func(t *testing.T) {
		svc, _, cacheRepo, memRepo, tokenizer := newTestService()

		username := "user1"
		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 1, Username: username, Password: string(hash), Tipo: "cliente"}

		cacheRepo.On("GetByUsername", mock.Anything, username).Return(mockUser, nil).Once()
		tokenizer.On("GenerateToken", username, int64(1), "cliente").Return("token123", nil).Once()

		resp, err := svc.Login(context.Background(), username, password)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.Equal(t, "token123", resp.Token)
		assert.Equal(t, "cliente", resp.Tipo)

		memRepo.AssertNotCalled(t, "GetByUsername", mock.Anything)
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _, cacheRepo, _, tokenizer := newTestService()

		username := "admin"
		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 2, Username: username, Password: string(hash), Tipo: "administrador"}

		cacheRepo.On("GetByUsername", mock.Anything, username).Return(mockUser, nil).Once()
		tokenizer.On("GenerateToken", username, int64(2), "administrador").Return("admin_token", nil).Once()

		resp, err := svc.Login(context.Background(), username, password)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.UserID)
		assert.Equal(t, "admin_token", resp.Token)
		assert.Equal(t, "administrador", resp.Tipo)
	})

	t.Run("invalid password", func(t *testing.T) {
		svc, _, cacheRepo, _, tokenizer := newTestService()

		username := "user1"
		hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 1, Username: username, Password: string(hash), Tipo: "cliente"}

		cacheRepo.On("GetByUsername", mock.Anything, username).Return(mockUser, nil).Once()

		resp, err := svc.Login(context.Background(), username, "wrong")

		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Equal(t, usersDomain.LoginResponse{}, resp)

		tokenizer.AssertNotCalled(t, "GenerateToken", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("user not found", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		// Solo el sentinel ErrUserNotFound colapsa a credenciales inválidas (RV6)
		cacheRepo.On("GetByUsername", mock.Anything, "missing").Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByUsername", mock.Anything, "missing").Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByUsername", mock.Anything, "missing").Return(usersDAO.User{}, usersRepo.ErrUserNotFound).Once()

		resp, err := svc.Login(context.Background(), "missing", "password")

		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Equal(t, usersDomain.LoginResponse{}, resp)
	})

	t.Run("infra error is NOT invalid credentials (RV6)", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, tokenizer := newTestService()

		// MySQL caído: connection refused NO es 401 — se propaga para que el
		// controller responda 5xx
		infraErr := errors.New("dial tcp 172.19.0.2:3306: connect: connection refused")
		cacheRepo.On("GetByUsername", mock.Anything, "user1").Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByUsername", mock.Anything, "user1").Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByUsername", mock.Anything, "user1").Return(usersDAO.User{}, infraErr).Once()

		resp, err := svc.Login(context.Background(), "user1", "password")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Contains(t, err.Error(), "error getting user for login")
		assert.Equal(t, usersDomain.LoginResponse{}, resp)

		tokenizer.AssertNotCalled(t, "GenerateToken", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("empty credentials", func(t *testing.T) {
		svc, _, _, _, _ := newTestService()

		resp, err := svc.Login(context.Background(), "", "password")
		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Equal(t, usersDomain.LoginResponse{}, resp)

		resp, err = svc.Login(context.Background(), "user", "")
		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Equal(t, usersDomain.LoginResponse{}, resp)
	})

	t.Run("token generation error", func(t *testing.T) {
		svc, _, cacheRepo, _, tokenizer := newTestService()

		username := "user1"
		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 1, Username: username, Password: string(hash), Tipo: "cliente"}

		cacheRepo.On("GetByUsername", mock.Anything, username).Return(mockUser, nil).Once()
		tokenizer.On("GenerateToken", username, int64(1), "cliente").Return("", errors.New("token error")).Once()

		resp, err := svc.Login(context.Background(), username, password)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error generating token")
		assert.Equal(t, usersDomain.LoginResponse{}, resp)
	})
}

// RV9: BCRYPT_COST fuera del rango sano se clampea al default en vez de
// romper todos los registros (32 hace fallar GenerateFromPassword; 0 era el
// único caso defendido antes).
func TestService_BcryptCostClamp(t *testing.T) {
	for _, cost := range []int{32, 0} {
		mainRepo := usersRepo.NewMock()
		cacheRepo := usersRepo.NewMock()
		memRepo := usersRepo.NewMock()
		svc := service.NewService(mainRepo, cacheRepo, memRepo, tokenizers.NewMock(), cost)

		var created usersDAO.User
		mainRepo.
			On("Create", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { created = args.Get(1).(usersDAO.User) }).
			Return(int64(1), nil).
			Once()
		cacheRepo.On("Create", mock.Anything, mock.Anything).Return(int64(1), nil).Once()
		memRepo.On("Create", mock.Anything, mock.Anything).Return(int64(1), nil).Once()

		_, err := svc.Create(context.Background(), usersDomain.LoginRequest{Username: "u", Password: "password"})

		assert.NoError(t, err, "cost %d must not fail", cost)
		hashCost, costErr := bcrypt.Cost([]byte(created.Password))
		assert.NoError(t, costErr)
		assert.Equal(t, bcrypt.DefaultCost, hashCost, "cost %d must clamp to default", cost)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.Password), []byte("password")))
	}
}
