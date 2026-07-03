# ProyectoBackend — Hotel Search & Booking Microservices Platform

> Resumen arquitectónico del proyecto. El foco de este documento es mostrar **qué patrones de arquitectura usé y por qué**, no entrar en línea-por-línea. El proyecto partió como un trabajo de la facultad y lo evolucioné agregando una capa de gateway con load balancing, caché multi-nivel, autenticación distribuida y arquitectura event-driven.

---

## TL;DR — qué hay acá

Un sistema de búsqueda y reserva de hoteles con **3 microservicios Go independientes** (Users, Hotels, Search), comunicados por **HTTP síncrono** y **eventos asíncronos en RabbitMQ**. Cada servicio tiene su propia base de datos (**MySQL, MongoDB, Solr** — poliglot persistence). Todo el tráfico pasa por un **API Gateway en Nginx** que hace routing, rate limiting, CORS y **load balancing sobre 3 réplicas del Users API**. El frontend es una SPA en React 19.

Los conceptos arquitectónicos que demuestra el proyecto:

- Microservicios con bounded contexts
- API Gateway + Load Balancing (`least_conn`)
- CQRS-lite con eventual consistency (Hotels = write model, Search = read model)
- Event-driven architecture (RabbitMQ pub/sub)
- Caché multi-nivel (L1 in-process + L2 distribuida)
- Repository pattern con dependency inversion
- JWT distribuido + RBAC
- Concurrencia con goroutines, channels y mutexes
- Containerización con multi-stage builds

---

## Arquitectura

```
        Frontend (React 19 + Vite + MUI v7) — :5173
                          │
                          ▼
              Nginx API Gateway — :80
        (LB least_conn · rate limit · CORS · gzip)
                          │
        ┌─────────────────┼──────────────────┐
        ▼                 ▼                  ▼
   Users API x3      Hotels API         Search API
   (Go + Gin)        (Go + Gin)         (Go + Gin)
        │                 │                  │
        ▼                 ▼                  ▼
    MySQL 8          MongoDB 6           Solr 9
    + Memcached      + ccache LRU            ▲
    + ccache             │                   │
                         │  publica          │ consume
                         └──► RabbitMQ ──────┘
                              (hotels-news,
                               durable queue)
```

**10 contenedores** en total: Nginx, 3× Users-API, Hotels-API, Search-API, MySQL, Memcached, MongoDB, RabbitMQ, Solr.

---

## Stack


| Capa       | Tecnología                                                          |
| ---------- | ------------------------------------------------------------------- |
| Gateway    | Nginx (LB, rate limit, CORS, security headers)                      |
| Backend    | Go 1.22/1.23 · Gin · GORM · mongo-driver · solr-go · streadway/amqp |
| Datos      | MySQL 8 · MongoDB 6 · Solr 9                                        |
| Caché      | ccache (L1 in-process) · Memcached (L2 distribuida)                 |
| Mensajería | RabbitMQ 3 (AMQP, queue durable, mensajes `Persistent`)             |
| Auth       | JWT HS256 + bcrypt                                                  |
| Frontend   | React 19 · Vite · MUI v7 · React Router · Axios · react-hook-form   |
| Infra      | Docker · docker-compose con healthchecks                            |
| Tests      | Go testing + testify/mock + httptest                                |


---

## Patrones arquitectónicos aplicados

### Microservicios con bounded contexts

Cada servicio tiene un dominio acotado y **su propia base de datos** (anti-pattern: compartir DB entre servicios). La única forma de comunicación entre ellos es HTTP o eventos.


| Servicio   | Dominio                                  | DB      | Réplicas |
| ---------- | ---------------------------------------- | ------- | -------- |
| Users API  | Identidad, auth, autorización            | MySQL   | 3 (LB)   |
| Hotels API | CRUD hoteles + reservas + disponibilidad | MongoDB | 1        |
| Search API | Búsqueda full-text                       | Solr    | 1        |


### Arquitectura en capas (Clean lite)

