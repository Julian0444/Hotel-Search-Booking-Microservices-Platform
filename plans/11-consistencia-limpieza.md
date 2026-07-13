# Plan 11 — Consistencia y limpieza de código (incluye el rename coordinado C11)

> **Alcance:** C1, C2, C3, C5, C6, C7, C9, C11, C13, CQ4, I8
> **Base:** sin playbook — redactado desde los IDs de las Secciones 2 y 6
> **Prerequisitos:** **plan 02 obligatorio para C11** (el módulo de contratos mantiene el typo hasta acá y el golden test protege el cambio) y **plan 07 recomendado** (`/api/v1` ya en su lugar: el rename es el primer breaking change que la estrategia de versionado absorbe). CQ4 usa el módulo compartido del 02.
> **Esfuerzo:** ~1 día

## Contexto

Bolsa de correcciones chicas de consistencia — cada una es una señal barata de repo pulido — más **un ítem grande y peligroso**: el rename `AvaiableRooms` (C11), que atraviesa el contrato JSON/BSON, Solr, el frontend y el golden contract test, y por eso tiene acá su procedimiento de cambio coordinado atómico.

## Pasos

### 1. CORS válido y convergido (C1, CQ4-parte)

- `hotels-api/cmd/main.go:60-67` y `search-api/.../middlewares.go:9-10` usan `AllowOrigins:["*"]` **con** `AllowCredentials:true` — combinación que el browser rechaza. La auth es por header Bearer (no cookies) → configuración correcta: **`*` + `AllowCredentials:false`**, o allowlist + true.
- Unificar con el patrón `CORS_ALLOWED_ORIGINS` por env que ya usa users-api, y converger las 3 implementaciones divergentes en **una** (en `platform-contracts` o un paquete compartido — parte de CQ4).

### 2. Layering y paquetes consistentes (CQ4)

- Elegir una convención y aplicarla: `internal/services/<dominio>` (como users/search) y nombre de paquete = nombre de dir (hoy `package middleware` vive en `internal/middlewares/` en hotels-api — `auth.go:2`).
- Es un refactor mecánico de mover archivos + ajustar imports; `go build ./...` y el CI del plan 02 lo validan.

### 3. Panel "microservices admin" (C2)

`microservices_controller.go:43-274`: uptimes hardcodeados, health por hash, logs mock, scale/restart que no hacen nada. **Decisión recomendada (opción a del plan): hacerlo real y read-only**:

- Health real: `GET /readyz` (plan 05) a cada servicio desde el backend; devolver estado + latencia reales.
- **Quitar** scale/restart/logs mock (acciones de escritura fuera).
- Ajustar la pestaña del frontend admin al nuevo shape (solo status).
- Alternativa mínima si no hay tiempo: rotular la respuesta y el README como "demo/mock" — pero la opción (a) convierte una vergüenza en una feature.

### 4. Endpoint público con PII (C3)

`GET /hotels/:id/reservations` (`hotels-api/cmd/main.go:71`, `hotels_controller.go:237-252`) devuelve `user_id` de todos. Moverlo detrás de `Authenticate() + AdminOnly()`. Actualizar la tabla de endpoints del README (el plan 12 hace la pasada completa).

### 5. Respetar `PORT` (C5)

`hotels-api/cmd/main.go:108`: `router.Run(":8081")` → `router.Run(":" + config.Port)` (o el `http.Server` del plan 08 si ya corrió — verificar que use `config.Port`). Revisar que compose/k8s fijen `PORT=8081` explícito (el plan 09 fijó `containerPort: 8081` — mantener coherencia).

### 6. Errores tipados en users-api (C6)

`users_controller.go:89-97` decide 400/409/500 con `strings.Contains(err.Error(), "Duplicate")`:

- Sentinels en el dominio: `ErrUsernameTaken`, `ErrValidation`, `ErrUserNotFound`.
- En el repo MySQL, detectar el duplicado de forma tipada:

```go
var mysqlErr *mysql.MySQLError
if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { return domain.ErrUsernameTaken }
```

- Controller: `errors.Is` → status; el mensaje al cliente sale del envelope del plan 07, nunca del texto del driver.

### 7. TTL en memcached (C7)

`users_memcached.go:92,98,128-140`: los items se escriben sin `Expiration` → viven para siempre (una cuenta borrada puede seguir logueando desde L2). Setear `Expiration` explícito (ej. 300s) en todos los `memcache.Item`.

### 8. DELETE de usuario inexistente → 404 (C9)

`users_mysql.go:105-110`: chequear `result.RowsAffected == 0` → `ErrUserNotFound` → 404 (consistente con `GetByID`).

### 9. Mocks fuera del binario (C13)

