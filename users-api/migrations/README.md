# Migraciones de users-api (golang-migrate)

Migraciones SQL versionadas para MySQL. El formato es el de
[golang-migrate](https://github.com/golang-migrate/migrate): `NNNN_nombre.up.sql` / `.down.sql`.

## Cómo se ejecutan

- **Docker Compose (flujo normal):** el service one-shot `migrate` (imagen `migrate/migrate`)
  corre `up` contra el service `mysql` apenas está healthy, y las 3 réplicas de users-api
  arrancan recién cuando terminó (`depends_on: service_completed_successfully`).
  Corre en **una sola** instancia a propósito: 3 réplicas migrando en paralelo pelean por el
  advisory lock de golang-migrate y pueden dejar el schema "dirty".
- **A mano (CLI o docker):**

  ```bash
  # con el CLI instalado (brew install golang-migrate) — puerto host 3307, DB users-api
  migrate -path users-api/migrations -database "mysql://root:$MYSQL_ROOT_PASSWORD@tcp(localhost:3307)/users-api" up

  # sin CLI, con la misma imagen del compose
  docker run --rm -v ./users-api/migrations:/migrations --network hotel-search-booking-microservices-platform_app-network \
    migrate/migrate -path /migrations -database "mysql://root:$MYSQL_ROOT_PASSWORD@tcp(mysql:3306)/users-api" up
  ```

  Ojo: si `MYSQL_ROOT_PASSWORD` tiene caracteres reservados de URL (`@`, `/`, `:`, `#`),
  hay que URL-encodearlos en el DSN.

- **`make seed`** re-ejecuta el one-shot (idempotente: `no change` si ya está al día).

## Relación con GORM AutoMigrate

`AutoMigrate` quedó gateado por la env `AUTO_MIGRATE` (default `true` para el dev pelado con
`go run`; el compose la setea en `false` porque ahí migra el one-shot). La 0001 es copia exacta
del schema que generaba GORM (verificado con `SHOW CREATE TABLE`), con `IF NOT EXISTS` para que
un volumen creado antes de las migraciones converja sin dirty flag.

## Seed de usuarios demo (0002)

| Usuario | Password | Rol | Fuente |
|---------|----------|-----|--------|
| `demo` | `DemoCliente123` | cliente | migración `0002` (idempotente por `ON DUPLICATE KEY`) |
| `${ADMIN_USERNAME}` | `${ADMIN_PASSWORD}` | administrador | seed env-driven al arranque de users-api (plan 01) — **única fuente del admin** |

El hash de la 0002 se genera con la lib bcrypt de la app (prefijo `$2a$`); `htpasswd` genera
`$2y$` y la lib lo rechaza. Para regenerarlo:

```bash
cd users-api && go run <<'EOF' /dev/stdin 'NuevaPassword'
package main
import ("fmt"; "os"; "golang.org/x/crypto/bcrypt")
func main() { h, _ := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost); fmt.Println(string(h)) }
EOF
```

## Re-seed desde cero

```bash
docker compose down -v && docker compose up -d --build
```

Esto también re-siembra los hoteles demo de Mongo (`hotels-api/seed/mongo-init.js`, que solo
corre con volumen nuevo).
