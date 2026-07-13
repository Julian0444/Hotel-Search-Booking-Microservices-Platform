# Plan 02 — CI/CD, quality gates y módulo de contratos

> **Alcance:** I4, I6, I9, SD1, CQ1, CQ2, CQ3, T1, T2, T3, T4, T6
> **Base:** playbook 7.7 de `plantofinish.md` (el único "implementable-as-written") + ítems afines sin playbook (I4, I6, I9, T3, T6)
> **Prerequisitos:** ninguno duro. Hacerlo **temprano**: da `go test -race`/lint como red de seguridad antes de los refactors grandes, y el módulo de contratos que el plan 06 aprovecha.
> **Esfuerzo:** ~1 día

## Contexto

No hay `.github/workflows` (I4), ni linter (CQ1: `gofmt -l` lista 7 archivos sucios), ni `go.work` (CQ3), y el struct `Hotel` está duplicado 4× con el contrato de evento `HotelNew` duplicado en ambos extremos del pipeline (CQ2): renombrar un tag rompe RabbitMQ sin error de compilación. Además `govulncheck` reporta **12 CVEs en search-api** y `npm audit` **16 en el frontend** (SD1).

## Correcciones de la Sección 7.0 que aplican

- **CQ1**: golangci-lint es **v2**: `gofmt`/`goimports` son *formatters*, no linters; el schema v1 plano no carga. Usar `version: "2"`, linters bajo `linters.enable`, formatters bajo `formatters.enable`.
- **SD1**: search-api tiene 12 CVEs hoy → el gate se pondría rojo inmediatamente: `continue-on-error: true` en la leg de search-api **hasta** que el plan 06 bumpee gin/x/net. En modo workspace `go install ...@latest` está prohibido → instalar govulncheck con `GOWORK=off`.
- **CQ2**: mantener el typo `AvaiableRooms`/`hotel_id` en el módulo compartido — arreglarlo es C11 (plan 11); hacerlo acá rompe RabbitMQ.

## Pasos

### 1. `go.work` raíz + unificación de Go y module path (CQ3, I6)

- Unificar la versión de Go: search-api está en `1.22`; users/hotels en `1.23.0` + `toolchain 1.24.11`. Llevar los 3 `go.mod` a la misma versión.
- Renombrar el módulo de search-api del path pelado `search-api` al canónico `github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/search-api` (ajustar imports internos con `gofmt -w`/sed).
- Crear el workspace:

```
go 1.23

use (
    ./hotels-api
    ./users-api
    ./search-api
    ./platform-contracts
)
```

### 2. Módulo compartido `platform-contracts` (CQ2)

- Nuevo módulo `platform-contracts/` con los tipos `Hotel` y `HotelNew` como única fuente de verdad (copiar los actuales **con el typo intacto** — ver corrección 7.0).
- En hotels-api y search-api: importar vía **type-alias bridge** para cero cambios en call-sites:

```go
// hotels-api/internal/domain/hotels/hotels_domain.go
import contracts "github.com/Julian0444/.../platform-contracts"

type Hotel = contracts.Hotel
type HotelNew = contracts.HotelNew
```

- `require` + `replace ../platform-contracts` local en cada `go.mod` (el replace se resuelve solo con go.work en dev; el replace explícito cubre builds fuera del workspace, p.ej. Docker).

### 3. Linter + formato (CQ1)

`.golangci.yml` raíz, **schema v2**:

```yaml
version: "2"
linters:
  enable: [govet, staticcheck, errcheck, ineffassign, unused]
formatters:
  enable: [gofmt, goimports]
```

Correr `gofmt -w` sobre los 7 archivos sucios (`gofmt -l ./users-api ./hotels-api ./search-api` los lista) y dejar `golangci-lint run ./...` verde en los 3 módulos.

### 4. GitHub Actions CI (I4, T4, SD1)

`.github/workflows/ci.yml`:

