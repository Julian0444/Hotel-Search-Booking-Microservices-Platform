# Plan 04 — Dominio: no-overbooking atómico + reserva rica + caché correcta

> **Alcance:** D1, D2, D3, D4, DM1, DM2, DM5, C10, R3
> **Base:** playbook 7.2 de `plantofinish.md` + ítems afines (D2, D3, R3, C10)
> **Prerequisitos:** plan 03 (índices/ping/pool ya en su lugar; el refactor de ctx no toca estas firmas pero conviene la red de CI del 02)
> **Esfuerzo:** 1–2 días (L) — el paquete más importante después de seguridad

## Contexto

- **D1**: `CreateReservation` no verifica capacidad — overbooking incondicional (`hotels_service.go:205-225`, `hotels_mongo.go:207`).
- **DM1/DM2**: `Reservation` es un CRUD de 6 campos: sin estado (cancelar = hard `DeleteOne`), sin cantidad, sin dinero.
- **D2**: `Update` de hotel devuelve 500 y **no publica el evento** si el hotel expiró de la caché (`hotels_service.go:150-166`, `hotels_cache.go:201-207`).
- **D3**: la caché reporta hoteles libres como "no disponibles" (lista de reservas ausente ⇒ `false`; `hotels_cache.go:456-460, 388-394`).
- **D4**: semántica de fechas divergente Mongo vs caché (`hotels_mongo.go:399-414` vs `hotels_cache.go:474-489`).
- **R3**: lecturas devuelven 500 si falla el `cacheRepository.Create` post-DB (`hotels_service.go:57-58, 283, 316, 349` — incoherente con `:237-240` que loguea y sigue).
- **DM5**: `ReservationNew` está definido pero nunca se publica (`reservations.go:14-17`).
- **C10**: nada de esto tiene tests de repositorio.

## Correcciones de la Sección 7.0 que aplican

- **D1 (la corrección más importante del documento):** NO usar transacción Mongo — el compose corre **Mongo standalone** (`WithTransaction` falla con "Transaction numbers are only allowed on a replica set") y aunque funcionara, dos `InsertOne` concurrentes no colisionan. Usar **contador de inventario por hotel-noche** con índice único y `findOneAndUpdate` atómico + compensación.
- **DM5:** el `Queue.Publish` actual está atado a la cola `hotels-news` que search-api consume esperando `HotelNew` — publicar `ReservationNew` ahí rompe el `json.Unmarshal` del consumidor. Método `PublishReservation` separado sobre una cola **distinta** (`reservations-news`).

## Pasos

### 1. Modelos (DM1, DM2)

En `hotels_dao.go` (y dominio espejo):

- `Reservation` += `Status string`, `NumRooms int`, `NumGuests int`, `TotalPrice int64` (centavos — **nunca** float para dinero), `Currency string`, `CreatedAt time.Time`, `CancelledAt *time.Time`.
- Constantes `StatusConfirmed = "confirmed"`, `StatusCancelled = "cancelled"`.
- Nuevo `Inventory{HotelID string, Date string, Booked int, Capacity int}` (fecha como `"2006-01-02"`).
- Sentinel en el dominio: `var ErrNoAvailability = errors.New("no availability for the requested dates")`.

### 2. Índice único del inventario

En `NewMongo`: `CreateOne` sobre `{hotel_id: 1, date: 1}` con `SetUnique(true)` en la colección `reservation_inventory` (env `MONGO_COLLECTION_INVENTORY`). Sin cambios de connection string ni replica set.

### 3. Claim atómico por noche + compensación (D1)

```go
func (r Mongo) claimNight(ctx context.Context, c *mongo.Collection, hotelID, day string, rooms, cap int) (bool, error) {
  filter := bson.M{"hotel_id": hotelID, "date": day, "booked": bson.M{"$lte": cap - rooms}}
  update := bson.M{"$inc": bson.M{"booked": rooms}, "$setOnInsert": bson.M{"capacity": cap}}
  opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
  // en mongo.IsDuplicateKeyError reintentar UNA sola vez, no loopear: la carrera del upsert solo
  // ocurre si el doc no existía; en el reintento el doc ya existe y el filtro decide
  // (matchea → claim ok; no matchea → noche llena → ok=false)
}
```

En `CreateReservation` del repo: recorrer las noches del rango; si una falla, **liberar** (`$inc booked:-rooms`) las ya reclamadas y devolver `ErrNoAvailability`; recién entonces `InsertOne` de la reserva (si falla, liberar todo).

*Gotchas críticos (7.0):* usar `SetReturnDocument(options.After)` — el default `Before` devuelve `ErrNoDocuments` en un upsert-insert exitoso y parecería fallo. **No** poner `hotel_id`/`date` en `$setOnInsert` (ya están en el filtro → error "conflict").

### 4. Cancelación = soft delete idempotente (DM1)

