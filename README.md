# 🏨 Hotel Search & Booking Microservices Platform

A full-stack hotel search and booking platform built with a microservices architecture. Three independent Go APIs communicate through an Nginx API Gateway and RabbitMQ, backed by a React SPA frontend.

---

## 🏗️ Architecture

```
                              ┌──────────────────────────┐
                              │   Frontend (React 19)    │
                              │     Vite + MUI v7        │
                              │      Port 5173           │
                              └────────────┬─────────────┘
                                           │
                              ┌────────────▼─────────────┐
                              │   Nginx API Gateway      │
                              │   Port 80  │  Port 8090  │
                              │  (routing, rate limiting, │
                              │   load balancing, CORS)   │
                              └──┬─────────┼──────────┬──┘
                                 │         │          │
              ┌──────────────────▼──┐  ┌───▼────────┐ │  ┌───────────────────┐
              │     Users API       │  │ Hotels API │ │  │    Search API      │
              │  (Load Balanced x3) │  │  Port 8081 │ │  │    Port 8082       │
              │     Port 8082       │  └─────┬──────┘ │  └──────┬────────────┘
              └──┬──────┬──────┬────┘        │        │         │
                 │      │      │             │        │         │
           ┌─────▼┐ ┌───▼─┐ ┌─▼─────┐       │        │         │
           │ API-1│ │API-2│ │ API-3 │       │        │         │
           └──────┘ └─────┘ └───────┘       │        │         │
                 │                           │        │         │
          ┌──────▼───────┐           ┌───────▼──┐     │   ┌─────▼──────┐
          │    MySQL 8   │           │ MongoDB 6│     │   │  Solr 9    │
          │  Port 3307   │           │ Port 27017│    │   │ Port 8983  │
          └──────────────┘           └──────────┘     │   └────────────┘
                 │                        │           │         │
          ┌──────▼───────┐               │      ┌────▼─────────▼────┐
          │  Memcached   │               └──────┤    RabbitMQ 3     │
          │ Port 11211   │                      │  Port 5672/15672  │
          └──────────────┘                      └───────────────────┘
                                                  Hotels API publishes
                                                  Search API consumes
```

---

## 🛠️ Tech Stack

| Category        | Technology                                                     |
|-----------------|----------------------------------------------------------------|
| **Backend**     | Go 1.22/1.23 · Gin · GORM · mongo-driver · solr-go · amqp     |
| **Frontend**    | React 19 · Vite 7 · MUI v7 · React Router 6 · Axios · react-hook-form |
| **Databases**   | MySQL 8 · MongoDB 6 · Apache Solr 9                           |
| **Cache**       | Memcached 1.6 (distributed L2) · ccache (in-process L1)       |
| **Messaging**   | RabbitMQ 3 (AMQP)                                             |
| **Infra**       | Docker · Docker Compose · Nginx (API Gateway + Load Balancer)  |
| **Auth**        | JWT (shared secret across services) · bcrypt                   |
| **Testing**     | Go testing · httptest · testify · mock repositories            |

---

## 📁 Project Structure

```
├── docker-compose.yml          # Full orchestration (10 services)
├── nginx.conf                  # API Gateway configuration
│
├── users-api/                  # User management & authentication
│   ├── cmd/main.go             # Entrypoint
│   ├── internal/
│   │   ├── config/             # Env vars configuration
│   │   ├── controllers/users/  # HTTP handlers (Gin)
│   │   ├── services/users/     # Business logic (bcrypt, JWT)
│   │   ├── repositories/users/ # MySQL + Cache L1 + Memcached L2
│   │   ├── dao/users/          # Data access objects
│   │   ├── domain/users/       # Domain models
│   │   ├── tokenizers/         # JWT token generation
│   │   └── utils/              # CORS middleware
│   └── Dockerfile              # Multi-stage build
│
├── hotels-api/                 # Hotel & reservation management
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── config/             # Env vars configuration
│   │   ├── controllers/
│   │   │   ├── hotels/         # Hotel & reservation handlers
│   │   │   └── microservices/  # Admin panel service management
│   │   ├── services/           # Business logic + cache-aside
│   │   ├── repositories/hotels/# MongoDB + ccache
│   │   ├── clients/queues/     # RabbitMQ producer
│   │   ├── middlewares/        # JWT auth + role-based access
│   │   ├── dao/hotels/         # Data access objects
│   │   └── domain/hotels/      # Domain models
│   └── dockerfile
│
├── search-api/                 # Full-text hotel search
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── config/             # Env vars configuration
│   │   ├── controllers/search/ # Search handler
│   │   ├── services/search/    # Search logic + event handling
│   │   ├── repositories/hotels/# Solr + Hotels API HTTP client
│   │   ├── clients/queues/     # RabbitMQ consumer
│   │   ├── dao/hotels/         # Data access objects
│   │   ├── domain/hotels/      # Domain models
│   │   └── utils/              # CORS middleware
│   └── Dockerfile
│
└── frontend/                   # React SPA
    └── src/
        ├── components/         # Layout, HotelCard, SearchBar
        ├── pages/              # Home, Search, Login, Register,
        │                       # HotelDetail, MyReservations, Admin
        ├── services/           # Axios API clients
        ├── context/            # AuthContext (JWT state)
        ├── hooks/              # useAuth
        ├── constants/          # Routes, config, amenities
        ├── theme/              # MUI custom theme
        └── utils/              # Helpers & validators
```

