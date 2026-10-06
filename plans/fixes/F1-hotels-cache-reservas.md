# Fix F1 — Caché de reservas de hotels-api: invalidación en vez de edición

> **Alcance:** RV1 (listas envenenadas), RV2 (data race), RV3 (cancel en miss), RV4 (`GetAvailability` traga errores), RV5 (mocks alineados)
> **Prerequisitos:** ninguno. Conviene hacerlo **antes** de los planes 06/07 (el 07 vuelve a pasar por estos getters).
> **Esfuerzo:** ~medio día (M)
> Snippets validados contra el código del 2026-07-11.

## Contexto

El plan 04 dejó la caché de reservas (`hotels-api/internal/repositories/hotels/hotels_cache.go`) con dos bugs estructurales:

1. **RV1 — Envenenamiento**: `updateHotelReservationsList` / `updateUserReservationsList` / `updateUserHotelReservationsList` (`hotels_cache.go:23-142`) **crean** la lista agregada cuando no existe, conteniendo solo esa reserva. Los getters (`hotels_cache.go:350-398`) tratan "la clave existe" como "lista completa". Flujo real: usuario con 10 reservas históricas no cacheadas crea una reserva (`hotels_service.go:299` → `Cache.CreateReservation` → `updateXxxList`) → `GET /users/U/reservations` devuelve **1 reserva en vez de 11** hasta el TTL. El guard del service (`offset == 0 && len < limit`, `hotels_service.go:373`) protege la clave consultada, pero el pobla-caché por-ítem crea listas parciales para las **otras dos** claves.
2. **RV2 — Data race**: ccache guarda el slice por referencia; `updateXxxList` lo **muta in place** (`reservations[i] = reservation` en `:39`, `append(reservations[:i], reservations[i+1:]...)` en `:51`) mientras `paginateReservations` (`:338-347`) devuelve subslices del mismo backing array a requests concurrentes. `POST /reservations` + `GET /users/:id/reservations` simultáneos = race real (los tests `-race` no lo agarran porque no ejercitan esa concurrencia).

La solución de ambos es la misma: **las listas agregadas nunca se editan — se invalidan en escrituras y solo se escriben completas (como copia) desde el service**.

## Decisión de diseño

- Escrituras de reserva en caché (`CreateReservation`, cancelación): setear la key individual `reservation:{id}` + **`Delete` de las 3 keys de listas** del par hotel/usuario. Nada de mutar slices → RV1 y RV2 mueren juntos.
- Población en lectura: el service ya sabe cuándo una página es la lista completa; en vez del loop por-ítem con `CreateReservation`, llama a un **setter de lista completa** nuevo, que guarda una **copia** del slice.
- El setter no puede ir en la interfaz `Repository` (Mongo tendría que implementarlo sin sentido): se agrega una interfaz `CacheRepository` que **embebe** `Repository`, y el campo `cacheRepository` del service cambia de tipo. `repositories.Cache` la satisface sin tocar `main.go`; `MockCache` necesita los métodos nuevos.

## Pasos

### 1. Interfaz `CacheRepository` en el service

En `hotels-api/internal/services/hotels_service.go` (la interfaz `Repository` está en `:19-32`):

```go
// CacheRepository extiende Repository con los setters de listas completas:
// las listas agregadas solo se escriben enteras (desde el service, que sabe
// cuándo una página es la lista completa) y se invalidan en cada escritura.
type CacheRepository interface {
	Repository
	SetReservationsByHotelID(ctx context.Context, hotelID string, reservations []hotelsDAO.Reservation)
	SetReservationsByUserID(ctx context.Context, userID string, reservations []hotelsDAO.Reservation)
	SetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string, reservations []hotelsDAO.Reservation)
}
```

Cambiar el tipo del campo y del parámetro (`hotels_service.go:39-52`): `cacheRepository CacheRepository` en el struct y en `NewService`. `cmd/main.go` compila sin cambios (pasa el `repositories.Cache` concreto).

### 2. Reemplazar las `updateXxxList` por invalidación

En `hotels_cache.go`: **borrar** las tres funciones `updateHotelReservationsList` / `updateUserReservationsList` / `updateUserHotelReservationsList` (`:22-142`) y agregar:

```go
// invalidateReservationLists borra las listas agregadas del par hotel/usuario.
// Las listas nunca se editan in-place (RV1: cachear parciales envenena; RV2:
// mutar slices compartidos con lectores concurrentes es un data race): toda
// escritura invalida y la próxima lectura repobla completa desde Mongo.
func (repository Cache) invalidateReservationLists(reservation hotelsDAO.Reservation) {
	repository.client.Delete(fmt.Sprintf("reservations:hotel:%s", reservation.HotelID))
	repository.client.Delete(fmt.Sprintf("reservations:user:%s", reservation.UserID))
	repository.client.Delete(fmt.Sprintf("reservations:hotel:%s:user:%s", reservation.HotelID, reservation.UserID))
}
```

`CreateReservation` (`:278-289`) queda:

```go
func (repository Cache) CreateReservation(ctx context.Context, reservation hotelsDAO.Reservation) (string, error) {
	key := fmt.Sprintf("reservation:%s", reservation.ID)
	repository.client.Set(key, reservation, repository.duration)
	repository.invalidateReservationLists(reservation)
	return reservation.ID, nil
}
```

### 3. Setters de listas completas (con copia)

En `hotels_cache.go`:

```go
// storeReservationsList guarda una lista agregada COMPLETA bajo key, como
// copia (los llamadores conservan su slice; nadie comparte backing array con
// la caché), y cachea también cada reserva individual. Una lista vacía se
// guarda igual: "sé que no hay reservas" también es un hit válido.
func (repository Cache) storeReservationsList(key string, reservations []hotelsDAO.Reservation) {
	list := make([]hotelsDAO.Reservation, len(reservations))
	copy(list, reservations)
	for _, r := range list {
		repository.client.Set(fmt.Sprintf("reservation:%s", r.ID), r, repository.duration)
	}
	repository.client.Set(key, list, repository.duration)
}

func (repository Cache) SetReservationsByHotelID(_ context.Context, hotelID string, reservations []hotelsDAO.Reservation) {
	repository.storeReservationsList(fmt.Sprintf("reservations:hotel:%s", hotelID), reservations)
}

func (repository Cache) SetReservationsByUserID(_ context.Context, userID string, reservations []hotelsDAO.Reservation) {
	repository.storeReservationsList(fmt.Sprintf("reservations:user:%s", userID), reservations)
}

func (repository Cache) SetReservationsByUserAndHotelID(_ context.Context, hotelID, userID string, reservations []hotelsDAO.Reservation) {
	repository.storeReservationsList(fmt.Sprintf("reservations:hotel:%s:user:%s", hotelID, userID), reservations)
}
```

> Nota de comportamiento: ahora una lista completa **vacía** se cachea (antes `updateXxxList` borraba la key al quedar vacía). Es correcto y ahorra viajes a Mongo; `IsHotelAvailable` con lista vacía cuenta cero ocupación (disponible), igual que con lista ausente.

### 4. Población en los getters del service

En los 3 getters de listas (`hotels_service.go:361-445`), reemplazar el loop por-ítem por el setter (el guard queda igual). Ejemplo para `GetReservationsByHotelID` (`:370-379`):

```go
		// Se guarda en la cache SOLO si esta página es la lista completa
		// (offset 0 y menos resultados que el límite). Best-effort (R3).
		if offset == 0 && int64(len(reservationsDAO)) < limit {
			service.cacheRepository.SetReservationsByHotelID(ctx, hotelID, reservationsDAO)
		}
```

Análogo con `SetReservationsByUserAndHotelID` (`:400-407`) y `SetReservationsByUserID` (`:428-435`).

### 5. Cancelación: reflejar en caché vía la reserva devuelta por Mongo (RV3)

Hoy `Cache.CancelReservation` (`hotels_cache.go:312-334`) en **miss** de la key individual borra esa key y retorna — las listas agregadas pueden conservar la copia `confirmed` hasta el TTL. El service ya tiene la reserva cancelada que devuelve Mongo: usarla. En `Service.CancelReservation` (`hotels_service.go:344-347`), reemplazar:

```go
	// Caché best-effort: setea la copia cancelada e invalida las listas
	// agregadas (funciona igual con la key individual evicted — RV3)
	if _, err := service.cacheRepository.CreateReservation(ctx, cancelled); err != nil {
		slog.Warn("error updating cancelled reservation in cache (continuing)", "reservation_id", id, "error", err)
	}
```

(`cancelled` es el `hotelsDAO.Reservation` que ya devuelve `mainRepository.CancelReservation`.) `Cache.CancelReservation` queda sin call-sites de producción pero la interfaz `Repository` la exige (Mongo la necesita): simplificarla a "setear la copia cancelada si estaba + invalidar listas" reutilizando lo anterior, con un comentario de que el service usa el path de `CreateReservation`.

