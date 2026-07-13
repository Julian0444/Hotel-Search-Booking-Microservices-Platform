package users

import (
	"context"

	"github.com/stretchr/testify/mock"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
)

// Mock implementa la interfaz Repository para testing.
type Mock struct {
	mock.Mock
}

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) GetAll(ctx context.Context, limit, offset int) ([]usersDAO.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usersDAO.User), args.Error(1)
}

func (m *Mock) CountAll(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Mock) GetByID(ctx context.Context, id int64) (usersDAO.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(usersDAO.User), args.Error(1)
}

func (m *Mock) GetByUsername(ctx context.Context, username string) (usersDAO.User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(usersDAO.User), args.Error(1)
}

func (m *Mock) Create(ctx context.Context, user usersDAO.User) (int64, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Mock) Update(ctx context.Context, user usersDAO.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *Mock) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