---

## ⚙️ Microservices

### Users API
Handles user registration, authentication, and JWT token generation. Runs **3 load-balanced instances** behind Nginx (`least_conn` algorithm).

- **Stack:** Go 1.23 · Gin · GORM · MySQL 8 · Memcached · ccache
- **Cache strategy:** Three-tier read-through — L1 (in-process ccache) → L2 (Memcached) → MySQL, with backfill on cache miss
- **Auth:** Generates JWT tokens with `user_id`, `username`, and `tipo` (role) claims; passwords hashed with bcrypt
- **Roles:** `cliente` (default) and `administrador`

### Hotels API
Manages hotel CRUD operations and the reservation system. Publishes hotel lifecycle events to RabbitMQ.

- **Stack:** Go 1.23 · Gin · MongoDB 6 · ccache · RabbitMQ
- **Cache strategy:** Cache-aside pattern with LRU eviction (ccache, 30s TTL)
- **Events:** Publishes `CREATE`, `UPDATE`, `DELETE` events for hotels to the `hotels-news` queue
- **Auth:** Validates JWT tokens from Users API (shared secret); role-based middleware (`AdminOnly`, `LoggedUserOnly`)
- **Concurrency:** Availability checks run in parallel using goroutines (one per hotel)

### Search API
Provides full-text hotel search powered by Apache Solr. Consumes RabbitMQ events to keep the search index synchronized.

- **Stack:** Go 1.22 · Gin · solr-go · RabbitMQ
- **Event-driven sync:** Listens to `hotels-news` queue — on hotel create/update/delete events, updates the Solr index accordingly
- **Hotels API client:** Fetches hotel details via HTTP when processing events

---

## 📋 Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Node.js](https://nodejs.org/) v18+ and npm (for the frontend)
- ~4 GB RAM available for Docker services

---

## 🚀 Installation & Setup

### 1. Clone the repository

```bash
git clone https://github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform.git
cd Hotel-Search-Booking-Microservices-Platform
```

### 2. Configure environment & start backend services

```bash
cp .env.example .env   # local demo credentials (JWT secret, DB passwords, admin seed)
docker compose up -d --build
```

The first admin account is created automatically at startup from `ADMIN_USERNAME` / `ADMIN_PASSWORD` in your `.env` — public registration always creates customer accounts.

This starts **10 containers**: Nginx, MySQL, Memcached, MongoDB, RabbitMQ, Solr, Users API (x3), Hotels API, and Search API.

Wait for all services to be healthy:

```bash
docker compose ps
```

### 3. Start the frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend runs at `http://localhost:5173` and proxies API requests through Vite to the Nginx gateway.

### Alternative: Kubernetes (kind)

The same platform can run on Kubernetes: Deployments with real readiness/liveness probes, a Service replacing the static nginx upstream (scaling = `kubectl scale`, no YAML duplication), an HPA, and graceful rolling deploys. Dev-only StatefulSets keep the demo self-contained — in production the datastores would be managed services. See [`k8s/README.md`](k8s/README.md) for the quickstart and the honest dev-vs-prod notes.

---

## 🌐 API Endpoints (Gateway — Port 80)

The API is versioned under a `/api/v1` prefix (URI versioning: a breaking change to any contract ships as `/api/v2` alongside `v1`). Success responses use a single envelope — `{"data": ...}` for single resources, `{"data": [...], "meta": {"total", "limit", "offset"}}` for lists — and every error (services and gateway alike) has the shape `{"error": {"code", "message", "trace_id"}}` with stable machine-readable codes. Health endpoints are not versioned.

