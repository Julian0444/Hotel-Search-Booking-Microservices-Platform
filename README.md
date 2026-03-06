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

### 2. Start backend services

```bash
docker compose up -d --build
```

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

---

## 🌐 API Endpoints (Gateway — Port 80)

| Method   | Endpoint                                      | Service    | Auth     | Description                     |
|----------|-----------------------------------------------|------------|----------|---------------------------------|
| `POST`   | `/users`                                      | Users API  | —        | Register a new user             |
| `POST`   | `/login`                                      | Users API  | —        | Login, returns JWT              |
| `GET`    | `/users`                                      | Users API  | —        | List all users                  |
| `GET`    | `/users/:id`                                  | Users API  | —        | Get user by ID                  |
| `DELETE` | `/users/:id`                                  | Users API  | —        | Delete user                     |
| `GET`    | `/hotels/:id`                                 | Hotels API | —        | Get hotel details               |
| `GET`    | `/hotels/:id/reservations`                    | Hotels API | —        | List hotel reservations         |
| `POST`   | `/hotels/availability`                        | Hotels API | —        | Check availability (multi)      |
| `POST`   | `/reservations`                               | Hotels API | JWT      | Create reservation              |
| `DELETE` | `/reservations/:id`                           | Hotels API | JWT      | Cancel reservation              |
| `GET`    | `/users/:id/reservations`                     | Hotels API | JWT      | User's reservations             |
| `GET`    | `/search?q=...`                               | Search API | —        | Full-text hotel search          |
| `POST`   | `/admin/hotels`                               | Hotels API | Admin    | Create hotel                    |
| `PUT`    | `/admin/hotels/:id`                           | Hotels API | Admin    | Update hotel                    |
| `DELETE` | `/admin/hotels/:id`                           | Hotels API | Admin    | Delete hotel                    |
| `GET`    | `/health`                                     | Gateway    | —        | Gateway health check            |

---

## ✨ Key Features

- **Load Balancing** — Nginx distributes Users API traffic across 3 instances using `least_conn` with automatic failover (`max_fails=3`, `fail_timeout=30s`)
- **Multi-Level Caching** — Users API: L1 (ccache) → L2 (Memcached) → MySQL. Hotels API: ccache (LRU) → MongoDB
- **Event-Driven Architecture** — Hotels API publishes CRUD events to RabbitMQ; Search API consumes them to keep the Solr index in sync
- **Shared JWT Authentication** — Users API issues tokens, Hotels API validates them with the same secret; role-based access control (`cliente` / `administrador`)
- **Rate Limiting** — API requests: 10 req/s. Login endpoint: 5 req/min. Connection limit: 20 per IP
- **Security Headers** — X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy
- **CORS Configuration** — Centralized CORS handling at the gateway level with origin whitelist
- **Gzip Compression** — Enabled for JSON, XML, JavaScript, and CSS responses
- **Monitoring** — Nginx status and JSON config endpoint on port 8090
- **Protected Frontend Routes** — React ProtectedRoute component with role-based access

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
