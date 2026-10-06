package users

import (
	"context"
	"testing"
	"time"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
)

// Benchmarks del hot path del read-through: el hit en la caché L1 (ccache)
// es lo que responde la enorme mayoría de los GetByID/GetByUsername.
func BenchmarkCacheGetByID(b *testing.B) {
	cache := NewCache(CacheConfig{TTL: time.Minute})
	if _, err := cache.Create(context.Background(), usersDAO.User{ID: 1, Username: "benchuser", Password: "x", Tipo: "cliente"}); err != nil {
		b.Fatalf("seeding cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := cache.GetByID(context.Background(), 1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheGetByUsername(b *testing.B) {
	cache := NewCache(CacheConfig{TTL: time.Minute})
	if _, err := cache.Create(context.Background(), usersDAO.User{ID: 1, Username: "benchuser", Password: "x", Tipo: "cliente"}); err != nil {
		b.Fatalf("seeding cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := cache.GetByUsername(context.Background(), "benchuser"); err != nil {
			b.Fatal(err)
		}
	}
}
