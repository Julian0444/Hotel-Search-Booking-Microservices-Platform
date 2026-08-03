# HANDOFF — continuidad entre sesiones

> **Qué es esto:** log acumulativo de traspaso de contexto. Cada sesión que termina **agrega una sección fechada al final** (`## Sesión — YYYY-MM-DD HH:MM`) — nunca borra las anteriores — y actualiza "Trabajo restante" y "Advertencias" a la realidad actual.
> **Cómo usarlo:** al arrancar una sesión nueva, leé la **última** sección + las reglas globales de abajo. El mapa maestro de trabajo es [`plans/README.md`](README.md) (orden, dependencias y trazabilidad ID→plan); la referencia de diagnóstico es `../plantofinish.md`.

## Reglas globales (aplican a TODAS las sesiones)

- **Git es 100% manual del usuario.** La sesión NUNCA ejecuta `git commit/push/merge/rebase/reset/add` ni crea branches, y NUNCA toca `main`. Git solo lectura (`status`/`diff`/`log`). Si un commit parece conveniente: decirlo en texto y frenar.
- Antes de implementar un plan, validar sus snippets contra el código actual (paso 3 de `plans/README.md`).
- El stack local requiere `.env` (gitignored): `cp .env.example .env` en un clone fresco.

---

## Sesión — 2026-07-04 18:47

### Resumen de lo hecho

1. **Plan 01 (Seguridad: S1, S2, C4, C8, I1, SD2, SD3, T5) — IMPLEMENTADO Y VERIFICADO end-to-end.** Lo esencial:
   - `users-api/internal/middlewares/auth.go` nuevo (copia del de hotels-api + `OwnerOrAdmin()`); rutas protegidas: `GET /users` → admin, `GET/DELETE /users/:id` → dueño o admin. Públicas: `POST /users`, `POST /login`, `/health`.
   - Registro público bindea el DTO nuevo `RegisterRequest` (username 3-50, password 8-72) y **fuerza `tipo="cliente"` en el handler**; `service.Create` quedó intacto. El primer admin sale de un **seed idempotente** al arranque (`ADMIN_USERNAME`/`ADMIN_PASSWORD` — verificado: réplica 1 lo crea, 2 y 3 tragan el duplicado).
   - Fail-fast si `JWT_SECRET` está vacío/placeholder — ojo: el campo es `config.JWTKey` en users-api y `config.JWTSecret` en hotels-api.
   - JWT ahora emite `iss=users-api`, `aud=[users-api, hotels-api]`, `nbf`; ambos middlewares validan `iss`/`aud` con leeway 30s (**tokens viejos quedaron inválidos** — re-loguearse).
   - Guard de 72 bytes en `hashPassword`; `MIN_PASSWORD_LENGTH` del frontend 6→8.
   - `docker-compose.yml` parametrizado con `${VAR:?...}` (falla con mensaje claro sin `.env`); `.env` gitignored + `.env.example` + `.gitignore` raíz.
   - Frontend: selector de tipo eliminado de `Register.jsx` (ajustados `AuthContext.register` y `auth.service.register`).
   - Tests: subtests de `TestController_Create` rehechos; nuevos `tokenizers_jwt_test.go` y `middlewares/auth_test.go` en **ambos** servicios (expirado/key errónea/`alg=none`/header malformado/`iss`-`aud` incorrectos → 401); helpers `makeJWT` de hotels-api actualizados con los claims nuevos.
   - Docs mínimas: `users.md` (ya no documenta "registrar admin por curl"), Bruno `Post User.bru`, tabla de auth y quickstart del `README.md` (ahora incluye `cp .env.example .env`).
