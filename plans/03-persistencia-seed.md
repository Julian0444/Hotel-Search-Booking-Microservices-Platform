# Plan 03 — Persistencia + seed automatizado

> **Alcance:** DB1, DB2, DB3, DB4, DB5, R1, P7
> **Base:** playbook 7.3 de `plantofinish.md` (R1 fusionado con DB4 según la corrección 7.0)
> **Prerequisitos:** plan 02 recomendado (CI con `-race` como red de seguridad para el refactor de `context`)
> **Esfuerzo:** ~1 día (L)

## Contexto

La dimensión que un entrevistador backend pica más fuerte y una de las más flojas: esquema con `AutoMigrate` sin historial (DB1), **cero índices** (DB2), sin pool/timeouts de driver (DB3), `context` nunca llega a la DB en users-api (DB4/R1 — si MySQL se cuelga, el login cuelga a todas las réplicas para siempre), listas sin paginación en DB (DB5), y el quickstart deja una app vacía sin admin (P7).

## Correcciones de la Sección 7.0 que aplican

- **P7**: el seed de usuarios **no puede** ir en un initdb de Mongo (los usuarios viven en MySQL) ni en `/docker-entrypoint-initdb.d` de MySQL (correría antes de que exista la tabla). Partir en dos: (a) Mongo initdb siembra hoteles + índices; (b) admin+cliente demo vía **migración golang-migrate 0002**, con hash bcrypt generado con la lib Go de la app (`$2a$` — htpasswd genera `$2y$` que la lib rechaza).
- **DB4/R1**: no es "agregar ctx a unas líneas" — es un refactor de interfaz completo: interfaz `Repository`, los 4 implementadores (mysql/cache/memcached/mock), service + helpers de caché, interfaz `Service`, handlers y ambos archivos de test. ccache/memcached no tienen API con ctx → aceptan-e-ignoran (documentarlo para que no parezca bug).

## Pasos

### 1. Migraciones versionadas (DB1)

- `users-api/migrations/0001_create_users.up.sql` / `.down.sql` que matchee **exacto** el schema que crea GORM hoy: tabla `users`, `tipo ENUM('cliente','administrador') DEFAULT 'cliente'`, `UNIQUE (username)`.
- Ejecución: por **CLI en CI o contenedor one-shot** en el compose (`migrate/migrate` con `depends_on: mysql: service_healthy`) — **no** desde las 3 réplicas (contención del advisory lock → schema "dirty"). Si preferís programático: `go:embed` de las SQL (el Dockerfile multi-stage solo copia el binario) y gate `RUN_MIGRATIONS=true` en **una sola** instancia.
- Gatear `AutoMigrate` con env `AUTO_MIGRATE` (default true para dev pelado, `false` en compose donde migra el one-shot). Documentar el flujo en un `migrations/README.md` corto.

### 2. Índices (DB2)

- hotels-api: función `EnsureIndexes` idempotente llamada en el startup (`NewMongo`): compuesto `{hotel_id: 1, check_in: 1, check_out: 1}` + `{user_id: 1}` en la colección de reservas.
  *Gotcha:* los nombres de campo deben matchear los **bson tags** reales (`check_in`, no `check_in_time` — verificar en `hotels_dao.go`).
- MySQL: el unique de `username` ya queda como índice vía la migración 0001.
- (El colapso del loop-por-noche de `IsHotelAvailable` en una sola aggregation lo resuelve el plan 04 con el inventario.)

### 3. Pool de conexiones + ping (DB3)

```go
// users-api
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(25); sqlDB.SetMaxIdleConns(25); sqlDB.SetConnMaxLifetime(5 * time.Minute)
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
if err := sqlDB.PingContext(ctx); err != nil { log.Fatalf("mysql unreachable: %v", err) }

// hotels-api (hotels_mongo.go:45-56)
opts := options.Client().ApplyURI(uri).SetMaxPoolSize(50).SetServerSelectionTimeout(5 * time.Second)
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
client, err := mongo.Connect(ctx, opts)
// + client.Ping(ctx, nil) fail-fast
```

### 4. `context` end-to-end en users-api (DB4 + R1)

Orden de edición (cada paso deja de compilar hasta el siguiente — hacerlos seguidos):