Aplica a **users-api y search-api** (no solo users): `users_mock.go`, `tokenizers_mock.go` y los `*_mock.go` de search-api importan testify en paquetes de producción. Encontrarlos todos con `grep -rln testify --include='*.go' users-api hotels-api search-api | grep -v _test`. Renombrar a `*_test.go` (si algún test de otro paquete los importa, moverlos a un subpaquete `testonly/` o duplicar en `_test.go` del consumidor). Verificar con `go list -deps ./<módulo>/cmd/... | grep testify` → vacío en los binarios.

### 10. C11 — Rename `AvaiableRooms` → `AvailableRooms`: **cambio coordinado atómico**

**El ítem más peligroso del plan.** El typo vive en el contrato JSON/BSON completo y está propagado a Solr, frontend y golden test. Hacerlo **como un único cambio atómico** (el usuario lo versiona luego como una sola unidad — la sesión no ejecuta git), con el stack corriendo para verificar al final. Puntos de contacto — la lista completa:

1. **`platform-contracts`** (plan 02 lo dejó con el typo a propósito): `AvaiableRooms` → `AvailableRooms` en el struct `Hotel` **y su tag** (verificar el literal exacto del tag en el código — json y bson pueden diferir). Los type-alias de hotels-api y search-api arrastran el cambio automáticamente.
2. **hotels-api**: `grep -rn "Avaiable" hotels-api/` — domain (`hotels_domain.go:17`), dao (`hotels_dao.go:17`), service, controller, caché y cualquier `bson.M` literal que use el nombre del campo (p.ej. proyecciones o updates).
3. **Datos existentes en Mongo**: migración de rename (usar el literal del tag BSON viejo, verificado en el paso 2):

```js
// hotels-api/seed/rename-available-rooms.js  (mongosh, una vez)
db = db.getSiblingDB('hotels-api');
db.hotels.updateMany({}, { $rename: { "<tag_bson_viejo>": "available_rooms" } });
```

   Y actualizar `seed/mongo-init.js` (plan 03) al nombre nuevo.
4. **search-api / Solr**: `grep -rn "Avaiable" search-api/` (espejo del grep de hotels-api del punto 2) — el mapeo del documento en `hotels_solr.go` + el campo en `schema.xml`. Tras el deploy: **reindexar** (`POST /reindex` del plan 06) o `docker compose down -v` para recrear el core.
5. **Frontend**: `grep -rn "avaiable\|Avaiable" frontend/src/` — service layer, HotelDetail, formularios del admin.
6. **Golden contract test (plan 02/T2)**: si el golden incluye el payload del hotel (fixture de `GET /hotels/:id` o del doc Solr en `testdata/`), actualizarlo **deliberadamente como parte de este mismo cambio** — es el único momento legítimo en que los goldens cambian; el test en rojo antes de actualizar el golden confirma que protege el wire format.
7. **Docs/colecciones**: Bruno, `users.md`/`README` si muestran ejemplos de payload (la pasada completa es el plan 12; acá solo que no quede el typo en ejemplos ejecutables).
8. **Estrategia de rotura**: con `/api/v1` (plan 07) el rename queda dentro de la misma versión — aceptable porque no hay consumidores externos; documentar la decisión (el usuario la refleja al versionar). Si el plan 07 no corrió aún, NO hacer C11 todavía.

Verificación específica C11 (antes de dar por cerrado el cambio):

```bash
grep -rni "avaiable" --exclude-dir=node_modules . | grep -v plantofinish | grep -v plans/   # → 0 hits
docker compose down -v && docker compose up -d --build
# flujo completo: crear hotel con available_rooms → aparece en /search con el campo → reservar → panel admin lo muestra
go test ./... && cd frontend && npm run build
```

### 11. Perfil del frontend en compose (I8)

`docker-compose.yml:281-294`: descomentar el bloque del frontend bajo un profile:

```yaml
  frontend:
    profiles: ["frontend"]
    ...
```

→ `docker compose --profile frontend up` opcional. Reconciliar el conteo de servicios donde se mencione (README dice "10", hay 11 — el número final lo fija el plan 12/P6).

## Verificar

```bash
go build ./... && go test -race ./...        # via go.work
golangci-lint run ./...
grep -rni "avaiable" --exclude-dir=node_modules . | grep -vE "plantofinish|plans/"   # 0
go list -deps ./users-api/cmd/... ./search-api/cmd/... | grep -c testify    # 0 (C13, ambos binarios)

# C1: preflight CORS desde el browser (o curl -H 'Origin: http://localhost:5173' -X OPTIONS ...) sin el combo inválido
# C2: la pestaña admin muestra health real; matar search-api → aparece "down" de verdad
# C3: curl -i localhost/api/v1/hotels/$HID/reservations → 401 sin token
# C7: borrar un usuario y probar login inmediato → 401 (antes: podía seguir desde L2)
# C9: curl -i -X DELETE .../users/99999 (admin) → 404
# I8: docker compose --profile frontend up -d → frontend arriba; sin profile → no
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
