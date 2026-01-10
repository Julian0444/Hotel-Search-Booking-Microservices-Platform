## PLAN DE TRABAJO PARA LA IA (hotels-api)

Este archivo es el “recordatorio operativo” para trabajar en `hotels-api` sin romper contratos.
Debe mantenerse alineado con `README_HOTELS.md`, `cmd/main.go` y `RULES.md`.

---

## 0) Contexto rápido del microservicio

**Qué hace:** gestiona hoteles + reservas, con:
- API HTTP con **Gin** (controllers reales).
- Persistencia principal en **MongoDB**.
- Caché in-memory con **ccache** (cache-aside + listas agregadas para reservas).
- Eventos a **RabbitMQ** para **CRUD de hoteles** (no de reservas, por ahora).
- Auth con **JWT** y roles (`tipo`) vía middlewares.

**Rutas clave:** ver `cmd/main.go` (fuente de verdad de routing).

---

## 1) Checklist antes de tocar código

La IA debe:
- Leer `README_HOTELS.md`, `RULES.md` y `cmd/main.go` (rutas y middlewares).
- Identificar capa afectada: `controllers/` vs `services/` vs `repositories/` vs `domain/` vs `dao/`.
- Revisar naming/contratos:
  - Campo del modelo: `AvaiableRooms` (sí, está escrito así).
  - Claims JWT esperados: `user_id` y `tipo`.
- Ejecutar tests y formateo al terminar cambios:
  - `gofmt -w ./...`
  - `go test ./...`

---

## 2) Cómo agregar/modificar un endpoint (patrón recomendado)

Pasos generales (mantener separación de capas):
- **Controller**: parsea payload/params, aplica status codes, delega.
- **Service**: lógica de negocio/orquestación; DAO ↔ Domain; cache-aside; llamadas a repos; eventos si aplica.
- **Repository**: acceso a datos (Mongo/cache); sin lógica de negocio.
- **Tests**: controllers con `httptest` + JWT real; services con repos mocks.
- **Docs**: actualizar `README_HOTELS.md` si cambia contrato o rutas.

---

## 3) Auth/JWT: cómo debe trabajar la IA

En runtime, las rutas se agrupan así (ver `cmd/main.go`):
- **Públicas**: lectura de hotel/reservas por hotel, disponibilidad, health.
- **Usuario logueado**: crear/cancelar reservas, listar reservas por usuario.
- **Admin (`/admin`)**: CRUD hoteles + endpoints de “microservices”.

Los tests de controller deben:
- Usar `Authorization: Bearer <token>` (no “hackear” `ctx.Set("userID")`).
- Probar mínimo: 401 (sin token), 403 (rol o user mismatch), 200/201 (happy path), 400 (payload inválido).

---

## 4) Cache: decisiones actuales del código (no inventar comportamiento)

**Dónde está la verdad:** `internal/services/hotels_service.go` + repos `internal/repositories/hotels/*`.

Reglas actuales (resumen):
- Lecturas **cache-aside**:
  - `GetHotelByID`: cache → fallback main → set cache.
  - `GetReservationByID`: cache → fallback main → set cache (si falla set, no debe fallar la respuesta).
- Listas de reservas:
  - `GetReservationsByHotelID` / `GetReservationsByUserID` / `GetReservationsByUserAndHotelID`:
    - cache → fallback main → se guardan reservas en cache (por cada reserva) para poblar listas.
- Disponibilidad:
  - `GetAvailability`: intenta cache; si falla, usa main. (No cachea el resultado de main actualmente).
- Escrituras:
  - Hoteles: main → cache → publish evento.
  - Reservas: main → cache (no hay evento).

**Importante:** el TTL por defecto está configurado en `internal/config/config.go` (30s).

---

## 5) Eventos (RabbitMQ): alcance actual

Eventos publicados hoy:
- Solo **hoteles**: `CREATE`, `UPDATE`, `DELETE` (mensaje `HotelNew`).

Notas:
- Si RabbitMQ no está disponible, `Publish` falla y el service devuelve error (impacta el CRUD admin de hoteles).
- No inventar eventos de reservas salvo requerimiento explícito.

---

## 6) Estrategia de testing (la que queremos mantener)

Objetivo: feedback rápido en PRs.

- **Service tests** (`internal/services/*_test.go`):
  - repos mock (main + cache) + queue mock
  - validar cache population y lógica de fechas (checkout excluido)
- **Controller tests** (`internal/controllers/*_test.go`):
  - Gin + `httptest`
  - rutas reales (incluyendo `/admin`) + JWT real en headers
  - set mínimo de casos (happy + 400 + 401/403)

---

## 7) Backlog recomendado (si el usuario lo pide)

Ideas que suelen aparecer al “profesionalizar” el servicio:
- Config por env vars (hoy `internal/config/config.go` usa constantes).
- Publicar eventos también para reservas (si hay consumidores).
- OpenAPI/Swagger (contrato HTTP).
- Rate limiting y observabilidad (traces/metrics/logs).
- E2E en CI (docker compose) cuando el sistema completo esté listo.

---

## 8) Resultado esperado de cualquier cambio

Cada PR/cambio debe:
- Mantener arquitectura (ver `RULES.md`).
- Mantener contratos HTTP y de interfaces.
- Incluir tests razonables.
- Actualizar documentación si cambia comportamiento.
- Ser idempotente y fácil de revisar (gofmt, commits chicos).