### 6. `DeleteReservationsByHotelID` sin `updateXxxList`

La versión actual (`hotels_cache.go:522-547`) llama a las funciones borradas. Reescribir:

```go
func (repository Cache) DeleteReservationsByHotelID(ctx context.Context, hotelID string) error {
	reservations, err := repository.GetReservationsByHotelID(ctx, hotelID, int64(^uint64(0)>>1), 0)
	if err == nil {
		for _, r := range reservations {
			repository.client.Delete(fmt.Sprintf("reservation:%s", r.ID))
			repository.client.Delete(fmt.Sprintf("reservations:user:%s", r.UserID))
			repository.client.Delete(fmt.Sprintf("reservations:hotel:%s:user:%s", hotelID, r.UserID))
		}
	}
	repository.client.Delete(fmt.Sprintf("reservations:hotel:%s", hotelID))
	return nil
}
```

### 7. `Cache.GetAvailability`: propagar errores (RV4)

Hoy (`hotels_cache.go:434-442`) un error por-hotel se mapea a `available=false` con error nil → input inválido (fecha mal formada, `check_out <= check_in`) devuelve `200 {"h1": false}` si los hoteles están cacheados, y 500 vía Mongo si no. Unificar: si la caché no puede responder, que lo diga y el service caiga a Mongo (`hotels_service.go:450-457` ya hace el fallback):

```go
	for i := 0; i < len(hotelIDs); i++ {
		r := <-results
		if r.err != nil {
			// La caché no puede responder por este hotel: se propaga y el
			// service cae a Mongo (el canal tiene buffer: no hay goroutine leak)
			return nil, fmt.Errorf("cache availability failed for hotel %s: %w", r.hotelID, r.err)
		}
		availability[r.hotelID] = r.available
	}
```

(El 400 para input inválido es tema del plan 07/RV19; acá solo se elimina la respuesta mentirosa.)

### 8. Mocks y tests existentes (RV5)

- `hotels_mock.go`: `MockCache` (`:47`) debe implementar `CacheRepository` → agregar los 3 `SetReservationsByXxx` (guardar la lista para que los getters del mock la devuelvan). Aprovechar para acercar la semántica del mock a la real (hoy sus getters escanean todo como una DB — por eso los tests de service no podían ver RV1). `Mock` (el principal, `:41`) NO necesita cambios: sigue siendo `Repository`.
- `hotels_service_test.go`: los tests del flujo de cancelación que asserten llamadas a `MockCache.CancelReservation` pasan a assertar `CreateReservation` con la reserva cancelada (paso 5).
- `availability_suite_test.go`: la suite corre contra la caché real seedeando con `CreateReservation` (`:37,85`) — con la semántica nueva eso **invalida** las listas y `IsHotelAvailable` vería lista ausente (siempre disponible). Adaptar el harness de caché: tras crear las reservas del caso, poblar la lista del hotel con `SetReservationsByHotelID` (acumulando las reservas del caso). El lado Mongo de la suite no cambia.

### 9. Tests nuevos (regresión de RV1/RV2)

En `hotels_cache_test.go` (o donde vivan los tests de caché):

- **RV1**: `CreateReservation` de una reserva del user U / hotel H con caché vacía → `GetReservationsByUserID(U)`, `GetReservationsByHotelID(H)` y `GetReservationsByUserAndHotelID(H,U)` deben devolver **error (miss)**, nunca una lista de 1.
- **RV1 bis**: `SetReservationsByUserID(U, [r1, r2])` → `CreateReservation(r3 de U)` → el getter debe dar **miss** (lista invalidada).
- **RV2**: test con `N` goroutines mezclando `SetReservationsByHotelID`, `CreateReservation` y `GetReservationsByHotelID` sobre el mismo hotel — pasa limpio bajo `-race`.

## Verificar

```bash
cd hotels-api && go test -race ./... && cd ..
make lint && make test    # los 4 módulos siguen verdes

# En vivo (compose levantado, JWT de cliente demo):
# 1. Usuario con reservas previas NO cacheadas (reiniciar hotels-api para vaciar L1)
docker compose restart hotels-api
# 2. Crear una reserva nueva y pedir la lista INMEDIATAMENTE (antes era 1 sola):
#    POST /reservations ... ; GET /users/{id}/reservations → todas las reservas
# 3. Cancelar y re-pedir: la cancelada aparece con status cancelled (sin esperar TTL)
```

Al terminar: tick en `plans/fixes/README.md` y sección nueva en `plans/HANDOFF.md`.
