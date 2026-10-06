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
// El id se serializa como string (A7): el `user_id` canónico del contrato es
// string en toda la plataforma (es lo que viaja en el JWT y lo que usa
// hotels-api en las reservas); la PK int64 queda como detalle interno.
type User struct {
	ID       int64  `json:"id,string"`
	Username string `json:"username"`
	Tipo     string `json:"tipo"`
}

// LoginResponse es la respuesta al endpoint /login.
// Incluye el JWT token compatible con hotels-api.
// user_id como string, ver User (A7).
type LoginResponse struct {
	UserID   int64  `json:"user_id,string"`
	Username string `json:"username"`
	Token    string `json:"token"`
	Tipo     string `json:"tipo"`
}