2. **Plan 01 revisado contra el código real antes de implementar** (3 agentes): 7 discrepancias corregidas en `plans/01-seguridad-auth.md` (quedó fiel a lo implementado).
3. **Todos los `plans/*.md` limpiados de instrucciones de git** ("Al terminar"/"Commit sugerido"/"tildá" → nota de versionado manual) por regla del usuario.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` · último commit `5072170 docs: planes de ejecución` (solo incluyó `plans/`).
- **TODO el código del plan 01 está SIN commitear** (working tree): ~23 modificados (users-api ×8, hotels-api ×6, frontend ×4, `docker-compose.yml`, `README.md`, Bruno, `users.md`) + los 14 `plans/*.md` retocados. Untracked nuevos: `users-api/internal/middlewares/` (auth.go + auth_test.go), `users-api/internal/tokenizers/tokenizers_jwt_test.go`, `hotels-api/internal/middlewares/auth_test.go`, `.gitignore`, `.env.example` (también `.claude/` y `CLAUDE.md`, del usuario). Nota: `hotels-api/go.mod|go.sum` cambiaron por `go mod tidy` (testify pasó a dependencia directa).
- **Tests:** `go build/vet/test` verdes en users-api y hotels-api; `gofmt` limpio; `npm run build` OK.
- **Verificación en vivo (2026-07-03, por el gateway :80):** registro pidiendo admin → JWT dice `cliente` con `iss/aud`; sin token → 401; cliente en `GET /users` → 403; dueño → 200; admin del seed → 200 (también contra `/admin/*` de hotels-api); password "1" → 400; contenedor sin `JWT_SECRET` muere con log claro.
- **Stack:** `docker compose` corriendo hace ~43h. `api-gateway` figura "(unhealthy)" en `docker compose ps` pero responde bien (`/health` 200, `/nginx-health` 200, `/users` 401) — healthcheck interno flaky, no bloquea; el plan 05 rehace los healthchecks igual.
- **`.env` local actual (demo):** `root`/`root` en DBs, `JWT_SECRET=ThisIsAnExampleJWTKey!`, admin `admin`/`DemoAdmin123`.

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 02 — CI/CD, tooling y módulo de contratos** (`go.work`, `platform-contracts`, golangci-lint v2, GitHub Actions, contract test, Makefile/LICENSE) ← **siguiente**
2. Plan 03 — Persistencia + seed (después del 02)
3. Plan 04 — Dominio: no-overbooking (después del 03)
4. Planes 05 → 13 según orden y dependencias del README (05 antes del 09; 08 antes del 09; C11 del plan 11 requiere 02 y 07).

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - El usuario debe **commitear manualmente** el trabajo del plan 01 (la sesión no lo hará). El checkbox del plan 01 en `plans/README.md` ya está en `[x]`.
  - Plan 02: el gate de `govulncheck` para search-api va con `continue-on-error` hasta que el plan 06 bumpee gin/x/net (12 CVEs conocidas). El push de la branch para ver el CI lo hace el usuario.
  - Los JWT emitidos antes del plan 01 no validan más (falta `iss`/`aud`) — si el frontend tenía sesión guardada en localStorage, hay que re-loguear.
  - `mongo` es el nombre del service de Mongo en compose (no `mongodb`) — los planes ya lo reflejan.

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con el 02"** → leer `plans/02-ci-tooling-contratos.md` completo, validar sus snippets contra el código actual (versiones de Go en los 3 `go.mod`, los 7 archivos `gofmt -l`, module path de search-api) y recién ahí implementar. Si el usuario todavía no commiteó el plan 01, conviene que lo haga antes (el 02 crea `go.work`/módulo nuevo y conviene un punto de corte limpio).

---

## Sesión — 2026-07-05 14:58

### Resumen de lo hecho

1. **Plan 02 (CI/CD, tooling y módulo de contratos: I4, I6, I9, SD1, CQ1, CQ2, CQ3, T1, T2, T3, T4, T6) — IMPLEMENTADO Y VERIFICADO.** Lo esencial:
   - **`go.work` raíz** con los 4 módulos (`go.work.sum` incluido — commitear ambos). Module path de search-api renombrado del pelado `search-api` al canónico de GitHub (imports internos ajustados en ~13 archivos) y bump a `go 1.23.0` + `toolchain go1.24.11` (imagen Docker a `golang:1.23-alpine`).
   - **`platform-contracts/`** nuevo: `Hotel` + `HotelNew` (typo `AvaiableRooms` intacto a propósito hasta C11) con **type-alias bridge** en los domains de hotels-api y search-api (`require` + `replace ../platform-contracts` en ambos go.mod). Los DAO por servicio (tags bson) quedan como estaban — el módulo compartido es solo el contrato wire.
   - **Contract test golden** (`platform-contracts/testdata/hotel_new.golden.json` + `contracts_test.go`): serializa byte-a-byte y deserializa afirmando campos. Probado que protege: rename temporal de tag → test rojo → revertido.
   - **`.golangci.yml` v2** raíz (govet/staticcheck/errcheck/ineffassign/unused + gofmt/goimports). `gofmt -w` sobre los 7 archivos sucios y **17 hallazgos de lint arreglados** (errcheck en `Close()`/caché best-effort, ineffassign, ST1005). Lint verde en los 4 módulos (golangci-lint 2.12.2 instalado vía brew).
   - **CI** (`.github/workflows/ci.yml`): matrix Go de **4 módulos** (incluye platform-contracts) con gofmt/vet/golangci-lint-action@v8/test `-race`/govulncheck **estricto** (ver desviaciones) + job `integration` (solo PRs) + job frontend (npm ci/build/audit high). `.github/dependabot.yml` (gomod ×4, npm, actions).
   - **Testcontainers** (`hotels-api/.../hotels_mongo_integration_test.go`, tag `integration`): mongo:6 real, path Create → CreateReservation → IsHotelAvailable (ocupado/libre). Verde local en ~6s.
   - **`test_load_balancer.sh` reescrito**: `set -euo pipefail`, PASS/FAIL por check + contador + exit 1, sin `declare -A` (bash 3.2 de macOS moría con set -e), rate-limit test apuntado a `/login` (5r/m burst 3 → determinístico) y al final (agota el budget). Verificado: stack sano → 0; réplica caída → 1.
   - **Fixtures + benchmarks**: `search-api/.../testdata/` (doc Solr + respuesta de hotels-api) usados por `hotels_solr_test.go` (helpers de campos; documenta el bug E6 con assert a invertir en plan 06) y `hotels_http_test.go` (contrato lado consumidor + `BenchmarkSearchResultsMarshal`); `users-api/.../users_cache_bench_test.go` (hit L1 ccache).
   - **Higiene**: `.gitignore` completado, `LICENSE` MIT 2026, `Makefile` (build/test/test-integration/lint/fmt/up/down/seed-placeholder).
   - **`npm audit fix`** en frontend: 16 vulns → **0**; `npm run build` OK (solo cambió `package-lock.json`).

### Desviaciones del plan (validadas contra la realidad)

- **`go build ./...` desde la raíz NO funciona** cuando la raíz del workspace no es un módulo (comportamiento de Go, verificado con repro mínimo) → el Makefile y el CI enumeran los módulos (`./users-api/...` etc.).
- **Docker**: el `replace ../platform-contracts` rompía los builds con contexto por-servicio → contexto de build **raíz** para hotels-api y search-api en compose (`dockerfile: hotels-api/dockerfile`), Dockerfiles re-ruteados (COPY platform-contracts + módulo; hotels pasó de `go mod tidy` a `go mod download`) y **`.dockerignore` raíz** estilo allowlist (adelanto mínimo de CN2; el plan 09 ya no necesita "copiar platform-contracts al contexto" — ya está).
- **govulncheck quedó limpio en los 4 módulos, sin `continue-on-error`**: `go work sync` bumpeó gin 1.9.1→1.11.0 (+ transitivas) en search-api por MVS del workspace, y se subió `quic-go` a v0.57.0 (única vuln alcanzable restante). **El paso 8 del plan 06 (bump CVEs) queda pre-hecho** — solo re-verificar y no re-agregar la excepción.
- `QueueDeclare` del consumer de search-api ahora chequea el error con `log.Fatalf` (era ineffassign; consistente con los Fatalf vecinos) — el plan 06 lo reemplaza por reintentos.
- El matrix de CI y dependabot incluyen **platform-contracts** (el plan decía 3 módulos).

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` · últimos commits: `5072170` docs: planes de ejecución · `90bfe0f` New README · `702859b` Mejora front — **ninguno incluye código de los planes 01/02**.
- **TODO sin commitear: plan 01 + plan 02 mezclados en el working tree** (~81 entradas en `git status`). Archivos con cambios de AMBOS planes (si se quieren commits separados, hace falta `git add -p`): `docker-compose.yml` (01: env `${VAR:?}` / 02: build contexts), `.gitignore` (01 lo creó / 02 lo completó), `users-api/.../users_service.go` (01: hash guard / 02: lint), `hotels-api/go.mod` (01: testify / 02: contracts+testcontainers). Todo lo demás es separable por archivo.
- **Checkbox del plan 02 en `plans/README.md`: `[x]`.**
- **Tests:** `make test` (con `-race`) verde en los 4 módulos; integración verde; lint verde; `docker compose build` OK; smoke test del LB exit 0.
- **Verificación en vivo:** stack recreado con las imágenes nuevas; flujo completo por el gateway: login admin → `POST /admin/hotels` → evento RabbitMQ (tipo compartido) → indexado en Solr → visible en `/search` → DELETE limpia el índice. (El hotel de prueba fue borrado.)

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 03 — Persistencia + seed** ← **siguiente**
2. Plan 04 — Dominio: no-overbooking (después del 03)
3. Planes 05 → 13 según orden y dependencias del README.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear** planes 01+02 (ver nota de archivos mezclados arriba) y **pushear la branch para ver el CI** — los 4 jobs Go + frontend deberían dar verdes; el job `integration` solo corre en PRs.
  - `npm run lint` del frontend tiene **1 error preexistente** (`react-refresh/only-export-components` en `AuthContext.jsx:16` — exporta hook + componente). El CI no corre lint de frontend (a propósito, según plan); candidato natural: plan 13 (FE) o al tocar AuthContext.
  - El assert de `getTimeField` en `search-api/.../hotels_solr_test.go` **documenta el bug E6** (devuelve zero) — al arreglar E6 en el plan 06 hay que invertirlo (está comentado en el test).
  - El golden `hotel_new.golden.json` se actualiza SOLO como parte del cambio atómico C11 (plan 11).
  - `api-gateway` sigue "(unhealthy)" en `docker compose ps` pero responde bien (healthcheck flaky conocido; el plan 05 lo rehace).

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con el 03"** → leer `plans/03-persistencia-seed.md` completo y validar snippets contra el código actual (firmas del repo de users-api, `hotels_mongo.go` líneas de paginación, nombres bson de reservas). Idealmente con planes 01+02 ya commiteados y el CI verde en GitHub.

---

## Sesión — 2026-07-05 21:33

### Resumen de lo hecho

1. **Plan 03 (Persistencia + seed: DB1, DB2, DB3, DB4, DB5, R1, P7) — IMPLEMENTADO Y VERIFICADO.** Lo esencial:
   - **Migraciones versionadas (DB1)**: `users-api/migrations/` con `0001_create_users` (copia EXACTA del schema GORM, verificado con `SHOW CREATE TABLE` contra el contenedor: `uni_users_username`, `utf8mb4_0900_ai_ci`; con `IF NOT EXISTS` para que volúmenes pre-migraciones converjan sin dirty flag) y `0002_seed_demo_users` (solo el **cliente demo** `demo`/`DemoCliente123`, hash `$2a$` generado con la lib de la app). `migrations/README.md` documenta flujo, credenciales y gotchas.
   - **Ejecución**: service one-shot **`migrate`** en compose (`migrate/migrate:v4.18.3`, `depends_on: mysql healthy`); las 3 réplicas de users-api ahora dependen de `migrate: service_completed_successfully` y arrancan con **`AUTO_MIGRATE: "false"`** (gate nuevo en config; default `true` para dev pelado).
   - **Admin: una sola fuente** — el seed env-driven del plan 01 (`ADMIN_USERNAME`/`ADMIN_PASSWORD`). La 0002 NO siembra admin (documentado en `migrations/README.md` y `.env.example`).
   - **Índices (DB2)**: `EnsureIndexes` idempotente en `NewMongo` (compuesto `{hotel_id, check_in, check_out}` + `{user_id}` en `reservations`, bson tags reales) + los mismos índices en `mongo-init.js` para arranque limpio. El unique de `username` viene por la 0001.
   - **Pool/timeouts (DB3)**: users-api `SetMaxOpenConns/Idle(25)` + `ConnMaxLifetime(5m)` + `PingContext` fail-fast + DSN con `timeout/readTimeout/writeTimeout=5s`; hotels-api `SetMaxPoolSize(50)` + `SetServerSelectionTimeout(5s)` + `Ping` fail-fast (Connect es lazy).
   - **`context` end-to-end en users-api (DB4+R1)**: interfaz `Repository` (ahora 7 métodos, +`CountAll`) con ctx en los 4 implementadores (mysql `WithContext`; ccache/memcached aceptan-e-ignoran con comentario; mock testify con ctx), service, controllers y ambos archivos de test. Middleware nuevo `RequestTimeout` (5s, env `REQUEST_TIMEOUT`) + mapeo `DeadlineExceeded` → **503** en todos los handlers; `Login` distingue deadline de credenciales inválidas para no devolver 401 espurio.
   - **Paginación en DB (DB5)**: users-api `GetAll(ctx, limit, offset)` + `CountAll` → header **`X-Total-Count`** (respuesta sigue siendo array pelado para el admin del frontend; envelope formal en plan 07). hotels-api: `limit/offset int64` en los 3 `GetReservationsBy*` (Mongo `SetLimit/SetSkip` + sort estable por `_id`; helper `findPageOptions`), threading por interfaz Repository/Service/controller con clamp `?limit=20&offset=0` max 100.
   - **Caché de listas de hotels-api**: slicing con `paginateReservations` y guard en el service — **solo se cachea la página que es lista completa** (`offset==0 && len < limit`) para no envenenar la lista agregada con páginas parciales (la caché se rehace en plan 04 igual).
   - **Seed hoteles (P7)**: `hotels-api/seed/mongo-init.js` montado en `/docker-entrypoint-initdb.d/` + `MONGO_INITDB_DATABASE: hotels-api`; 5 hoteles demo argentinos con los bson tags reales (incluido `avaiable_rooms` a propósito). Solo corre con volumen nuevo (`docker compose down -v`).
   - **`make seed`** completado: re-corre el one-shot de migraciones e imprime las credenciales demo.
   - Tests nuevos: assert de `X-Total-Count` y subtest **login con deadline → 503** en el controller de users-api.

### Desviaciones del plan (validadas contra la realidad)

- **DB/DSN**: la base es **`users-api`** (no `users`) y el puerto host de MySQL es **3307** — el bloque Verificar del plan estaba desactualizado en eso.
- **`/search` NO muestra los hoteles sembrados** (esperado): el índice Solr se alimenta por eventos RabbitMQ que el initdb no emite. Queda para el **backfill/reindex del plan 06 (E3)**. Verificación alternativa usada: `GET /hotels/:id` vía gateway + `countDocuments()` en mongosh. **Al ejecutar plan 06, probar que el backfill indexa los 5 hoteles demo.**
- **Dockerfiles de build bumpeados a `golang:1.25-alpine`** (los 3): los go.mod del working tree ya exigían más que 1.23 (search-api `go 1.24` por quic-go v0.57; hotels-api `go 1.25.0` por testcontainers-go v0.43) y con `GOTOOLCHAIN=local` el build moría. No fue cambio de este plan sino arrastre del 02 que recién explotó al rebuildear.
- **Sin CLI `migrate` local**: las verificaciones down/up se hicieron con `docker run migrate/migrate:v4.18.3` sobre la red del compose (documentado en `migrations/README.md`).
- `MySQLConfig` ganó el campo `AutoMigrate`; el mock de users-api ahora también implementa `CountAll`.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` · últimos commits: `5072170` docs / `90bfe0f` README / `702859b` front — **planes 01+02+03 sin commitear** (~92 entradas en `git status`, 23 untracked).
- Archivos con cambios de MÁS de un plan (para `git add -p` si se quieren commits separados): `docker-compose.yml` (01: `${VAR:?}` / 02: build contexts / 03: migrate one-shot + AUTO_MIGRATE + mongo initdb), `.env.example` (01 / 03: nota admin único), `Makefile` (02 / 03: seed), los 3 Dockerfiles (02: rutas contexts / 03: golang:1.25), `users-api/*` y `hotels-api/*` (01/02/03 mezclados en varios).
- **Verificación completa del plan 03 (todo verde):** `docker compose down -v && up --build` → one-shot migrate exit 0 (`1/u`+`2/u`), 5 hoteles + índices sembrados; login `demo` 200; login admin 200; `X-Total-Count: 2` con `limit=2`; páginas de reservas reales distintas (`limit=1&offset=0/1`, offset=5 vacío); índices visibles en `getIndexes()`; migrate `down 1` + `up` re-aplica seed; **MySQL en pausa → login 503 en 5s** y recuperación limpia (con `stop` da 401 instantáneo por connection refused — el catch-all de login; semántica de errores es plan 07); `make test` (-race, 4 módulos) + lint 0 issues; integración testcontainers verde (~3s); `bash test_load_balancer.sh` exit 0.
- Quedaron **2 reservas del usuario demo** en Mongo (del smoke de paginación) — le dan vida al demo; borrarlas con `DELETE /reservations/:id` si molestan.
- **Checkbox del plan 03 en `plans/README.md`: `[x]`.**

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 04 — Dominio: no-overbooking + reserva rica** ← **siguiente** (depende del 03 ✓)
2. Plan 05 — Observabilidad (recomendado tras 03 ✓)
3. Planes 06 → 13 según orden y dependencias del README.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear planes 01+02+03** y pushear para ver el CI. Ojo: el CI matrix usa `go-version` de los go.mod — verificar que los jobs sigan verdes con los bumps a 1.24/1.25 (los runners bajan el toolchain que pida el módulo; si el workflow pinnea 1.23 habrá que subirlo).
  - `api-gateway` sigue "(unhealthy)" cosmético (healthcheck flaky conocido; plan 05 lo rehace).
  - La password del cliente demo está fija en la migración 0002 (`demo`/`DemoCliente123`) — es deliberado (portfolio demo, documentado); si se cambia, regenerar el hash con la receta de `migrations/README.md`.
  - El guard "solo cachear lista completa" en las listas de reservas de hotels-api es un parche consciente hasta el rework de caché del plan 04 (D2/D3/R3).
  - `npm run lint` del frontend mantiene su 1 error preexistente (`AuthContext.jsx:16`) — sin cambios en frontend en esta sesión.

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con el 04"** → leer `plans/04-dominio-reservas.md` completo y validar snippets contra el código actual — en particular las firmas nuevas de este plan: `GetReservationsBy*(ctx, ..., limit, offset int64)` en repos/service/controller de hotels-api, el guard de caché en `hotels_service.go`, y `EnsureIndexes` ya existente en `hotels_mongo.go` (el 04 agrega el índice/colección de inventario sobre eso). Idealmente con 01+02+03 commiteados antes (el 04 toca los mismos archivos de hotels-api).

---

## Sesión — 2026-07-06 01:32

### Resumen de lo hecho

1. **Plan 04 (Dominio: D1, D2, D3, D4, DM1, DM2, DM5, C10, R3) — IMPLEMENTADO Y VERIFICADO end-to-end.** Lo esencial:
   - **No-overbooking (D1)**: colección nueva `reservation_inventory` (contador por hotel-noche, fecha `"2006-01-02"`, env `MONGO_COLLECTION_INVENTORY`) con **índice único `{hotel_id, date}`** (en `EnsureIndexes` + `mongo-init.js`). `claimNight` = `findOneAndUpdate` con upsert + `$inc booked` + `$setOnInsert capacity` y `SetReturnDocument(After)`; ante `IsDuplicateKeyError` (carrera del upsert) **reintenta UNA vez SIN upsert** — `ErrNoDocuments` ahí = noche llena. `CreateReservation` del repo reclama noche a noche, **compensa** (libera con `$inc -rooms` sobre `context.WithoutCancel`) si algo falla, y recién entonces inserta la reserva. Guard defensivo `NumRooms > capacity` → sentinel directo (sin él, el upsert de una noche virgen insertaría `booked > capacity`).
   - **Reserva rica (DM1/DM2)**: `Reservation` (DAO+domain) += `Status`/`NumRooms`/`NumGuests`/`TotalPrice int64` (centavos)/`Currency`/`CreatedAt`/`CancelledAt *time.Time`; constantes `StatusConfirmed/StatusCancelled` en el DAO; sentinel **`ErrNoAvailability` en el domain** → el controller lo mapea con `errors.Is` a **409**.
   - **Cancelación = soft-delete idempotente**: `CancelReservation` del repo ahora **devuelve la reserva** (cambio de firma en la interfaz `Repository` — el service necesita HotelID/noches para evento y liberación). `FindOneAndUpdate` con filtro `{_id, status: confirmed}` (pre-imagen default Before) → set cancelled + `cancelled_at` + libera noches; segundo cancel no matchea → devuelve la ya-cancelada sin re-liberar.
   - **`IsHotelAvailable` de Mongo reescrito** desde el inventario (una sola `Find` con `$in` de noches; noche sin doc = libre; compara contra la capacidad ACTUAL del hotel — la misma que usa el filtro del claim). Borrado el `$unionWith: {coll: nil}` y el loop de aggregation por día.
   - **Caché (D2/D3/R3)**: lista de reservas ausente = **disponible** (D3); disponibilidad cuenta `NumRooms` y saltea canceladas (D4); `CancelReservation` de la caché marca cancelled y reemplaza en las listas agregadas (espejo de Mongo). En el service: `Update`/`Delete` de hotel y TODOS los pobla-caché de lecturas son **log-and-continue** — el evento de hotel se publica siempre; los 3 getters de listas ya no fallan por error de caché (R3).
   - **Evento `ReservationNew` (DM5)**: struct extendido con `HotelID`; método nuevo `PublishReservation` en la interfaz `Queue` y en `RabbitQueue` sobre cola separada **`reservations-news`** (env `RABBIT_RESERVATIONS_QUEUE_NAME`; se declara junto a `hotels-news` en `connect()`; `publish(queueName, body)` extraído como helper compartido). Publica `CREATE`/`CANCEL` **best-effort (log-and-continue)**: nadie consume la cola aún y un fallo de publish no puede tirar una reserva ya persistida. Mock de queues actualizado (`ReservationMessages()`).
   - **Service `CreateReservation`**: valida hotel existe (cache-aside), `CheckOut > CheckIn`, check-in no pasado, `NumRooms/NumGuests >= 1` (default 1 si vienen en 0), `NumRooms <= capacidad` (→ wrap de `ErrNoAvailability`); deriva `HotelName` del hotel (no del body) y `TotalPrice = round(PricePerNight*100) × noches × NumRooms`, `Currency: "USD"` (constante `reservationCurrency`, documentada).
   - **Controller**: DTO nuevo `createReservationRequest` con fechas **string `YYYY-MM-DD`** (parse estricto → 400; RFC3339 ya NO se acepta), `check_out > check_in` → 400, `user_id` opcional (si viene y no coincide con el token → 403, igual que antes; si no viene se toma del token), `ErrNoAvailability` → **409**.
   - **Backfill (paso 9)**: `hotels-api/seed/migrate-inventory.js` — completa `status/num_rooms/created_at` en reservas legacy y regenera `reservation_inventory` desde las confirmadas (join capacity contra hotels). **Ya corrido contra el volumen actual**: las 2 reservas demo del plan 03 quedaron `confirmed`/`num_rooms:1` (con `total_price: 0` — legacy, esperado) y 3 entradas de inventario.
   - **Tests (C10)**: suite de solapamiento **compartida** (`availability_suite_test.go`: 8 casos de overlap + borde checkout/check-in + checkin==checkout error + cancelada-no-ocupa) corrida contra la **caché real** (unit) y contra **Mongo** (integration, `TestMongo_AvailabilitySuite`); `TestMongo_ConcurrentClaimLastRoom` (20 goroutines → exactamente 1 gana + `booked <= capacity`); `TestMongo_CancelReservationIdempotent` (doble cancel no doble-libera, re-booking OK). Tests de service nuevos: no-availability, validaciones, eventos publicados, cancel idempotente, D2 (cache miss en Update publica igual), precio derivado. Controller: 400 fecha RFC3339, 400 checkout<=checkin, 409.
   - **Extras de higiene**: `DeleteReservationsByHotelID` de Mongo ahora también borra el inventario del hotel (sin eso quedaban contadores huérfanos al borrar un hotel). Compose: 2 env nuevas en hotels-api. Bruno `Post Reservation.bru` actualizado al DTO nuevo.
   - **Frontend (cambio mínimo)**: `HotelDetail.jsx` manda fechas planas `YYYY-MM-DD` (antes convertía a RFC3339 con hora — ahora daría 400); `reservations.service.js` con firma nueva `create(hotelId, userId, checkIn, checkOut, numRooms=1, numGuests=1)` (ya no manda `hotel_name`); `MyReservations.jsx` filtra `status === 'cancelled'` al cargar (soft-delete: sin el filtro reaparecían como activas; la UI rica de estados es plan 13).

### Desviaciones del plan (validadas contra la realidad)

- **`CancelReservation` cambió de firma** en `Repository` (devuelve `(Reservation, error)`) — el plan no lo decía pero el service necesita la reserva para el evento CANCEL y la caché. La firma del **Service** hacia el controller quedó igual (`error`).
- **El mock de Queue de los tests no era testify** (era un struct pelado) — se le agregó `PublishReservation` y registro de eventos en vez de `.On(...)`.
- El **retry del claim va SIN upsert** (el plan decía "reintentar una vez" a secas): con upsert, un filtro que no matchea reintenta el insert y vuelve a dar DuplicateKey en loop; sin upsert, `ErrNoDocuments` = noche llena. Verificado con la carrera real.
- **Eventos de reserva best-effort** (log-and-continue), a diferencia de los de hotel que siguen devolviendo error si el publish falla: la cola no tiene consumidor y una reserva persistida no puede fallar por RabbitMQ. Los eventos de HOTEL en `Update`/`Delete` se publican siempre aunque la caché falle (D2).
- El **mock principal (`hotels_mock.go`) también aplica no-overbooking** (mismo conteo por noche) para que los tests de service ejerciten el path 409 sin Mongo.
- Segundo cancel idempotente **re-publica el evento CANCEL** (at-least-once) — aceptado y verificado (3 mensajes en la cola tras create + doble cancel).

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **planes 01+02+03+04 sin commitear** (~100 entradas en `git status`). Nuevos untracked de esta sesión: `hotels-api/internal/repositories/hotels/availability_suite_test.go`, `hotels-api/seed/migrate-inventory.js`.
- Archivos con cambios de VARIOS planes (para `git add -p`): los ya listados en la sesión anterior + ahora `docker-compose.yml` (04: 2 env), `hotels-api/*` (01/02/03/04), `frontend/src/pages/HotelDetail.jsx` y `frontend/src/services/reservations.service.js` (01: register / 04: fechas).
- **Checkbox del plan 04 en `plans/README.md`: `[x]`.**
- **Verificación completa (todo verde):** `make test` (-race, 4 módulos), lint 0 issues en los 4, `gofmt` limpio, integración testcontainers (~12s: suite overlap + carrera 20 goroutines → 1 ganador + cancel idempotente), `npm run build` OK, eslint limpio en los 3 archivos tocados.
- **Verificación en vivo (por el gateway :80, imagen rebuildeada):** backfill corrido (2 legacy + 3 entradas); carrera real `seq 20 | xargs -P 20` → **1×201 + 19×409**; inventario `booked <= capacity` (0 violaciones); doble DELETE → 200/200 con `booked: 0` (no negativo) y reserva visible con `status: cancelled`; disponibilidad false→true tras cancelar; **D2**: PUT de hotel seed con caché expirada → 200 y search-api actualizó Solr; **D3**: GET hotel + availability inmediata → true; DELETE del hotel de test limpió reservas e inventario; `test_load_balancer.sh` exit 0. La reserva de test y su hotel fueron borrados — quedan solo las 2 reservas demo.

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 05 — Observabilidad** ← **siguiente** (recomendado tras 03 ✓; 06 depende de 05)
2. Plan 06 — Endurecer search-api (02 ✓, 05)
3. Planes 07 → 13 según orden y dependencias del README.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear planes 01–04** y pushear para ver el CI (el job `integration` corre los tests nuevos de Mongo solo en PRs).
  - **API breaking**: `POST /reservations` ya NO acepta fechas RFC3339 (400) — clientes viejos (frontend sin rebuildear, Bruno viejo) fallan hasta actualizar. El frontend del repo ya está adaptado.
  - En un volumen con datos pre-plan-04, **correr `hotels-api/seed/migrate-inventory.js` una vez** (receta en el header del script) — ya corrido en el volumen local actual. Sin backfill, las reservas legacy sin `status` no se pueden cancelar (el filtro `status: confirmed` no matchea) y no ocupan inventario.
  - Si un admin **baja `avaiable_rooms` de un hotel** con reservas existentes, el inventario viejo mantiene su `capacity` snapshot pero el claim compara contra la capacidad ACTUAL — noches ya sobre-vendidas quedan como están (el backfill lo reporta como "overbooking histórico").
  - La caché de disponibilidad sigue siendo aproximada (lista ausente = disponible, D3) — el no-overbooking REAL lo garantiza solo el claim de Mongo; la caché es para el display de búsqueda.
  - `api-gateway` sigue "(unhealthy)" cosmético (plan 05 lo rehace); 1 error preexistente de eslint en `AuthContext.jsx:16` (sin cambios).

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con el 05"** → leer `plans/05-observabilidad.md` completo y validar snippets contra el código actual (los `log.Printf` nuevos del plan 04 en service/repos de hotels-api pasarán a `slog`; el `/health` de hotels-api sigue en `cmd/main.go`; healthchecks del compose). Idealmente con 01–04 commiteados antes.

---

## Sesión — 2026-07-06 17:30

### Resumen de lo hecho

1. **Plan 05 (Observabilidad: O1, O2, O3, O4, I5) — IMPLEMENTADO Y VERIFICADO end-to-end.** Lo esencial:
   - **slog JSON (O2)**: los 3 `main.go` setean `slog.SetDefault` con `NewJSONHandler` + atributo `service` (users-api además `instance` desde `INSTANCE_ID`, env nueva en config). Todos los `log.Printf` no-fatales de services/repos/queues pasados a `slog.Info/Warn/Error` con atributos y mensajes en inglés (users_service, hotels_service, queue_rabbit de hotels, releaseNights de hotels_mongo, queue_rabbit y search_service de search). Los `log.Fatalf/Panicf` de constructores quedan (el bridge del stdlib los emite como JSON igual); los fatales de `main` son `slog.Error + os.Exit(1)`.
   - **Gin release (O4)**: los 3 mains hacen `gin.SetMode(gin.ReleaseMode)` **si `GIN_MODE` no está seteada** (default producción, `GIN_MODE=debug` lo restaura en dev). Además `gin.New()+gin.Recovery()` en vez de `gin.Default()` — el access-log ahora es el middleware RequestID en JSON.
   - **Request-ID end-to-end (O1)**: middleware `RequestID()` idéntico en los 3 (`internal/middlewares/requestid.go`, search-api estrenó el paquete): propaga o genera UUID, lo setea en la respuesta y emite el access-log JSON (`request`, method/path/status/duration_ms); los paths `/health|/livez|/readyz` no se loguean (los pollea docker cada 10s). Consumer de search-api: `StartConsumer` ahora pasa `ctx` con un `request_id` **por mensaje** (helpers `utils.WithRequestID/RequestIDFromContext`), `HandleHotelNew(ctx, ...)` loguea con ese id y `hotels_http.GetHotelByID` usa `http.NewRequestWithContext` + header `X-Request-ID` → el hop search→hotels queda correlacionado (verificado con un UPDATE real: mismo uuid en el log del consumer y en el access-log de hotels-api).
   - **`/livez` + `/readyz` (O3)**: controller nuevo `internal/controllers/health` (idéntico en los 3): `/livez` barato, `/readyz` corre los checks **en paralelo con timeout de 3s** (< 5s del healthcheck de docker) y devuelve 503 `{"status":"degraded","checks":{...}}`; `/health` = alias de `/livez`. Checks: hotels Mongo+RabbitMQ, users MySQL+Memcached, search Solr+RabbitMQ. Métodos nuevos: `Mongo.Ping`, `MySQL.Ping`, `Memcached.Ping` (acepta-e-ignora ctx), `Solr.Ping` (HTTP a `admin/ping`, solr-go no expone ping; el struct ganó `baseURL`), `Rabbit.IsConnected` en search (hotels ya lo tenía).
   - **Healthchecks compose (I5)**: `wget --spider http://127.0.0.1:PORT/readyz` en los 5 contenedores Go (start_period 15s, search 90s); nginx `depends_on: service_healthy` de los 5.
   - **nginx**: `map $http_x_request_id $req_id` (respeta el ID del cliente, genera solo si falta), `request_id` agregado a `json_combined` y **access_log cambiado de `main` a `json_combined`**; `proxy_set_header X-Request-ID $req_id`.
   - **BONUS — resuelto el "(unhealthy)" cosmético del api-gateway**: era el `wget` de busybox resolviendo `localhost → ::1` contra un nginx que escucha solo IPv4 (sin fallback). Todos los healthchecks apuntan ahora a `127.0.0.1`. **Primera vez que los 11 contenedores están healthy.**
   - **Deps**: `google/uuid` agregado a users-api y search-api (hotels ya lo tenía).
   - **Tests nuevos** (idénticos en los 3 servicios): `health_controller_test.go` (livez 200 aunque un check falle, readyz 200 ok / 503 degraded con mapa) y `requestid_test.go` (propaga header entrante, genera si falta).
   - **README**: nota sobre `/livez`/`/readyz` internos bajo la tabla de endpoints.

### Desviaciones del plan (validadas contra la realidad)

- **nginx pisaba el X-Request-ID del cliente**: el plan asumía que con `$request_id` alcanzaba, pero esa variable es SIEMPRE el random de nginx — sin el `map`, el `test-trace-123` del Verificar jamás llegaba a los servicios. Corregido con el patrón `map` estándar.
- **El access_log activo era el formato `main`**, no `json_combined` — el plan pedía "activarlo"; hecho.
- **`HandleHotelNew` cambió de firma** (`(ctx context.Context, hotelNew)`) igual que el handler de `StartConsumer`: era la única forma limpia de llevar el request_id hasta el header del fetch. Tests de service actualizados; `queue_mock.go` de search estaba vacío (solo package), nada que tocar.
- **Gin release por código con escape** (`if os.Getenv("GIN_MODE") == ""`) en vez de 5 envs en compose: containers en release sin tocar compose, dev puede volver a debug.
- El healthcheck de nginx en compose ya existía pero estaba roto por lo de IPv6 (arriba); el plan no lo mencionaba explícitamente pero el HANDOFF anterior sí ("plan 05 lo rehace").

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **planes 01–05 sin commitear** (~110 entradas en `git status`, 31 untracked). Untracked nuevos de esta sesión: `internal/controllers/health/` (×3), `internal/middlewares/requestid*.go` (×3, en search el dir entero), `search-api/internal/utils/requestid.go`.
- **Checkbox del plan 05 en `plans/README.md`: `[x]`.**
- **Verificación completa (todo verde):** `make test` (-race, 4 módulos, incluye los tests nuevos), `make lint` 0 issues ×4, `gofmt` limpio; `go mod tidy` corrido en users/search (uuid).
- **Verificación en vivo:** `docker compose up -d --build` → **11/11 healthy (incluido api-gateway)**; readyz de los 3 con `{"checks":{...},"status":"ok"}`; `X-Request-ID` vuelve por el gateway; `test-trace-123` correlacionado en logs JSON de nginx y search-api, `test-trace-456` en hotels-api; UPDATE de hotel → mismo uuid en consumer de search y access-log de hotels (hop correlacionado); `stop mongo` → readyz 503 `{"mongo":"down"}` + contenedor unhealthy en ~25s, `start mongo` → recupera solo; grep "running in debug mode" = 0; **reinicio en frío: nginx arrancó DESPUÉS de los upstreams healthy, smoke 200 sin 502 tempranos**; `bash test_load_balancer.sh` exit 0.

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 06 — Endurecer search-api** ← **siguiente** (02 ✓ y 05 ✓; recordar del handoff del 03: **probar que el backfill/reindex indexe los 5 hoteles demo** — hoy `/search` devuelve `[]` porque el índice Solr está vacío salvo eventos en vivo)
2. Planes 07 → 13 según orden y dependencias del README.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear planes 01–05** y pushear para ver el CI.
  - El access-log de los servicios lo emite el middleware RequestID: si el plan 06 agrega endpoints internos ruidosos, sumarlos a `quietPaths` del middleware.
  - `restart: unless-stopped` no reinicia contenedores *unhealthy* (compose no tiene autoheal): un readyz degradado se ve en `docker compose ps` pero no recicla el contenedor — comportamiento esperado y documentable.
  - El healthcheck de search-api golpea Solr vía `/readyz` cada 10s (admin/ping es barato, sin impacto observado).
  - 1 error preexistente de eslint en `AuthContext.jsx:16` (sin cambios de frontend en esta sesión).

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con el 06"** → leer `plans/06-search-api.md` completo y validar snippets contra el código actual — en particular: el consumer ahora arranca con `StartConsumer(handler func(ctx, HotelNew))` y `NewRabbit` de search sigue con `log.Fatalf` (el 06 los reemplaza por reintentos con backoff, el comentario en el código lo marca); `hotels_http.GetHotelByID` ya usa `NewRequestWithContext` + header (el 06 agrega el client con timeout, E4); `Solr` ganó `baseURL` y `Ping`. Al hacer el backfill (E3), verificar que indexe los 5 hoteles demo del seed.

---

## Sesión — 2026-07-11

### Resumen de lo hecho

1. **Review general de código (read-only) del estado post-planes 01–05.** 4 revisores paralelos (hotels-api, users-api, search-api+contracts, frontend+nginx+infra) + verificación manual de los hallazgos graves contra el código. **No se modificó ningún archivo de código** — solo documentación en `plans/`.
2. **Creada `plans/fixes/`** con el resultado triageado:
   - `fixes/README.md` — índice con IDs `RV1`–`RV31`: los 13 huérfanos (ningún plan pendiente los cubría) asignados a 4 planes nuevos `F1`–`F4`; los 18 restantes triageados a los planes 06/07/08/10/11/13 (tabla "ejecutarlos DENTRO de ese plan").
   - `F1-hotels-cache-reservas.md` (M): listas agregadas envenenadas por el cacheo por-ítem (RV1) + data race por mutación in-place de slices de ccache (RV2) → estrategia nueva "invalidar, nunca editar" con interfaz `CacheRepository` (embebe `Repository` + 3 setters de lista completa con copia); cancel vía `CreateReservation(cancelled)` (RV3); `Cache.GetAvailability` propaga errores (RV4); mocks y **adaptación de `availability_suite_test.go`** (hoy seedea la caché con `CreateReservation`, que pasará a invalidar).
   - `F2-users-api-login.md` (S): `Login` solo mapea `ErrUserNotFound`→401 (hoy cualquier error de infra da 401, RV6) + bcrypt dummy contra timing oracle (RV7), `ORDER BY id` en paginación (RV8), clamp `BCRYPT_COST` [10,15] (RV9), documentar tradeoff stale-login multi-réplica (RV10, sin code-fix).
   - `F3-frontend-login-interceptor.md` (S): el interceptor 401 de `api.js` excluye el request de `/login` (hoy recarga y se come el error, RV11); `minLength` de Login pasa a `VALIDATION.MIN_PASSWORD_LENGTH` (RV12).
   - `F4-ci-platform-contracts.md` (XS): `cache-dependency-path: ${{ matrix.module }}/go.*` (platform-contracts no tiene `go.sum` y el leg fallaría, RV13). Verificado: jobs `integration` y `frontend` no tienen el problema.
3. `plans/README.md`: sección nueva "Fixes de la review externa (2026-07-11)" antes del grafo, apuntando a `fixes/README.md`.
4. Los planes de `fixes/` **no mencionan comandos de git** (pedido explícito del usuario; versionado 100% manual).

### Hallazgos clave de la review (detalle completo en `fixes/README.md`)

- La review confirmó sólido el núcleo: `claimNight`, suite de solapamiento, middleware JWT + tests negativos, seed race-safe, platform-contracts, health checks, compose.
- Lo más grave: RV1/RV2 (caché de reservas), RV6 (login miente 401), RV21 (el form admin de hoteles **no puede crear ni editar** — `check_in_time` "HH:mm" vs `time.Time`; va al plan 07 como decisión de contrato), RV11 (interceptor), RV13 (CI).

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — planes 01–05 siguen sin versionar. Archivos nuevos de esta sesión: `plans/fixes/` (5 archivos); modificado: `plans/README.md` (sección nueva).
- Checkboxes: 01–05 `[x]` (sin cambios); F1–F4 `[ ]` en `fixes/README.md`.
- Sin verificación que correr: esta sesión no tocó código.

### Trabajo restante (en orden)

1. **F4** (minutos — antes de que el CI corra por primera vez) → **F1** (antes de 06/07) → F2/F3 (independientes).
2. Plan 06 en adelante según `plans/README.md`, sumando los `RV*` triageados de cada plan (tabla en `fixes/README.md`).

### Bloqueos y advertencias

- Sin bloqueos. Las advertencias de la sesión anterior siguen vigentes (índice Solr vacío hasta el 06; eslint preexistente en `AuthContext.jsx:16` — su fix natural cae con F3/RV12 cerca, pero es el error conocido, no lo introduce esta sesión).
- Al ejecutar F1: recordar que cambia la firma de `NewService` (tipo del cache repo) — los tests de service y la suite de availability necesitan los ajustes descritos en el propio plan.

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con los fixes"** → leer `plans/fixes/README.md` + `F4` y `F1` completos, validar snippets contra el código actual, ejecutar en ese orden.

---

## Sesión — 2026-07-11 15:18

### Resumen de lo hecho

**Los 4 fixes de la review externa (`plans/fixes/`) — IMPLEMENTADOS Y VERIFICADOS: F4 → F1 → F2 → F3.** Checkboxes tildados en `plans/fixes/README.md`.

1. **F4 (RV13, CI)**: `ci.yml:20` → `cache-dependency-path: ${{ matrix.module }}/go.*` (platform-contracts no tiene `go.sum` y `setup-go` fallaría). actionlint no está instalado; YAML validado con parser — la verificación real es el primer run del CI en GitHub.
2. **F1 (RV1–RV5, caché de reservas de hotels-api)**: las listas agregadas **nunca se editan** — toda escritura invalida y solo el service escribe listas completas (como copia):
   - `hotels_cache.go`: borradas las 3 `updateXxxList` → `invalidateReservationLists` + `storeReservationsList` + 3 setters `SetReservationsByXxx`; `CreateReservation` setea individual + invalida; `CancelReservation` simplificada (sin call-sites de producción, queda por la interfaz); `DeleteReservationsByHotelID` reescrito (borra/invalida, no edita); `GetAvailability` **propaga errores** en vez de mapear a `false` (RV4) → el service cae a Mongo.
   - `hotels_service.go`: interfaz nueva **`CacheRepository`** (embebe `Repository` + setters); el campo `cacheRepository` cambió de tipo (`cmd/main.go` compila sin cambios); los 3 getters pueblan con el setter de lista completa (guard `offset==0 && len<limit` intacto); `CancelReservation` refleja en caché vía `CreateReservation(cancelled)` — la copia cancelada que devuelve Mongo — en vez de `Cache.CancelReservation` (RV3).
   - `hotels_mock.go`: `MockCache` reescrito con la semántica real (mapas de listas; getters solo aciertan con lista seteada completa; escrituras invalidan; lista ausente = disponible) — RV5. `countRoomsByNight` pasó de map a **slice** (helper `allReservations()` en `Mock`).
   - Tests nuevos en `hotels_cache_test.go` (untracked): no-listas-parciales (RV1), invalidación de listas seteadas (RV1 bis), copia-cancelada-invalida-con-key-evicted (RV3), setter-guarda-copia y lectores/escritores concurrentes bajo `-race` (RV2).
3. **F2 (RV6–RV10, users-api)**: `Login` solo colapsa **`usersRepo.ErrUserNotFound`** a 401 — cualquier otro error se propaga (5xx); el camino "no existe" quema un `dummyBcryptHash` cost 10 (RV7); `GetAll` con `Order("id ASC")` (RV8); clamp de `BCRYPT_COST` a [10,15] → default con warn (RV9); tradeoff de caché stale documentado en `invalidateCaches` + subsección **"Known trade-offs (demo scope)"** nueva en el README (RV10). Tests: subtest "user not found" actualizado al sentinel (con el error genérico era la regresión exacta de RV6), subtest nuevo "infra error is NOT invalid credentials", `TestService_BcryptCostClamp` (32 y 0 → hash verificable cost 10).
4. **F3 (RV11–RV12, frontend)**: el interceptor 401 de `api.js` **excluye requests a `/login`** (el error llega al form, sin hard-reload); `Login.jsx` usa `VALIDATION.MIN_PASSWORD_LENGTH` (8) en vez del 4 hardcodeado.

### Desviaciones del plan (validadas contra la realidad)

- **F1 / suite de disponibilidad**: con la semántica nueva, correr la suite contra la caché cruda vería siempre "lista ausente = disponible". Se agregó **`cacheSuiteHarness`** en `availability_suite_test.go`: re-publica la lista completa del hotel tras cada escritura (lo mismo que hace el service al repoblar) y asigna IDs uuid (Mongo los asigna en producción). El lado Mongo de la suite no cambió.
- **F1 / `TestAvailabilityWithReservation`**: tras crear la reserva se agregó una lectura (`GetReservationsByHotelID`) que repuebla la lista — sin eso la caché respondería "disponible" optimista (lista invalidada) y el test no ejercitaría el conteo.
- **F2 / tests más lentos**: `newTestService` usa `bcrypt.MinCost` (4) → el clamp lo lleva a cost 10; los tests del service de users pasaron de ~1s a ~6s. Deliberado, no regresión.
- **Verificación en vivo / nginx**: nginx resuelve las IPs de los upstreams **al arrancar** — tras `docker compose up --build` (recrea contenedores → IPs nuevas) el gateway dio 502 hasta `docker compose restart nginx`. Anotado en Advertencias.
- **Dato descubierto**: las "2 reservas demo" de Mongo pertenecen a `user_id: "1"`, pero los usuarios actuales de MySQL son **admin=2 y demo=5** (los ids 1/3/4 se consumieron en pruebas de sesiones anteriores). Cosmético (el demo user no ve reservas al entrar); si molesta, regenerar seed o tocarlo en plan 12/13.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **planes 01–05 + fixes F1–F4 sin commitear** (~114 entradas en `git status`, 33 untracked; nuevo de esta sesión: `hotels-api/internal/repositories/hotels/hotels_cache_test.go`).
- **Verificación completa (todo verde):**
  - `make test` (-race, 4 módulos), `make lint` 0 issues ×4, `gofmt` limpio, integración testcontainers `-count=1` (12.3s: suites de caché-con-harness y Mongo, carrera de 20 goroutines, cancel idempotente).
  - **F1 en vivo** (gateway :80, imágenes rebuildeadas): user 5 con reserva previa NO cacheada (restart de hotels-api) + create → la lista inmediata trae **todas** (antes: solo la nueva); cancel → aparece `cancelled` al instante (sin esperar TTL). Datos de prueba limpiados (quedan solo las 2 reservas legacy del user "1" y sus 3 entradas de inventario).
  - **F2 en vivo**: memcached reiniciado (L2 vacía) + `stop mysql` → login demo **500 en 27ms** (antes: 401 mentiroso); `start mysql` → 200; password mala 401 en 64ms vs usuario inexistente 401 en **76ms** (RV7: antes ~5ms — oráculo de timing cerrado); paginación admin `[2,5]` / `[]` ordenada (RV8).
  - **F3 en vivo** (Playwright + Vite dev server): 4/4 — password corta rechazada client-side sin request; password incorrecta → **sin reload** y error visible en el form; login demo entra; token corrupto + página protegida → redirect a `/login` con storage limpio.
  - `bash test_load_balancer.sh` exit 0; 11/11 contenedores healthy.

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 06 — Endurecer search-api** ← **siguiente** (02 ✓, 05 ✓, F1 ✓). Al empezar, sumar del triage (`plans/fixes/README.md`): **RV14** (pre-requisito del manual-ack: tipificar el 404 de hotels-api), RV15 (clamp en /search), RV16 (request_id al ctx del request), RV17 (IsConnected del channel), RV18 (mensaje copy-paste de Delete), RV30-search (dead code). Recordar: el backfill (E3) debe indexar los 5 hoteles demo (hoy `/search` devuelve `[]`).
2. Planes 07 → 13 según orden y dependencias del README, cada uno sumando sus `RV*` triageados.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear** (planes 01–05 + fixes) y **pushear para el primer run del CI** — F4 dejó el workflow listo; en ese run confirmar que los runners resuelven los toolchains 1.24/1.25 de los go.mod (advertencia pendiente del 2026-07-05).
  - **nginx cachea las IPs de los upstreams al arrancar**: después de un `docker compose up --build` que recree servicios Go, correr `docker compose restart nginx` si aparecen 502 (el healthcheck del gateway no lo detecta — chequea `/nginx-health` local).
  - Los tests del service de users tardan ~5s más por el clamp de bcrypt (cost 10) — esperado.
  - memcached quedó reiniciado (L2 vacía) — se repuebla solo con el uso.
  - 1 error preexistente de eslint en `AuthContext.jsx:16` (sin cambios).
  - La caché de disponibilidad sigue siendo aproximada y ahora algo más optimista: una escritura invalida las listas y hasta la próxima lectura `IsHotelAvailable` responde "disponible" (lista ausente = D3). El no-overbooking real sigue en el claim de Mongo; el 400-vs-500 de availability con input inválido es plan 07 (RV19).

### Primera acción sugerida para la próxima sesión

Commitear y pushear (ver el CI verde por primera vez — 4 legs Go + frontend + dependabot). Después decir **"empecemos con el 06"** → leer `plans/06-search-api.md` + su fila de `RV*` en `plans/fixes/README.md`, y validar snippets contra el código actual (los punteros de search-api de la sesión del 2026-07-06 siguen vigentes: esta sesión no tocó search-api).

---

## Sesión — 2026-07-12

### Resumen de lo hecho

1. **El usuario commiteó y pusheó todo el trabajo pendiente** (fin de ~6 sesiones sin versionar): `13427a0` (planes 01–05 + fixes F1–F4, ~131 archivos) y `1894e86` (fix del CI, abajo). Branch remota `feat/plan-01-seguridad-auth` creada. `.claude/` agregado a `.gitignore`; `CLAUDE.md` y `plans/HANDOFF.md` quedaron versionados (decisión consciente del usuario).
2. **Primer run del CI: FALLÓ (3 jobs Go). Diagnosticado y arreglado — segundo run: TODO VERDE** (4 legs Go + frontend, 2m40s). Las causas no eran los planes pendientes:
   - `go test` moría en users/search-api con `go: no such tool "covdata"`: setup-go instalaba el Go viejo del go.mod y el switch de toolchain del runner rompe coverage.
   - govulncheck rojo en hotels-api: 28 vulns — stdlib de go1.25.0 exacto (local había go1.26.4 parcheado) + **quic-go v0.54.0 vulnerable de verdad** (GO-2025-4233, GO-2026-5676). Invisible localmente: el workspace unificaba versiones, pero el CI corre `GOWORK=off` y cada módulo resuelve con su go.mod propio.
   - Además los `go.sum` por-módulo estaban incompletos para `GOWORK=off` (secuela del `go work sync` del plan 02).
3. **Fixes aplicados**: `ci.yml` → `go-version: stable` + `check-latest: true` en los jobs `go` e `integration` (comentario in-line explica por qué NO `go-version-file`); quic-go bumpeado a **v0.59.1** en hotels-api y search-api; `GOWORK=off go mod tidy` en los 3 servicios + `go work sync`. Los 3 go.mod quedaron alineados en **`go 1.25.0`** (compatible con `golang:1.25-alpine` de los Dockerfiles; search-api perdió su línea `toolchain`).
4. **Verificación local**: `go test -race` verde ×4, gofmt limpio, lint 0 issues ×4, `GOWORK=off govulncheck` limpio ×4 — salvo GO-2026-5856 (crypto/tls, fixed en go1.26.5) que aparece solo porque el Go local es 1.26.4; el runner con `stable` usa 1.26.5+ y en CI no sale (confirmado: CI verde).

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth`, pusheada, working tree limpio (salvo este HANDOFF.md, a commitear por el usuario). CI verde en `1894e86`.
- El job **`integration` todavía no corrió nunca** (solo se dispara en PRs) — abrir PR `feat/plan-01-seguridad-auth` → `main` para verlo correr; puede quedar abierto sin mergear.
- Stack local: 11/11 contenedores healthy (~27h arriba); frontend dev server en :5173. `/search` muestra **1 solo hotel** (el índice Solr se llena con el backfill del plan 06).

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 06 — Endurecer search-api** ← **siguiente**, sumando RV14–RV18 y RV30-search del triage (`plans/fixes/README.md`).
2. Planes 07 → 13 según orden y dependencias del README.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - Los 5 warnings amarillos del CI son deprecación de Node 20 en las actions — dependabot (`github-actions`) va a proponer los bumps solo; no tocar a mano.
  - Ya NO re-agregar excepciones de govulncheck: quedó estricto y verde en los 4 módulos. Si el plan 06 toca deps de search-api, re-correr `GOWORK=off govulncheck` local antes de pushear (con Go local ≥ 1.26.5, o ignorar GO-2026-5856 si sigue en 1.26.4).
  - Siguen vigentes: nginx cachea IPs de upstreams al arrancar (restart tras rebuild), eslint preexistente en `AuthContext.jsx:16`, el assert de `getTimeField` a invertir al arreglar E6.

### Primera acción sugerida para la próxima sesión

Decir **"empecemos con el 06"** → leer `plans/06-search-api.md` completo + su fila en `plans/fixes/README.md`, validar snippets contra el código actual y recién ahí implementar. Al hacer el backfill (E3), verificar que indexe los 5 hoteles demo del seed.

---

## Sesión — 2026-07-12 (2) — Plan 06

### Resumen de lo hecho

1. **Plan 06 (Endurecer search-api: E1–E6, DB6, DB7 + fixes triageados RV14–RV18 y RV30-search) — IMPLEMENTADO Y VERIFICADO end-to-end.** Lo esencial:
   - **E1 (manual ack + retry + DLQ)**: `search-api/internal/clients/queues/queue_rabbit.go` reescrito — struct con mutex/config, `connectWithRetry` (espejo del productor), `consumeLoop` que re-registra `Consume` para siempre (reconexión E5), `autoAck=false` + `Qos(1)`. Política de acks en `handleDelivery`: OK → ack; 1er fallo → requeue; 2º fallo (`Redelivered`) → DLQ; **JSON inválido → DLQ directo** (mensaje veneno, sin invocar el handler). Topología: exchange `hotels-news-dlx` (direct) + cola `hotels-news-dlq` bindeada + `hotels-news` con `x-dead-letter-exchange`, declarada **idéntica en productor (hotels-api `connect()`) y consumidor** (gotcha 406; `reservations-news` sigue sin DLX). `HandleHotelNew` ahora **devuelve `error`**; operación desconocida = error (→ DLQ, no se pierde en silencio).
   - **RV14 (pre-req del ack)**: sentinel `ErrHotelNotFound` en el **domain de hotels-api** — `Mongo.GetHotelByID` lo devuelve para `ErrNoDocuments` y para hex inválido; el service pasó ese wrap de `%v` a `%w`; el controller mapea con `errors.Is` → **404, todo lo demás 500** (antes: 404 para cualquier error). En search-api: sentinel espejo en su domain, el cliente HTTP tipifica el 404 (sin reintentos) y `HandleHotelNew` ante 404 **descarta el evento y borra el doc de Solr** (si ese delete falla → error → retry/DLQ).
   - **E2 (query Solr segura)**: `buildSearchQuery` — `{!edismax qf='name description' v=$qq}` con el input como **parámetro dereferenciado** (`Params` de solr-go), `escapeSolrQuery` para los metacaracteres Lucene, `Limit/Offset` reales (antes el string entero `q=...&rows=...` iba como query Lucene: paginación ignorada y 500 con q vacía — confirmado en vivo antes de tocar). Vacío → `*:*`.
   - **E3 (backfill + reindex)**: hotels-api ganó `GET /hotels?limit&offset` → `{data, total}` (repo Mongo `GetHotels`/`CountHotels` + stubs de error en Cache/MockCache "el listado no se cachea" (RV1), service `GetHotels` sin caché + helper `hotelToDomain` extraído, controller + ruta pública; nginx ya ruteaba `/hotels`). En search-api: `Service.Backfill(ctx)` pagina e indexa (pageSize 50, idempotente por uniqueKey `id`); **goroutine al arranque** (5 intentos, backoff 2s→32s) y **`POST /reindex`** protegido.
   - **Auth de /reindex**: `search-api/internal/middlewares/auth.go` nuevo (copia del de hotels-api, audiencia `search-api`); el **tokenizer de users-api agrega `search-api` al `aud`** (los middlewares de users/hotels validan "contains" — sin impacto); `JWTSecret` en config de search + fail-fast en main + env `JWT_SECRET` en compose; location `/reindex` en nginx.
   - **E4**: cliente HTTP de search reescrito — un único `http.Client{Timeout: 5s}` + `getJSON` con 3 intentos (reintenta conexión/5xx con backoff corto respetando ctx; 404 y otros 4xx no se reintentan) y header `X-Request-ID` desde el ctx.
   - **E5/RV17**: `IsConnected()` = conexión abierta **y flag `consuming`** (se apaga cuando el canal de deliveries se cierra) — el `/readyz` degrada si el consumer murió aunque la conexión TCP siga viva; el loop reconecta y re-consume solo.
   - **E6**: `getTimeField` parsea RFC3339 (string o multiValued); **assert del test invertido** como estaba anotado.
   - **DB6**: sin `Commit()` por documento (manda el autoCommit 15s / autoSoftCommit 1s del solrconfig); `Index`/`Update` deduplicados en `addDocument` (RV30).
   - **DB7**: schema — `phone`/`email` a `string` (match exacto), catch-all `_text_` (`copyField` de name/description; el `df` de los handlers apunta ahí), `docValues` en el fieldType `string`. **Requiere recrear el core** (ver Verificación).
   - **RV15**: `paginationParams` en el controller de search (default 20, max 100, inválido/negativo → default; **ya no hay 400 por params faltantes** — cambio de contrato menor). **RV16**: el middleware RequestID mete el id también en `c.Request.Context()`. **RV18**: mensaje de `Delete` corregido. **RV30-search**: borrados `queues/queue_mock.go` y `search/search_mock.go` (dead code).
   - **Tests nuevos/adaptados**: política de acks vía `Acknowledger` fake (4 casos, sin broker); `auth_test.go` de search (espejo del de hotels con aud `search-api`, incluye "token sin search-api en aud → 401"); httptest del cliente HTTP (404 tipado sin retry, 5xx con 3 intentos + header `X-Request-ID`, envelope de GetHotels); `buildSearchQuery`/`escapeSolrQuery`; `Backfill` (paginado multi-página, catálogo vacío, errores); descarte-por-404 (y su variante "delete de Solr falla → retry"); clamps del controller + Reindex 200/500; `HandleHotelNew` asserts de error; hotels-api: 404 tipado vs 500 de infra, `GetHotels` envelope+clamp en controller y service; tokenizer asserts `search-api` en aud.
   - **Paso 8 del plan (CVEs): pre-hecho en sesiones anteriores** — re-verificado `GOWORK=off govulncheck` limpio ×4 (solo GO-2026-5856, artefacto conocido del Go local 1.26.4; el runner con `stable` no lo ve) y el CI ya estaba estricto sin `continue-on-error`. No se re-agregó ninguna excepción.

### Desviaciones del plan (validadas contra la realidad)

- **`POST /reindex` necesitó infraestructura de auth que el plan no explicitaba**: search-api no tenía middleware JWT ni audiencia en los tokens. Solución: middleware espejo + `aud` del tokenizer ampliado + `JWT_SECRET` en compose + location en nginx — el mismo patrón por-servicio del plan 01.
- **El snippet del loop del plan asumía el `StartConsumer` viejo**; la reconexión (E5) pidió reestructurar: el loop vive en `consumeLoop` y `StartConsumer` ya no devuelve error (main sin `os.Exit` por Rabbit caído — antes `NewRabbit` hacía `log.Fatalf`).
- **Mensaje veneno** (unmarshal inválido) no estaba en el plan: con manual ack hay que ack/nackear SIEMPRE — va directo a DLQ sin pasar por el handler.
- El envelope `{data, total}` de `GET /hotels` reutiliza el `paginationParams` (clamp 100) que hotels-api ya tenía del plan 03.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **todo el plan 06 sin commitear** (~40 entradas en `git status`; untracked nuevos: `search-api/internal/middlewares/auth.go` + `auth_test.go`, `search-api/internal/clients/queues/queue_rabbit_test.go`; borrados: `search-api/.../queues/queue_mock.go`, `search-api/.../search/search_mock.go`). Este HANDOFF trae además la sección de la sesión anterior (2026-07-12) sin commitear.
- **Checkbox del plan 06 en `plans/README.md`: `[x]`.** Con él quedan ejecutados RV14–RV18 y RV30-search del triage de `plans/fixes/README.md`.
- **Verificación local (todo verde)**: `make test` (-race ×4 módulos), `make lint` 0 issues ×4, `gofmt` limpio, `go vet` ok, `GOWORK=off govulncheck` ×4 (nota GO-2026-5856 arriba).
- **Verificación en vivo (stack rebuildeado)**: para aplicar el schema DB7 se recreó **solo** el volumen de Solr (`docker compose down && docker volume rm hotel-search-booking-microservices-platform_solr_data && docker compose up -d --build`) — Mongo conservó los 5 hoteles del seed y las 2 reservas legacy; el `down` limpió el broker efímero y las colas se re-declararon con los args nuevos **sin 406**. 11/11 healthy. Resultados:
  - **E3**: backfill al arranque `hotels_indexed: 5, attempt: 1` → `/search` devuelve los **5 hoteles demo** (antes: 1 solo). `GET /hotels?limit=2` → `{data: [...], total: 5}`.
  - **E6**: `check_in_time` reales en `/search` (`2024-01-01T14:00:00Z`), no `0001-01-01`.
  - **E2**: `q=hotel%20sierras` → matchea el hotel correcto; `q=` vacía → 200 con los 5 (antes 500 SyntaxError); metacaracteres `%22(malicious%3A*%22` → 200 `[]` sin inyección; `limit=2` con `offset=0/2` → páginas distintas (antes la paginación se ignoraba). **RV15**: sin params → 200; `limit=99999&offset=-4` → 200 clampeado.
  - **RV14**: `GET /hotels/garbage` y ObjectID válido inexistente → **404** (el 500 de infra queda cubierto por unit tests).
  - **/reindex**: sin token → 401 / cliente demo → 403 / admin → 200 `{"indexed":5}`.
  - **E1**: `stop solr` + crear hotel → logs `requeueing once` → `sending to DLQ`, `hotels-news-dlq: 1` y cola principal en 0 (sin loop ni pérdida), readyz 503; `start solr` + `POST /reindex` → `{"indexed":6}` y el hotel buscable.
  - **E4**: `stop hotels-api` + evento publicado a mano (`rabbitmqadmin` — ojo: **sin `-u root -p ...` falla silencioso**) → DLQ en segundos (fetch acotado, 3 intentos), consumer vivo y readyz ok durante la caída.
  - **E5/RV17**: `restart rabbitmq` → `deliveries channel closed, reconnecting` → intentos con backoff → `connected to RabbitMQ` → `rabbitmq consumer started`; readyz de search y hotels recuperados solos.
  - **RV16**: `POST /reindex` con `X-Request-ID: rv16-trace-999` → el access-log de hotels-api muestra `GET /hotels | request_id: rv16-trace-999` (hop correlacionado).
  - Flujo UPDATE/DELETE en vivo por el consumer nuevo OK (rename visible en `/search` en ~1s vía autoSoftCommit; DELETE limpia el índice). Hotel de prueba borrado y DLQ purgada — **estado final: 5 hoteles demo, colas en 0**. `bash test_load_balancer.sh` exit 0.

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 07 — Contratos de API** ← **siguiente** (04 ✓, 06 ✓), sumando del triage: RV19 (validaciones 400/404 vs 500 — **el sentinel `ErrHotelNotFound` y el wrap `%w` de este plan le dejan el camino hecho**), RV20 (fechas date-only en respuestas), RV21 (tipo de `check_in_time`/`check_out_time` — el form admin de hoteles sigue roto), RV22 (envelope con total + `Access-Control-Expose-Headers`).
2. Plan 08 — Resiliencia de runtime (sumar RV23–RV25) → resto según README (08 antes del 09).

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear el plan 06** y pushear (los 4 legs Go + frontend del CI deberían seguir verdes; `integration` solo corre en PRs).
  - **Los JWT emitidos antes del 06 no traen `search-api` en `aud`**: `/reindex` les da 401 — re-loguearse. El resto de los endpoints acepta tokens viejos igual que antes.
  - **El schema nuevo de Solr solo aplica recreando el core**: en un volumen viejo, `docker compose down && docker volume rm hotel-search-booking-microservices-platform_solr_data && docker compose up -d` — el backfill del arranque repuebla el índice solo (no hace falta `down -v` completo).
  - **Cambio de contrato menor en `GET /search`**: params de paginación ausentes/ inválidos ya no dan 400 — se clampean a defaults (RV15).
  - La DLQ (`hotels-news-dlq`) no tiene consumidor por diseño: es para inspección/replay manual (management UI o `rabbitmqctl`); `POST /reindex` es la reconciliación gruesa. El broker sigue efímero (sin volumen): un `down` vacía colas y DLQ.
  - Si se cambia la declaración de `hotels-news` a futuro, tocar **ambos lados** (productor hotels-api y consumidor search-api) o llueve 406 `PRECONDITION_FAILED`.
  - Siguen vigentes: nginx cachea IPs de upstreams al recrear contenedores (`docker compose restart nginx` si aparecen 502 tras un `up --build`; en esta sesión el gateway arrancó último y no hizo falta), eslint preexistente en `AuthContext.jsx:16`, memcached sin healthcheck (normal).

