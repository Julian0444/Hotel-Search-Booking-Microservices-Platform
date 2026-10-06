package hotels

import (
	"context"
	"errors"
)

var ErrIdempotencyConflict = errors.New("idempotency key already used with a different payload")
var ErrLegacyIdempotency = errors.New("legacy idempotency record requires migration")
var ErrInventoryInconsistent = errors.New("inventory inconsistent with confirmed reservations; migration required")

type idempotencyContextKey struct{}

// Request-scoped metadata, never persisted separately from the reservation.
// Only a successful transaction creates a durable key; validation failures
// and aborted transactions do not consume it.
type IdempotencyOperation struct {
	Key      string
	Replayed bool
}

func WithIdempotency(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyContextKey{}, &IdempotencyOperation{Key: key})
}
func IdempotencyFromContext(ctx context.Context) *IdempotencyOperation {
	operation, _ := ctx.Value(idempotencyContextKey{}).(*IdempotencyOperation)
	return operation
}