Los 3 servicios siguen la misma estructura:

```
cmd/main.go                ← composition root: inyecta dependencias
└── internal/
    ├── config/            ← env vars (12-factor)
    ├── controllers/       ← HTTP handlers (Gin)
    ├── services/          ← lógica de negocio
    ├── repositories/      ← acceso a datos (mysql/mongo/solr/cache/mock)
    ├── dao/               ← modelos de persistencia (tags bson/gorm)
    ├── domain/            ← modelos de negocio (tags json)
    ├── middlewares/       ← auth, CORS, RBAC
    └── clients/queues/    ← adaptadores RabbitMQ
```

### Repository pattern + Dependency Inversion

Cada `Service` depende de **interfaces locales**, no de implementaciones concretas. Esto permite tener múltiples implementaciones intercambiables del mismo repositorio:

- **Users API** tiene 4 implementaciones del repo: `MySQL`, `Cache` (ccache), `Memcached`, `Mock`.
- **Hotels API** tiene 3: `Mongo`, `Cache` (ccache LRU), `Mock`.
- **Search API** tiene 3: `Solr`, `HTTP` (cliente al Hotels API), `Mock`.

El `main.go` (composition root) decide qué inyectar. Los tests usan los mocks sin levantar ninguna DB.

### CQRS-lite + Eventual consistency

- **Write side**: Hotels API + MongoDB (fuente de verdad de hoteles y reservas).
- **Read side**: Search API + Solr (índice full-text optimizado para búsqueda).
- **Sync**: eventos RabbitMQ.

Cuando se crea/actualiza/elimina un hotel, Hotels API publica un evento (`HotelNew{operation, hotel_id}`) en `hotels-news`. Search API consume el evento y reindexa Solr — y para CREATE/UPDATE hace un HTTP GET al Hotels API para obtener el documento completo (patrón **"thin event + lookup"**).

### API Gateway pattern

Nginx centraliza:

- Routing por path (`/users → users_api`, `/hotels → hotels_api`, `/search → search_api`).
- Load balancing `least_conn` sobre 3 réplicas de Users API con failover automático (`max_fails=3, fail_timeout=30s`) y `keepalive 32`.
- Rate limiting: 10 r/s API general, **5 r/min en `/login`** (mitiga brute force).
- CORS centralizado con whitelist de orígenes.
- Security headers (X-Frame-Options, X-Content-Type-Options, etc.).
- Gzip, monitoring (`/nginx_status`, `/status`).

---

## Concurrencia

La parte que más quería trabajar del proyecto. Hay 5 mecanismos concurrentes coexistiendo:

**1. Fan-out / fan-in con goroutines + channels** — en `hotels-api/.../hotels_mongo.go`, cuando el frontend pide disponibilidad de N hoteles, abro 1 goroutine por hotel y agrego los resultados por un buffered channel. Cada `IsHotelAvailable` corre un aggregation pipeline en Mongo; en paralelo el tiempo total se aproxima al del hotel más lento, no a la suma.

**2. Consumer RabbitMQ en goroutine dedicada** — en `search-api/.../queue_rabbit.go` el consumer corre en una goroutine durante toda la vida del proceso. Permite que Gin arranque inmediatamente y atienda HTTP, mientras el consumer sigue procesando eventos en background.

**3. Producer RabbitMQ con `sync.RWMutex` + reconexión** — en `hotels-api/.../queue_rabbit.go` implementé:

- Conexión inicial con **backoff exponencial** (5 reintentos, factor 2, max 30s).
- Una goroutine que escucha `NotifyClose` y reconecta sola ante caída del broker.
- `sync.RWMutex` para acceso concurrente seguro a `connection`/`channel`.
- `Publish` con hasta 3 reintentos por mensaje.

Es el patrón **self-healing infrastructure**.

**4. Gin maneja cada request en su propia goroutine** (default de `net/http`) y por encima **Nginx distribuye conexiones entre las 3 réplicas** con `least_conn` y keepalive.