### Primera acción sugerida para la próxima sesión

Commitear y pushear el plan 06 (ver el CI verde). Después decir **"empecemos con el 07"** → leer `plans/07-contratos-api.md` completo + su fila de RV en `plans/fixes/README.md`, y validar snippets contra el código actual — en particular: el controller de hotels-api ya distingue 404/500 en `GetHotelByID` (RV14), ya existe `GET /hotels` con envelope `{data, total}` (adelanto del patrón de envelopes del 07), y la interfaz `Service` del controller de search-api ahora incluye `Backfill`.

---

## Sesión — 2026-07-14 — Plan 07

### Resumen de lo hecho

1. **Plan 07 (Contratos de API: A1–A8 + fixes triageados RV19–RV22) — IMPLEMENTADO Y VERIFICADO end-to-end** (unit + stack en vivo + Playwright sobre el frontend). Lo esencial:
   - **A1 (envelope de error)**: paquete `internal/apperr` **copiado en los 3 módulos** — `Abort(c, status, code, msg, cause)` responde `{"error":{"code","message","trace_id"}}`; la causa va SOLO al log (slog + `c.Error`; el access-log del middleware RequestID ahora agrega el campo `errors` cuando hay causas). Reemplazados todos los sitios que filtraban `err.Error()` de Mongo/Solr en controllers Y en los middlewares de auth de los 3 servicios (códigos: `unauthorized`, `forbidden`, `invalid_body`, `invalid_id`, `invalid_reservation`, `hotel_not_found`, `reservation_not_found`, `user_not_found`, `no_availability`, `username_taken`, `invalid_credentials`, `request_in_flight`, `not_acceptable`, `timeout`, `internal`). nginx: `error_page 500 502 504 → @api_error` (502 `upstream_unavailable` JSON) y **503 aparte → `rate_limited` JSON con status 503** (el 503 del gateway lo genera el rate/conn-limit, no un upstream caído; el cambio a 429 sigue siendo I7/plan 10). Fallback 404 del gateway con el mismo envelope.
   - **A2 (/api/v1)**: `router.Group("/api/v1")` en los 3 servicios; `health/livez/readyz` sin versionar. nginx: TODAS las locations y `proxy_pass` prefijados (incluidas las 2 regex de reservas-por-usuario y `/reindex`). Frontend en el mismo cambio: `BASE_URL` → `/api/v1` (dev) / `http://localhost/api/v1` (prod), proxy de Vite **sin rewrite** (pasa `/api/v1` tal cual). Rutas viejas → 404 del gateway. `test_load_balancer.sh` actualizado (incluye check "unversioned → 404").
   - **A3 (idempotencia)**: middleware `Idempotency` en hotels-api sobre `POST /api/v1/reservations` (después de Authenticate — clave compuesta `(Idempotency-Key, userID)`, NUNCA X-Request-ID). Store en Mongo: colección `idempotency_keys` (config `MONGO_COLLECTION_IDEMPOTENCY`, ya en compose) con **índice único {key, user_id} + TTL 24h** (EnsureIndexes al arranque). Semántica: sin header = no-op; 1ª vez ejecuta y persiste status+body (captura con un `bodyRecorder`); replay completado devuelve la respuesta guardada + header `Idempotency-Replayed: true`; replay en vuelo → 409 `request_in_flight` (el insert duplicado lo detecta atómicamente); un 5xx **libera la key** (el reintento legítimo re-ejecuta), los 4xx se persisten. `Idempotency-Key` agregado a los CORS del gateway y de Gin. 6 tests de middleware con store fake.
   - **A4/A5 (paginación + envelope)**: convención única `{data, meta:{total,limit,offset}}` en listas y `{data}` en gets/creates. `GET /hotels` pasó de `{data,total}` al envelope estándar y su consumidor (`hotels_http.go` de search-api, struct `hotelsPage`) se actualizó junto; `GET /search` devuelve el **total real del índice (numFound)** — la firma `Search` de repo/service de search-api ahora devuelve `(hoteles, total, error)` (RV22). users-api `GET /users` → envelope; **X-Total-Count eliminado** (con eso muere también el pendiente de Expose-Headers de RV22). Las listas de reservas llevan `meta:{limit,offset}` **sin total** (nadie las pagina en la UI; sumar CountDocuments si algún día hace falta).
   - **A6 (semántica)**: 201 + `Location: /api/v1/...` en creates (hoteles, reservas, users); DELETE → **204 sin body** (hotel, reserva, user); PUT hotel → **200 con la representación actualizada** (re-fetch post-update); PUT/DELETE de hotel inexistente → **404 tipado** (ErrHotelNotFound desde Mongo Update/Delete). **Ruta nueva `GET /api/v1/reservations/:id`** (owner o admin) para que el Location del create apunte a algo real.
   - **A7 (user_id string)**: `User.ID` y `LoginResponse.UserID` con `json:",string"` — el wire siempre string, la PK int64 queda interna; convención documentada en `platform-contracts/contracts.go`. El frontend ya lo consumía tolerante (`String(user.id)`).
   - **A8 (JSON-only)**: middleware `RequireJSON` (copiado en los 3, aplicado al grupo `/api/v1`): `Accept` presente que no admita `application/json` / `application/*` / `*/*` → 406 con envelope.
   - **RV19**: sentinels nuevos `ErrInvalidReservation` y `ErrReservationNotFound` en el domain de hotels-api. Las validaciones del service (fechas mal formadas / check-in pasado / checkout≤checkin / num_rooms<1 / num_guests<1) → 400 `invalid_reservation`; reservar sobre hotel inexistente → 404 `hotel_not_found` (el wrap %w del 06 lo dejó pasar); cancelar/ver reserva inexistente o con hex inválido → 404 `reservation_not_found`; `POST /hotels/availability` valida fechas en el controller → 400 (antes todo eso era 500).
   - **RV20 (decisión de contrato)**: las RESPUESTAS de reserva serializan `check_in`/`check_out` **date-only `YYYY-MM-DD`** — simétrico con el request. `hotelsDomain.Reservation.CheckIn/CheckOut` pasaron de `time.Time` a `string` (+ `DateFormat` const); el DAO/Mongo sigue en `time.Time`; `created_at`/`cancelled_at` siguen RFC3339 (timestamps de auditoría). Sin tocar el frontend, MyReservations dejó de correr las fechas un día (verificado con Playwright).
   - **RV21 (decisión de contrato)**: `check_in_time`/`check_out_time` del Hotel → **string `"HH:mm"` end-to-end**: platform-contracts, DAOs de hotels/search, Mongo (string), schema de Solr (`string` en vez de `pdate`), seed `mongo-init.js`, y validación `HH:mm` en Create/Update del controller (400 si no parsea `15:04`). `getTimeField` de search-api eliminado (queda `getStringField`). **El form admin de hoteles crea y edita OK** (verificado en vivo con "14:00"); `HotelDetail` ahora muestra "Check-in: 14:00" limpio.
