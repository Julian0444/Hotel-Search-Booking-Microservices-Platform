package users

import (
	"errors"
	"fmt"
	"log"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"

	"golang.org/x/crypto/bcrypt"
)

// Repository define las operaciones de persistencia de usuarios.
type Repository interface {
	GetAll() ([]usersDAO.User, error)
	GetByID(id int64) (usersDAO.User, error)
	GetByUsername(username string) (usersDAO.User, error)
	Create(user usersDAO.User) (int64, error)
	Update(user usersDAO.User) error
	Delete(id int64) error
}

// Tokenizer define la generación de JWT tokens.
type Tokenizer interface {
	GenerateToken(username string, userID int64, tipo string) (string, error)
}

// Errores exportados para manejo en controller.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Service encapsula la lógica de negocio de usuarios.
type Service struct {
	mainRepository      Repository
	cacheRepository     Repository
	memcachedRepository Repository
	tokenizer           Tokenizer
	bcryptCost          int
}

// NewService crea una nueva instancia del servicio.
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

// GetAll retorna todos los usuarios (sin passwords).
func (s Service) GetAll() ([]usersDomain.User, error) {
	users, err := s.mainRepository.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting all users: %w", err)
	}

	result := make([]usersDomain.User, 0, len(users))
	for _, user := range users {
		result = append(result, s.toUser(user))
	}

	return result, nil
}

// GetByID retorna un usuario por ID (sin password).
func (s Service) GetByID(id int64) (usersDomain.User, error) {
	user, err := s.getByIDFromCaches(id)
	if err != nil {
		return usersDomain.User{}, fmt.Errorf("error getting user by ID: %w", err)
	}

	return s.toUser(user), nil
}

// Create registra un nuevo usuario con password hasheado.
func (s Service) Create(request usersDomain.LoginRequest) (int64, error) {
	if request.Username == "" {
		return 0, fmt.Errorf("username is required")
	}
	if request.Password == "" {
		return 0, fmt.Errorf("password is required")
	}

	// Default tipo = cliente
	tipo := request.Tipo
	if tipo == "" {
		tipo = "cliente"
	}
	if err := validateTipo(tipo); err != nil {
		return 0, err
	}

	// Hash password
	passwordHash, err := s.hashPassword(request.Password)
	if err != nil {
		return 0, err
	}

	newUser := usersDAO.User{
		Username: request.Username,
		Password: passwordHash,
		Tipo:     tipo,
	}

	// Persistir en DB
	id, err := s.mainRepository.Create(newUser)
	if err != nil {
		return 0, fmt.Errorf("error creating user: %w", err)
	}

	// Best-effort: poblar caches
	newUser.ID = id
	s.populateCaches(newUser)

	return id, nil
}

// Delete elimina un usuario por ID.
func (s Service) Delete(id int64) error {
	if err := s.mainRepository.Delete(id); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	// Best-effort: invalidar caches
	s.invalidateCaches(id)

	return nil
}

// Login valida credenciales y retorna un JWT token.
func (s Service) Login(username, password string) (usersDomain.LoginResponse, error) {
	if username == "" || password == "" {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	user, err := s.getByUsernameFromCaches(username)
	if err != nil {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	token, err := s.tokenizer.GenerateToken(user.Username, user.ID, user.Tipo)
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

// --- Métodos internos ---

// getByIDFromCaches busca en L1 -> L2 -> DB y puebla caches en miss.
func (s Service) getByIDFromCaches(id int64) (usersDAO.User, error) {
	// L1: cache in-process
	if user, err := s.cacheRepository.GetByID(id); err == nil {
		return user, nil
	}

	// L2: memcached
	if user, err := s.memcachedRepository.GetByID(id); err == nil {
		s.cacheRepository.Create(user) // best-effort
		return user, nil
	}

	// DB: source of truth
	user, err := s.mainRepository.GetByID(id)
	if err != nil {
		return usersDAO.User{}, err
	}

	s.populateCaches(user)
	return user, nil
}

// getByUsernameFromCaches busca en L1 -> L2 -> DB y puebla caches en miss.
func (s Service) getByUsernameFromCaches(username string) (usersDAO.User, error) {
	// L1: cache in-process
	if user, err := s.cacheRepository.GetByUsername(username); err == nil {
		return user, nil
	}

	// L2: memcached
	if user, err := s.memcachedRepository.GetByUsername(username); err == nil {
		s.cacheRepository.Create(user) // best-effort
		return user, nil
	}

	// DB: source of truth
	user, err := s.mainRepository.GetByUsername(username)
	if err != nil {
		return usersDAO.User{}, err
	}

	s.populateCaches(user)
	return user, nil
}

// populateCaches agrega el usuario a L1 y L2 (best-effort).
func (s Service) populateCaches(user usersDAO.User) {
	if _, err := s.cacheRepository.Create(user); err != nil {
		log.Printf("warn: cache create failed (user_id=%d): %v", user.ID, err)
	}
	if _, err := s.memcachedRepository.Create(user); err != nil {
		log.Printf("warn: memcached create failed (user_id=%d): %v", user.ID, err)
	}
}

// invalidateCaches elimina el usuario de L1 y L2 (best-effort).
func (s Service) invalidateCaches(id int64) {
	if err := s.cacheRepository.Delete(id); err != nil {
		log.Printf("warn: cache delete failed (user_id=%d): %v", id, err)
	}
	if err := s.memcachedRepository.Delete(id); err != nil {
		log.Printf("warn: memcached delete failed (user_id=%d): %v", id, err)
	}
}

// validateTipo verifica que el tipo sea válido.
func validateTipo(tipo string) error {
	switch tipo {
	case "cliente", "administrador":
		return nil
	default:
		return fmt.Errorf("invalid tipo: %s", tipo)
	}
}

// hashPassword genera un hash bcrypt del password.
func (s Service) hashPassword(plain string) (string, error) {
	cost := s.bcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("error hashing password: %w", err)
	}
	return string(hash), nil
}

// toUser convierte el DAO a domain (sin password).
func (s Service) toUser(user usersDAO.User) usersDomain.User {
	return usersDomain.User{
		ID:       user.ID,
		Username: user.Username,
		Tipo:     user.Tipo,
	}
}
