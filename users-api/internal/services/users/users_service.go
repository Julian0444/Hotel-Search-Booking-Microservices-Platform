package users

import (
	"errors"
	"fmt"
	"log"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	GetAll() ([]usersDAO.User, error)
	GetByID(id int64) (usersDAO.User, error)
	GetByUsername(username string) (usersDAO.User, error)
	Create(user usersDAO.User) (int64, error)
	Update(user usersDAO.User) error
	Delete(id int64) error
}

type Tokenizer interface {
	GenerateToken(username string, userID int64, tipo string) (string, error)
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoFieldsToUpdate   = errors.New("no fields to update")
)

type Service struct {
	mainRepository      Repository
	cacheRepository     Repository
	memcachedRepository Repository
	tokenizer           Tokenizer
	bcryptCost          int
}

func NewService(
	mainRepository Repository,
	cacheRepository Repository,
	memcachedRepository Repository,
	tokenizer Tokenizer,
	bcryptCost int,
) Service {
	return Service{
		mainRepository:      mainRepository,
		cacheRepository:     cacheRepository,
		memcachedRepository: memcachedRepository,
		tokenizer:           tokenizer,
		bcryptCost:          bcryptCost,
	}
}

func (service Service) GetAll() ([]usersDomain.UserResponse, error) {
	users, err := service.mainRepository.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting all users: %w", err)
	}

	result := make([]usersDomain.UserResponse, 0, len(users))
	for _, user := range users {
		result = append(result, service.toUserResponse(user))
	}

	return result, nil
}

func (service Service) GetByID(id int64) (usersDomain.UserResponse, error) {
	user, err := service.getByIDDAO(id)
	if err != nil {
		return usersDomain.UserResponse{}, fmt.Errorf("error getting user by ID: %w", err)
	}

	return service.toUserResponse(user), nil
}

func (service Service) GetByUsername(username string) (usersDomain.UserResponse, error) {
	user, err := service.getByUsernameDAO(username)
	if err != nil {
		return usersDomain.UserResponse{}, fmt.Errorf("error getting user by username: %w", err)
	}

	return service.toUserResponse(user), nil
}

func (service Service) Create(request usersDomain.UserCreateRequest) (int64, error) {
	if request.Username == "" {
		return 0, fmt.Errorf("username is required")
	}
	if request.Password == "" {
		return 0, fmt.Errorf("password is required")
	}

	tipo := request.Tipo
	if tipo == "" {
		tipo = "cliente"
	}
	if err := validateTipo(tipo); err != nil {
		return 0, err
	}

	passwordHash, err := service.hashPassword(request.Password)
	if err != nil {
		return 0, err
	}

	newUser := usersDAO.User{
		Username: request.Username,
		Password: passwordHash,
		Tipo:     tipo,
	}

	id, err := service.mainRepository.Create(newUser)
	if err != nil {
		return 0, fmt.Errorf("error creating user: %w", err)
	}

	newUser.ID = id
	if _, err := service.cacheRepository.Create(newUser); err != nil {
		log.Printf("warn: cache create failed (user_id=%d): %v", newUser.ID, err)
	}
	if _, err := service.memcachedRepository.Create(newUser); err != nil {
		log.Printf("warn: memcached create failed (user_id=%d): %v", newUser.ID, err)
	}

	return id, nil
}

func (service Service) Update(id int64, request usersDomain.UserUpdateRequest) error {
	if request.Username == nil && request.Password == nil && request.Tipo == nil {
		return ErrNoFieldsToUpdate
	}

	existingUser, err := service.mainRepository.GetByID(id)
	if err != nil {
		return fmt.Errorf("error retrieving existing user: %w", err)
	}

	updated := usersDAO.User{
		ID:       existingUser.ID,
		Username: existingUser.Username,
		Password: existingUser.Password,
		Tipo:     existingUser.Tipo,
	}

	if request.Username != nil {
		if *request.Username == "" {
			return fmt.Errorf("username is required")
		}
		updated.Username = *request.Username
	}

	if request.Tipo != nil {
		if *request.Tipo == "" {
			return fmt.Errorf("tipo is required")
		}
		if err := validateTipo(*request.Tipo); err != nil {
			return err
		}
		updated.Tipo = *request.Tipo
	}

	if request.Password != nil {
		if *request.Password == "" {
			return fmt.Errorf("password is required")
		}
		hash, err := service.hashPassword(*request.Password)
		if err != nil {
			return err
		}
		updated.Password = hash
	}

	if err := service.mainRepository.Update(updated); err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	// Best-effort: limpiar mapping viejo (username) y setear mapping nuevo en ambos caches.
	// Truco: “seed” con existingUser para asegurar que Delete(id) pueda borrar el username viejo.
	if _, err := service.cacheRepository.Create(existingUser); err != nil {
		log.Printf("warn: cache seed failed (user_id=%d): %v", existingUser.ID, err)
	}
	if _, err := service.memcachedRepository.Create(existingUser); err != nil {
		log.Printf("warn: memcached seed failed (user_id=%d): %v", existingUser.ID, err)
	}
	if err := service.cacheRepository.Delete(id); err != nil {
		log.Printf("warn: cache delete failed (user_id=%d): %v", id, err)
	}
	if err := service.memcachedRepository.Delete(id); err != nil {
		log.Printf("warn: memcached delete failed (user_id=%d): %v", id, err)
	}
	if err := service.cacheRepository.Update(updated); err != nil {
		log.Printf("warn: cache update failed (user_id=%d): %v", id, err)
	}
	if err := service.memcachedRepository.Update(updated); err != nil {
		log.Printf("warn: memcached update failed (user_id=%d): %v", id, err)
	}

	return nil
}