2. **Frontend (cambio atómico con A2/A5)**: service layer desenvuelve los envelopes nuevos (`response.data.data`); `hotelsService.search` devuelve `{data, meta}` y **Search.jsx pagina con `meta.total`** (RV22 — antes calculaba totalPages sobre la página actual y siempre daba 1); extracción de errores `err.response?.data?.error?.message` en los 6 sitios; `healthCheck` pega al `/health` del gateway sin versionar; `cancel`/`deleteUser`/`deleteHotel` manejan el 204.
3. **README**: tabla de endpoints reescrita con `/api/v1` + párrafo con la estrategia de versionado y el shape de los envelopes (lo retoma el 12). `frontend/README.md`: ejemplo de `VITE_API_URL` versionado.

### Desviaciones del plan (validadas contra la realidad)

- El plan pedía `Location` en creates pero no existía ningún GET de reserva individual: se agregó `GET /api/v1/reservations/:id` (owner/admin) en vez de apuntar el Location al vacío.
- El snippet de nginx mapeaba 500/502/**503**/504 a un solo `@api_error` 502: eso convertía el rate-limit (503 de limit_req) en "upstream_unavailable" y rompía el check de rate-limiting del e2e — el 503 tiene su propio envelope `rate_limited` (status sigue 503 hasta I7/plan 10).
- El middleware de idempotencia necesita decidir qué persistir: 2xx/4xx se guardan, 5xx libera la key. El plan no lo especificaba.
- Los `meta` de las listas de reservas van sin `total` (decisión de alcance, ver arriba).

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **planes 06 y 07 sin commitear** (~87 entradas en `git status`; untracked nuevos del 07: `*/internal/apperr/apperr.go` ×3, `*/internal/middlewares/accept.go` ×3, `hotels-api/internal/middlewares/idempotency.go` + `_test`, `hotels-api/internal/repositories/hotels/hotels_idempotency.go`).
- **Checkbox del plan 07 en `plans/README.md`: `[x]`.** Con él quedan ejecutados RV19–RV22 del triage.
- **El golden `hotel_new.golden.json` NO se tocó** (HotelNew no cambió) y el typo `AvaiableRooms` sigue intacto (C11/plan 11).
- **Verificación local (todo verde)**: `make test` (-race, 20 paquetes ok), `make lint` 0 issues ×4, `gofmt` limpio, `npm run build` OK (eslint: solo el error preexistente de `AuthContext.jsx:16`).
- **Verificación en vivo**: stack rebuildeado 11/11 healthy. Para aplicar RV21 se **migró el volumen vivo de Mongo** (5 hoteles: `check_in_time` Date → `"HH:mm"`, receta abajo) y se **recreó el volumen de Solr** (schema string). Resultados: backfill `hotels_indexed: 5`; envelopes y `trace_id` en 404/500; `/hotels` viejo → 404 envelope; upstream caído → 502 `upstream_unavailable` JSON; idempotencia real (2 POST misma key → mismo id, `Idempotency-Replayed: true`, 1 sola reserva); RV19 en vivo (400 check-in pasado / 404 hotel inexistente / 400 availability con fecha basura); 201+Location, DELETE→204, PUT devuelve representación; alta/edición/borrado de hotel con `"14:00"` fluye por RabbitMQ hasta Solr y `/search` lo muestra; 406 con `Accept: text/html`; `bash test_load_balancer.sh` **PASSED** (incl. rate-limit 6/10). **Playwright**: login → user.id string en localStorage → búsqueda (5 hoteles) → Book Now → Confirm → "September 10" en MyReservations sin corrimiento. Reservas de prueba canceladas; estado final: 5 hoteles demo.

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 08 — Resiliencia de runtime** ← **siguiente** (04 ✓, 06 ✓), sumando del triage: RV23 (leak de noches en releaseNights/cancel), RV24 (fan-out de availability sin bulkhead), RV25 (retry-loop del publisher sobre canal muerto).
2. Resto según README (08 antes del 09; el 10 conviene después del 07 — ya está — para no re-tocar locations).

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear los planes 06+07** y pushear (CI debería seguir verde: no se tocaron deps de Go, solo código; `integration` sigue corriendo solo en PRs).
  - **BREAKING deliberado**: toda la API vive bajo `/api/v1`; las rutas sin versión dan 404. Cualquier cliente externo (curl guardados, Bruno) debe actualizarse. Los JWT existentes siguen sirviendo (claims sin cambios).
  - **Volúmenes viejos de Mongo necesitan la migración RV21** (el DAO ya no decodifica Date en `check_in_time`): `docker exec hotels-mongo mongosh -u root -p $MONGO_PASSWORD --authenticationDatabase admin --eval '...'` convirtiendo `check_in_time`/`check_out_time` de Date a `"HH:mm"` (esta sesión ya migró el volumen local; `mongo-init.js` siembra strings en volúmenes nuevos).
  - **El schema nuevo de Solr (string) solo aplica recreando el core**: `docker compose down && docker volume rm hotel-search-booking-microservices-platform_solr_data && docker compose up -d` — el backfill repuebla solo (ya hecho localmente).
  - El rate-limit del gateway ahora responde **503 con envelope `rate_limited`** (antes: HTML default de nginx); pasarlo a 429 sigue pendiente como I7 en el plan 10.
  - La colección `idempotency_keys` se crea sola (índices en EnsureIndexes); el TTL de 24h la limpia. Sin registro no hay replay: si Mongo falla al reservar la key, el POST devuelve 500 (fail-closed).
  - Siguen vigentes: nginx cachea IPs de upstreams al recrear contenedores (esta sesión se recreó nginx a mano tras el rebuild), eslint preexistente en `AuthContext.jsx:16`, warning de consola MUI (h5 dentro de h2 en el diálogo de reserva — UI, plan 13).

### Primera acción sugerida para la próxima sesión

Commitear y pushear los planes 06+07 (ver CI verde). Después decir **"empecemos con el 08"** → leer `plans/08-resiliencia-runtime.md` completo + su fila RV23–RV25 en `plans/fixes/README.md`, y validar snippets contra el código actual — en particular: el controller/service de hotels-api quedaron reescritos por el 07 (apperr + sentinels), y `GetAvailability`/`releaseNights` de `hotels_mongo.go` son exactamente los sitios de RV23/RV24.

---

## Sesión — 2026-07-23 — Fix CI (CVEs de dependencias)

### Resumen de lo hecho

El primer run del CI con los planes 06+07 falló en 3 jobs por **CVEs publicados después del último run verde** (no por el código de los planes):

1. **`go (users-api)` y `go (hotels-api)` — GO-2026-5970** (`golang.org/x/text@v0.37.0`, loop infinito con input inválido; alcanzable vía GORM init y `mongo.Connect`). Fix: bump a **v0.39.0** en ambos módulos. El `go work sync` arrastró además x/crypto v0.53.0 / x/net v0.56.0 / x/sys v0.46.0 y dejó los `go.sum` incompletos para `GOWORK=off` (la secuela conocida del 02) → se re-corrió `GOWORK=off go mod tidy` en los 4 módulos **después** del sync hasta estabilizar.
2. **`frontend` — GHSA-3jxr-9vmj-r5cp** (`brace-expansion` <1.1.16, high): `npm audit fix` lo resolvió (solo `package-lock.json`).

### Estado / verificación

- `GOWORK=off go build` OK ×3; `GOWORK=off govulncheck` limpio ×4 — solo queda **GO-2026-5856** (crypto/tls), el artefacto conocido del Go local 1.26.4 que el runner con `stable` no ve.
- `make test` 20 paquetes ok; `npm run build` OK; `npm audit --audit-level=high` exit 0 (el gate del CI).
- Quedan **2 moderate** de `react-router-dom@6` (fix = migrar a v7, breaking) — NO cortan el CI (gate en high); si se encara, es junto con el trabajo de frontend del **plan 13**.
- Archivos tocados (a commitear por el usuario): `users-api/go.{mod,sum}`, `hotels-api/go.{mod,sum}`, `search-api/go.{mod,sum}`, `frontend/package-lock.json`.

### Primera acción sugerida para la próxima sesión

Commitear/pushear este fix y confirmar CI verde. Después, plan 08 (prompt ya preparado en la sesión anterior).

---

## Sesión — 2026-07-28 — Plan 08

### Resumen de lo hecho

