package users

import (
	"encoding/json"
	"errors"
	"fmt"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcachedConfig struct {
	Host string
	Port string
}

type Memcached struct {
	client *memcache.Client
}

func idKey(id int64) string {
	return fmt.Sprintf("user:id:%d", id)
}

func usernameKey(username string) string {
	return fmt.Sprintf("user:username:%s", username)
}

func NewMemcached(config MemcachedConfig) Memcached {
	// Connect to Memcached
	address := fmt.Sprintf("%s:%s", config.Host, config.Port)
	client := memcache.New(address)

	return Memcached{client: client}
}

func (repository Memcached) GetAll() ([]usersDAO.User, error) {
	// In Memcached, you typically don’t have a way to retrieve "all" keys
	// You might need to store the list of all IDs in a separate cache entry
	return nil, fmt.Errorf("GetAll not supported in Memcached")
}

func (repository Memcached) GetByID(id int64) (usersDAO.User, error) {
	// Retrieve the user from Memcached
	key := idKey(id)
	item, err := repository.client.Get(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return usersDAO.User{}, fmt.Errorf("cache miss for user ID %d", id)
		}
		return usersDAO.User{}, fmt.Errorf("error fetching user from memcached: %w", err)
	}

	// Deserialize the data
	var user usersDAO.User
	if err := json.Unmarshal(item.Value, &user); err != nil {
		return usersDAO.User{}, fmt.Errorf("error unmarshaling user: %w", err)
	}
	return user, nil
}

func (repository Memcached) GetByUsername(username string) (usersDAO.User, error) {
	// Assume we store users with "username:<username>" as key
	key := usernameKey(username)
	item, err := repository.client.Get(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return usersDAO.User{}, fmt.Errorf("cache miss for username %s", username)
		}
		return usersDAO.User{}, fmt.Errorf("error fetching user by username from memcached: %w", err)
	}

	// Deserialize the data
	var user usersDAO.User
	if err := json.Unmarshal(item.Value, &user); err != nil {
		return usersDAO.User{}, fmt.Errorf("error unmarshaling user: %w", err)
	}

	return user, nil
}

func (repository Memcached) Create(user usersDAO.User) (int64, error) {
	// Serialize user data
	data, err := json.Marshal(user)
	if err != nil {
		return 0, fmt.Errorf("error marshaling user: %w", err)
	}

	// Store user with ID as key and username as an alternate key
	idKey := idKey(user.ID)
	if err := repository.client.Set(&memcache.Item{Key: idKey, Value: data}); err != nil {
		return 0, fmt.Errorf("error storing user in memcached: %w", err)
	}

	// Set key for username as well for easier lookup by username
	usernameKey := usernameKey(user.Username)
	if err := repository.client.Set(&memcache.Item{Key: usernameKey, Value: data}); err != nil {
		return 0, fmt.Errorf("error storing username in memcached: %w", err)
	}

	return user.ID, nil
}

func (repository Memcached) Update(user usersDAO.User) error {
	// Assume update is similar to Create: overwrite the existing user
	// Serialize user data
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error marshaling user: %w", err)
	}

	// Store user with ID as key
	idKey := idKey(user.ID)
	if err := repository.client.Set(&memcache.Item{Key: idKey, Value: data}); err != nil {
		return fmt.Errorf("error updating user in memcached: %w", err)
	}

	// Also update the username key
	usernameKey := usernameKey(user.Username)
	if err := repository.client.Set(&memcache.Item{Key: usernameKey, Value: data}); err != nil {
		return fmt.Errorf("error updating username in memcached: %w", err)
	}

	return nil
}

func (repository Memcached) Delete(id int64) error {
	// Best-effort delete: no fallar por cache miss
	keyByID := idKey(id)
	item, err := repository.client.Get(keyByID)
	if err == nil {
		var user usersDAO.User
		if err := json.Unmarshal(item.Value, &user); err == nil {
			_ = repository.client.Delete(usernameKey(user.Username))
		}
	}
	_ = repository.client.Delete(keyByID)
	return nil
}
