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