| Method   | Endpoint                                      | Service    | Auth     | Description                     |
|----------|-----------------------------------------------|------------|----------|---------------------------------|
| `POST`   | `/api/v1/users`                               | Users API  | —        | Register a new user (customer role only) |
| `POST`   | `/api/v1/login`                               | Users API  | —        | Login, returns JWT              |
| `GET`    | `/api/v1/users`                               | Users API  | Admin    | List all users                  |
| `GET`    | `/api/v1/users/:id`                           | Users API  | Owner/Admin | Get user by ID               |
| `DELETE` | `/api/v1/users/:id`                           | Users API  | Owner/Admin | Delete user (204)            |
| `GET`    | `/api/v1/hotels?limit=&offset=`               | Hotels API | —        | List hotels (paginated)         |
| `GET`    | `/api/v1/hotels/:id`                          | Hotels API | —        | Get hotel details               |
| `GET`    | `/api/v1/hotels/:id/reservations`             | Hotels API | —        | List hotel reservations         |
| `POST`   | `/api/v1/hotels/availability`                 | Hotels API | —        | Check availability (multi)      |
| `POST`   | `/api/v1/reservations`                        | Hotels API | JWT      | Create reservation (201 + `Location`; supports `Idempotency-Key`) |
| `GET`    | `/api/v1/reservations/:id`                    | Hotels API | JWT      | Get reservation (owner/admin)   |
| `DELETE` | `/api/v1/reservations/:id`                    | Hotels API | JWT      | Cancel reservation (204)        |
| `GET`    | `/api/v1/users/:id/reservations`              | Hotels API | JWT      | User's reservations             |
| `GET`    | `/api/v1/search?q=...`                        | Search API | —        | Full-text hotel search          |
| `POST`   | `/api/v1/reindex`                             | Search API | Admin    | Rebuild the Solr index from Hotels API |
| `POST`   | `/api/v1/admin/hotels`                        | Hotels API | Admin    | Create hotel (201 + `Location`) |
| `PUT`    | `/api/v1/admin/hotels/:id`                    | Hotels API | Admin    | Update hotel (returns updated representation) |
| `DELETE` | `/api/v1/admin/hotels/:id`                    | Hotels API | Admin    | Delete hotel (204)              |
| `GET`    | `/health`                                     | Gateway    | —        | Gateway health check            |

> Each Go service also exposes internal (not routed through the gateway) health endpoints: `/livez` (process liveness, `/health` is an alias) and `/readyz` (pings its own dependencies — e.g. Mongo+RabbitMQ for Hotels API — and returns `503` with a per-check status map if any is down). Docker Compose healthchecks hit `/readyz`, and nginx only starts once every upstream is healthy.

---

## ✨ Key Features

- **Load Balancing** — Nginx distributes Users API traffic across 3 instances using `least_conn` with automatic failover (`max_fails=3`, `fail_timeout=30s`)
- **Multi-Level Caching** — Users API: L1 (ccache) → L2 (Memcached) → MySQL. Hotels API: ccache (LRU) → MongoDB
- **Event-Driven Architecture** — Hotels API publishes CRUD events to RabbitMQ; Search API consumes them (manual ack, one retry, then a `hotels-news-dlq` dead-letter queue) to keep the Solr index in sync, and rebuilds the index on startup / `POST /reindex` by paging `GET /hotels`
- **Shared JWT Authentication** — Users API issues tokens; Hotels API and Search API validate them with the same secret and their own audience; role-based access control (`cliente` / `administrador`)
- **Rate Limiting** — API requests: 10 req/s. Login endpoint: 5 req/min. Connection limit: 20 per IP
- **Security Headers** — X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy
- **CORS Configuration** — Centralized CORS handling at the gateway level with origin whitelist
- **Gzip Compression** — Enabled for JSON, XML, JavaScript, and CSS responses
- **Monitoring** — Nginx status and JSON config endpoint on port 8090
- **Protected Frontend Routes** — React ProtectedRoute component with role-based access

### Known trade-offs (demo scope)

- **Stale cache window on user deletion** — with 3 users-api replicas and an in-process L1 cache, deleting a user only invalidates the replica that served the DELETE: for up to `CACHE_DURATION` (30s) the deleted user can still log in through another replica and obtain a fresh 24h JWT, and stateless JWTs mean already-issued tokens are never revoked. Accepted for this demo; production mitigations: pub/sub cache invalidation, a token denylist, or short-lived tokens + refresh tokens.

---

## 🧪 Testing

Each microservice includes unit tests for both the service and controller layers, using mock repositories.

```bash
# Users API
cd users-api && go test ./... -v

# Hotels API
cd hotels-api && go test ./... -v

# Search API
cd search-api && go test ./... -v
```

**Test strategy:**
- **Controller tests:** Gin + `httptest`, real JWT tokens in headers, covers 401/403/400/200 scenarios
- **Service tests:** Mock repositories (main + cache + queue), validates cache-aside behavior and business logic

---

## 🔗 Access URLs (Local Development)

| Service            | URL                            |
|--------------------|--------------------------------|
| Frontend           | http://localhost:5173           |
| API Gateway        | http://localhost                |
| Gateway Monitoring | http://localhost:8090/status    |
| Nginx Status       | http://localhost:8090/nginx_status |
| RabbitMQ Dashboard | http://localhost:15672 (root/root) |
| Solr Admin UI      | http://localhost:8983/solr      |
| MongoDB            | localhost:27017                 |
| MySQL              | localhost:3307                  |

---

## 👤 Contact

**Developer:** Julian Irusta Roure
