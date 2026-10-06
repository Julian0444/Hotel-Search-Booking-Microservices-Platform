package utils

import "context"

// requestIDKey es la clave privada del request_id en el context: el consumer
// de RabbitMQ genera un id por mensaje (no hay request HTTP inbound del que
// leerlo) y el cliente HTTP de hotels-api lo reenvía como header (O1).
type requestIDKey struct{}

// WithRequestID devuelve un context que transporta el request_id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext extrae el request_id del context ("" si no hay).
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
