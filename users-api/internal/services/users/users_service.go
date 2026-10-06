package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"

	"golang.org/x/crypto/bcrypt"
)

// Repository define las operaciones de persistencia de usuarios.
// Todos los métodos reciben context para propagar deadlines/cancelación hasta
// la DB (DB4/R1); las implementaciones de caché lo aceptan y lo ignoran.
type Repository interface {
	GetAll(ctx context.Context, limit, offset int) ([]usersDAO.User, error)
	CountAll(ctx context.Context) (int64, error)
	GetByID(ctx context.Context, id int64) (usersDAO.User, error)
	GetByUsername(ctx context.Context, username string) (usersDAO.User, error)
	Create(ctx context.Context, user usersDAO.User) (int64, error)
	Update(ctx context.Context, user usersDAO.User) error
	// Delete recibe el DAO completo: las cachés borran su key por username
	// sin depender de un lookup por id que puede haber sido evictado (RV28).
	Delete(ctx context.Context, user usersDAO.User) error
}

// Tokenizer define la generación de JWT tokens.
type Tokenizer interface {
	GenerateToken(username string, userID int64, tipo string) (string, error)
}

// Errores exportados para manejo en controller.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// dummyBcryptHash es un hash válido de un valor descartable, usado solo para
// igualar el timing del camino "usuario inexistente" en Login (RV7). Cost 10,
// como el default de producción.
var dummyBcryptHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

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

// GetAll retorna una página de usuarios (sin passwords) y el total en DB.
func (s Service) GetAll(ctx context.Context, limit, offset int) ([]usersDomain.User, int64, error) {
	users, err := s.mainRepository.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting all users: %w", err)
	}

	total, err := s.mainRepository.CountAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting users: %w", err)
	}

	result := make([]usersDomain.User, 0, len(users))
	for _, user := range users {
		result = append(result, s.toUser(user))
	}

	return result, total, nil
}

// GetByID retorna un usuario por ID (sin password).
func (s Service) GetByID(ctx context.Context, id int64) (usersDomain.User, error) {
	user, err := s.getByIDFromCaches(ctx, id)
	if err != nil {
		return usersDomain.User{}, fmt.Errorf("error getting user by ID: %w", err)
	}

	return s.toUser(user), nil
}

// Create registra un nuevo usuario con password hasheado.
func (s Service) Create(ctx context.Context, request usersDomain.LoginRequest) (int64, error) {
	if request.Username == "" {
		return 0, fmt.Errorf("%w: username is required", usersDomain.ErrValidation)
	}
	if request.Password == "" {
		return 0, fmt.Errorf("%w: password is required", usersDomain.ErrValidation)
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
	id, err := s.mainRepository.Create(ctx, newUser)
	if err != nil {
		return 0, fmt.Errorf("error creating user: %w", err)
	}

	// Best-effort: poblar caches
	newUser.ID = id
	s.populateCaches(ctx, newUser)

	return id, nil
}

// Delete elimina un usuario por ID.
func (s Service) Delete(ctx context.Context, id int64) error {
	// Resolver el usuario primero (RV28): el username hace determinística la
	// invalidación de la key user:username:* en L1/L2, y un id inexistente
	// corta acá con ErrUserNotFound (C9) sin tocar nada.
	user, err := s.mainRepository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error getting user for delete: %w", err)
	}

	if err := s.mainRepository.Delete(ctx, user); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	// Best-effort: invalidar caches
	s.invalidateCaches(ctx, user)

	return nil
}