1. Interfaz `Repository`: todos los métodos reciben `ctx context.Context` primero.
2. `users_mysql.go`: `r.db.WithContext(ctx).…` en los 6 métodos (`:61,69,80,91,98,105`).
3. `users_cache.go` / `users_memcached.go`: firma `(_ context.Context, ...)` — **ignoran ctx a propósito** (ccache/memcached no lo soportan); dejar un comentario de una línea diciéndolo.
4. Mock: `m.Called(ctx, ...)`.
5. Interfaz `Service` + implementación + helpers de caché: enhebrar ctx.
6. Controllers: pasar `c.Request.Context()`.
7. Tests (ambos archivos): agregar `mock.Anything` como primer arg en cada `.On(...)`.

**R1 además:**
- DSN de MySQL (`users_mysql.go:40`): agregar `timeout=5s&readTimeout=5s&writeTimeout=5s`.
- Deadline por request: middleware que envuelve `c.Request.Context()` con `context.WithTimeout(…, 5*time.Second)`, y mapear `context.DeadlineExceeded` → **503** (fallar rápido en vez de colgar el login).

### 5. Paginación a nivel DB (DB5)

- `users_mysql.go`: `GetAll(ctx, limit, offset)` con `db.Limit(limit).Offset(offset)` + `CountAll(ctx)`.
- Defaults y clamp en el controller (`?limit=20&offset=0`, max 100).
- Devolver el total en el header **`X-Total-Count`** — no un envelope `{results,total}`, que rompería el admin del frontend que espera un array. (El envelope formal llega con el plan 07/A5 bajo `/api/v1`.)
- Listas de reservas de hotels-api: mismo patrón `options.Find().SetLimit().SetSkip()` (`hotels_mongo.go:271/287/303`).

### 6. Seed: hoteles + usuarios demo (P7)

a) **Hoteles**: `hotels-api/seed/mongo-init.js` montado en `/docker-entrypoint-initdb.d/` del contenedor de Mongo + `MONGO_INITDB_DATABASE: hotels-api` en compose:

```js
db = db.getSiblingDB('hotels-api');
if (db.hotels.countDocuments() === 0) {
  db.hotels.insertMany([ /* 4-6 hoteles demo con fotos, precio, AvaiableRooms */ ]);
}
// índices de reservas también acá (refuerza DB2 en arranque limpio)
```

b) **Usuarios demo**: migración `0002_seed_demo_users.up.sql` (INSERT del admin + un cliente, `ON DUPLICATE KEY UPDATE username=username` para idempotencia). El hash bcrypt se genera con la lib de la app:

```bash
cd users-api && cat > /tmp/hashgen.go <<'EOF'
package main
import ("fmt"; "os"; "golang.org/x/crypto/bcrypt")
func main() { h,_ := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost); fmt.Println(string(h)) }
EOF
go run /tmp/hashgen.go 'DemoAdmin123'   # → $2a$... (pegarlo en la 0002)
```

- Coordinar con el seed env-driven del plan 01: si ya existe el seed por env, la migración 0002 puede sembrar **solo el cliente demo** y dejar el admin al seed — elegir **una** fuente para el admin y documentarla.
- Documentar `docker compose down -v` para re-seed, y completar el target `make seed` del Makefile (plan 02).

## Verificar

```bash
# Migraciones suben y bajan limpio
migrate -path users-api/migrations -database "mysql://root:$MYSQL_ROOT_PASSWORD@tcp(localhost:3306)/users" up
migrate -path users-api/migrations -database "mysql://..." down 1 && migrate ... up

# Arranque limpio siembra todo
docker compose down -v && docker compose up -d --build
curl -s http://localhost/search?q=\*&offset=0&limit=10   # hoteles demo presentes
curl -s -X POST http://localhost/login -d '{"username":"demo-admin","password":"DemoAdmin123"}' # 200

# Índices creados
docker compose exec mongo mongosh hotels-api --eval 'db.reservations.getIndexes()'

# Paginación
curl -si -H "Authorization: Bearer $ATOKEN" 'http://localhost/users?limit=2&offset=0' | grep -i x-total-count

# ctx cancela: bajar MySQL y pegarle al login → 503 en ~5s, no cuelgue infinito
docker compose stop mysql && time curl -i -X POST http://localhost/login -d '{...}'

cd users-api && go test -race ./...
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