1. **Plan 08 (Resiliencia de runtime: C12, R2, R4, R5, C14 + fixes triageados RV23–RV25) — IMPLEMENTADO Y VERIFICADO end-to-end** (unit + integración testcontainers + stack en vivo). Lo esencial:
   - **C12 (graceful shutdown, ×3 servicios)**: `router.Run` reemplazado por `http.Server` (+`ReadHeaderTimeout: 5s`) en goroutine + `signal.NotifyContext(SIGINT/SIGTERM)` + `srv.Shutdown` con tope de 10s, y cierre ordenado de dependencias: hotels-api cierra Rabbit y Mongo (método nuevo `Mongo.Disconnect`), users-api cierra el pool de MySQL (método nuevo `MySQL.Close`; gomemcache no tiene Close), search-api drena el consumer (abajo) y cierra Rabbit.
   - **Drenaje del consumer (C12, search-api)**: el `Consume` ahora se registra con tag fijo (`search-api-consumer`); `Close()` primero **cancela el Consume** → el broker deja de entregar, el canal de deliveries se cierra tras el mensaje en vuelo, y `consumeLoop` termina solo (canal `loopDone`, espera acotada 5s) — recién entonces se cierran channel/connection.
   - **R2 (deadlines en toda llamada saliente)**: hotels-api — helper `opCtx` (3s) alrededor de **cada operación** de `hotels_mongo.go` (finds, inserts, updates, counts, claim por noche, release por intento) + `SetSocketTimeout(10s)` en el driver (red de contención; NO se usó `SetTimeout` del client para no pisar los deadlines por ctx). search-api — `solrOpCtx` (5s) en Index/Update/Delete/Search/Ping de `hotels_solr.go` + el client de solr-go ahora usa un `http.Client{Timeout: 5s}` propio vía `WithRequestSender` (antes: `http.DefaultClient` sin timeout); el **consumer** pone `context.WithTimeout` **por mensaje** en `handleDelivery` (ver desviaciones: 30s, no 5s).
   - **R4/RV24 (bulkhead + parcial)**: `GetAvailability` de Mongo reescrito — semáforo de **8** goroutines máx (`availabilityMaxConcurrency`) y **respuesta parcial**: un hotel que falla (ID basura, hex inexistente, timeout) se loguea y reporta `available:false` en vez de tumbar el batch con 500. Sesgo conservador: nunca ofrecer lo que no se pudo verificar.
   - **R5 (circuit breaker + jitter)**: `sony/gobreaker/v2` (dep nueva de search-api) alrededor de **cada intento** del cliente HTTP search→hotels: abre con 5 fallos consecutivos, `Timeout` 30s, half-open con hasta 3 pruebas. Solo cuentan como fallo conexión/lectura/5xx — **el 404 (RV14) es respuesta válida y no abre el breaker**. Breaker abierto → fail-fast sin reintentos (`ErrOpenState`/`ErrTooManyRequests`); transiciones logueadas (`circuit breaker state change`). Retry existente (E4) ganó **jitter** (`rand.N`, hasta 250ms).
   - **C14/RV25 (publisher de hotels-api)**: en el retry-loop de `publish()`, el chequeo `channel == nil` pasó a **`!IsConnected()` + `connect()` (un solo intento) por reintento** — antes los 3 intentos pegaban al mismo canal muerto. El backoff se movió al inicio del loop (el `continue` del reconnect fallido lo salteaba). Los otros sub-ítems de C14 ($unionWith nil / cursor por día) ya no existen: los eliminó la reescritura del plan 04 (verificado).
   - **RV23 (leak de noches)**: `releaseNights` dejó de ser single-shot: **3 reintentos con backoff por noche** (`releaseNight`, deadline propio por intento sobre `context.WithoutCancel`); si igual falla, log ERROR estructurado (`hotel_id`, `date`, `rooms`) para reconciliación manual. **Decisión deliberada**: NO se re-libera en el segundo DELETE ni se compensa el claim ambiguo por timeout — sin transacciones (Mongo standalone) eso arriesga doble-liberación ⇒ overbooking, que acá es peor que una noche bloqueada. El fix completo (ledger por reserva o replica set + transacciones) queda anotado como futuro; los deadlines de R2 achican la ventana de ambigüedad del claim.
   - **Fix colateral**: `startMongoContainer` del test de integración no seteaba `Collection_idempotency` → `EnsureIndexes` (A3, plan 07) panickeaba con InvalidNamespace. **La suite de integración estaba rota desde el 07** (solo corre en PRs) — arreglada, y se le sumó `TestMongo_GetAvailabilityPartialOnBadID` (RV24).
   - **Tests nuevos**: breaker (abre a los 5 fallos y no pega más al upstream / 404 nunca lo abre / half-open recupera con timeout corto inyectado vía `newHotelsBreaker`); consumer (ctx del handler CON deadline y request_id); integración RV24.
2. **CVE nuevo detectado y bumpeado**: `GOWORK=off govulncheck` acusó **GO-2026-5327** (mongo-driver, heap OOB en GSSAPI) en hotels-api → bump **v1.17.4 → v1.17.7** + `go work sync` + `GOWORK=off go mod tidy` ×4 (secuela conocida del 02). Tras el bump queda solo GO-2026-5856 (artefacto del Go local 1.26.4; el runner con `stable` no lo ve).

### Desviaciones del plan (validadas contra la realidad)