**5. Mutex en `MockQueue`** — hasta los mocks son thread-safe; `Publish` toma el lock y `Messages()` devuelve una copia del slice. Permite tests con goroutines sin race conditions.

---

## Caching multi-nivel

Dos estrategias distintas según el servicio:

### Users API — 3 niveles (L1 + L2 + DB)

```
GET user ─► L1 (ccache, in-process)   ── hit? return
              miss
            └► L2 (Memcached, compartida)  ── hit? back-fill L1 → return
                 miss
               └► MySQL (source of truth)   ── back-fill L1+L2 → return
```

**Por qué dos niveles**: L1 es local al proceso (~30s TTL, instantánea). L2 está compartida entre las 3 réplicas del Users API — si una réplica trae el dato de MySQL, las otras lo encuentran en L2 sin volver a la DB. Sin L2, cada réplica iría a MySQL la primera vez.

**Doble indexación** en cada nivel: clave por ID (`user:id:N`) y por username (`user:username:foo`), porque busco por ambos según el endpoint.

### Hotels API — ccache LRU con listas agregadas

Cache-aside con `MaxSize=100k` items, eviction LRU. Mantiene índices denormalizados:

- `hotel:<id>`
- `reservations:hotel:<id>`
- `reservations:user:<id>`
- `reservations:hotel:<id>:user:<id>`

Al crear una reserva, sincroniza las 3 listas (write-through con denormalización — escritura más cara, lectura mucho más rápida).

**Decisión**: invalidación best-effort. Si la cache falla, logueo pero no rompo la operación. La fuente de verdad es la DB.

---

## Seguridad

### JWT distribuido

- **Users API emite tokens** HS256 con claims `user_id`, `username`, `tipo` (rol), `iat`, `exp`. Firma con `JWT_SECRET`.
- **Hotels API valida tokens** con el mismo secreto. Rechaza explícitamente cualquier método de firma que no sea HMAC — defensa contra el **"algorithm confusion attack"** (`alg: none`, `alg: RS256` con la key pública).
- El JWT se inyecta en `gin.Context` (`userType`, `userID`) para que los handlers lo usen.

### RBAC + ownership checks

Tres middlewares Gin:

- `Authenticate()` — valida JWT y pone claims en contexto.
- `AdminOnly()` — exige `tipo == "administrador"`.
- `LoggedUserOnly()` — exige token válido.

Y por encima, los handlers verifican **ownership**: un usuario solo puede crear/cancelar/ver sus propias reservas. Los admins son la excepción.

### Bcrypt + nunca exponer passwords

- `bcrypt.GenerateFromPassword` para crear, `CompareHashAndPassword` para verificar (timing-safe).
- El mapping DAO → Domain en el service **omite el campo password** (`Service.toUser()`).

### Defensa en gateway

- Rate limit estricto en login (5 r/min).
- Connection limit por IP (20).
- Security headers (X-Frame-Options, nosniff, XSS-Protection, Referrer-Policy).
- CORS con whitelist de orígenes.

---

## Event-driven architecture

### Configuración de la cola

Tanto producer como consumer declaran la misma queue con `durable=true`, y el producer publica con `DeliveryMode: Persistent`. **Los mensajes sobreviven a un reinicio del broker**.

### Flujo de un CREATE de hotel

```
1. Admin → POST /admin/hotels → Nginx → Hotels API
2. Hotels API crea en MongoDB → obtiene ID
3. Hotels API crea en ccache
4. Hotels API publica HotelNew{op:CREATE, id} en RabbitMQ
5. ───────────── (eventual consistency boundary) ─────────────
6. Search API consumer goroutine recibe el evento
7. Search API hace HTTP GET a Hotels API por el ID
8. Search API indexa el documento en Solr + commit
9. Próxima búsqueda /search devuelve el hotel
```

Pasos 1-4 son síncronos en la request. Los demás son asíncronos (típicamente <100ms).

