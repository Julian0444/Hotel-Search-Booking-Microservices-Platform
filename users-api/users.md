## Users API — Contexto de desarrollo (hasta ahora)

Este documento resume **lo que ya está implementado** en `users-api` y **qué falta construir** para que la API quede lista para integrarse con `hotels-api` (especialmente para emitir JWT con rol/admin).

---

## Objetivo de `users-api`

- **Responsabilidad principal**: gestionar usuarios (CRUD) y autenticación (`/login`) para emitir **JWT**.
- **Integración con `hotels-api`**: `hotels-api` protege rutas con JWT y espera estos claims:
  - **`user_id`**: ID del usuario (numérico o string; el middleware lo convierte a string).
  - **`tipo`**: `"cliente"` o `"administrador"`.
    - En `hotels-api`, `AdminOnly()` permite solo si `tipo == "administrador"`.

---

## Estructura actual (lo que existe hoy)

- **`users-api/internal/dao/users/users_dao.go`**: modelo GORM `User` (MySQL).
  - Campos: `ID`, `Username` (único), `Password` (hash), `Tipo` (enum `cliente|administrador`).

- **`users-api/internal/domain/users/users_domain.go`**: modelos HTTP (requests/responses).
  - `UserCreateRequest`, `UserUpdateRequest`, `UserResponse`, `LoginRequest`, `LoginResponse`.

- **`users-api/internal/repositories/users/`**: repositorios (fuente de datos y caches).
  - `users_mysql.go`: MySQL (GORM) → **source of truth**.
  - `users_cache.go`: cache L1 in-process (`ccache`) → rápida, no compartida entre instancias.
  - `users_memcached.go`: cache L2 distribuida (Memcached) → compartida entre instancias.
  - `users_mock.go`: mock con `testify/mock` para tests de service.

- **`users-api/internal/tokenizers/`**: generación de JWT.
  - `tokenizers_jwt.go`: HS256, claims `user_id`, `tipo`, `iat`, `exp`.
  - `tokenizers_mock.go`: mock para tests.

- **`users-api/cmd/main.go`**: existe, pero **solo compilará** cuando estén implementados los paquetes que importa (config/services/controllers/utils).

---

## Qué se validó en `repositories` (tests)

En `users-api/internal/repositories/users` **no hay tests unitarios** (`*_test.go`) todavía.

Lo que se ejecutó fue un **smoke-test de compilación**:

- `go test ./internal/repositories/...`

Esto sirve para confirmar que **los paquetes compilan** y que las firmas/métodos son consistentes, pero **no valida lógica de negocio**.

Más adelante, la validación real se hace en:
- **Tests de service** (orquestación MySQL + caches + tokenizer).
- **Tests HTTP de controller** (httptest) verificando status codes y payloads.

---

## Repositories — Por qué hay `Cache` y `Memcached`

Ambos son “cache”, pero de distinto tipo:

- **Cache L1 (`users_cache.go`)**:
  - Vive en memoria **dentro del proceso**.
  - Ideal para lecturas frecuentes (`GetByID`, `GetByUsername`).
  - No se comparte entre múltiples instancias/containers.

- **Cache L2 (`users_memcached.go`)**:
  - Vive en un servicio externo Memcached.
  - Es compartida entre instancias.
  - Útil cuando escalas horizontalmente.

Patrón habitual en producción: **L1 (in-process) → L2 (memcached/redis) → DB**.

### Sobre `GetAll`
`GetAll` **no es buen candidato para cache** (se invalida fácil, crece mucho, y Memcached no permite listar keys).
Lo correcto es:
- `GetAll`/listados → **MySQL** (idealmente paginado).
- Cachear solo lecturas puntuales → **GetByID / GetByUsername**.

### Keys de cache
En cache/memcached se usan keys consistentes:
- `user:id:<id>`
- `user:username:<username>`

---

## Tokenizer JWT — Requisitos y compatibilidad con `hotels-api`

En `users-api/internal/tokenizers/tokenizers_jwt.go`:

- **Algoritmo**: HS256.
- **Claims**:
  - `username`
  - `user_id`
  - `tipo`
  - `iat` (issued-at)
  - `exp` (expiration)

Esto es importante porque:
- `exp` es el claim estándar que hace que `token.Valid` funcione correctamente en el middleware de `hotels-api`.
- `tipo` y `user_id` son exactamente lo que `hotels-api/internal/middlewares/auth.go` consume.

### Tipo/rol
- Para probar endpoints `/admin/*` en `hotels-api`, el token debe traer:
  - `tipo = "administrador"`
- Para endpoints de usuario (reservas), el middleware deja `userID` en el contexto y los controllers validan ownership.

---

## Qué falta para que `users-api` quede “funcionando” (siguiente trabajo)

Orden recomendado (secuencial):

### 1) Config
- Crear `users-api/internal/config/` para leer env vars:
  - MySQL host/port/db/user/pass
  - Memcached host/port
  - `JWT_SECRET` (debe coincidir con `hotels-api` en local/docker)
  - TTL/duración del JWT
  - Puerto del server

### 2) Services
Crear `users-api/internal/services/users` con lógica:
- Hash de password (bcrypt).
- Login: validar password y emitir JWT.
- CRUD:
  - Reads intentan L1 → L2 → DB (y rellenan caches).
  - Writes actualizan DB y hacen best-effort update/invalidate en caches.

### 3) Controllers (Gin)
Crear `users-api/internal/controllers/users` con rutas:
- `GET /users`
- `GET /users/:id`
- `POST /users` (crear usuario; opcional aceptar `tipo`)
- `PUT /users/:id`
- `POST /login` (retorna `token`, `tipo`, `user_id`)
- `GET /health`

### 4) Tests (mejor práctica)
- **Service tests**: mocks de repos + mock tokenizer.
- **Controller tests** (httptest): mock del service y tabla de casos (200/201/400/401/404/500).

### 5) Docker / Compose mínimo para desarrollo
- `mysql` + `memcached` + `users-api`.
- Compartir `JWT_SECRET` con `hotels-api` para que ambos entiendan los tokens.

---

## Cómo debería usarse para probar `hotels-api`

Flujo mínimo:

1) Crear usuario admin en `users-api` (tipo `administrador`).
2) `POST /login` con ese usuario → obtienes `token`.
3) En Bruno/Postman:
   - `Authorization: Bearer <token>`
   - Probar rutas admin de `hotels-api` (`/admin/hotels`, `/admin/microservices`, etc.).

---

## Notas para siguientes agentes

- Mantener **compatibilidad de claims JWT** con `hotels-api` (`user_id`, `tipo`, `exp`).
- Evitar “inventar” `GetAll` en cache/memcached.
- Mantener deletes de cache como **best-effort** (cache miss no debe romper flujo).
- Para multi-módulo: este repo usa **módulos separados** (`hotels-api/go.mod`, `users-api/go.mod`), así que para comandos:
  - `cd users-api` → `go test ./...`, `go run ./cmd/main.go`
  - `cd hotels-api` → idem