```yaml
name: ci
on: [push, pull_request]
jobs:
  go:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        module: [users-api, hotels-api, search-api]
    defaults: { run: { working-directory: ${{ matrix.module }} } }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: ${{ matrix.module }}/go.mod }
      - run: test -z "$(gofmt -l .)"
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v8   # CLI v2, lee el .golangci.yml raíz
        with: { working-directory: ${{ matrix.module }} }
      - run: go test -race -coverprofile=coverage.out ./...
      - name: govulncheck
        continue-on-error: ${{ matrix.module == 'search-api' }}  # hasta el bump del plan 06
        run: |
          GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
  frontend:
    runs-on: ubuntu-latest
    defaults: { run: { working-directory: frontend } }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: npm, cache-dependency-path: frontend/package-lock.json }
      - run: npm ci
      - run: npm run build
      - run: npm audit --audit-level=high
```

- Correr `npm audit fix` en el frontend (16 hallazgos, 7 high — path-traversal en rollup/vite) antes de activar el gate.
- `.github/dependabot.yml` con ecosistemas `gomod` (×3 dirs), `npm` y `github-actions`.
- El badge en el README lo agrega el plan 12 (P9).

### 5. Contract test productor↔consumidor (T2)

Golden JSON del evento (p.ej. `platform-contracts/testdata/hotel_new.golden.json`):

```json
{"operation":"CREATE","hotel_id":"abc-123"}
```

Test que (a) serializa el `HotelNew` del productor y compara byte-a-byte contra el golden, y (b) deserializa el golden con el tipo del consumidor y afirma los campos. Si el wire format deriva, el test falla en CI. (Cuando el plan 11 haga el rename C11, actualizar este golden será parte de ese cambio atómico.)

### 6. Skeleton de tests de integración (T1)

Suite `//go:build integration` con testcontainers-go en hotels-api, ejercitando el path real `Create → CreateReservation → IsHotelAvailable` contra un Mongo real (el plan 04 la va a extender con los casos de fechas). Job de CI aparte (`go test -tags=integration`), permitido en `pull_request` solamente o manual, para no encarecer cada push.

### 7. Endurecer `test_load_balancer.sh` (T3)

Evidencia: `test_load_balancer.sh:255-300` — sin `set -e` ni `exit 1`, siempre sale 0.

- `set -euo pipefail` al inicio; acumular fallos en un contador y `exit 1` si hubo alguno; cada check imprime PASS/FAIL.
- Así puede correr en CI como smoke test opcional (job manual con `docker compose up`).

### 8. Fixtures y benchmarks (T6)

- `testdata/` por módulo con documentos JSON reutilizables (p.ej. el doc Solr de un hotel, respuesta de hotels-api) en lugar de datos inline.
- 1–2 `Benchmark` en hot paths: cache-aside de users-api y serialización de resultados de search.

### 9. Higiene de repo (I9)

- Completar `.gitignore` raíz (el plan 01 lo creó para `.env`): binarios, `coverage.out`, `dist/`, `node_modules/`, `.DS_Store`.
- `LICENSE` MIT (año actual, tu nombre).
- `Makefile` raíz:

```make
.PHONY: build test lint up down seed
build: ; go build ./... # via go.work
test:  ; go test -race ./...
lint:  ; golangci-lint run ./...
up:    ; docker compose up -d --build
down:  ; docker compose down -v
```

(El target `seed` se completa en el plan 03.)

## Verificar

```bash
go build ./...                      # una sola pasada gracias a go.work
golangci-lint run ./...             # verde
go test -race ./...                 # verde en los 3 módulos
grep -rn "AvaiableRooms" platform-contracts/   # el typo sigue (a propósito, hasta C11)

# El contract test protege el wire format: renombrar temporalmente un tag JSON
# de HotelNew en platform-contracts → go test falla → revertir.

# CI: el push de una branch lo hace el USUARIO manualmente (la sesión no ejecuta git);
# una vez pusheada, verificar los 4 jobs verdes (search-api govulncheck en amarillo permitido).

bash test_load_balancer.sh; echo "exit=$?"   # con el stack sano → 0; matando un servicio → ≠0
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
