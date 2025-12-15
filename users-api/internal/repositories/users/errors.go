package users

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrCacheMiss    = errors.New("cache miss")
)
