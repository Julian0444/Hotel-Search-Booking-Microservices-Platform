package users

import (
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

func (repository Cache) GetAll() ([]usersDAO.User, error) {
	// Since it's not typical to cache all users in one request, you might skip caching here
	// Alternatively, you can cache a summary list if needed
	return nil, fmt.Errorf("GetAll not implemented in cache")
}

func (repository Cache) GetByID(id int64) (usersDAO.User, error) {
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
	return usersDAO.User{}, fmt.Errorf("%w: user ID %d", ErrCacheMiss, id)
}

func (repository Cache) GetByUsername(username string) (usersDAO.User, error) {
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
	return usersDAO.User{}, fmt.Errorf("%w: username %s", ErrCacheMiss, username)
}

func (repository Cache) Create(user usersDAO.User) (int64, error) {
	// Cache user by ID and by username after creation
	idKey := fmt.Sprintf("user:id:%d", user.ID)
	userKey := fmt.Sprintf("user:username:%s", user.Username)

	// Set user in cache
	repository.client.Set(idKey, user, repository.ttl)
	repository.client.Set(userKey, user, repository.ttl)

	// Return the user ID as if it was created successfully
	return user.ID, nil
}

func (repository Cache) Update(user usersDAO.User) error {
	// Update both the ID and username keys in cache
	idKey := fmt.Sprintf("user:id:%d", user.ID)
	userKey := fmt.Sprintf("user:username:%s", user.Username)

	// Set the updated user in cache
	repository.client.Set(idKey, user, repository.ttl)
	repository.client.Set(userKey, user, repository.ttl)

	return nil
}

func (repository Cache) Delete(id int64) error {
	// Delete user by ID and username from cache
	idKey := fmt.Sprintf("user:id:%d", id)

	// Best-effort: also delete the username key if we can read it from the cached user.
	item := repository.client.Get(idKey)
	if item != nil && !item.Expired() {
		if user, ok := item.Value().(usersDAO.User); ok {
			userKey := fmt.Sprintf("user:username:%s", user.Username)
			repository.client.Delete(userKey)
		}
	}

	repository.client.Delete(idKey)

	return nil
}