`FindOneAndUpdate` con filtro `{_id, status: StatusConfirmed}` → set `status: cancelled` + `cancelled_at`, devolver la pre-imagen y liberar sus noches (`$inc booked:-num_rooms`). El filtro por `status:confirmed` hace la operación **idempotente**: un segundo cancel no matchea → no doble-libera. Reemplaza el `DeleteOne` de `hotels_mongo.go:248-266`.

### 5. `IsHotelAvailable` desde el inventario (D4)

Reescribir para leer el contador por noche — modelo canónico: **por noche, checkout excluido** — y **borrar** el `$unionWith: {coll: nil}` frágil y el loop de aggregation por día (`hotels_mongo.go:399-479`). Esto también arregla la divergencia D4: la caché y el mock deben implementar la **misma** semántica (paso 7).

### 6. Service: validaciones, precio, evento (DM2, DM5)

En `CreateReservation` del service (`hotels_service.go:205-225`):

- Validar: hotel existe, `CheckOut > CheckIn`, fechas no pasadas, `NumRooms >= 1 && NumRooms <= capacidad`.
- Derivar `HotelName` del hotel (no del body) y `TotalPrice`: `round(PricePerNight*100) * noches * NumRooms` (int64 centavos), `Currency: "USD"` (o ARS — elegir y documentar).
- Extender el struct `ReservationNew` (`reservations.go:14-17`) agregando el campo **`HotelID`** (el struct actual no lo tiene) y publicar `{Operation: "CREATE"|"CANCEL", ReservationID, HotelID}` vía nuevo método `PublishReservation` sobre cola `reservations-news` (declararla en el productor; nadie la consume aún — search puede aprovecharla como stretch).
- `PublishReservation` amplía la **interfaz `Queue`** → actualizar también el **mock de Queue** y los tests del service que lo usan (nuevo `.On("PublishReservation", ...)`).

### 7. Caché best-effort + semántica unificada (D2, D3, R3)

- **D2** (`hotels_service.go:150-166`): en `Update`/`Delete`, tratar la caché como best-effort — loguear-y-continuar ante error (o upsert en vez de update), y **publicar el evento siempre**. Nunca fallar una escritura de DB exitosa por un miss de caché.
- **R3** (`hotels_service.go:57-58, 283, 316, 349`): poblar caché en lecturas también log-and-continue — una escritura de caché nunca falla una lectura ya resuelta (imitar el patrón de `:237-240`).
- **D3** (`hotels_cache.go:456-460, 388-394`): lista de reservas ausente = **cero reservas = disponible** (o caer a Mongo ante miss). Sincronizar caché y mock con el modelo nuevo: contar `NumRooms`, saltar canceladas, checkout excluido.

### 8. Controller

- DTO de creación con fechas **string `"2006-01-02"`** (resuelve la inconsistencia de formato), parseo explícito con error 400 claro.
- Mapear `ErrNoAvailability` → **409 Conflict**.

### 9. Backfill de datos existentes

`hotels-api/seed/migrate-inventory.js` (correr una vez con `mongosh`): setear `status:"confirmed"`, `num_rooms:1`, `created_at` en reservas viejas; poblar `reservation_inventory` desde las reservas confirmadas (join `ObjectId(r.hotel_id)` contra hotels para la capacity).

### 10. Tests de repositorio (C10)

- Suite de date-overlap (mismo set de casos: solapado total/parcial/borde checkout/checkin=checkout/rango pasado) que corre contra **ambas** implementaciones (caché con datos inyectados; Mongo vía la suite `integration` del plan 02 con testcontainers).
- Test del claim: N goroutines reservando el último cuarto → exactamente 1 gana.
- Test de cancelación idempotente.

## Verificar

```bash
# Carrera real: 20 requests paralelos a un hotel de 1 habitación → exactamente 1×201, resto 409
seq 1 20 | xargs -P 20 -I{} curl -s -o /dev/null -w "%{http_code}\n" \
  -X POST http://localhost/reservations \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"hotel_id":"'$HID'","check_in":"2026-08-01","check_out":"2026-08-03","num_rooms":1,"num_guests":2}' \
  | sort | uniq -c        # esperado: 1 × 201, 19 × 409

# El inventario nunca supera capacity
docker compose exec mongo mongosh hotels-api --eval \
  'db.reservation_inventory.find({$expr:{$gt:["$booked","$capacity"]}}).count()'   # → 0

# Cancelar libera y es idempotente
curl -s -X DELETE http://localhost/reservations/$RID -H "Authorization: Bearer $TOKEN"  # 200/204, status:cancelled
curl -s -X DELETE http://localhost/reservations/$RID -H "Authorization: Bearer $TOKEN"  # idempotente, no doble-libera

# D2: PUT de hotel con caché expirada (esperar >30s tras crear) → 200 y search-api recibe el UPDATE
# D3: GET del hotel (cachea el hotel) e inmediatamente pedir disponibilidad → disponible (antes: falso negativo 30s)

cd hotels-api && go test -race ./... && go test -tags=integration ./...
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
