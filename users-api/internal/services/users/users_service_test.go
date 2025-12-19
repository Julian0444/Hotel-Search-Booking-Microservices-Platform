package users_test

import (
	"errors"
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

func strPtr(v string) *string { return &v }

func TestService(t *testing.T) {
	t.Run("GetAll - Success", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mockUsers := []usersDAO.User{
			{ID: 1, Username: "user1", Password: "hash1", Tipo: "cliente"},
			{ID: 2, Username: "user2", Password: "hash2", Tipo: "administrador"},
		}
		mainRepo.On("GetAll").Return(mockUsers, nil).Once()

		result, err := svc.GetAll()

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(1), result[0].ID)
		assert.Equal(t, "user1", result[0].Username)
		assert.Equal(t, "cliente", result[0].Tipo)
		assert.Equal(t, "administrador", result[1].Tipo)

		mainRepo.AssertExpectations(t)
		cacheRepo.AssertExpectations(t)
		memRepo.AssertExpectations(t)
	})

	t.Run("GetAll - Error", func(t *testing.T) {
		svc, mainRepo, _, _, _ := newTestService()

		mainRepo.On("GetAll").Return(nil, errors.New("db error")).Once()

		result, err := svc.GetAll()

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "error getting all users: db error", err.Error())
		mainRepo.AssertExpectations(t)
	})

	t.Run("GetByID - Success from Cache (L1)", func(t *testing.T) {
		svc, _, cacheRepo, memRepo, _ := newTestService()

		cacheRepo.On("GetByID", int64(1)).Return(usersDAO.User{ID: 1, Username: "user1", Password: "hash", Tipo: "cliente"}, nil).Once()

		result, err := svc.GetByID(1)

		assert.NoError(t, err)
		assert.Equal(t, "user1", result.Username)
		assert.Equal(t, "cliente", result.Tipo)
		memRepo.AssertNotCalled(t, "GetByID", mock.Anything)
	})

	t.Run("GetByID - Cache miss, Memcached hit (L2)", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mockUser := usersDAO.User{ID: 1, Username: "user1", Password: "hash", Tipo: "cliente"}
		cacheRepo.On("GetByID", int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByID", int64(1)).Return(mockUser, nil).Once()
		cacheRepo.On("Create", mockUser).Return(int64(1), nil).Once()

		result, err := svc.GetByID(1)

		assert.NoError(t, err)
		assert.Equal(t, "user1", result.Username)
		mainRepo.AssertNotCalled(t, "GetByID", mock.Anything)
	})

	t.Run("GetByID - Cache miss, Memcached miss, DB hit", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mockUser := usersDAO.User{ID: 1, Username: "user1", Password: "hash", Tipo: "cliente"}
		cacheRepo.On("GetByID", int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByID", int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByID", int64(1)).Return(mockUser, nil).Once()
		cacheRepo.On("Create", mockUser).Return(int64(1), nil).Once()
		memRepo.On("Create", mockUser).Return(int64(1), nil).Once()

		result, err := svc.GetByID(1)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.ID)
		assert.Equal(t, "cliente", result.Tipo)
	})

	t.Run("GetByID - DB error", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		cacheRepo.On("GetByID", int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByID", int64(1)).Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByID", int64(1)).Return(usersDAO.User{}, errors.New("db error")).Once()

		result, err := svc.GetByID(1)

		assert.Error(t, err)
		assert.Equal(t, "error getting user by ID: db error", err.Error())
		assert.Equal(t, usersDomain.UserResponse{}, result)
	})

	t.Run("GetByUsername - Cache miss, DB hit", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mockUser := usersDAO.User{ID: 10, Username: "u", Password: "hash", Tipo: "cliente"}
		cacheRepo.On("GetByUsername", "u").Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByUsername", "u").Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByUsername", "u").Return(mockUser, nil).Once()
		cacheRepo.On("Create", mockUser).Return(int64(10), nil).Once()
		memRepo.On("Create", mockUser).Return(int64(10), nil).Once()

		result, err := svc.GetByUsername("u")

		assert.NoError(t, err)
		assert.Equal(t, int64(10), result.ID)
		assert.Equal(t, "u", result.Username)
	})

	t.Run("Create - Success (hash bcrypt + caches)", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		var created usersDAO.User
		mainRepo.
			On("Create", mock.Anything).
			Run(func(args mock.Arguments) { created = args.Get(0).(usersDAO.User) }).
			Return(int64(1), nil).
			Once()

		cacheRepo.On("Create", mock.MatchedBy(func(u usersDAO.User) bool {
			return u.ID == 1 && u.Username == "newuser" && u.Tipo == "cliente" && u.Password != ""
		})).Return(int64(1), nil).Once()

		memRepo.On("Create", mock.MatchedBy(func(u usersDAO.User) bool {
			return u.ID == 1 && u.Username == "newuser" && u.Tipo == "cliente" && u.Password != ""
		})).Return(int64(1), nil).Once()

		id, err := svc.Create(usersDomain.UserCreateRequest{Username: "newuser", Password: "password"})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
		assert.Equal(t, "newuser", created.Username)
		assert.Equal(t, "cliente", created.Tipo)
		assert.NotEqual(t, "password", created.Password)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.Password), []byte("password")))
	})

	t.Run("Create - Invalid tipo", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		id, err := svc.Create(usersDomain.UserCreateRequest{Username: "u", Password: "p", Tipo: "hacker"})

		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		mainRepo.AssertNotCalled(t, "Create", mock.Anything)
		cacheRepo.AssertNotCalled(t, "Create", mock.Anything)
		memRepo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("Create - DB error", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Create", mock.Anything).Return(int64(0), errors.New("db error")).Once()

		id, err := svc.Create(usersDomain.UserCreateRequest{Username: "u", Password: "p"})

		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		cacheRepo.AssertNotCalled(t, "Create", mock.Anything)
		memRepo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("Update - No fields", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		err := svc.Update(1, usersDomain.UserUpdateRequest{})

		assert.ErrorIs(t, err, service.ErrNoFieldsToUpdate)
		mainRepo.AssertNotCalled(t, "GetByID", mock.Anything)
		cacheRepo.AssertNotCalled(t, "Update", mock.Anything)
		memRepo.AssertNotCalled(t, "Update", mock.Anything)
	})

	t.Run("Update - Success (username + password)", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		oldHash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.MinCost)
		existing := usersDAO.User{ID: 1, Username: "old", Password: string(oldHash), Tipo: "cliente"}
		mainRepo.On("GetByID", int64(1)).Return(existing, nil).Once()

		var updated usersDAO.User
		mainRepo.
			On("Update", mock.Anything).
			Run(func(args mock.Arguments) { updated = args.Get(0).(usersDAO.User) }).
			Return(nil).
			Once()

		// seed + delete + update (best-effort) en ambos caches
		cacheRepo.On("Create", existing).Return(int64(1), nil).Once()
		memRepo.On("Create", existing).Return(int64(1), nil).Once()
		cacheRepo.On("Delete", int64(1)).Return(nil).Once()
		memRepo.On("Delete", int64(1)).Return(nil).Once()
		cacheRepo.On("Update", mock.Anything).Return(nil).Once()
		memRepo.On("Update", mock.Anything).Return(nil).Once()

		err := svc.Update(1, usersDomain.UserUpdateRequest{
			Username: strPtr("new"),
			Password: strPtr("newpass"),
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), updated.ID)
		assert.Equal(t, "new", updated.Username)
		assert.Equal(t, "cliente", updated.Tipo)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("newpass")))
	})

	t.Run("Delete - Success (best-effort cache)", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Delete", int64(1)).Return(nil).Once()
		cacheRepo.On("Delete", int64(1)).Return(nil).Once()
		memRepo.On("Delete", int64(1)).Return(nil).Once()

		err := svc.Delete(1)

		assert.NoError(t, err)
		mainRepo.AssertExpectations(t)
		cacheRepo.AssertExpectations(t)
		memRepo.AssertExpectations(t)
	})

	t.Run("Delete - DB error", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		mainRepo.On("Delete", int64(1)).Return(errors.New("db error")).Once()

		err := svc.Delete(1)

		assert.Error(t, err)
		cacheRepo.AssertNotCalled(t, "Delete", mock.Anything)
		memRepo.AssertNotCalled(t, "Delete", mock.Anything)
	})

	t.Run("Login - Success from Cache", func(t *testing.T) {
		svc, _, cacheRepo, memRepo, tokenizer := newTestService()

		username := "user1"
		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 1, Username: username, Password: string(hash), Tipo: "cliente"}

		cacheRepo.On("GetByUsername", username).Return(mockUser, nil).Once()
		tokenizer.On("GenerateToken", username, int64(1), "cliente").Return("token", nil).Once()

		resp, err := svc.Login(username, password)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.Equal(t, "token", resp.Token)
		assert.Equal(t, "cliente", resp.Tipo)
		memRepo.AssertNotCalled(t, "GetByUsername", mock.Anything)
	})

	t.Run("Login - Invalid password", func(t *testing.T) {
		svc, _, cacheRepo, _, tokenizer := newTestService()

		username := "user1"
		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 1, Username: username, Password: string(hash), Tipo: "cliente"}

		cacheRepo.On("GetByUsername", username).Return(mockUser, nil).Once()

		resp, err := svc.Login(username, "wrong")

		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Equal(t, usersDomain.LoginResponse{}, resp)
		tokenizer.AssertNotCalled(t, "GenerateToken", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Login - User not found", func(t *testing.T) {
		svc, mainRepo, cacheRepo, memRepo, _ := newTestService()

		cacheRepo.On("GetByUsername", "missing").Return(usersDAO.User{}, errors.New("miss")).Once()
		memRepo.On("GetByUsername", "missing").Return(usersDAO.User{}, errors.New("miss")).Once()
		mainRepo.On("GetByUsername", "missing").Return(usersDAO.User{}, errors.New("not found")).Once()

		resp, err := svc.Login("missing", "password")

		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Equal(t, usersDomain.LoginResponse{}, resp)
	})

	t.Run("Login - Token generation error", func(t *testing.T) {
		svc, _, cacheRepo, _, tokenizer := newTestService()

		username := "user1"
		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 1, Username: username, Password: string(hash), Tipo: "cliente"}

		cacheRepo.On("GetByUsername", username).Return(mockUser, nil).Once()
		tokenizer.On("GenerateToken", username, int64(1), "cliente").Return("", errors.New("token error")).Once()

		resp, err := svc.Login(username, password)

		assert.Error(t, err)
		assert.Equal(t, "error generating token: token error", err.Error())
		assert.Equal(t, usersDomain.LoginResponse{}, resp)
	})

	t.Run("Login - Administrator user", func(t *testing.T) {
		svc, _, cacheRepo, _, tokenizer := newTestService()

		username := "admin"
		password := "password"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		mockUser := usersDAO.User{ID: 2, Username: username, Password: string(hash), Tipo: "administrador"}

		cacheRepo.On("GetByUsername", username).Return(mockUser, nil).Once()
		tokenizer.On("GenerateToken", username, int64(2), "administrador").Return("admin_token", nil).Once()

		resp, err := svc.Login(username, password)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.UserID)
		assert.Equal(t, "admin_token", resp.Token)
		assert.Equal(t, "administrador", resp.Tipo)
	})
}
