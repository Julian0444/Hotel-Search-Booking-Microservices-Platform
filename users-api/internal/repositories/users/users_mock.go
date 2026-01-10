package users

import (
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

func (m *Mock) GetAll() ([]usersDAO.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usersDAO.User), args.Error(1)
}

func (m *Mock) GetByID(id int64) (usersDAO.User, error) {
	args := m.Called(id)
	return args.Get(0).(usersDAO.User), args.Error(1)
}

func (m *Mock) GetByUsername(username string) (usersDAO.User, error) {
	args := m.Called(username)
	return args.Get(0).(usersDAO.User), args.Error(1)
}

func (m *Mock) Create(user usersDAO.User) (int64, error) {
	args := m.Called(user)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Mock) Update(user usersDAO.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *Mock) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}
