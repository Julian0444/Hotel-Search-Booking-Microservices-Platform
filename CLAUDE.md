# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Docker Compose — 10 containers)

```bash
cp .env.example .env          # required: compose uses ${VAR:?} and fails without it
docker compose up -d --build
docker compose ps             # wait until healthy (Solr takes ~90s)
docker compose --profile frontend up -d   # optional: built SPA on http://localhost:5173 (I8)
```

Both users-api and hotels-api `log.Fatal` at startup if `JWT_SECRET` is unset or still the code default.

### Go tests (no Docker needed — mock repositories + httptest)

There is a root `go.work` (users-api, hotels-api, search-api, platform-contracts) and a root Makefile. Note the workspace root is not itself a module, so from the root you must enumerate modules (or use the Makefile targets):

```bash
make build && make test           # all modules
go build ./users-api/... ./hotels-api/... ./search-api/... ./platform-contracts/...
cd users-api && go test ./internal/services/users/ -run TestLogin -v   # single test
```

### Frontend

```bash
cd frontend
npm install
npm run dev      # port 5173; proxies /api/* → nginx gateway on localhost:80, stripping the /api prefix
npm run lint     # eslint
npm run build
```

## Architecture

Three independent Go microservices + React SPA, orchestrated by `docker-compose.yml` with nginx (`nginx.conf`) as the API gateway on port 80 — routing, rate limiting, CORS, and load balancing of 3 users-api replicas (`least_conn`). All client traffic goes through the gateway; see the endpoint table in README.md.

- **users-api** (Go, port 8082 ×3): registration/login, issues JWTs. Read-through cache: L1 ccache (in-process) → L2 Memcached → MySQL (GORM).
- **hotels-api** (Go, port 8081): hotel CRUD + reservations. Cache-aside ccache → MongoDB. Publishes `CREATE`/`UPDATE`/`DELETE` events to the RabbitMQ `hotels-news` queue.
- **search-api** (Go, port 8082): consumes `hotels-news`, fetches the full hotel from hotels-api over HTTP, syncs the Solr `hotels` core; serves `GET /search`.

Each Go service has the same layered layout, wired in `cmd/main.go`: `internal/config` (env vars) → `controllers` (Gin handlers) → `services` (business logic) → `repositories` (DB + cache) → `dao`/`domain` (models), plus service-specific `clients/queues` (RabbitMQ), `middlewares`, `tokenizers`.

### Auth flow (spans services)

`users-api/internal/tokenizers/tokenizers_jwt.go` mints HS256 JWTs with claims `user_id`, `username`, `tipo` (role: `cliente` | `administrador`), `iss: "users-api"`, `aud: ["users-api", "hotels-api"]`. Both users-api and hotels-api have an `internal/middlewares/auth.go` that validates the shared `JWT_SECRET` and each service's own audience, with role guards (`AdminOnly`, `LoggedUserOnly`). Public registration always creates `cliente`; the first admin is seeded idempotently at users-api startup from `ADMIN_USERNAME`/`ADMIN_PASSWORD`.

### Test conventions

- Controller tests: Gin + `httptest` with real JWTs in headers, covering 401/403/400/200.
- Service tests: mock repositories (main + cache + queue) validating cache-aside behavior.

## The plans/ workflow (how work is organized here)

This repo is being hardened as a portfolio piece via a fixed sequence of plans:

- `plantofinish.md` — master audit; every finding has an ID (S1, D2, C11, …). Do not edit.
- `plans/README.md` — **source of truth**: execution order, dependency graph, checkbox status, and the ID→plan traceability table.
- `plans/NN-*.md` — 13 self-contained plans, each executable in a fresh session.

When executing a plan: validate its code snippets against the current code first (earlier plans may have moved things), run its **Verificar** block when done, tick its checkbox in `plans/README.md`, and commit. Plan 01 (security/auth) is complete.

## Known deliberate quirks — do not "fix" in passing

- The historical misspelling of `AvailableRooms`/`available_rooms` was renamed in plan 11 (C11) as one atomic coordinated change (contracts, BSON, Solr schema, frontend, seeds, goldens). Existing Mongo data needs the one-off migration `hotels-api/seed/rename-available-rooms.js` (the only file that still carries the old field name, on purpose).
- search-api's module path and Go version were unified with the other services in plan 02 (full GitHub path, shared toolchain via `go.work`).
- Plans and much of the documentation are in Spanish; code comments mix Spanish and English. Keep that style.
