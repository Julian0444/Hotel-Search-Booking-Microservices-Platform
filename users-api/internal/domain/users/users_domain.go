package users

// LoginRequest se usa para autenticación y como input del service de creación.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Tipo     string `json:"tipo,omitempty"` // "cliente" | "administrador"
}

// RegisterRequest es el body del registro público (POST /users).
// El campo Tipo se acepta pero se ignora: el handler siempre fuerza "cliente";
// los administradores se crean únicamente vía seed (ADMIN_USERNAME/ADMIN_PASSWORD).
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Tipo     string `json:"tipo,omitempty"`
}

// User representa la información pública de un usuario (sin password).
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Tipo     string `json:"tipo"`
}

// LoginResponse es la respuesta al endpoint /login.
// Incluye el JWT token compatible con hotels-api.
type LoginResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
	Tipo     string `json:"tipo"`
}
