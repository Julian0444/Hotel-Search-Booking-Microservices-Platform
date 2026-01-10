package users

// LoginRequest se usa para autenticación y registro de usuarios.
// Para registro, el campo Tipo es opcional (default: "cliente").
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Tipo     string `json:"tipo,omitempty"` // Solo usado en registro: "cliente" | "administrador"
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