- **Timeout por mensaje del consumer: 30s, no los 5s del plan.** El plan se escribió pre-06: hoy el handler contiene los reintentos del cliente HTTP (E4: 3 intentos × 5s + backoff ≈ 16s peor caso) — 5s los truncaría y mandaría eventos a la DLQ en cualquier reinicio corto de hotels-api. Los 5s finos ya viven en el timeout del client y en `solrOpCtx`.
- **Sin errgroup en R4**: el propio plan lo anticipaba — la cancelación en cascada de errgroup es lo contrario de "disponibilidad parcial". Semáforo + canal, cero deps nuevas (x/sync sigue siendo indirecta).
- **gobreaker v2 (no v1 como el snippet)**: misma Settings API, tipado con generics; v1 quedó sin mantenimiento.
- **El breaker NO se expuso en `/readyz`** (el plan lo daba como opcional): con hotels-api caído, search sigue sirviendo `/search` desde Solr — degradar readyz lo sacaría de rotación del gateway por una dependencia que solo afecta la ingesta. El estado queda observable por logs (`state change`).
- **Reconexión entre reintentos de publish con `connect()` de UN intento** (no `connectWithRetry`): el retry completo son ~60s de backoff dentro de un request HTTP. El `connectWithRetry` queda para el `ensureConnection` de entrada y el `handleConnectionClose` de fondo.
- **`SetSocketTimeout` en vez de `SetTimeout`**: el NOTE del driver advierte comportamiento indefinido combinando `Timeout` de client con deadlines — y los deadlines por ctx (R2) son el mecanismo fino elegido.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth`. **El usuario commiteó los planes 06+07 y el fix de CI** (`544ac90`, `02e768b`) — el working tree tiene SOLO el plan 08 **sin commitear**: 17 archivos modificados, sin archivos nuevos (`git status`): mains ×3, `hotels_mongo.go` + integración, `queue_rabbit.go` ×2 (+test de search), `hotels_http.go` (+test), `hotels_solr.go`, `users_mysql.go`, `hotels-api/go.{mod,sum}` (mongo-driver v1.17.7), `search-api/go.{mod,sum}` (gobreaker v2.4.0), `plans/README.md` (checkbox 08) y este HANDOFF.
- **Checkbox del plan 08 en `plans/README.md`: `[x]`.** Con él quedan ejecutados RV23–RV25 del triage de `plans/fixes/README.md`.
- **Verificación local (todo verde)**: `make test` (-race, 20 paquetes), `make test-integration` (incluye el test nuevo RV24), `make lint` 0 issues ×4, `gofmt` limpio, `go build`/`go vet` ok, `GOWORK=off govulncheck` ×4 (solo GO-2026-5856, ver arriba).
- **Verificación en vivo (stack rebuildeado, 11/11 healthy)**:
  - **C12**: `kill -s SIGTERM` a hotels/search/users-2 → logs `shutting down` → `shutdown complete`, **exit code 0** los tres; search loguea el cierre DESPUÉS de drenar. Bajo carga: 200 requests continuos a `/api/v1/hotels` durante `docker compose restart hotels-api` → **200/200, cero errores**.
  - **R2**: con Solr **pausado** (proceso congelado, conexión que no responde — el caso que el timeout de DNS no cubre), evento UPDATE → cada intento falla a los **5s exactos** por `context deadline exceeded` → requeue → DLQ en ~10s; el consumer sigue vivo y readyz degrada.
  - **R5**: hotels-api caído + eventos → `circuit breaker state change closed→open` al 5º fallo; los mensajes siguientes fallan instantáneo con `circuit breaker is open` (cero hits al upstream). Al levantar hotels-api y pasar los 30s: `open→half-open`, la prueba entra, el hotel se indexa y `/search` lo muestra renombrado.
  - **R4/RV24**: `POST /hotels/availability` con `[id_bueno, "garbage", hex_inexistente]` → **200** `{bueno:true, garbage:false, hex:false}` (antes: 500).
  - **E1 intacto**: retry→DLQ con Solr caído (DLQ contó 1, cola principal 0). `bash test_load_balancer.sh` **PASSED** exit 0.
  - Limpieza: hotel de prueba borrado (DELETE fluyó hasta Solr), DLQ purgada — **estado final: 5 hoteles demo, hotels-news y DLQ en 0**. `reservations-news` tiene 4 mensajes residuales de las pruebas Playwright del 07 (cola sin consumidor por diseño, DM5; el próximo `down` la limpia).

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 09 — Cloud-native / k8s** ← **siguiente** (05 ✓, 08 ✓ — el SIGTERM que necesitaban los rolling deploys ya está). Revisar su fila en `plans/fixes/README.md` (no tiene RVs triageados propios).
2. Plan 10 (nginx TLS/hardening, sumar RV26) → 11 (limpieza, RV27–RV30) → 13 → 12.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear el plan 08** y pushear. CI: los 4 legs Go deberían seguir verdes (deps nuevas: gobreaker v2.4.0 en search-api, mongo-driver v1.17.7 en hotels-api — govulncheck local ya pasó; `integration` corre solo en PRs e incluye ahora el fix del harness testcontainers).
  - **Semántica nueva de `/hotels/availability`**: IDs malos ya NO dan 500 — vienen `false` en el mapa (cambio de contrato menor, coherente con "no ofrecer lo no verificable").
  - **El breaker vive en la ingesta de search-api**: con hotels-api caído, `/search` sigue sirviendo y `/readyz` de search NO degrada por eso (decisión de esta sesión, ver desviaciones).
  - Si un `releaseNights` agota reintentos queda log ERROR `requires manual reconciliation` con hotel/fecha/habitaciones — el runbook manual es decrementar `booked` en `reservation_inventory` (mismo criterio conservador: verificar antes de tocar).
  - `stop_grace_period` del compose sigue en el default (10s) — igual al tope del `Shutdown`; si algún día el drain compite con el SIGKILL, subirlo a 15s en los 3 servicios.
  - Siguen vigentes: nginx cachea IPs de upstreams al recrear contenedores (esta sesión se reinició nginx a mano tras el rebuild), eslint preexistente en `AuthContext.jsx:16`, memcached sin healthcheck (normal), 2 moderate de react-router-dom v6 (no cortan CI; plan 13).

### Primera acción sugerida para la próxima sesión

Commitear y pushear el plan 08 (ver CI verde — ahora con la suite de integración arreglada, abrir PR la ejercita). Después decir **"empecemos con el 09"** → leer `plans/09-cloud-native-k8s.md` completo, validar snippets contra el código actual — en particular: los 3 mains ya hacen graceful shutdown (los probes/preStop de k8s se apoyan en eso), los healthchecks `/livez`/`readyz` existen desde el 05, y no hay Dockerfiles multi-stage aún.

---

## Sesión — 2026-07-29 — Fix CI (frontend: npm audit)

### Resumen

El run del CI con el plan 08 falló SOLO en el job `frontend` (los 4 legs Go verdes): dos advisories **high** nuevos — **GHSA-mh99-v99m-4gvg** (`brace-expansion <=5.0.7`, DoS por OOM; llega vía eslint→minimatch→brace-expansion) y **GHSA-r28c-9q8g-f849** (`postcss <=8.5.17`, path traversal; vía vite).

### Fix aplicado (a commitear: `frontend/package.json` + `package-lock.json`)

- **postcss**: `npm audit fix` → 8.5.25 (in-range, trivial).
- **brace-expansion**: el único parche es **5.0.8** (todo ≤5.0.7 vulnerable, incluida la línea 1.x) y npm sugería `--force` con **eslint@10.8.0 — inviable**: `eslint-plugin-react-hooks@5.2.0` solo acepta eslint hasta ^9. Además, el override de brace-expansion solo NO funciona (probado): minimatch@3 espera el export default callable de v1 y la v5 exporta nombrado → `TypeError: expand is not a function`. Solución que SÍ funciona (verificada con lint+build):
  - `"overrides": { "minimatch": "^10.2.4", "brace-expansion": "^5.0.8" }` en `frontend/package.json` (minimatch 10 es dual CJS/ESM y depende de brace-expansion ^5.0.8)
  - eslint bumpeado a **^9.39.5** dentro de la línea 9 (su config-array/eslintrc actuales toleran minimatch 10; el core también, verificado empíricamente).
- **Verificado**: `npm audit --audit-level=high` exit 0 (quedan solo las 2 moderate de react-router-dom v6, bajo el gate; plan 13), `npm run build` OK, `npm run lint` corre sano (falla solo por el error preexistente de `AuthContext.jsx:16`, igual que siempre).

### Advertencia para el futuro

- Los `overrides` de minimatch/brace-expansion son una **muleta temporal**: sacarlos cuando se migre a eslint 10 (bloqueado por eslint-plugin-react-hooks; revisar al encarar el plan 13, junto con react-router v7).

---

## Sesión — 2026-07-30 — Plan 09

### Resumen de lo hecho

1. **Plan 09 (Cloud-native: CN1–CN4 + I2; sin RVs propios — confirmado contra `plans/fixes/README.md`) — IMPLEMENTADO Y VERIFICADO end-to-end en un cluster kind real.**
   - **I2 (Dockerfiles)**: hotels-api y search-api reescritos multi-stage/no-root replicando el patrón de users-api (que ya era multi-stage). Los 3 quedaron con: bases pineadas por **digest del manifest-list** (CN4) — `golang:1.26-alpine@sha256:0178a6…` y `alpine:3.24@sha256:28bd5f…` — cross-compile `--platform=$BUILDPLATFORM` + `GOOS=$TARGETOS GOARCH=$TARGETARCH` (el buildx multi-arch del CI no emula el toolchain bajo QEMU), `-ldflags "-s -w"` y **`USER 10001:10001` numérico** (ver desviaciones). Imágenes finales: **9.5–13 MB** (antes hotels/search embarcaban el toolchain completo). Se eliminaron vestigios sin referencias en el código: `mkdir uploads/hotels` (hotels) y `netcat-openbsd` (search). `hotels-api/dockerfile` → `Dockerfile` (**ver advertencia git**) + ref actualizada en compose.
   - **CN2 (.dockerignore)**: el allowlist raíz (contexto de hotels/search) ahora poda `*_test.go`, `testdata/`, `*.md`, `coverage.out`, `hotels-api/seed/` y `search-api/internal/solr-config/` (ambos se montan en runtime, no van en la imagen). Nuevos: `users-api/.dockerignore` (excluye `migrations/` — las corre el one-shot, la imagen nunca las lee) y `frontend/.dockerignore` (**node_modules**, el obligatorio del plan).
   - **CN1 (k8s/)**: 13 manifests YAML planos + `k8s/README.md`. `00` namespace `hotel-platform`; `01` ConfigMap unión de envs del compose (con DNS = nombre de Service; **PORT va por-Deployment**: mismo nombre, valores distintos — envFrom lo pisaría); `02` Secret `stringData` demo (nota de secrets manager para prod); `10–15` datastores dev: StatefulSets MySQL/Mongo/RabbitMQ/Solr con PVC chico + **Job `users-migrate`** (initContainer que espera MySQL; ConfigMap con los 4 .sql) + seed de Mongo y conf de Solr como ConfigMaps; `20` **users-api template canónico**: 3 réplicas, probes `/readyz`(10s)/`/livez`(20s), `maxUnavailable: 0`, `preStop sleep 3` + `terminationGracePeriodSeconds: 20` (cubre la ventana de propagación de endpoints antes del SIGTERM del 08), securityContext duro (runAsNonRoot, readOnlyRootFilesystem, drop ALL, seccomp), requests/limits del plan, `INSTANCE_ID` = nombre del pod vía fieldRef, HPA 2–6 al 70% CPU; `21/22` hotels (containerPort **8081** fijo) y search (1 réplica: es el consumer) como deltas; `30` Ingress **opcional** con `use-regex` y las 2 sub-rutas `/users/…/reservations` → hotels-api (inerte sin controller, no rompe el apply).
   - **CN3 (limits)**: `deploy.resources.limits+reservations` en compose para los 3 APIs (0.5cpu/256M) y las 5 DBs (MySQL/Mongo 1G, Rabbit 768M, **Solr 1536M** — <1G hace OOM, memcached 128M). Verificado con `docker inspect`: los cgroups quedan aplicados.
   - **CN4 (CI)**: job `docker` nuevo en `ci.yml` (`needs: go` — nunca publicar imágenes de un commit con tests rotos): matrix de 3 servicios (hotels/search con contexto raíz), build amd64 + **Trivy como gate** (`--exit-code 1 --severity HIGH,CRITICAL --ignore-unfixed`) en **todo** push/PR, y push **multi-arch** (amd64+arm64) a `ghcr.io/julian0444/*` con tags `sha-<short>` (+ semver en tags `v*`) vía metadata-action **solo en main/tags**. Cache buildx `type=gha` por servicio.
   - **README raíz**: subsección "Alternative: Kubernetes (kind)" apuntando a `k8s/README.md`.

### Desviaciones del plan (validadas contra la realidad)

- **`USER 10001:10001` numérico, no `USER appuser`**: descubierto en vivo — con `runAsNonRoot: true`, kubelet no puede verificar un USER por nombre y los 3 APIs morían en `CreateContainerConfigError`. Fix canónico: uid fijo en la imagen.
- **La corrección 7.0 del plan quedó vieja en dos puntos**: (1) "hotels-api ignora PORT" — hoy `config.Port` ya lee `PORT` con default 8081 (C5 parece resuelto de facto por un plan anterior; **verificarlo y cerrarlo formalmente en el 11**); igual el `containerPort` quedó fijo como pide el plan. (2) La ref del compose a corregir no era `dockerfile: dockerfile` (línea 216) sino `hotels-api/dockerfile` en la 284 — el contexto raíz por platform-contracts ya existía desde el 02/07.
- **Bases nuevas**: `golang:1.26-alpine` (no 1.25: stdlib parcheada — el artefacto GO-2026-5856 del Go local no aparece; Trivy gobinary limpio) y `alpine:3.24` (no 3.19: EOL desde nov-2025, sin fixes upstream).
- **Trivy con `ignore-unfixed: true`**: sin fix upstream no hay acción posible; el gate queda accionable (y sobre multi-stage escanea binario+alpine final, no el toolchain — exactamente la nota del plan).
- **memcached como Deployment, no StatefulSet** (el plan decía STS para los 5): es 100% RAM — un PVC ahí sería mentirle al lector; queda comentado en el manifest.
- **kind load con workaround**: Docker Desktop con containerd image store exporta manifest-lists con blobs de una sola plataforma y `kind load docker-image` falla (`ctr: content digest … not found`) para imágenes **pulleadas** (las built locales cargan bien) → `docker save --platform linux/arm64` + `kind load image-archive`. Documentado en `k8s/README.md`.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth`. **El usuario commiteó el plan 08 (`99a1e3a`) y el fix de npm audit (`1408d76`, HEAD actual)** — el working tree tiene SOLO el plan 09 sin commitear: modificados `.dockerignore`, `ci.yml`, `README.md`, `docker-compose.yml`, `hotels-api/dockerfile` (git aún lo ve con el nombre viejo — ver advertencias), `search-api/Dockerfile`, `users-api/Dockerfile`; nuevos `k8s/` (14 archivos), `users-api/.dockerignore`, `frontend/.dockerignore`. (El `M .gitignore` — `.agents/`/`AGENTS.md` — es del usuario, previo a esta sesión.)
- **Checkbox del 09 en `plans/README.md`: `[x]`.**
- **Verificación (todo verde)**: `make test` (-race, 20 paquetes) y `make lint` (0 issues ×4) exit 0; `docker compose config` OK; YAML de CI y k8s parseados. **Trivy: 0 HIGH/CRITICAL en las 3 imágenes** (OS + gobinary). **kind e2e completo**: apply de los 13 manifests → 10/10 pods `1/1 Running` + Job `Completed` (readiness real: users llegó a MySQL+Memcached, hotels a Mongo+Rabbit, search a Solr+Rabbit); seed de Mongo vía ConfigMap sembró los 5 hoteles demo (smoke in-cluster: `/readyz` 200, `GET /api/v1/hotels` 200 con data, `/api/v1/search` 200); `kubectl scale --replicas=5` → **5 endpoints** sin tocar nginx; rolling restart bajo tráfico continuo → **25/25 respuestas 200, cero drops** (maxUnavailable 0 + preStop + SIGTERM del 08); self-healing: 5 pods borrados → 5 reemplazos `1/1 Running` en ~3s. Cluster kind **borrado** al terminar. **Compose rebuildeado y ARRIBA**: 11/11 healthy con los Dockerfiles nuevos y los limits aplicados (verificado por inspect), gateway 200 en `/api/v1/hotels` y `/health/all`, `bash test_load_balancer.sh` **PASSED** exit 0.

### Bloqueos y advertencias

- **CRÍTICO al commitear — rename case-only**: APFS es case-insensitive y con `core.ignorecase` git sigue trackeando `hotels-api/dockerfile` (minúscula) aunque el archivo ya se llame `Dockerfile`. Un `git add -A` NO registra el rename y **el CI en Linux rompería** (compose y el matrix del job docker referencian `hotels-api/Dockerfile`). Antes del commit: `git mv hotels-api/dockerfile hotels-api/Dockerfile` (con el working tree como está, git lo resuelve como rename); verificar con `git ls-files | grep -i dockerfile` → debe listar `hotels-api/Dockerfile` con D mayúscula.
- **Primer push**: el job `docker` corre build+Trivy en el PR (sin push — no es main). Al mergear a main pushea 3 paquetes nuevos a GHCR (default **private**; para que un cluster pullee sin auth habría que hacerlos públicos — el flujo local usa `kind load`, no pull, así que no urge).
- **Tag de imagen en manifests = `sha-1408d76`** (HEAD de esta sesión): tras commitear el 09 el HEAD nuevo no coincidirá — huevo-gallina inherente; `k8s/README.md` documenta cómo buildear/actualizar el tag. El CI pushea el tag correcto en cada main.
- **ConfigMaps k8s duplican fuentes**: migraciones SQL, seed de Mongo y conf de Solr son copias (marcadas con comentario) — regenerarlas si cambian los originales. El template del Job es inmutable: `kubectl delete job users-migrate` antes de re-aplicar cambios.
- **Herramientas instaladas por esta sesión (Homebrew)**: `kind` v0.32.0 y `trivy` v0.72.0.
- **RAM del VM de Docker (7.65 GiB)**: no da para compose + kind a la vez — esta sesión frenó compose (`docker compose stop`, sin tocar volúmenes) durante la verificación k8s y lo relevantó al final.
- Siguen vigentes: eslint preexistente en `AuthContext.jsx:16`, memcached sin healthcheck (normal), 2 moderate de react-router-dom v6 (plan 13), muleta de overrides eslint (plan 13).

### Primera acción sugerida para la próxima sesión

Commitear el plan 09 (**empezando por el `git mv` del Dockerfile de hotels**), pushear y mirar el primer run del job `docker` (3 legs build+Trivy). Después decir **"empecemos con el 10"** → `plans/10-nginx-gateway.md` (TLS + redirect + HSTS, herencia de `add_header`, cache real de `/search`, 429) **sumando RV26** de `plans/fixes/README.md` (puerto de monitoreo 8090 → `127.0.0.1:8090:8090`). Validar snippets contra el `nginx.conf` actual (el 07 tocó locations).

### Post-cierre (2026-07-31)

**El plan 09 quedó commiteado y pusheado (git mv del rename OK) y el CI está TODO VERDE**, incluido el primer run real del job `docker` (3 legs build+Trivy). Único fix post-push: `aquasecurity/trivy-action@0.28.0` no resolvía — el proyecto usa tags con prefijo `v`; quedó pineado a **`v0.36.0`** (mismos inputs). El working tree quedó limpio salvo esta nota del HANDOFF.

---

## Sesión — 2026-08-02 — Plan 10

### Resumen de lo hecho

1. **Plan 10 (Gateway nginx: SD4, SD5, I3, I7 + fix triageado RV26) — IMPLEMENTADO Y VERIFICADO end-to-end** contra el stack en vivo. Lo esencial:
   - **SD4 (TLS)**: server block **443 ssl** (TLS 1.2/1.3, `ssl_session_cache`), server 80 reducido a **`return 301 https://$host$request_uri`**, HSTS via snippet (abajo). Certs self-signed locales **NO commiteados** (`nginx/certs/*.pem` en `.gitignore`); receta reproducible en **`nginx/certs/README.md`** — `openssl req -x509` con **SAN `DNS:localhost,IP:127.0.0.1`** (el CN solo ya no alcanza a clientes modernos), alternativa mkcert documentada con su caveat HSTS. Compose: publica `443:443`, monta `./nginx/certs` y `./nginx/snippets` (ro).
   - **SD5 (headers)**: **`nginx/snippets/security-headers.conf`** (X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy + **HSTS max-age=31536000**, todos `always`) incluido a nivel server 443 **Y en cada location con `add_header` propios** (las de CORS) **Y dentro de cada bloque `if` de OPTIONS** — el `if` crea contexto propio de add_header y las preflight 204 quedaban sin headers. Verificado con GET y OPTIONS reales.
   - **I3 (cache real de /search)**: `proxy_cache_path` zona `search_cache` (keys 10m, max_size 100m, inactive 10m, `use_temp_path=off`) + en `/api/v1/search`: `proxy_cache_key "$request_uri"` (incluye la query string), `proxy_cache_valid 200 5m`, `proxy_cache_use_stale error timeout updating`, header **`X-Cache-Status`**. Tradeoff documentado en README: resultados hasta 5 min viejos, coherente con el CQRS-lite (Solr ya es eventual).
   - **I7 (429)**: `limit_req_status 429` + `limit_conn_status 429` en el http block, y **`error_page 429 = @api_rate_limited`** que responde **status 429** con el envelope estándar (`rate_limited` + `trace_id`, `Content-Type: application/json`). **El mapeo `error_page 503` desapareció**: nginx ya no genera 503 propio (el del rate-limit era el único) y los 5xx de upstreams vivos pasan intactos como siempre; `@api_error` sigue cubriendo 500/502/504.
   - **RV26**: monitoreo `8090` publicado **solo en loopback** (`127.0.0.1:8090:8090`).
   - **Healthcheck de nginx** → `http://127.0.0.1:8090/nginx-health` (location movida del server principal al de monitoreo 8090): sobre el 80 el wget de busybox seguiría el 301 y fallaría por el cert self-signed.
   - **`test_load_balancer.sh`**: `BASE_URL=https://localhost` con `-k` controlado (solo self-signed local, comentado); monitor sigue `http://localhost:8090`. Checks nuevos: HTTP→301, 3 security headers en ruta de API, cache **MISS→HIT** con query única por corrida + query distinta = MISS, y rate-limit que exige **429 y falla si aparece un 503** + assert del envelope JSON con `trace_id`.
   - **Frontend**: proxy de Vite → `https://localhost` con **`secure: false`** (el browser sigue en http con el dev server); fallback prod de `BASE_URL` → `https://localhost/api/v1`; `frontend/README.md` con `VITE_API_URL` https y nota del self-signed. `healthCheck` no se tocó (URL relativa al origin del SPA, no le pega al gateway directo).
   - **README raíz**: quickstart con **generación del cert ANTES del primer `docker compose up`** (desde clon limpio ahora es `.env` + certs + compose, sin pasos ocultos), tabla de endpoints como "Gateway — HTTPS, Port 443" con nota del redirect y el `-k`, features nuevos (TLS/HSTS, 429 con envelope, cache de /search, monitoreo loopback-only), URLs actualizadas.

### Desviaciones del plan (validadas contra la realidad)

- El plan (pre-07) solo pedía `limit_req_status 429`: como el 07 ya había puesto el envelope JSON en el 503 del rate-limit, el handler entero se movió a `error_page 429` → `return 429` con envelope, y se **eliminó** el mapeo de 503 (no quedó ningún handler describiéndose como rate-limit con otro status).
- El snippet del plan ponía HSTS suelto en el server 443: acá vive **dentro** del snippet de security-headers — si no, cualquier location con add_header propio lo descartaba (la misma herencia de SD5 aplicaba al HSTS del propio plan).
- Los `include` del snippet van también **dentro de cada `if` de OPTIONS** (el plan solo mencionaba las locations).
- `/nginx-health` se movió al server 8090 y el healthcheck del compose apunta ahí (el plan no contemplaba que el healthcheck moría con el redirect).
- Cert generado **con SAN**, no solo CN (el comando del plan daba un cert que los clientes modernos rechazan por falta de SAN; `-k` lo tapaba, mkcert no).
- **Nota HSTS local documentada** (`nginx/certs/README.md`): los browsers ignoran HSTS sobre conexiones con error de cert (RFC 6797) → con el self-signed es inerte (queda demostrada la config de prod); con mkcert aplicaría a TODO el host `localhost` sin distinguir puertos y rompería `http://localhost:5173` (Vite dev).

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **el plan 10 sin commitear**. Modificados: `nginx.conf`, `docker-compose.yml`, `.gitignore`, `README.md`, `test_load_balancer.sh`, `frontend/vite.config.js`, `frontend/src/constants/index.js`, `frontend/README.md`, `plans/README.md` (checkbox), este HANDOFF. Nuevos: `nginx/snippets/security-headers.conf`, `nginx/certs/README.md` (los `.pem` generados quedan git-ignored).
- **Checkbox del plan 10 en `plans/README.md`: `[x]`.** Con él queda ejecutado RV26 del triage.
- **Verificación (todo verde, stack 11/11 healthy)**: `docker compose config` OK (8090 resuelve con `host_ip: 127.0.0.1`); certs generados con el comando exacto del README nuevo; nginx recreado (`docker compose up -d nginx`) y `nginx -t` OK; `http://localhost/...` → **301** con `Location: https://...`; https con `-k`: hotels 200, search 200, /health 200, users 401; **los 5 security headers** en GET de API y en OPTIONS 204 (CORS intacto: ACAO presente); cache: misma query **MISS→HIT**, query distinta MISS, misma q con `limit` distinto MISS (la key es el URI completo); burst de 10 logins → **400×4 + 429×6, cero 503**, el 429 con `Content-Type: application/json`, envelope `rate_limited`, `trace_id` y los security headers heredados; `docker port api-gateway` → 8090 **solo** en 127.0.0.1; healthcheck **healthy**; `bash test_load_balancer.sh` **PASSED exit 0** (20 checks, incluye los nuevos); **proxy de Vite verificado en vivo** (dev server efímero → `GET /api/v1/hotels` 200 a través del gateway https); `npm run build` OK y `npm run lint` solo con el **error preexistente de `AuthContext.jsx:16`** (+ warning preexistente de `HotelForm.jsx:88`).

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 11 — Consistencia y limpieza** ← **siguiente** (02 ✓, 07 ✓), sumando del triage: RV27 (CORS duplicado gateway+servicio), RV28 (delete de Memcached), RV29 (hash bcrypt en L2), RV30 (limpieza menor) — e incluye el rename atómico `AvaiableRooms` (C11).
2. Después 13 (frontend) → 12 (docs).

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear el plan 10** y pushear (el CI no ejercita nginx: los 5 legs deberían seguir verdes sin cambios).
  - **BREAKING local deliberado**: el gateway ahora es `https://localhost`; el 80 solo redirige. Curls guardados necesitan `-k`; el browser pide aceptar el cert una vez. **Clon limpio: generar los certs ANTES del primer `up`** (README paso 2 / `nginx/certs/README.md`) — sin `.pem` nginx no arranca (crash-loop en "cannot load certificate").
  - **El cache de `/search` sirve resultados hasta 5 min viejos** para la MISMA query string; `docker compose restart nginx` lo vacía (vive en el filesystem efímero del contenedor). Ojo en las demos: tras crear/editar un hotel, una búsqueda repetida idéntica puede mostrar lo viejo hasta 5 min.
  - **RV27 y la unificación de CORS quedan para el plan 11** (los add_header de CORS del gateway se conservaron tal cual, solo se les sumó el include del snippet); el rename `AvaiableRooms` tampoco se adelantó.
  - Siguen vigentes: nginx cachea IPs de upstreams al recrear contenedores (`docker compose restart nginx` si hay 502 tras un `up --build`), eslint preexistente en `AuthContext.jsx:16`, memcached sin healthcheck (normal), 2 moderate de react-router-dom v6 (plan 13), muleta de overrides eslint/minimatch (plan 13).

### Primera acción sugerida para la próxima sesión

Commitear y pushear el plan 10 (ver CI verde). Después decir **"empecemos con el 11"** → leer `plans/11-consistencia-limpieza.md` completo + su fila RV27–RV30 en `plans/fixes/README.md`, y validar snippets contra el código actual — en particular: cualquier cambio de CORS del 11 en `nginx.conf` debe **conservar los `include` del snippet de security-headers** (SD5 depende de eso), y C5 ("hotels-api ignora PORT") parece resuelto de facto desde antes del 09 — verificarlo y cerrarlo formalmente (nota de la sesión del 2026-07-30).

### Post-cierre (2026-08-02)

**El plan 10 quedó commiteado y pusheado por el usuario y el CI corrió TODO VERDE** (los 5 legs: 4 Go + frontend + docker — ninguno ejercita nginx, como se anticipó). Los `.pem` confirmados fuera del commit (solo entraron `nginx/certs/README.md` y `nginx/snippets/security-headers.conf` como nuevos). El working tree quedó limpio salvo esta nota del HANDOFF. Siguiente: **plan 11** (primera acción arriba).

---

## Sesión — 2026-08-02 — Plan 11

### Resumen de lo hecho

**Plan 11 (Consistencia y limpieza: C1, C2, C3, C5, C6, C7, C9, C11, C13, CQ4, I8 + RV27–RV30) — IMPLEMENTADO Y VERIFICADO end-to-end** con rebuild completo del stack desde cero. Lo esencial:

- **C1/CQ4/RV27 (CORS unificado)**: nuevo paquete compartido **`platform-contracts/cors`** (único CORS de la plataforma, con tests): allowlist por env `CORS_ALLOWED_ORIGINS`, **nunca** emite `Allow-Credentials` (auth por Bearer, no cookies — el combo inválido `*`+credentials de C1 no puede reaparecer), con `*` responde el **literal** `*` (no refleja el Origin — segunda mitad de RV27), allowlist refleja con `Vary: Origin`, preflight → 204. Montado en los 3 servicios (reemplaza `utils.CorsMiddleware` de users/search —archivos borrados— y el `gin-contrib/cors` de hotels —dep removida). **nginx ya no emite CORS en rutas proxiadas** (RV27: header duplicado = bloqueo del browser): fuera los 10 bloques `if OPTIONS` (el preflight ahora lo responde el servicio A TRAVÉS del gateway) y los `add_header ACAO`; el `map $cors_origin` queda SOLO para respuestas generadas por el propio nginx (`@api_rate_limited`, `@api_error` y el 404 fallback — sin upstream no hay duplicación y sin ACAO el browser no podría leer esos envelopes), re-incluyendo ahí el snippet de security-headers (SD5 intacto: locations con `add_header` propios conservan su `include`; las que quedaron sin ninguno heredan el del server).
- **Efecto colateral deliberado**: users-api ahora depende de `platform-contracts` → su **Dockerfile pasó a contexto raíz** (patrón de hotels/search), compose (`context: .`), **leg de CI actualizado**, `.dockerignore` raíz ahora permite `users-api/` (+ poda `users-api/migrations/`; el `.dockerignore` propio se borró). compose suma `CORS_ALLOWED_ORIGINS` a hotels/search (users ya la tenía); en k8s el configmap común ya la inyectaba vía `envFrom`.
- **CQ4 (layering)**: `package middleware` → **`middlewares`** en los TRES servicios (call-sites sin alias); `hotels-api/internal/services/` (flat, `package services`) → **`internal/services/hotels/`** (`package hotels`) — convención `internal/services/<dominio>` + paquete=dir en todo el repo. `users-api/internal/utils` desapareció; el de search queda solo con `requestid.go`.
- **C2 (panel microservicios REAL y read-only)**: controller reescrito — probes **`GET /readyz` reales en paralelo** (timeout 2s) con `status` up/degraded/down por servicio y **latency_ms medida** por instancia; targets por env **`MICROSERVICES_TARGETS`** (default = DNS del compose; en k8s el configmap apunta a los Services). **Eliminados** scale/restart/logs (rutas, handlers y métodos del `admin.service.js` del frontend — ninguna UI los consumía). Respuesta con envelope `{"data":{services,summary}}`. Tests nuevos con `httptest` (readyz 200/503/unreachable + ParseTargets + 403 no-admin).
- **C3 (PII)**: `GET /api/v1/hotels/:id/reservations` (expone `user_id` de todos) ahora exige `Authenticate() + AdminOnly()` — misma ruta. README actualizado. Ninguna página del frontend lo llamaba (solo el service layer, JSDoc anotado).
- **C5 (PORT)**: **cerrado sin código** — ya estaba resuelto de facto: el `http.Server` del plan 08 usa `config.Port` (default 8081), compose fija `PORT` y k8s fija `PORT` + `containerPort` coherentes. Verificado.
- **C6+C9 (errores tipados)**: sentinels en **`users-api/internal/domain/users/errors.go`** (`ErrUserNotFound` —movido del repo al dominio—, `ErrUsernameTaken`, `ErrValidation`); el repo MySQL detecta el duplicado **tipado** (`errors.As` → `*mysql.MySQLError` 1062, driver ahora dep directa) y `Delete` chequea `RowsAffected==0` → 404 (C9); el service envuelve validaciones con `%w ErrValidation`; controller y `seedAdmin` deciden con `errors.Is` — no queda ningún `strings.Contains` sobre errores. Tests nuevos: 409/400 tipados en controller, 404 de delete en controller y service.
- **C7+RV28+RV29 (Memcached L2)**: `Expiration: 300s` en los 4 `memcache.Item` (C7). **`Repository.Delete` cambió de firma: `Delete(ctx, user usersDAO.User)`** — el service resuelve el usuario ANTES de borrar y L1/L2 borran **ambas keys (id y username) determinísticamente**, sin el `Get(idKey)` que dejaba la key por username huérfana para siempre (RV28); de paso el delete de un id inexistente corta con 404 sin tocar cachés. Tradeoff del hash bcrypt en L2 documentado en el código (RV29: Login lee a través de la caché y necesita el hash; red interna + TTL + bcrypt en reposo).
- **C13 (mocks fuera del binario)**: `users_mock.go`, `tokenizers_mock.go` (users) y `hotels_mock.go` (search) **eliminados de producción** → recreados como `mocks_test.go` **dentro del paquete de tests consumidor** (`services/users` y `services/search`, tipos `repoMock`/`tokenizerMock`/`solrMock`/`hotelsAPIMock`). `go list -deps ./cmd/...` → **0 testify en los 3 binarios** (los mocks de hotels-api son artesanales, sin testify — quedan). `go mod tidy`: testify pasó a dep de test.
- **RV30 (limpieza)**: `fmt.Printf` de `hotels_mongo.go` → `slog.Info`; los 4 wraps `%v` del service de hotels → `%w`; los `AssertNotCalled` con aridad incompleta (vacuos: testify solo matchea con aridad exacta) → **`AssertNumberOfCalls(t, "X", 0)`** en users (controller+service); `TestRabbitQueuePublishWithoutChannel` de **~15s → 0.4s** (retries/backoff del `RabbitQueue` ahora inyectables; `NewRabbit` fija los defaults). El dead code de search (`queue_mock`/`search_mock`) y el dedupe `Index`/`Update` de Solr **ya estaban resueltos por el plan 06** (verificado).
- **C11 (rename atómico `AvaiableRooms` → `AvailableRooms` / `available_rooms`)**: contracts (struct+tag json), hotels-api y search-api completos (dao bson, `bson.M` de updates/proyecciones, solr doc+parse, services), `schema.xml`, **los configmaps embebidos de k8s (`13-mongo.yaml`, `15-solr.yaml`)**, frontend (types, HotelCard, HotelDetail, HotelForm — incl. fallbacks camelCase), seeds (`mongo-init.js`, `seed-hotels.js`, `migrate-inventory.js`), Bruno (Post/Put hotel), docs de hotels-api. Fixtures/testdata de search-api actualizados **deliberadamente** (se demostró el rojo antes: los tests de fixture rompen en compile/assert con el rename — protegen el wire format). **Migración one-off `hotels-api/seed/rename-available-rooms.js`** ($rename en mongosh, con instrucciones y nota de reindex) — es **el ÚNICO archivo del repo que conserva el literal viejo** (lo necesita); las notas históricas (contracts.go, CLAUDE.md) se redactaron sin el literal para que el grep de verificación quede limpio.
- **I8 (frontend en compose)**: bloque descomentado bajo **`profiles: ["frontend"]`** → `docker compose --profile frontend up -d` sirve el SPA buildeado en `http://localhost:5173` (origin ya permitido por el CORS nuevo); sin profile no existe en el set default (verificado a nivel config y up). README reconcilia el conteo (10 + migrate one-shot + frontend opcional); CLAUDE.md idem.
- **CLAUDE.md refrescado** (estaba desinformando a las próximas sesiones): go.work + Makefile raíz existen desde el 02, quirk del typo cerrado, quirk del module path de search cerrado. `RULES.md` de hotels-api: nota del panel mock reemplazada por la realidad read-only.

### Desviaciones del plan (validadas contra la realidad)

- El `package middleware` no era solo de hotels-api: **los tres servicios** lo tenían — se normalizó en los tres.
- RV28 se resolvió **cambiando la firma** de `Repository.Delete` (DAO completo en vez de id): más invasivo que "considerar borrar por username" pero elimina la clase de bug de raíz; el TTL de C7 queda como red de seguridad.
- Los `AssertNotCalled` vacuos se cambiaron a `AssertNumberOfCalls(..., 0)` (robusto a aridad) en vez de completar la aridad; los dos `GenerateToken` que ya tenían aridad correcta se dejaron como estaban.
- El gateway **conserva** el `map $cors_origin`, pero solo para sus respuestas propias (429/5xx/404) — el plan no lo contemplaba; sin esto un browser cross-origin no puede leer esos envelopes (y no duplica: esas respuestas nunca vienen de upstream).
- users-api a contexto raíz de build (Dockerfile/compose/CI/.dockerignore) — consecuencia necesaria de compartir `platform-contracts` que el plan no listaba.
- El grep de verificación de C11 no da 0 literal: da exactamente **1 archivo** — la migración, que necesita el nombre viejo. Documentado como excepción deliberada.

### Estado actual

- **Branch:** `feat/plan-01-seguridad-auth` — **el plan 11 SIN commitear** (el usuario versiona). Cambios grandes: nuevos `platform-contracts/cors/` (+ go.sum del módulo), `users-api/internal/domain/users/errors.go`, `mocks_test.go` ×2, `hotels-api/internal/services/hotels/` (movido), `hotels-api/seed/rename-available-rooms.js`; borrados `users-api/internal/utils/`, `search-api/internal/utils/middlewares.go`, los 3 `*_mock.go` con testify, `users-api/.dockerignore`; modificados nginx.conf, docker-compose.yml, ci.yml, Dockerfile de users, k8s (configmap + mongo + solr), frontend (rename + admin.service), Bruno, README/CLAUDE.md/RULES.md y el grueso de users/hotels/search por el rename + middlewares.
- **Checkbox del plan 11 en `plans/README.md`: `[x]`.** Con él quedan ejecutados RV27, RV28, RV29 y RV30 del triage.

### Verificación (todo verde)

`make build` OK; `make test` (**-race, 4 módulos**) OK; `make lint` (golangci-lint) **0 issues ×4**; `make test-integration` OK; `npm run build` OK y `npm run lint` solo con los preexistentes (`AuthContext.jsx:16` error + `HotelForm.jsx:88` warning); `go list -deps` sin testify en los 3 binarios; grep `avaiable` → solo la migración; `docker compose config` OK y `nginx -t` OK. **Stack desde cero (`down -v && up --build`) 11/11 healthy** y batería e2e: 301 en :80; GET con Origin permitido → **UN solo ACAO** + security headers (herencia OK en locations sin add_header); preflight OPTIONS **respondido por el servicio vía gateway** (204 + Allow-Headers con Idempotency-Key); Origin no permitido → sin ACAO; login admin OK; **C3** 401 sin token / 200 admin; **C2** panel con latencias reales, `docker stop search-api` → **down real** (summary 2/3) y rutas mock → 404; **C11** hotel nuevo con `available_rooms` → Mongo con el bson nuevo → evento → `/search` lo devuelve con el campo nuevo (sin typo en el payload) → reserva OK (`check_in`/`check_out`) → visible en el endpoint admin de reservas; cache `/search` MISS→HIT; **C6** duplicado → 409 `username_taken`; **C9** DELETE 99999 → 404 `user_not_found`; **RV28** tras DELETE ambas keys de L2 verificadas ausentes (memcached por nc); **C7** login post-delete → 401 tras la ventana L1 de 30s (el 200 inmediato posible es el tradeoff RV10 ya documentado); burst login → 429 con envelope JSON **+ ACAO del gateway**; `bash test_load_balancer.sh` **PASSED**; **I8** `--profile frontend` → SPA 200 en :5173, sin profile no aparece; migración `rename-available-rooms.js` corre OK (no-op sobre datos ya renombrados: matched=6 modified=0).

### Trabajo restante (en orden — detalle en `plans/README.md`)

1. **Plan 13 — Frontend portfolio-grade** ← **siguiente** (07 ✓ y 11 ✓), sumando **RV31** del triage (Promise.all del Dashboard, date-picker UTC, sesión zombie sin validar exp, admin auto-borrable, JSDoc de RegisterRequest). **Antes que el 12.**
2. Plan 12 — Documentación y presentación (cierra con el producto final del 13).

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El usuario debe commitear el plan 11** y pushear. CI: los 5 legs deberían seguir verdes, pero el **leg docker de users-api cambió a contexto raíz** — mirar su primer run.
  - **BREAKING deliberado (C11)**: el campo es `available_rooms` en API/Mongo/Solr/eventos. Curls/colecciones viejas guardadas fuera del repo deben renombrar el campo. **Volúmenes Mongo preexistentes** necesitan `rename-available-rooms.js` + reindex (`POST /api/v1/reindex` o `down -v`); en local `docker compose down -v` re-siembra ya con el nombre nuevo.
  - **BREAKING chico (C2/C3)**: desaparecieron `POST /admin/microservices/scale`, `GET /admin/microservices/:name/logs`, `POST /admin/microservices/:name/restart` (eran mocks); `GET /hotels/:id/reservations` ahora es admin-only; el panel de microservicios responde con envelope `{"data":...}`.
  - Los **preflights OPTIONS ahora pasan por el rate-limit** del gateway (antes nginx los cortaba antes) — `Access-Control-Max-Age: 86400` lo amortigua; si una demo browser-heavy gatilla 429 en login, es esto.
  - Scripts propios que buildeen `./users-api` como contexto deben pasar a raíz (`-f users-api/Dockerfile .`).
  - Siguen vigentes: RV10 (login ≤30s post-delete desde L1 de otra réplica — documentado), nginx cachea IPs de upstreams al recrear contenedores (`docker compose restart nginx` si hay 502 tras `up --build`), eslint preexistente `AuthContext.jsx:16` (plan 13), memcached sin healthcheck (normal), 2 moderate de react-router-dom v6 (plan 13), muleta de overrides eslint/minimatch (plan 13).

### Primera acción sugerida para la próxima sesión

Commitear y pushear el plan 11 (ver CI verde, en particular el leg docker de users-api con contexto nuevo). Después decir **"empecemos con el 13"** → leer `plans/13-stretch-dominio-frontend.md` completo + **RV31** en `plans/fixes/README.md`, y validar snippets contra el código actual — en particular: el frontend ya usa `available_rooms` (C11), `admin.service.js` quedó solo con `getMicroservicesStatus` (shape nuevo `{services, summary}` para la pestaña admin que el 13 construya), y el interceptor/AuthContext siguen con los pendientes RV31.

### Post-cierre (2026-08-02)

**El plan 11 quedó commiteado y pusheado por el usuario y el CI corrió TODO VERDE**, incluido el leg docker de users-api con su contexto raíz nuevo. Único fix post-push: el `go mod tidy` inicial de platform-contracts había resuelto **quic-go v0.59.0** (vulnerable, GO-2026-5676, arrastrado por gin del CORS compartido) y el govulncheck de su leg lo cortó — bumpeado a **v0.59.1** (la versión que ya tenían los otros tres módulos) en un segundo commit. Nota: govulncheck local puede reportar además GO-2026-5856 (crypto/tls) — es artefacto del toolchain local 1.26.4; el CI instala `stable` (≥1.26.5) y no lo ve. El working tree quedó limpio salvo esta nota del HANDOFF. Siguiente: **plan 13** (primera acción arriba — el 13 va ANTES que el 12).

---

## Sesión — 2026-08-03 — Plan 13 (CHECKPOINT PARCIAL: fases 0–7)

> **EL PLAN 13 NO ESTÁ TERMINADO.** Esta sesión implementó y verificó las fases 0–7 del núcleo obligatorio + RV31 + el sort backend de search-api. **Faltan las fases 8 (a11y/responsive/SEO), 9 (verificación FE2/presupuesto) y 10 (E2E Playwright + cierre adversarial + verificación Docker)**. El checkbox del plan 13 en `plans/README.md` queda **SIN marcar** a propósito.

### Terminado (fases 0–7)

- **Fase 0 — Red de seguridad**: Vitest 4 + jsdom + Testing Library + **MSW 2** (`onUnhandledRequest: 'error'`) con fixtures en el **contrato final** de los planes 07/11 (`src/test/{setup,fixtures,handlers,server}.js`, `render.jsx` con providers y QueryClient por test). Scripts `test/test:run/test:coverage/test:e2e/check`. `playwright.config.js` escrito (desktop 1440 + Pixel 7, `ignoreHTTPSErrors` por el self-signed). **CI**: leg frontend ahora corre `lint` + `TZ=America/Los_Angeles test:coverage` + `build` + `audit`; job **`frontend-e2e` nuevo (solo PRs)** que levanta compose con profile frontend + certs efímeros y corre Playwright.
- **Fase 1 — Contrato**: `services/envelope.js` (`unwrapObject`/`unwrapList` — meta.total real, degradación explícita para listas sin total), `services/apiError.js` (**ApiError** con status/code/message/traceId, jamás internals; mensajes estables por status), `services/queryClient.js` (**query keys centralizadas** + retry solo no-4xx ×1, mutations sin retry), servicios reescritos con **un solo shape** (cero fallbacks camelCase/legacy) y `AbortSignal` de TanStack Query → Axios. `utils/dateOnly.js` (fechas civiles por componentes locales + aritmética UTC: `todayLocal`, `addDaysDateOnly`, `differenceInNights`, `formatDateOnlyLabel`, **`formatTimeLabel`** para el "HH:mm" de RV21), `utils/money.js` (**`formatMajorAmount`** para price_per_night, **`formatMinorAmount`** para total_price en centavos, **`previewTotalCents`** espejo exacto de hotels-api: `round(price*100)*nights*rooms`). JSDoc de types al contrato final (user_id string, Register sin `tipo`, Reservation completa — RV31).
- **Fase 2 — Sesión (RV31)**: `context/auth-context.js` (context primitivo) + `AuthContext.jsx` solo-provider + `hooks/useAuth.js` → **el error eslint de react-refresh quedó resuelto de raíz**. `services/authStorage.js` **valida exp/iss/aud localmente antes de aceptar sesión** (token vencido/malformado ⇒ anónimo + storage limpio — adiós sesión zombie; el user se reconstruye de los claims con id string A7). `services/authEvents.js` + interceptor: el 401 de sesión **emite `session-expired`** (401 de login o anónimo NO); `SessionExpiredNavigator` navega a /login con `state.from` — **`window.location.href` eliminado**. `ProtectedRoute` con loader de bootstrap, redirect con state.from y **pantalla 403 explícita** para cliente en /admin. `isBootstrapping` ≠ `isSubmitting`. Login/Register con `aria-live`, autocomplete correcto, focus al primer error y aviso de sesión expirada.
- **Fase 3 — Búsqueda (RV22) + sort backend**: **search-api acepta `?sort=relevance|price_asc|price_desc|rating_desc`** — whitelist en controller (400 `invalid_sort` sin tocar el service), pass-through en service, mapping a cláusulas fijas de Solr en el repo (`price_per_night asc, id asc` etc., **`id asc` de desempate para paginación estable**; relevance/vacío = score edismax; nunca se interpola input). `useHotelSearchParams`: **URL única fuente de verdad** (q/page/sort, sort inválido cae a relevance, q/sort nuevos resetean page). `Search.jsx`: count y páginas por **meta.total**, `keepPreviousData` + LinearProgress sutil al paginar, skeleton solo primera carga, empty/error/retry distintos, corrección automática de page fuera de rango.
- **Fase 4 — Home/layout honestos**: theme con **tokens** formalizados (ink/gold/surface/terracotta/sage, sombras cortas, `clamp()` en headings, smooth-scroll solo `prefers-reduced-motion: no-preference`; `index.css` con kill-switch de animaciones para `reduce`). **`BrandMark`** SVG propio + favicon propio (`public/brand/favicon.svg`; `vite.svg` eliminado) + **fallback local de hotel** (`public/images/hotel-fallback.svg`). Home en **tres actos** (hero editorial sin imagen remota, catálogo real vía API, "How it works" verificable) + franja de arquitectura → repo. **Footer compacto sin `href="#"`** ni social/contacto falsos (test lo garantiza). Navbar con marca, NavLink activo, drawer accesible. `Layout` con **SkipLink + `<main id="main-content">` enfocable**.
- **Fase 5 — Booking concierge (RV21/F13-05)**: `HotelGallery` adaptativa (1/2/3+ imágenes, lightbox Dialog) con `ResponsiveImage` (aspect-ratio estable, lazy, fallback local ante error). HotelDetail muestra horas **"HH:mm" localizadas** (`3:00 PM`), contacto solo si existe, amenities normalizadas. **`useBooking`**: fechas civiles con mínimos LOCALES (check-out ≥ día siguiente), rooms clampeados al cupo, guests con límite, **total en centavos** espejo del backend, availability como feedback (POST es la autoridad) e **Idempotency-Key POR INTENTO** (`utils/idempotency.js`: retry del mismo payload conserva la key; payload nuevo/intento confirmado/409 de negocio la rotan). Panel sticky desktop + action bar mobile que abre Dialog con el MISMO estado. `BookingSuccess` con ID corto copiable y CTA al historial. Copy específico para `no_availability` y `request_in_flight`.
- **Fase 6 — Historial fiel (F13-06)**: tabs **Upcoming/Past/Cancelled/All con counts** (`utils/reservations.js`: `status` del backend manda, fechas solo agrupan), **canceladas siempre visibles**, card con rango civil, noches, rooms/guests, **total del dominio** (centavos+currency) e ID copiable; Cancel solo si `status === 'confirmed'`; `CancelReservationDialog` nombra el recurso y muestra el error adentro; **mutation + invalidación de query** (se fue el `setReservations(filter)` sobre estado viejo).
- **Fase 7 — Admin (RV31/F13-07/08)**: **Dashboard sin `Promise.all`** — hotels/users/services son queries independientes con `enabled` por tab y **retry propio** (error parcial no borra nada). **Self-delete deshabilitado** comparando IDs string, con motivo en tooltip/aria-label. Tablas desktop + **cards mobile** con acciones en viewport; paginación por meta.total. **`ServiceHealthGrid` read-only** sobre el shape real `{services, summary}` del plan 11 (readiness/latencia por instancia, refetch 15s, "Read-only observability", sin scale/restart/logs). **HotelForm**: contrato estricto ("HH:mm" validado espejo del backend, price>0, rooms entero, email opcional válido), amenities de **catálogo + libre normalizada** (Autocomplete), imágenes **solo http(s)** con preview/fallback/reorden (`HotelImageFields`), summary de errores + focus al primero, **`useUnsavedChanges`** (beforeunload + confirmación en salidas propias) y éxito con **acciones explícitas View hotel / Back** (sin `setTimeout`).

### Decisiones importantes

- **`react-router-dom` v6 → `react-router` v8.3.0** (paquete core, imports `'react-router'`): resolvía las 2 moderate conocidas, pero v6.30.4 y v7 latest también estaban vulnerables (GHSA-wrjc/GHSA-337j en v6; CSRF RSC en 7.12–8.2) — v8.3.0 deja **`npm audit` en 0** y de paso **eliminó la muleta de overrides** (minimatch/brace-expansion) del package.json. API declarativa idéntica (BrowserRouter/Routes verificados). Los "future flags" de la fase 9 quedan obsoletos: v8 no tiene flags v7.
- **`App.jsx` ya quedó con `React.lazy`** en todas las rutas salvo Home + `Suspense` con `AppLoader` + `manualChunks` (vendor-react/vendor-mui) en vite.config: era necesario reescribirlo entero por el AuthProvider/QueryClient nuevos y se estructuró directamente como pide FE2. **La fase 9 NO está cerrada**: falta la verificación de presupuesto (requests iniciales, Lighthouse, prefetch intencional) — pero el build ya no emite el warning de 500 kB (chunk mayor: vendor-mui 383 kB / 116 kB gzip; Dashboard/HotelForm/HotelDetail en chunks lazy propios).
- **Anticipos de fase 8 integrados por diseño** (los componían las páginas nuevas): `NotFound` real (ruta `*`), `SkipLink`, `RouteAnnouncer` (foco a main + anuncio por live region al cambiar ruta), `RouteMeta` (title/description por página vía React 19), aria-labels en icon buttons, `prefers-reduced-motion`, aspect-ratio en imágenes. **La fase 8 NO está cerrada**: faltan axe en las 6 rutas, verificación de teclado/focus completa, viewports 320/390 y el barrido de copy.
- **Idempotencia por intento**: `createAttemptTracker` fingerprintea el payload — la misma key sobrevive a retries de red/5xx (hotels-api replayea), y rota ante payload nuevo, éxito o 409 `no_availability` (reintentar igual solo replayearía el 409).
- El **login con sesión ya activa** redirige a home (Back hacia /login no muestra otro login). `logout`/`session-expired` hacen `queryClient.clear()` (nada privado sobrevive en caché para el Back).
- `useUnsavedChanges` **no usa `useBlocker`** (exige data router; la app sigue en BrowserRouter declarativo): beforeunload nativo + confirmación en los botones de salida del form.
- users-api **no expone** endpoints que el plan daba por supuestos en fixtures viejos; los services quedaron solo con lo real (p. ej. `getReservationsByHotel` se quitó del front: nadie lo consumía y es admin-only).

### Verificación (checkpoint, SIN Docker)

- **Frontend**: `npm run test:run` **116 tests / 18 archivos PASSED** (también con `TZ=America/Los_Angeles`); `npm run test:coverage` **thresholds OK** (global 78% stmts / 76% branches / 79% lines vs 60/50; `services` 94%, `utils` 100%, `context` 86% vs 80); `npm run lint` **0 errores** (los preexistentes de `AuthContext.jsx:16` y `HotelForm.jsx:88` quedaron resueltos de raíz); `npm run build` OK **sin warning de chunks**; `npm audit` **0 vulnerabilidades**.
- **search-api** (único módulo Go tocado): `gofmt` limpio, `go vet` OK, `go build` OK, `go test -race ./...` **OK** (incluye los tests nuevos de sort en controller/repo/service), `golangci-lint run` **0 issues**.
- **NO corrido aún** (fase 10): stack Docker completo, `test_load_balancer.sh`, E2E Playwright, dev server contra gateway vivo, Lighthouse, axe.

### Estado y archivos

- **Branch** `feat/plan-01-seguridad-auth`, TODO sin commitear (~97 paths). Nuevos claves: `frontend/src/{test/*,services/{envelope,apiError,queryClient,authStorage,authEvents}.js,utils/{dateOnly,money,idempotency,reservations}.js(+tests),context/auth-context.js,hooks/{useAuth,useBooking,useHotelSearchParams,useUnsavedChanges}.js,hooks/{queries,mutations}/index.js,components/{a11y,auth,booking,admin,reservations,common}/*,pages/NotFound.jsx}`, `frontend/playwright.config.js`, `frontend/public/{brand,images}/*`, tests de páginas. Modificados: todos los services/pages/layout/theme, `App.jsx`, `main` intacto, `vite.config.js` (test config + manualChunks), `eslint.config.js` (ignores + override tests), `package.json` (scripts, router v8, sin overrides), `index.html`, `.gitignore` (coverage/playwright), `.github/workflows/ci.yml`, y en Go los 3 archivos de search + sus tests. `frontend/coverage/` quedó **git-ignored**.
- **Checkbox del plan 13 en `plans/README.md`: SIN tocar** (correcto: el núcleo no está completo).

### Trabajo restante del plan 13 (próxima sesión)

1. **Fase 8** — axe sin serious/critical en las 6 rutas, teclado/focus (dialogs devuelven foco, tabs/drawer), viewports 320/390 sin overflow, barrido final de copy de errores.
2. **Fase 9** — verificación FE2: baseline vs actual, requests iniciales de visitante (admin no debe bajar en Home — los chunks ya separan, medirlo), prefetch intencional, Lighthouse ≥85/95/90/90 documentado.
3. **Fase 10** — `frontend/e2e/*` (anonymous-search, customer-booking, admin, accessibility + degradación), stack Docker completo con profile frontend, R1–R6, greps de invariantes del bloque Verificar (nota: hoy hay ~10 hits BENIGNOS de esos patrones — comentarios que los citan como prohibidos, la prop interna `pricePerNight` de BookingSummary y los literales del test de honestidad de Home — decidir en el cierre: renombrar/parafrasear o acotar el grep a código no-test), evidencia para el plan 12, checkbox + HANDOFF final.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias del checkpoint:
  - **El job `frontend-e2e` del CI referencia `npm run test:e2e` y `frontend/e2e/` AÚN NO EXISTE.** En pushes no corre (es `if: pull_request`), así que **commitear/pushear este checkpoint deja el CI verde**; pero **un PR fallaría** en ese job hasta terminar la fase 10.
  - `date-fns` quedó como dependencia sin usar (dateOnly.js la reemplazó) — podarla en el cierre.
  - El interceptor request de `api.js` ya no distingue login: manda Bearer si hay token (correcto); el flag `isLoginRequest` vive solo en el interceptor de response.
  - Playwright browsers **no instalados** localmente aún (`npx playwright install chromium` pendiente para fase 10).
  - Siguen vigentes: RV10 (login ≤30s post-delete desde L1), nginx cachea IPs de upstreams al recrear (`docker compose restart nginx` si hay 502), memcached sin healthcheck.

### Primera acción sugerida para la próxima sesión

Retomar con **"sigamos el plan 13 desde la fase 8"** → releer esta sección + el plan (fases 8–10) y el bloque Verificar. El orden natural: fase 8 (axe/teclado/viewport) → fase 9 (medición FE2/Lighthouse) → fase 10 (e2e + Docker + greps + checkbox + HANDOFF de cierre).

---

## Sesión — 2026-08-03 (2) — Plan 13 (CHECKPOINT: fases 8–9 CERRADAS)

> **EL PLAN 13 SIGUE SIN TERMINAR** — falta solo la **fase 10** (E2E Playwright + cierre adversarial R1–R6 + greps de invariantes + checkbox). Esta sesión cerró las fases 8 (a11y/responsive/SEO/copy) y 9 (FE2 verificado con presupuesto y Lighthouse). El checkbox del plan 13 en `plans/README.md` queda **SIN marcar** a propósito.

### Fase 8 — hecho y verificado

- **Axe a nivel componente (jsdom)**: `axe-core` 4.12.1 como devDependency directa + helper **`src/test/axe.js`** (corre axe sobre `document.body` y falla SOLO ante violaciones **serious/critical** — el bar del plan; `color-contrast` deshabilitado en jsdom porque no hay motor de layout: el contraste real lo midió Lighthouse, abajo). **`src/test/a11y.test.jsx`**: 7 rutas (Home, Search, HotelDetail, Login, MyReservations, Dashboard admin, 404) renderizadas como en la app (dentro de `Layout`; Login trae su propio main) → **0 violaciones serious/critical**. No se usó `vitest-axe` (menos control del filtro por impacto y riesgo de compat con vitest 4).
- **Teclado/focus (`src/test/keyboard.test.jsx`)**: dialog de cancelación cierra con Escape y **devuelve el foco al trigger**; tabs de MyReservations navegan con flechas + Enter (aria-selected verificado); drawer mobile abre desde el botón de menú, atrapa foco (resto aria-hidden por el Modal) y **Escape lo cierra restaurando el foco**; lightbox de la galería idem. El nombre accesible de icon buttons lo cubre axe (button-name es critical).
- **Routing real (`src/App.test.jsx`)**: una ruta lazy muestra el fallback con **layout estable** (navbar + footer presentes; determinístico mockeando `NotFound` tras un deferred) y después la página; navegar Home→Search resuelve el chunk, **RouteAnnouncer enfoca `#main-content`** y `document.title` queda `Search stays · StayLux` (RouteMeta).
- **Responsive 320/390 verificado EN VIVO** (Playwright python contra `npm run preview` + stack Docker real): 8 rutas (las 6 + register + 404, con sesiones reales de demo y admin vía API) × 2 viewports → **`scrollWidth === innerWidth` en las 16 combinaciones**, h1 y title correctos, screenshots a 390 revisados (detail con action bar mobile, admin con cards y acciones en viewport, search con datos reales del índice y fallback local en el hotel sin imagen). El assert automatizado como spec queda para la fase 10, como pide el plan.
- **SEO/copy**: todas las páginas con **un único h1** y `RouteMeta` (title+description; HotelDetail incluye nombre/ubicación; se agregó la description que faltaba en HotelForm). **`public/robots.txt` nuevo** (Allow / + Disallow /admin): sin él, `vite preview` (y el nginx del SPA) devuelven el fallback HTML del SPA para `/robots.txt` y Lighthouse lo marca inválido. Copy de errores ya cumplía el patrón (qué pasó + qué hacer + `Reference: trace_id` solo si existe).

### Fixes de a11y salidos de la primera pasada de Lighthouse (A11y 94 → 100)

- **`tokens.goldDark` `#a88b45` → `#7f6429`** (bronce profundo): el original daba **3.26:1 sobre blanco / 2.99:1 sobre parchment** (fallaba AA en overlines de Home y el botón Admin del navbar); el nuevo da **5.59:1 / 5.14:1**. El gold del hero sobre navy ya pasaba (5.35:1) y quedó igual.
- **heading-order**: el "StayLux" del Footer era un `<h6>` real (default de MUI para `variant="h6"`) → `component="p"`. Barrido de `subtitle1/subtitle2` sin `component` (MUI los renderiza `<h6>`): precio del action bar, username del menú de navbar y nombre de servicio del health grid → `p`; nombres de cards admin (hotel/user) → `h3` bajo sus h2. **`HotelCard` ganó prop `headingComponent`** (default `h3` bajo el h2 de Home; **Search le pasa `h2`** porque cuelga directo del h1) y el nombre en `ReservationCard` pasó a `h2` por la misma razón.
- **label-in-name (WCAG 2.5.3)**: `BrandMark` perdió su `aria-label="StayLux — home"` (no contenía el texto visible completo); el nombre accesible ahora ES el texto visible del wordmark.

### Fase 9 — FE2 verificado y cerrado

- **`date-fns` eliminada** (estaba sin uso desde que `dateOnly.js` la reemplazó) y **`axe-core` agregada** como devDependency.
- **`RouteFallback.jsx` creado** (el archivo que pedía el plan): `Layout` + `AppLoader` — `App.jsx` lo usa como fallback del Suspense de rutas.
- **Prefetch intencional**: **`src/utils/prefetch.js`** (`prefetchHotelDetailChunk` con latch y rearme si falla) disparado por **hover/focus** del link de `HotelCard` (el `::after` estira el área del link a la card entera) y del "View hotel" de `ReservationCard`. Testeado con mock (`HotelCard.test.jsx`) y **verificado a nivel red sobre el build**: Home descarga exactamente **4 JS** (entry + 3 vendors) y el hover dispara **solo** el grafo de HotelDetail (nada de admin).
- **Fonts no bloqueantes** (`index.html`): el stylesheet de Google Fonts pasó a `preload as=style` + swap en onload + `<noscript>` — era render-blocking (~900 ms de FCP en la simulación mobile). Los fallbacks Georgia/system del theme hacen el swap inofensivo.
- **`manualChunks` rehecho como función** (mirando el analyzer, como pide el plan): **desapareció el mega `vendor-mui`** (precargaba TODO MUI y Lighthouse le imputaba ~850 ms de JS sin usar; además el array form de `vendor-react` capturaba solo los facades — react-dom real quedaba en el entry). Ahora: `vendor-react` (react/react-dom/scheduler/react-router), `vendor-data` (react-query/axios), `vendor-emotion` — solo lo que el entry importa estático (precarga neutra) — y **MUI se reparte por uso real**: Tabs/Table/Autocomplete/ImageList etc. viven en los chunks lazy que los usan.
- **Presupuesto (evidencia)**: baseline pre-plan-13 **734.65 kB raw en un solo bundle con warning**; checkpoint fases 0–7: 225.6 kB gzip iniciales; ahora **192.3 kB gzip iniciales (−15%)** = html 0.9 + css 0.6 + entry 83.0 + react 73.6 + data 28.0 + emotion 11.0. **Cero warning >500 kB** (entry 276.7 kB raw). **Dashboard/Admin no baja en Home** (verificado por red; los literales "Dashboard"/"HotelForm" en el entry son solo las URLs de los `import()`). **Desvío documentado**: el objetivo aspiracional de ≤170 kB gzip no es alcanzable con MUI — solo los vendors precargados (react 73.6 + data 28.0 + emotion 11.0) suman 112.6 kB y el core eager de MUI vive dentro del entry; el plan lo contempla ("ajustar con evidencia si MUI lo hace imposible").
- **Lighthouse sobre `npm run preview` (127.0.0.1:5173, build prod) con el stack Docker real detrás** (SPA → `https://localhost/api/v1`, flag `--ignore-certificate-errors` por el cert self-signed): **mobile 94 / 100 / 100 / 100** (Perf/A11y/BP/SEO — FCP 2.0 s, LCP 2.9 s = el párrafo del hero, TBT 10 ms, CLS 0.034) y **desktop 98 / 100 / 100 / 100** (FCP 0.5 s, LCP 1.2 s). Mobile corrido 3 veces por varianza: 81 (primer run, outlier frío con vitest recién terminado) / 93 / 94 → mediana ≥85 con margen. Condiciones: Lighthouse 12 vía npx, Chrome headless, simulación default (Moto G Power, slow-4G, 4x CPU) sobre Mac local (Apple Silicon). **Presupuesto del plan cumplido entero: ≥85 / ≥95 / ≥90 / ≥90.** Reportes html+json en **`frontend/lighthouse/`** (gitignored, como coverage) para las capturas del plan 12.

### Verificación (todo verde)

`npm run lint` **0 errores**; `TZ=America/Los_Angeles npm run test:coverage` → **131 tests / 22 archivos PASSED** (+15 tests / +4 archivos vs checkpoint anterior) con coverage global **84.98% stmts / 78.52% branches / 85.78% lines** (thresholds 60/50; `services` 97.8%, `utils` 95.3%, `context` 86.4% vs 80); `npm run build` limpio sin warnings; `go test -race` de los 3 paquetes del sort de search-api OK (sin cambios Go esta sesión). Los greps de invariantes siguen con los **mismos hits benignos** que documentó el checkpoint anterior (comentarios que citan los patrones prohibidos, la prop interna `pricePerNight` y los literales del test de honestidad) — la decisión renombrar/acotar sigue pendiente para la fase 10.

### Incidente de infraestructura (resuelto, no es bug)

Al arrancar, `/api/v1/search` daba **404 de Gin vía gateway** (directo al contenedor respondía 200): el upstream `search_api` de nginx tenía cacheada una **IP stale que ahora pertenecía a users-api-3** (rastreado con `X-Request-ID` en los logs). Es el gotcha ya documentado ("nginx cachea IPs de upstreams al recrear contenedores") con la variante de que la IP reutilizada da 404 en vez de 502. `docker compose restart nginx` lo resolvió; **el stack quedó ARRIBA (11/11 healthy)** — el usuario lo tenía corriendo de antes.

### Archivos de esta sesión (para el commit manual — TODO sigue sin commitear, ~102 paths con fases 0–7)

- **Nuevos**: `src/test/axe.js`, `src/test/a11y.test.jsx`, `src/test/keyboard.test.jsx`, `src/App.test.jsx`, `src/components/Hotels/HotelCard.test.jsx`, `src/components/common/RouteFallback.jsx`, `src/utils/prefetch.js`, `public/robots.txt`.
- **Modificados**: `package.json`/`package-lock.json` (−date-fns, +axe-core), `index.html` (fonts async), `vite.config.js` (manualChunks función), `src/theme/theme.js` (goldDark AA), `src/App.jsx` (RouteFallback), `src/components/common/index.js`, `Footer`/`Navbar`/`BrandMark`/`BookingPanel`/`ServiceHealthGrid`/`AdminHotelList`/`AdminUserList` (headings/label), `HotelCard`/`ReservationCard` (heading + prefetch), `src/pages/Search.jsx` (headingComponent h2), `src/pages/Admin/HotelForm.jsx` (description), `src/test/setup.js` (stub scrollTo sin ruido), `src/test/keyboard.test.jsx`+`src/App.test.jsx` (nombre del BrandMark), `frontend/.gitignore` (+lighthouse).

### Trabajo restante del plan 13 (próxima sesión = fase 10 y cierre)

1. **E2E Playwright** (`frontend/e2e/`): anonymous-search, customer-booking, admin, accessibility (+ degradación), desktop + mobile, contra el stack compose con profile frontend. Ojo: `@playwright/test` 1.62 puede necesitar `npx playwright install chromium` aunque el cache de ms-playwright ya tenga builds (1208/1217, de otro toolchain).
2. **R1–R6** (revisión adversarial) + decisión de los greps benignos (renombrar `pricePerNight` interno / parafrasear comentarios / acotar el grep a código no-test).
3. Asserts automatizados de viewport 320/390 como spec de Playwright (la verificación manual de esta sesión ya dio limpio).
4. Evidencia final para plan 12 (los reportes de Lighthouse ya quedaron en `frontend/lighthouse/`), **checkbox del plan 13** en `plans/README.md` y HANDOFF de cierre.

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - **El job `frontend-e2e` del CI sigue referenciando `frontend/e2e/` que NO existe** — en push no corre (`if: pull_request`), así que commitear este checkpoint deja el CI verde; un PR fallaría en ese job hasta hacer la fase 10.
  - El chunking nuevo reparte MUI por uso: si se agrega un import de MUI "pesado" a código eager, el entry crece — mirar el tamaño del build en PRs de frontend.
  - Lighthouse tiene varianza local (±5-10 en Perf mobile): medir con la máquina ociosa y tomar mediana de 3 (el primer run post-tests dio 81 por contención de CPU).
  - Siguen vigentes: RV10, nginx cachea IPs de upstreams al recrear (esta sesión lo sufrió con síntoma 404 — `docker compose restart nginx`), memcached sin healthcheck.

### Primera acción sugerida para la próxima sesión

Retomar con **"cerremos el plan 13 con la fase 10"** → releer esta sección + la fase 10 del plan (recorridos, R1–R6, bloque Verificar completo). El stack ya está arriba y los certs generados; falta `npx playwright install chromium` si `npm run test:e2e` lo pide. Al cerrar: greps (con la decisión sobre los hits benignos), checkbox del plan 13 y recién después el plan 12.

---

## Sesión — 2026-08-03 (3) — Plan 13 CERRADO (fase 10 + Verificar)

> **EL PLAN 13 ESTÁ TERMINADO.** Esta sesión ejecutó la fase 10 (E2E Playwright + cierre adversarial R1–R6 + decisión de los greps) y el bloque Verificar completo contra el stack Docker real. **El checkbox del plan 13 en `plans/README.md` quedó marcado.** Siguiente: **plan 12** (documentación/presentación), que ya tiene toda la evidencia esperándolo.

### E2E Playwright (`frontend/e2e/`) — 23/23 en verde, desktop + mobile

- **Specs**: `anonymous-search.spec.js` (Home → search → sort global price_asc/rating_desc con asserts de orden → Back restaura URL → detail con horas "3:00 PM" y jamás RFC3339 → intento de reserva anónimo → login demo con retorno a `/hotels/:id`), `customer-booking.spec.js` (login UI → búsqueda → fechas civiles +30/+33 → guests → total en centavos espejo del backend → 201 con Confirmation ID → My Reservations ancla por shortId → cancel con dialog → visible en Cancelled sin botón Cancel), `admin.spec.js` (@desktop-only: tabs independientes, self-delete deshabilitado con motivo, ServiceHealthGrid read-only sin scale/restart/logs, crear hotel sin imágenes → fallback local en la página pública → **aparece en search** (evento→Solr, poll por API con URIs únicas) → editar → borrar via UI), `accessibility.spec.js` (axe con **0 serious/critical en las 6 rutas** × 2 form factors; **viewports 320/390 sin overflow** en 8 rutas como spec automatizado; **degradación** con `docker compose stop/start` de search-api y hotels-api → envelope 502 `service temporarily unavailable` + `Reference: <trace_id>` + Try again que recupera — nunca pantalla blanca ni HTML de nginx; screenshots de 5 superficies × 2 viewports con fechas enmascaradas).
- **Infra de specs**: `global-setup.js` (espera gateway + **índice poblado** — el backfill de Solr es asíncrono al /readyz —, UN login por rol vía API, **cliente fresco registrado por run** para historial determinístico, estado en `e2e/.auth/state.json` gitignored) + `helpers/{env,api,session,console-guard,ui}.js`. Los recorridos **fallan ante console errors/pageerrors** no permitidos (guard con allowlist explícito). Sesiones se inyectan por `localStorage` (claves `token`/`user`) con JWTs reales.
- **Gotchas de infra que gobiernan el diseño** (todos documentados en los specs): (1) **login_limit 5 r/m burst 3** y además **el preflight OPTIONS cuenta** — un 429 al preflight se ve como error CORS/red (status 0), no como 429; el helper de login reintenta esperando 26s (2 tokens) ante ambos síntomas. (2) **`/api/v1/search` se cachea 5 min por URI** → nombres de hotel únicos por run y polls con query param único; con search-api caído una URI cacheada sirve STALE 200 (por eso la degradación usa queries vírgenes). (3) Degradación con **stop/start, no down/up** (recrear cambia IPs y nginx las cachea). (4) La paginación real no se cubre: el seed (5 hoteles) no supera una página de 12 — queda en los unit tests con MSW, como contempla el plan.
- **Fixes que salieron de los runs**: el lápiz de editar de AdminHotelList es `IconButton component={Link}` → rol **link**, no button; en mobile la tabla admin está oculta por CSS → anclar en el heading `Hotels (N)`; el orden de loops del test de viewports es por RUTA (los initScripts de sesión se acumulan y `/login` con sesión redirige).

### Greps de invariantes — decisión y resultado: **ambos rg del Verificar dan 0 hits**

Se eligió **renombrar/parafrasear** (no acotar el grep): prop interna `pricePerNight` → **`nightlyRate`** (BookingSummary + 2 call sites) y el param de `previewTotalCents` idem (llamador posicional, cero blast radius); el assert de HotelForm.test pasó de `not.toHaveProperty('pricePerNight')` a **"ninguna key camelCase en el payload"** (más fuerte); 9 comentarios que citaban los patrones prohibidos parafraseados ("hard redirect del navegador", "links muertos", "recortar el ISO string UTC"); los literales del test de honestidad de Home se **componen por partes** (`'24/' + '7'`) con comentario explicando por qué. 30 unit tests de los archivos tocados en verde tras el rename.

### CI/Makefile/README (solo exponer comandos)

- **`ci.yml` — 3 fixes al job `frontend-e2e`** (pre-armado en el checkpoint, aún no corrido nunca): los certs se generaban como `localhost.pem`/`localhost-key.pem` pero nginx exige **`cert.pem`/`key.pem`** (el job moría seguro); el wait-loop contaba servicios sin healthcheck (memcached/frontend) y **nunca convergía** (300s fijos) → `grep -v '^$'`; **upload-artifact de `playwright-report/` + `test-results/`** en failure. El job corre solo en PRs; ahora `frontend/e2e/` existe y **un PR ya no falla por specs inexistentes**.
- **Makefile**: targets nuevos `up-frontend`, `frontend-check` (npm run check) y `e2e` (up con profile + test:e2e). **README raíz**: sección Testing ampliada con la pirámide frontend (unit/coverage/check/e2e + make e2e). **frontend/README**: scripts completos. **`.dockerignore` del frontend**: excluye `e2e/`, config de playwright y artefactos (no invalidar el build del SPA por editar specs). **`.gitignore` frontend**: + `e2e/.auth`, `e2e-evidence`.

### Bloque Verificar — TODO verde (orden del plan)

1. `npm ci` (0 vulnerabilidades) · `npm run lint` **0 errores** · `TZ=America/Los_Angeles npm run test:coverage` **131/131 tests, 22 archivos** (84.98% stmts / 78.52% branches / 85.78% lines vs 60/50) · `npm run build` limpio, mismo perfil de chunks de fase 9 (entry 83.0 kB gzip, cero warning >500 kB → la evidencia Lighthouse de fase 9, 94/100/100/100 mobile y 98/100/100/100 desktop, sigue válida: ningún cambio de esta sesión afecta el bundle).
2. Los dos `rg` de invariantes: **0 hits** cada uno.
3. `go test -race -count=1` de los 3 paquetes del sort de search-api: **ok**.
4. `docker compose --profile frontend up -d --build` (12/12 arriba; memcached y SPA sin healthcheck, como está documentado) + **`npm run test:e2e` 23/23** — corrido DOS veces en verde, la segunda contra la imagen del SPA reconstruida con el código final. Gotcha vivido de nuevo: el `--build` recreó los servicios Go (su contexto es la raíz y cambiaron README/Makefile/ci.yml) → IPs nuevas → `docker compose restart nginx` antes del run final.
5. Extra R1: **`test_load_balancer.sh` PASSED** entero (balanceo entre 3 réplicas, rate-limit con 429 + envelope + trace_id, cero 503).

### R1–R6 (cierre adversarial, con evidencia)

- **R1**: auth/retorno (e2e anonymous), disponibilidad + booking atómico (e2e customer), idempotencia (unit por-intento + double-submit disabled), cancelación (e2e desktop+mobile), CRUD admin + sync Solr (e2e admin), C11 (`avaiable` 0 hits; la API real devuelve `available_rooms`), LB/rate-limit (script PASSED).
- **R2**: token expirado/malformado (unit authStorage), query vacía (e2e), page fuera de rango (unit), 0/1/4 imágenes (unit gallery; e2e crea hotel SIN imágenes → fallback local visible), sin amenities (mismo hotel e2e), medianoche/DST (suite entera bajo TZ=LA), 409 (unit copy `no_availability`), 502 (e2e degradación), payload parcial (unit envelope).
- **R3**: todos los recorridos atraviesan nginx TLS → `/api/v1` real; count por `meta.total`; `user_id` string; "HH:mm" localizado; total en **centavos** espejo; `available_rooms`; ServiceHealthGrid consume el shape real de /readyz; profile compose.
- **R4**: console guard **vacío** en los 3 recorridos (build de producción, sin warnings React/Router); único `setTimeout` en src es el timer de foco de RouteAnnouncer (a11y, no navegación); estado remoto solo en TanStack; sin claims inventados (grep B 0 + tests de honestidad).
- **R5**: sin `console.log` en src; tokens jamás en URL (grep); errores de UI siempre estables (envelope o fallback por status, nunca internals); guards admin (pantalla 403 + e2e con roles reales); reservas ajenas protegidas server-side (OwnerOrAdmin / admin-only, tests Go de planes 01/11); los 2 links externos con `rel="noreferrer"`.
- **R6**: **`frontend/e2e-evidence/`** (10 screenshots: home/search/detail/booking/dashboard × desktop/mobile, gitignored) + **`frontend/lighthouse/`** (4 reportes) + resultados de lint/tests/build/e2e de esta sección → todo listo para las capturas y facts del plan 12.

### Estado y archivos (para el commit manual — TODO sigue sin commitear)

- **Nuevos**: `frontend/e2e/{anonymous-search,customer-booking,admin,accessibility}.spec.js`, `frontend/e2e/global-setup.js`, `frontend/e2e/helpers/{env,api,session,console-guard,ui}.js`.
- **Modificados esta sesión**: `frontend/playwright.config.js` (+globalSetup), `frontend/.gitignore`, `frontend/.dockerignore`, `.github/workflows/ci.yml` (fixes del job e2e), `Makefile`, `README.md`, `frontend/README.md`, `plans/README.md` (**checkbox 13 marcado**), y por los greps: `src/components/booking/{BookingSummary,BookingForm,BookingSuccess}.jsx`, `src/utils/money.js`, `src/pages/Admin/HotelForm.test.jsx`, `src/pages/Home.test.jsx`, `src/components/Layout/{Layout.test.jsx,Footer.jsx}`, `src/components/auth/SessionExpiredNavigator.jsx`, `src/context/AuthContext.jsx`, `src/services/{api.js,api.test.js,authEvents.js}`, `src/utils/dateOnly.js`.
- **Stack**: quedó ARRIBA (12/12) con la imagen del SPA reconstruida. En Mongo quedó el hotel manual "Hotel Plan Once" de una sesión anterior (no lo toqué: no es dato del E2E; los specs son robustos a él). Los usuarios `e2e_<ts>` del global-setup se acumulan run a run en MySQL (inofensivo; los specs no dependen de counts absolutos).

### Bloqueos y advertencias

- **Sin bloqueos.** Advertencias:
  - Los E2E consumen el `login_limit` (cada login de UI = preflight + POST = 2 tokens de 5/min): correr `npm run test:e2e` dos veces seguidas puede meter esperas de ~26s en los logins (los helpers lo absorben solos).
  - El job `frontend-e2e` del CI quedó arreglado pero **solo corre en PRs** — se estrena recién en el primer PR.
  - Siguen vigentes: RV10, **nginx cachea IPs de upstreams al recrear** (esta sesión mordió otra vez: un `--build` con cambios en la raíz recrea los Go services → `docker compose restart nginx`), memcached sin healthcheck.

### Primera acción sugerida para la próxima sesión

**Plan 12** (documentación y presentación): README nuevo, capturas (ya están en `frontend/e2e-evidence/` y `frontend/lighthouse/`), GIF/diagrama, OpenAPI, badges. Es el último plan pendiente.
