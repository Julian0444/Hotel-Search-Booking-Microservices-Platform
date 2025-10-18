package hotels

import (
	"context"
	"testing"
	"time"

	hotelsDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/hotels-api/internal/dao/hotels"
	"github.com/stretchr/testify/assert"
)

func TestCacheCRUDAndAvailability(t *testing.T) {
	ctx := context.Background()
	cache := NewCache(CacheConfig{MaxSize: 100, ItemsToPrune: 10, Duration: 1 * time.Minute})

	// Crear hotel y guardarlo
	h := hotelsDAO.Hotel{
		ID:            "cache-test-1",
		Name:          "Cache Test Hotel",
		AvaiableRooms: 2,
	}
	_, err := cache.Create(ctx, h)
	assert.NoError(t, err)

	// Obtener
	hgot, err := cache.GetHotelByID(ctx, "cache-test-1")
	assert.NoError(t, err)
	assert.Equal(t, h.Name, hgot.Name)

	// Actualizar
	err = cache.Update(ctx, hotelsDAO.Hotel{ID: "cache-test-1", Name: "Cache Test Hotel Updated"})
	assert.NoError(t, err)
	hgot, err = cache.GetHotelByID(ctx, "cache-test-1")
	assert.NoError(t, err)
	assert.Equal(t, "Cache Test Hotel Updated", hgot.Name)

	// Reservas para availability
	ci, _ := time.Parse("2006-01-02", "2025-10-20")
	co, _ := time.Parse("2006-01-02", "2025-10-21")
	res1 := hotelsDAO.Reservation{ID: "r1", HotelID: h.ID, UserID: "u1", CheckIn: ci, CheckOut: co}
	res2 := hotelsDAO.Reservation{ID: "r2", HotelID: h.ID, UserID: "u2", CheckIn: ci, CheckOut: co}

	// Guardar lista de reservas en la clave reservations:hotel:<id>
	_, err = cache.CreateReservation(ctx, res1)
	assert.NoError(t, err)
	_, err = cache.CreateReservation(ctx, res2)
	assert.NoError(t, err)
	// También guardar la lista completa para simplificar GetReservationsByHotelID
	cache.client.Set("reservations:hotel:"+h.ID, []hotelsDAO.Reservation{res1, res2}, cache.duration)

	avail, err := cache.IsHotelAvailable(ctx, h.ID, "2025-10-20", "2025-10-21")
	assert.NoError(t, err)
	// dado AvaiableRooms = 2 y tenemos 2 reservas, no hay disponibilidad
	assert.False(t, avail)

	// Borrar reservas
	err = cache.DeleteReservationsByHotelID(ctx, h.ID)
	assert.NoError(t, err)

	// Ahora debería ser disponible
	cache.client.Set("reservations:hotel:"+h.ID, []hotelsDAO.Reservation{}, cache.duration)
	avail, err = cache.IsHotelAvailable(ctx, h.ID, "2025-10-20", "2025-10-21")
	assert.NoError(t, err)
	assert.True(t, avail)

	// Delete hotel
	err = cache.Delete(ctx, h.ID)
	assert.NoError(t, err)
}