func (service Service) Delete(id int64) error {
	if err := service.mainRepository.Delete(id); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	// Best-effort cache invalidation
	if err := service.cacheRepository.Delete(id); err != nil {
		log.Printf("warn: cache delete failed (user_id=%d): %v", id, err)
	}
	if err := service.memcachedRepository.Delete(id); err != nil {
		log.Printf("warn: memcached delete failed (user_id=%d): %v", id, err)
	}

	return nil
}

func (service Service) Login(username string, password string) (usersDomain.LoginResponse, error) {
	if username == "" || password == "" {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	user, err := service.getByUsernameDAO(username)
	if err != nil {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	token, err := service.tokenizer.GenerateToken(user.Username, user.ID, user.Tipo)
	if err != nil {
		return usersDomain.LoginResponse{}, fmt.Errorf("error generating token: %w", err)
	}

	return usersDomain.LoginResponse{
		UserID:   user.ID,
		Username: user.Username,
		Token:    token,
		Tipo:     user.Tipo,
	}, nil
}

func (service Service) getByIDDAO(id int64) (usersDAO.User, error) {
	if user, err := service.cacheRepository.GetByID(id); err == nil {
		return user, nil
	}

	if user, err := service.memcachedRepository.GetByID(id); err == nil {
		if _, err := service.cacheRepository.Create(user); err != nil {
			log.Printf("warn: cache create after memcached hit failed (user_id=%d): %v", id, err)
		}
		return user, nil
	}

	user, err := service.mainRepository.GetByID(id)
	if err != nil {
		return usersDAO.User{}, err
	}

	if _, err := service.cacheRepository.Create(user); err != nil {
		log.Printf("warn: cache create after db hit failed (user_id=%d): %v", id, err)
	}
	if _, err := service.memcachedRepository.Create(user); err != nil {
		log.Printf("warn: memcached create after db hit failed (user_id=%d): %v", id, err)
	}

	return user, nil
}

func (service Service) getByUsernameDAO(username string) (usersDAO.User, error) {
	if user, err := service.cacheRepository.GetByUsername(username); err == nil {
		return user, nil
	}

	if user, err := service.memcachedRepository.GetByUsername(username); err == nil {
		if _, err := service.cacheRepository.Create(user); err != nil {
			log.Printf("warn: cache create after memcached hit failed (username=%s): %v", username, err)
		}
		return user, nil
	}

	user, err := service.mainRepository.GetByUsername(username)
	if err != nil {
		return usersDAO.User{}, err
	}

	if _, err := service.cacheRepository.Create(user); err != nil {
		log.Printf("warn: cache create after db hit failed (username=%s): %v", username, err)
	}
	if _, err := service.memcachedRepository.Create(user); err != nil {
		log.Printf("warn: memcached create after db hit failed (username=%s): %v", username, err)
	}

	return user, nil
}

func validateTipo(tipo string) error {
	switch tipo {
	case "cliente", "administrador":
		return nil
	default:
		return fmt.Errorf("invalid tipo: %s", tipo)
	}
}

func (service Service) hashPassword(plain string) (string, error) {
	cost := service.bcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("error hashing password: %w", err)
	}
	return string(hash), nil
}

func (service Service) toUserResponse(user usersDAO.User) usersDomain.UserResponse {
	return usersDomain.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Tipo:     user.Tipo,
	}
}
