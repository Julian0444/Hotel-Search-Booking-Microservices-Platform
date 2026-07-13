# Plan 13 — Stretch: dominio extendido + frontend (OPCIONAL)

> **Alcance:** DM3, DM4, DM6, DM7, FE1, FE2
> **Base:** sin playbook — Sección 6.10 de `plantofinish.md`. Todo lo de acá es **diferenciación incremental**: el proyecto ya se defiende sin este plan (ver nota de alcance en 6.12).
> **Prerequisitos:** plan 04 (lifecycle de reserva — DM4 extiende su máquina de estados; DM6 extiende su inventario), plan 07 (idempotencia — DM4 la reusa), **plan 11 para DM6** (el rename C11 ya aplicado — no construir room-types sobre el campo con typo), plan 12 idealmente ya hecho (si hacés esto después, actualizá docs/OpenAPI al final)
> **Esfuerzo:** ~2 días si se hace completo; cada ítem es independiente — cherry-pick en este orden de ROI: DM7 → DM3 → DM4 → DM6 → FE2 → FE1

## Contexto

El dominio quedó correcto tras el plan 04, pero sigue siendo chico. Estos ítems lo convierten en "diseño de sistema" para la entrevista: un segundo agregado con read model derivado (DM3), manejo de dinero y compensación (DM4), inventario por tipo de habitación (DM6), y un `User` que deja de ser anémico (DM7). FE1/FE2 son señales de frontend de baja prioridad para un rol backend.

## Pasos

### 1. `User` rico (DM7)

Evidencia: `users_dao.go:3-8`, `users_domain.go:12-16` — solo id/username/password/tipo; `Tipo` string libre (*primitive obsession*).

- Agregar `Email` (validado con binding `email`, único), `DisplayName`, `CreatedAt`.
- Rol como constante tipada: `type Role string; const (RoleClient Role = "cliente"; RoleAdmin Role = "administrador")` — usar en service/middleware en lugar de literales.
- Migración `000X_add_user_fields` (el flujo del plan 03); actualizar Register del frontend (campo email) y el seed demo.

### 2. Entidad `Review` + rating derivado (DM3)

Evidencia: `Hotel.Rating` es un número que tipea el admin (`hotels_domain.go:16`).

- Colección `reviews`: `{user_id, hotel_id, score 1..5, comment, created_at}` + índice único `{user_id, hotel_id}` (una review por usuario/hotel).
- Endpoints: `POST /api/v1/hotels/:id/reviews` (auth cliente; opcional: solo si tuvo reserva), `GET /api/v1/hotels/:id/reviews` (público, paginado).
- `Hotel.Rating` pasa a ser **derivado**: recomputar el promedio al crear/borrar review (aggregation `$avg` + update del hotel, y publicar el evento UPDATE para que search-api reindexe el rating). Quitar el campo del form del admin.
- Puntos de entrevista: segundo agregado + read model derivado + invalidación de caché.

### 3. Pago stub + saga de cancelación (DM4)

- Máquina de estados extendida (sobre el `Status` del plan 04): `pending_payment → confirmed → cancelled`, con `refunded` como estado terminal de la compensación.
- `PaymentService` **stub** (interfaz + implementación fake con latencia): `Authorize(amount) → paymentID`, `Capture(paymentID)`, `Refund(paymentID)`.
- Flujo: crear reserva → claim del inventario (plan 04) → `Authorize+Capture` → `confirmed`. Si el pago falla → **liberar las noches** (compensación) → error 402/409.
- Cancelación: `Refund` + liberar noches — una mini-saga; si el refund falla, reintentar con backoff y dejar la reserva en `cancel_pending` (hablar de esto en entrevista vale más que el código).
- Montos en **centavos int64** (ya así por DM2); endpoints de pago con `Idempotency-Key` (middleware del plan 07).

### 4. Room types (DM6) — el más grande, hacerlo último (requiere el plan 11: C11 ya aplicado)

Evidencia: un hotel es un solo `PricePerNight` + un solo contador de habitaciones (`hotels_domain.go:15-17`).

- `RoomType{ID, Name, Capacity, PricePerNight int64, TotalRooms}` como array embebido en `Hotel`.
- El inventario del plan 04 pasa de `{hotel_id, date}` a **`{hotel_id, room_type_id, date}`** (índice único triple); la reserva referencia `room_type_id` y el precio se snapshotea del tipo.
- Disponibilidad y búsqueda por tipo (capacidad de huéspedes); migración de datos: convertir el `AvailableRooms` actual en un tipo "standard" por hotel.
- Tocar: dao/domain/service/controller/caché/mock + frontend (selector de tipo en HotelDetail) + OpenAPI. Es efectivamente una segunda pasada sobre el plan 04 — presupuestar medio día largo.

### 5. Frontend (FE2, FE1)

- **FE2** (barato): `React.lazy` + `Suspense` por ruta en `App.jsx:16-22` — al menos el bundle del admin. Verificar chunks separados en `npm run build`.
- **FE1** (caro, opcional de verdad): migración a TypeScript. Para un rol backend, el plan original dice explícitamente que JSDoc está OK — solo hacerlo si sobra tiempo; si no, saltear sin culpa.

## Verificar

```bash
# DM7: registro exige email válido; el JWT/DTOs exponen display name; rol tipado compila sin literales sueltos
# DM3: crear 2 reviews (scores 2 y 4) → GET del hotel muestra rating 3.0; el hotel reindexado en search lo refleja
# DM4: pago fake que falla (flag del stub) → reserva no queda confirmed y las noches se liberan:
docker compose exec mongo mongosh hotels-api --eval 'db.reservation_inventory.find({booked:{$gt:0}})' # consistente
#      cancelar → refund + noches liberadas; segundo cancel → idempotente
# DM6: 2 tipos de habitación, agotar uno → el otro sigue reservable; overbooking test del plan 04 repetido por tipo
# FE2: npm run build → chunk separado para admin; la app carga igual

go test -race ./... && cd frontend && npm run build
# Actualizar OpenAPI/README con las entidades nuevas si el plan 12 ya corrió
```

## Al terminar

Con esto queda cubierto el 100% de los IDs de `plantofinish.md`. El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