// Login valida credenciales y retorna un JWT token.
func (s Service) Login(ctx context.Context, username, password string) (usersDomain.LoginResponse, error) {
	if username == "" || password == "" {
		return usersDomain.LoginResponse{}, ErrInvalidCredentials
	}

	user, err := s.getByUsernameFromCaches(ctx, username)
	if err != nil {
		if errors.Is(err, usersDomain.ErrUserNotFound) {
			// Igualar el costo del camino "no existe" al de un login real para
			// no filtrar existencia de usuarios por timing (RV7): el resultado
			// se descarta, solo importa quemar el mismo bcrypt.
			_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(password))
			return usersDomain.LoginResponse{}, ErrInvalidCredentials
		}
		// Infraestructura caída no es "credenciales inválidas" (RV6): se
		// propaga para que el controller responda 5xx en vez de 401 (el
		// deadline vencido de R1 cae acá y sigue mapeando a 503).
		return usersDomain.LoginResponse{}, fmt.Errorf("error getting user for login: %w", err)
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
func (s Service) getByIDFromCaches(ctx context.Context, id int64) (usersDAO.User, error) {
	// L1: cache in-process
	if user, err := s.cacheRepository.GetByID(ctx, id); err == nil {
		return user, nil
	}

	// L2: memcached
	if user, err := s.memcachedRepository.GetByID(ctx, id); err == nil {
		_, _ = s.cacheRepository.Create(ctx, user) // best-effort
		return user, nil
	}

	// DB: source of truth
	user, err := s.mainRepository.GetByID(ctx, id)
	if err != nil {
		return usersDAO.User{}, err
	}

	s.populateCaches(ctx, user)
	return user, nil
}

// getByUsernameFromCaches busca en L1 -> L2 -> DB y puebla caches en miss.
func (s Service) getByUsernameFromCaches(ctx context.Context, username string) (usersDAO.User, error) {
	// L1: cache in-process
	if user, err := s.cacheRepository.GetByUsername(ctx, username); err == nil {
		return user, nil
	}

	// L2: memcached
	if user, err := s.memcachedRepository.GetByUsername(ctx, username); err == nil {
		_, _ = s.cacheRepository.Create(ctx, user) // best-effort
		return user, nil
	}

	// DB: source of truth
	user, err := s.mainRepository.GetByUsername(ctx, username)
	if err != nil {
		return usersDAO.User{}, err
	}

	s.populateCaches(ctx, user)
	return user, nil
}

// populateCaches agrega el usuario a L1 y L2 (best-effort).
func (s Service) populateCaches(ctx context.Context, user usersDAO.User) {
	if _, err := s.cacheRepository.Create(ctx, user); err != nil {
		slog.Warn("cache create failed", "user_id", user.ID, "error", err)
	}
	if _, err := s.memcachedRepository.Create(ctx, user); err != nil {
		slog.Warn("memcached create failed", "user_id", user.ID, "error", err)
	}
}

// invalidateCaches elimina el usuario de L1 y L2 (best-effort). Recibe el DAO
// completo para que cada caché borre sus DOS keys (id y username) sin lookups
// intermedios (RV28).
//
// Tradeoff documentado (RV10): con 3 réplicas y L1 por proceso, esto solo
// limpia la réplica que atendió el DELETE — durante <=CACHE_DURATION (30s) un
// usuario borrado puede loguearse contra otra réplica (Login lee a través de
// la caché) y obtener un JWT nuevo de 24h; con JWT stateless sin revocación,
// el borrado tampoco corta tokens ya emitidos. Es inherente al diseño
// L1-por-réplica + JWT stateless y se acepta para el alcance del proyecto;
// en producción: invalidación por pub/sub, denylist de tokens o TTL corto +
// refresh tokens (ver "Known trade-offs" en el README).
func (s Service) invalidateCaches(ctx context.Context, user usersDAO.User) {
	if err := s.cacheRepository.Delete(ctx, user); err != nil {
		slog.Warn("cache delete failed", "user_id", user.ID, "error", err)
	}
	if err := s.memcachedRepository.Delete(ctx, user); err != nil {
		slog.Warn("memcached delete failed", "user_id", user.ID, "error", err)
	}
}

// validateTipo verifica que el tipo sea válido.
func validateTipo(tipo string) error {
	switch tipo {
	case "cliente", "administrador":
		return nil
	default:
		return fmt.Errorf("%w: invalid tipo %q", usersDomain.ErrValidation, tipo)
	}
}

// hashPassword genera un hash bcrypt del password.
func (s Service) hashPassword(plain string) (string, error) {
	// bcrypt solo opera sobre los primeros 72 bytes; rechazamos explícitamente
	// en vez de depender del comportamiento de la librería.
	if len(plain) > 72 {
		return "", fmt.Errorf("%w: password is too long (maximum 72 bytes)", usersDomain.ErrValidation)
	}

	cost := s.bcryptCost
	// Clamp defensivo (RV9): >31 hace fallar GenerateFromPassword (500 en
	// todos los registros) y 25-31 tarda minutos por hash; <MinCost lo salva
	// la lib pero mejor explícito. Rango sano para un servicio online: [10, 15].
	if cost < 10 || cost > 15 {
		slog.Warn("BCRYPT_COST out of sane range, using default", "configured", cost, "used", bcrypt.DefaultCost)
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