**Idempotencia parcial**: indexar el mismo hotel dos veces no genera duplicados porque `id` es `uniqueKey` en el schema de Solr.

---

## Containerización

### Multi-stage Dockerfile (ejemplo Users API)

- **Stage builder** con `golang:1.23-alpine` → compila con `CGO_ENABLED=0` y `-ldflags="-s -w"` (binario estático y stripped).
- **Stage final** con `alpine:3.19`, solo el binario + `ca-certificates` + `tzdata`.
- **Usuario no-root** (`adduser appuser`, `USER appuser`) — principio de menor privilegio.
- **Layer caching**: `go.mod`/`go.sum` se copian antes que el código para no invalidar `go mod download` con cada cambio.

### Orquestación

`docker-compose.yml` con **healthchecks** en DBs y `depends_on: condition: service_healthy` en los servicios. Evita que un API arranque antes que su DB esté lista.

### Configuración 12-factor

Toda la config viene por env vars (`config.go` en cada servicio). El mismo binario corre en cualquier entorno cambiando solo el compose.

---

## Frontend (resumen)

SPA en React 19 + Vite + MUI v7 con:

- **AuthContext** que gestiona el JWT en `localStorage` + state.
- `**<ProtectedRoute adminOnly>`** que redirige según autenticación y rol.
- **Axios interceptors** que inyectan el `Bearer` token en cada request automáticamente y, ante un 401, limpian sesión + redirigen a `/login`. Centraliza el manejo del token en un solo lugar.
- **Vite proxy** `/api` → `http://localhost` para evitar CORS en dev.

---

## En qué me enfoqué al evolucionar el proyecto

El proyecto original de la facultad tenía los 3 microservicios funcionando con MongoDB/MySQL/Solr y la cola RabbitMQ. Lo que agregué en esta versión:

1. **Capa de gateway con Nginx**: routing, CORS centralizado, security headers, rate limiting diferenciado (más estricto en login), gzip.
2. **Load balancing horizontal**: configuración para correr 3 réplicas del Users API con `least_conn` y failover (`max_fails`/`fail_timeout`).
3. **Caché L2 con Memcached** en el Users API (la versión original solo tenía L1) — necesaria justo porque ahora hay 3 réplicas y la L1 no se comparte.
4. **Reconexión automática del producer RabbitMQ** con backoff exponencial, mutex y `NotifyClose` — la versión original explotaba al primer hipo del broker.
5. **Multi-stage Dockerfile + usuario no-root** en Users API.
6. **Tests con `testify/mock`** y mocks realistas para los repositorios y la cola.
7. **Documentación arquitectónica** (este archivo + README + `LOAD_BALANCER.md`).

---

## Mapa de archivos clave


| Concepto                             | Archivo                                                                       |
| ------------------------------------ | ----------------------------------------------------------------------------- |
| Composition roots                    | `*/cmd/main.go`                                                               |
| Cache L1+L2+DB                       | `users-api/internal/services/users/users_service.go`                          |
| Goroutines fan-out/fan-in            | `hotels-api/internal/repositories/hotels/hotels_mongo.go` (`GetAvailability`) |
| RabbitMQ producer self-healing       | `hotels-api/internal/clients/queues/queue_rabbit.go`                          |
| RabbitMQ consumer goroutine          | `search-api/internal/clients/queues/queue_rabbit.go`                          |
| Event handler → Solr sync            | `search-api/internal/services/search/search_service.go` (`HandleHotelNew`)    |
| JWT issuer                           | `users-api/internal/tokenizers/tokenizers_jwt.go`                             |
| JWT middleware + RBAC                | `hotels-api/internal/middlewares/auth.go`                                     |
| Nginx gateway                        | `nginx.conf`                                                                  |
| Caché LRU con índices denormalizados | `hotels-api/internal/repositories/hotels/hotels_cache.go`                     |
| Multi-stage Dockerfile no-root       | `users-api/Dockerfile`                                                        |
| Healthchecks orquestados             | `docker-compose.yml`                                                          |


