package users

import (
	"context"
	"fmt"
	"time"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"

	"github.com/karlseguin/ccache"
)

type CacheConfig struct {
	TTL time.Duration // Cache expiration time
}

type Cache struct {
	client *ccache.Cache
	ttl    time.Duration
}

func NewCache(config CacheConfig) Cache {
	// Initialize ccache with default settings
	cache := ccache.New(ccache.Configure())
	return Cache{
		client: cache,
		ttl:    config.TTL,
	}
}

// Nota: ccache no tiene API con context; estos métodos aceptan ctx y lo
// ignoran a propósito para cumplir la interfaz Repository.
func (repository Cache) GetAll(_ context.Context, _, _ int) ([]usersDAO.User, error) {
	// Since it's not typical to cache all users in one request, you might skip caching here
	// Alternatively, you can cache a summary list if needed
	return nil, fmt.Errorf("GetAll not implemented in cache")
}

func (repository Cache) CountAll(_ context.Context) (int64, error) {
	return 0, fmt.Errorf("CountAll not implemented in cache")
}

func (repository Cache) GetByID(_ context.Context, id int64) (usersDAO.User, error) {
	// Convert ID to string for cache key
	idKey := fmt.Sprintf("user:id:%d", id)

	// Try to get from cache
	item := repository.client.Get(idKey)
	if item != nil && !item.Expired() {
		// Return cached value
		user, ok := item.Value().(usersDAO.User)
		if !ok {
			return usersDAO.User{}, fmt.Errorf("failed to cast cached value to user")
		}
		return user, nil
	}

	// If not found, return cache miss error
	return usersDAO.User{}, fmt.Errorf("cache miss for user ID %d", id)
}

func (repository Cache) GetByUsername(_ context.Context, username string) (usersDAO.User, error) {
	// Use username as cache key
	userKey := fmt.Sprintf("user:username:%s", username)

	// Try to get from cache
	item := repository.client.Get(userKey)
	if item != nil && !item.Expired() {
		// Return cached value
		user, ok := item.Value().(usersDAO.User)
		if !ok {
			return usersDAO.User{}, fmt.Errorf("failed to cast cached value to user")
		}
		return user, nil
	}

	// If not found, return cache miss error
	return usersDAO.User{}, fmt.Errorf("cache miss for username %s", username)
}

func (repository Cache) Create(_ context.Context, user usersDAO.User) (int64, error) {
	// Cache user by ID and by username after creation
	idKey := fmt.Sprintf("user:id:%d", user.ID)
	userKey := fmt.Sprintf("user:username:%s", user.Username)

	// Set user in cache
	repository.client.Set(idKey, user, repository.ttl)
	repository.client.Set(userKey, user, repository.ttl)

	// Return the user ID as if it was created successfully
	return user.ID, nil
}

func (repository Cache) Update(_ context.Context, user usersDAO.User) error {
	// Update both the ID and username keys in cache
	idKey := fmt.Sprintf("user:id:%d", user.ID)
	userKey := fmt.Sprintf("user:username:%s", user.Username)

	// Set the updated user in cache
	repository.client.Set(idKey, user, repository.ttl)
	repository.client.Set(userKey, user, repository.ttl)

	return nil
}

// Delete borra ambas keys directamente a partir del DAO (RV28): antes el
// username salía de un Get por id, y si esa entrada ya no estaba la key
// user:username:* quedaba huérfana.
func (repository Cache) Delete(_ context.Context, user usersDAO.User) error {
	repository.client.Delete(fmt.Sprintf("user:id:%d", user.ID))
	repository.client.Delete(fmt.Sprintf("user:username:%s", user.Username))
	return nil
}
