package users_test

// Mocks de testify SOLO en archivos _test (C13): antes vivían como código de
// producción en repositories/users y tokenizers, y metían testify (y su árbol
// de deps) en el binario de users-api.

import (
	"context"

	"github.com/stretchr/testify/mock"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
)

// repoMock implementa la interfaz Repository del service.
type repoMock struct {
	mock.Mock
}

func newRepoMock() *repoMock {
	return &repoMock{}
}

func (m *repoMock) GetAll(ctx context.Context, limit, offset int) ([]usersDAO.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usersDAO.User), args.Error(1)
}

func (m *repoMock) CountAll(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *repoMock) GetByID(ctx context.Context, id int64) (usersDAO.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(usersDAO.User), args.Error(1)
}

func (m *repoMock) GetByUsername(ctx context.Context, username string) (usersDAO.User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(usersDAO.User), args.Error(1)
}

func (m *repoMock) Create(ctx context.Context, user usersDAO.User) (int64, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(int64), args.Error(1)
}

func (m *repoMock) Update(ctx context.Context, user usersDAO.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *repoMock) Delete(ctx context.Context, user usersDAO.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// tokenizerMock implementa la interfaz Tokenizer del service.
type tokenizerMock struct {
	mock.Mock
}

func newTokenizerMock() *tokenizerMock {
	return &tokenizerMock{}
}

func (m *tokenizerMock) GenerateToken(username string, userID int64, tipo string) (string, error) {
	args := m.Called(username, userID, tipo)
	return args.String(0), args.Error(1)
}
