package users

type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Tipo     string `json:"tipo"`
}

type UserCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// Tipo es opcional. Si viene vacío se asume "cliente".
	Tipo string `json:"tipo"`
}

type UserUpdateRequest struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Tipo     *string `json:"tipo,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
	Tipo     string `json:"tipo"`
}
