# 🏨 Hotel Search & Booking Microservices Platform

Full-stack hotel search & booking portfolio: three Go microservices (database-per-service) behind an nginx API gateway, kept in sync through RabbitMQ, with a React SPA on top.

[![CI](https://github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/actions/workflows/ci.yml/badge.svg)](https://github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26.8-00ADD8?logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![Demo — login, search, booking, confirmation, history](docs/demo.gif)

*More screenshots (search, hotel detail, reservations, admin) in [`docs/screenshots/`](docs/screenshots/). The GIF and screenshots were recorded before the latest changes.*

**Verified locally:** 140 frontend unit/component tests, 32 Playwright E2E tests, and Go tests with real MongoDB, Solr and RabbitMQ integration. Results and limits: [`docs/CIERRE.md`](docs/CIERRE.md). Demo walkthrough, existing-data migration and deployment configuration: [`docs/OPERACION.md`](docs/OPERACION.md).

---

## 🎯 What this project demonstrates

- **Microservices done with intent** — three Go services with bounded domains and their own stores (MySQL, MongoDB, Solr), a shared contracts module for HTTP and event definitions, and one nginx gateway doing TLS, `least_conn` load balancing over 3 users-api replicas, rate limiting and stable JSON error envelopes with `trace_id`.
- **Event-driven search (CQRS-lite)** — hotel writes publish `CREATE`/`UPDATE`/`DELETE` events with broker confirms; search-api uses manual ack, delayed retries for transient failures and a dead-letter queue for invalid messages. A single writer reconciles missing, changed and deleted hotels on startup, every minute and on `POST /reindex`. Search covers names, cities and countries, with or without accents.
- **Transactional booking** — MongoDB replica-set transactions commit reservations, per-night inventory and optional **`Idempotency-Key`** records together. Cancellation releases inventory in the same transaction. Concurrent bookings, capacity changes and deletion share a hotel lock; real Mongo tests cover the last room, rollback, lost commit responses and replay after restart. Keys are scoped to a user and canonical payload for 24 hours.
- **Distributed auth** — users-api issues HS256 JWTs (`iss`/`aud` per service); every service validates its own audience with role guards (`cliente`/`administrador`) and owner-or-admin resource rules.
- **Focused caching** — users: in-process L1 → shared Memcached L2 → MySQL read-through. Hotel availability and history read Mongo directly; the gateway does not cache search responses.
- **Operability** — structured JSON logs with request IDs propagated end-to-end, `/livez` + `/readyz`, graceful shutdown, bounded publisher requests and background reconnection. RabbitMQ is optional for readiness: reservations and cancellations work while it is down.
- **Tested at every level** — Go unit/controller tests with `-race` and real JWTs, frontend Vitest+MSW suite, Playwright E2E against the real stack, API collections with assertions, CI with `govulncheck` and Trivy image scanning.

---

## 🏗️ Architecture

```mermaid
flowchart TB
    SPA["React 19 SPA + nginx/Vite proxy<br/>:5173"]
    subgraph GATEWAY["nginx API gateway — :443 TLS (80 → 301)"]
        NG["routing · least_conn LB · rate limiting<br/>JSON error envelopes · security headers"]
    end
    subgraph USERS["users-api (Go)"]
        U1[":8082 ×3 replicas"]
    end
    subgraph HOTELS["hotels-api (Go)"]
        H1[":8081"]
    end
    subgraph SEARCH["search-api (Go)"]
        S1[":8082"]
    end
    MYSQL[("MySQL 8<br/>users")]
    MEMC[("Memcached<br/>shared L2")]
    MONGO[("MongoDB 6 · rs0<br/>hotels · reservations<br/>inventory · idempotency")]
    SOLR[("Solr 9<br/>hotels core")]
    RABBIT[["RabbitMQ 3<br/>hotels-news (+ DLQ)"]]

    SPA -->|"same-origin /api/v1 → TLS gateway"| NG
    NG --> U1 & H1 & S1
    U1 --> MYSQL
    U1 --> MEMC
    H1 --> MONGO
    S1 --> SOLR
    H1 -.->|"publishes CREATE/UPDATE/DELETE"| RABBIT
    RABBIT -.->|"consumes (manual ack)"| S1
    S1 -->|"GET current hotel + paginated catalogue"| H1
```

The full write-up — service boundaries, transactional inventory, the event pipeline, caching, auth, resilience and deployment limits — lives in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

---

## 🛠️ Tech Stack

| Category        | Technology                                                     |
|-----------------|----------------------------------------------------------------|
| **Backend**     | Go 1.26.8 · Gin · GORM · mongo-driver · solr-go · amqp091-go     |
| **Frontend**    | React 19 · Vite 7 · MUI v7 · react-router 8 · TanStack Query 5 · Axios · react-hook-form |
| **Databases**   | MySQL 8 · MongoDB 6 · Apache Solr 9                           |
| **Cache**       | Memcached 1.6 (distributed L2) · ccache (in-process L1)       |
| **Messaging**   | RabbitMQ 3 (AMQP)                                             |
| **Infra**       | Docker · Docker Compose · Nginx (API Gateway + Load Balancer) · Kubernetes (historical reference) |
| **Auth**        | JWT HS256 (`iss`/`aud` per service) · bcrypt                   |
| **Testing**     | Go testing · httptest · testify · Vitest · MSW · Playwright · Bruno |

---

## 🚀 Quickstart

Prerequisites: [Docker](https://docs.docker.com/get-docker/) & Docker Compose, with about 6 GB RAM available to Docker. [Node.js](https://nodejs.org/) v22+ is needed for Vite/tests; Go 1.26.8+ for tools outside Docker.

For an existing installation, follow the [non-destructive migration steps](docs/OPERACION.md#si-ya-tenés-datos) first. `make down` preserves volumes; do not erase them to upgrade.

### 1. Clone the repository

```bash
git clone https://github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform.git
cd Hotel-Search-Booking-Microservices-Platform
```

### 2. Configure environment & generate the local TLS certificate

```bash
test -f .env || cp .env.example .env   # preserve existing credentials

# Generate the local TLS certificate only if the pair is missing.
# Full details in nginx/certs/README.md.
if [ ! -f nginx/certs/key.pem ] || [ ! -f nginx/certs/cert.pem ]; then
  mkdir -p nginx/certs
  openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
    -keyout nginx/certs/key.pem -out nginx/certs/cert.pem \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
fi
```

### 3. Start the backend

```bash
docker compose up -d --build
docker compose ps -a       # wait for healthy services and both one-shot jobs to exit 0
```

This starts nginx, MySQL, Memcached, MongoDB, RabbitMQ, Solr, users-api ×3, hotels-api and search-api, plus two one-shot jobs: `migrate` applies MySQL migrations and `mongo-init` initializes the single-node replica set. The five-hotel catalogue is seeded only in a new Mongo volume; Solr reconciles from that catalogue.

### 4. Log in with the demo credentials

| Role | Username | Password |
|------|----------|----------|
| Customer | `demo` | `DemoCliente123` (seeded by the migrations) |
| Admin | `ADMIN_USERNAME` from your `.env` | `ADMIN_PASSWORD` from your `.env` (seeded idempotently at users-api startup; public registration only ever creates customers) |

### 5. Start the frontend

```bash
cd frontend
npm ci
npm run dev                # http://localhost:5173 (proxies /api/* to the gateway)
```

Or, from the repository root, run the built SPA in Docker: `docker compose --profile frontend up -d --build`. Open **http://localhost:5173**. Both Vite and the built SPA proxy `/api/v1` under the same origin; no separate API hostname is needed in the browser.

### Kubernetes (historical reference)

The repository retains Kubernetes manifests from the earlier implementation. They are not a validated runtime for this version; Mongo replica-set setup and Solr configuration need updating before use. Docker Compose is the verified environment. See [`k8s/README.md`](k8s/README.md).

### Local URLs

| Service            | URL                            |
|--------------------|--------------------------------|
| Frontend           | http://localhost:5173           |
| API Gateway        | https://localhost (self-signed cert; port 80 redirects here) |
| Gateway Monitoring | http://localhost:8090/status (loopback only) |
| RabbitMQ Dashboard | http://localhost:15672 (`root` / `RABBIT_PASSWORD` from `.env`) |
| Solr Admin UI      | http://localhost:8983/solr      |

---

## 🌐 API Endpoints (Gateway — HTTPS, Port 443)

The gateway terminates TLS on port 443 (TLS 1.2/1.3, HSTS) and port 80 only
answers a `301` redirect to HTTPS. Locally the certificate is self-signed, so
use `curl -k` for direct gateway calls. The same-origin SPA needs no browser
certificate exception. A public-domain configuration is provided in the
[operations guide](docs/OPERACION.md#desarrollo-y-dominio-público); it was not deployed.

The API is versioned under a `/api/v1` prefix (URI versioning: a breaking change to any contract ships as `/api/v2` alongside `v1`). Success responses use a single envelope — `{"data": ...}` for single resources, `{"data": [...], "meta": {...}}` for lists — and every error (services and gateway alike) has the shape `{"error": {"code", "message", "trace_id"}}` with stable machine-readable codes. Health endpoints are not versioned.

| Method   | Endpoint                                      | Service    | Auth     | Description                     |
|----------|-----------------------------------------------|------------|----------|---------------------------------|
| `POST`   | `/api/v1/users`                               | Users API  | —        | Register a new user (customer role only) |
| `POST`   | `/api/v1/login`                               | Users API  | —        | Login, returns JWT              |
| `GET`    | `/api/v1/users`                               | Users API  | Admin    | List all users                  |
| `GET`    | `/api/v1/users/:id`                           | Users API  | Owner/Admin | Get user by ID               |
| `DELETE` | `/api/v1/users/:id`                           | Users API  | Owner/Admin | Delete user (204)            |
| `GET`    | `/api/v1/hotels?limit=&offset=`               | Hotels API | —        | List hotels (paginated)         |
| `GET`    | `/api/v1/hotels/:id`                          | Hotels API | —        | Get hotel details               |
| `GET`    | `/api/v1/hotels/:id/reservations`             | Hotels API | Admin    | List hotel reservations (PII: guests' `user_id`) |
| `POST`   | `/api/v1/hotels/availability`                 | Hotels API | —        | Check availability (multi)      |
| `POST`   | `/api/v1/reservations`                        | Hotels API | Logged user | Create reservation (201 + `Location`; supports `Idempotency-Key`) |
| `GET`    | `/api/v1/reservations/:id`                    | Hotels API | Owner/Admin | Get reservation              |
| `DELETE` | `/api/v1/reservations/:id`                    | Hotels API | Owner    | Cancel reservation (204, idempotent; admins cannot cancel others') |
| `GET`    | `/api/v1/users/:id/reservations`              | Hotels API | Owner/Admin | User's reservation history   |
| `GET`    | `/api/v1/users/:id/hotels/:hotel_id/reservations` | Hotels API | Owner/Admin | User's reservations in one hotel |
| `GET`    | `/api/v1/search?q=&sort=`                     | Search API | —        | Full-text search (`sort`: `relevance`/`price_asc`/`price_desc`/`rating_desc`) |
| `POST`   | `/api/v1/reindex`                             | Search API | Admin    | Reconcile the index, including orphan deletion |
| `POST`   | `/api/v1/admin/hotels`                        | Hotels API | Admin    | Create hotel (201 + `Location`) |
| `PUT`    | `/api/v1/admin/hotels/:id`                    | Hotels API | Admin    | Replace editable fields; accepts valid zeros/empty lists |
| `DELETE` | `/api/v1/admin/hotels/:id`                    | Hotels API | Admin    | Delete hotel (204); history prevents deletion (409)|
| `GET`    | `/api/v1/admin/microservices`                 | Hotels API | Admin    | Platform status (read-only, real `/readyz` probes) |
| `GET`    | `/health`                                     | Gateway    | —        | Gateway health check            |

**Full request/response contract:** OpenAPI 3.0 specs per service in [`docs/openapi/`](docs/openapi/) — [`users.yaml`](docs/openapi/users.yaml) · [`hotels.yaml`](docs/openapi/hotels.yaml) · [`search.yaml`](docs/openapi/search.yaml) (validated with `npx @redocly/cli lint docs/openapi/*.yaml`). Ready-to-run [Bruno](https://www.usebruno.com/) collections live in [`Bruno API tester/`](Bruno%20API%20tester/), with per-request assertions covering the whole table.

> Each Go service also exposes internal health endpoints: `/livez` (process liveness, `/health` is an alias) and `/readyz`. Readiness requires MySQL for users, Mongo for hotels and Solr for search. RabbitMQ/Memcached failures remain visible as degraded checks but do not block functions that can operate without them. A required dependency failure returns 503. Docker Compose healthchecks use `/readyz`.

---

## 🧪 Testing

```bash
# From the repository root: all Go modules and real integration tests
make build
make test                 # race detector included
make lint
make test-integration     # isolated Mongo rs0, Solr and RabbitMQ

# A single module
cd users-api && go test ./... -v
```

The frontend has its own test pyramid (Vitest + Testing Library + MSW for unit/component, Playwright for end-to-end):

```bash
cd frontend
npm run test:run        # unit/component suite
npm run test:coverage   # same suite with coverage thresholds
npm run check           # lint + coverage + production build (same gate as CI)

# End-to-end (requires the full stack with the frontend profile):
npx playwright install chromium   # first time only
cd ..                            # make targets live at the repository root
make e2e                          # or: docker compose --profile frontend up -d --build && cd frontend && npm run test:e2e
```

The Bruno collections double as API smoke tests — every request asserts its expected status (including the idempotent-replay and invalid-sort contract checks):

```bash
cd "Bruno API tester/Users API Collection"    # same for Hotels / Search
npx -y @usebruno/cli run --env local --insecure --env-var "admin_password=YOUR_ADMIN_PASSWORD"
```

**Test strategy:**

- **Controller tests:** Gin + `httptest`, real JWT tokens in headers, covers 401/403/400/200 scenarios
- **Service tests:** Mock repositories/queues for domain rules; real Mongo transactions for inventory, idempotency and ambiguous commits
- **Frontend unit/component:** MSW mocks the `/api/v1` contract (envelopes, errors with `trace_id`); axe runs on every main route
- **Frontend E2E:** Playwright drives the built SPA against the real gateway — anonymous search, customer booking/cancellation, admin CRUD with search sync, service-degradation states, and 320/390 px viewport checks
- **CI:** lint + unit tests (`-race`) + builds per Go module, frontend gate (lint/coverage/build/audit), `govulncheck`, Docker image builds with Trivy scanning; integration and E2E jobs on PRs or manual workflow dispatch (not every push)

---

## 🧠 Architecture decisions & trade-offs

[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) documents why transactions replace compensation, how thin events and a single reconciliation writer keep Solr recoverable, the remaining user-cache semantics and the auth model. Mongo uses one replica-set node; search has one writer; indexing is eventual; JWTs are not revoked. Hotel CRUD preserves its successful HTTP result when Mongo commits but publishing fails, and reconciliation repairs the remaining gap. The project calculates prices; it does not process payments.

## 🚢 How I'd take this to production

The demo runs on one machine by design. These are considerations for a future deployment, not implemented guarantees or pending scope for this portfolio:

1. **Secrets & TLS** — move `.env` to a secrets manager (the env contract is already 12-factor); swap the self-signed cert for a real one (config already split — only the files change).
2. **Managed datastores** — RDS/Cloud SQL for MySQL, a redundant MongoDB replica set for the transaction model, managed Solr/OpenSearch.
3. **Event reliability** — replace publish-after-write with a transactional outbox (or CDC); today bounded publishing + periodic reconciliation recover the index when the source is available.
4. **Token lifecycle** — short-lived access tokens + refresh tokens (or a gateway denylist); today JWTs live 24 h and are never revoked.
5. **Cache invalidation across replicas** — pub/sub invalidation for the per-replica L1 (today L1 invalidation is local; TTL and concurrent repopulation do not provide a universal 30-second stale bound).
6. **Observability** — Prometheus/Grafana metrics and OpenTelemetry traces; the `trace_id` plumbing and structured logs are already in place.
7. **Delivery** — the CI already builds and scans multi-arch images (GHCR + Trivy); validate deployment manifests and their datastore requirements before adding CD. This version has not been deployed to Kubernetes or a public domain.

---

## 👤 Contact

**Julian Irusta Roure** — [GitHub](https://github.com/Julian0444)

Licensed under the [MIT License](LICENSE).
