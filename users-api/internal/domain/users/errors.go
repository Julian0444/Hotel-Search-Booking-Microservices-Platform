package users

import "errors"

// Sentinels del dominio (C6): repos y service los envuelven con %w y el
// controller decide el status HTTP con errors.Is — nunca parseando el texto
// del driver (antes: strings.Contains(err.Error(), "Duplicate")).
var (
	// ErrUserNotFound -> 404. Lo produce el repo MySQL (GetByID/GetByUsername/
	// Delete sin filas afectadas, C9).
	ErrUserNotFound = errors.New("user not found")

	// ErrUsernameTaken -> 409. Lo produce el repo MySQL al detectar el error
	// 1062 (duplicate entry) de forma tipada con errors.As.
	ErrUsernameTaken = errors.New("username already taken")

	// ErrValidation -> 400. El service lo envuelve como prefijo
	// ("validation failed: <detalle>"); el detalle es texto propio, apto para
	// el cliente.
	ErrValidation = errors.New("validation failed")
)
