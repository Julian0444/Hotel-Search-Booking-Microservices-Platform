# Fixes de la review externa (2026-07-11) — índice y triage

> Origen: review general de código (4 revisores paralelos + verificación manual) hecha el 2026-07-11
> sobre el estado post-planes 01–05. Cada hallazgo tiene un ID `RV*`.
> **Regla de triage:** si un plan pendiente (06–13) ya iba a reescribir ese código, el hallazgo
> se ejecuta DENTRO de ese plan (tabla de abajo) y NO tiene plan propio acá. Solo los
> **hallazgos huérfanos** (ningún plan los cubría) tienen plan `F*` en esta carpeta.

## Cómo usar

1. Igual que los planes principales: leé la última sección de [`../HANDOFF.md`](../HANDOFF.md) antes,
   validá los snippets contra el código actual, corré el bloque **Verificar** al terminar
   y ticá el checkbox acá.
2. Orden recomendado: **F4 primero** (es de minutos y deja el workflow del CI listo para su primer run verde),
   después **F1** (el más grande; toca `hotels-api` antes de que los planes 06/07 pasen por ahí),
   y F2/F3 en cualquier momento (independientes entre sí y de todo lo demás).

## Planes de esta carpeta

| ✔ | Plan | Alcance en una línea | Esfuerzo |
|---|------|----------------------|----------|
| [x] | [F1 — Caché de reservas de hotels-api](F1-hotels-cache-reservas.md) | RV1–RV5: listas agregadas envenenadas, data race sobre slices compartidos, cancel en miss, `GetAvailability` que traga errores, mocks alineados | M (~medio día) |
| [x] | [F2 — users-api: login y detalles](F2-users-api-login.md) | RV6–RV10: error de infra disfrazado de 401, timing oracle, `ORDER BY` en paginación, clamp de `BCRYPT_COST`, documentar tradeoff de caché stale en login | S (~1-2 h) |
| [x] | [F3 — Frontend: interceptor 401 y minLength](F3-frontend-login-interceptor.md) | RV11–RV12: el interceptor se come el error de login con hard-reload; minLength de password inconsistente | S (<1 h) |
| [x] | [F4 — CI: leg de platform-contracts](F4-ci-platform-contracts.md) | RV13: `cache-dependency-path` apunta a un `go.sum` que no existe | XS (minutos) |

## Hallazgos de los planes F*

| ID | Hallazgo (resumen) | Plan |
|----|--------------------|:----:|
| RV1 | `updateXxxList` crea listas agregadas parciales que los getters sirven como completas | F1 |
| RV2 | Data race: mutación in-place de slices compartidos guardados en ccache | F1 |
| RV3 | `Cache.CancelReservation` en miss de la key individual deja la copia `confirmed` en las listas | F1 |
| RV4 | `Cache.GetAvailability` mapea errores por-hotel a `false` con error nil (200 para input inválido) | F1 |
| RV5 | `MockCache` con semántica de DB idealizada: no puede detectar RV1/RV3 | F1 (parcial) |
| RV6 | `Login` colapsa cualquier error de infraestructura (no solo not-found) a 401 | F2 |
| RV7 | Timing oracle: username inexistente responde sin costo bcrypt → enumeración | F2 |
| RV8 | `GetAll` pagina sin `ORDER BY`: páginas no determinísticas | F2 |
| RV9 | `BCRYPT_COST` sin bounds: >31 rompe todos los registros, 25–31 es DoS | F2 |
| RV10 | Usuario borrado puede loguearse ≤30s desde otra réplica (L1 stale) y obtener JWT de 24h | F2 (**documentar**, no code-fix: inherente a L1-por-réplica + JWT stateless) |
| RV11 | Interceptor 401 hace hard-redirect también en el propio login: el error nunca se ve | F3 |
| RV12 | `Login.jsx` exige minLength 4; `Register.jsx` y el backend exigen 8 | F3 |
| RV13 | CI: `cache-dependency-path: platform-contracts/go.sum` no existe (módulo sin deps) | F4 |

## Hallazgos triageados a planes pendientes (ejecutarlos DENTRO de ese plan)

Al empezar cada plan de abajo, leer su fila acá y sumar el hallazgo a su alcance:

| ID | Hallazgo (resumen) | Va al plan |
|----|--------------------|:----------:|
| RV14 | hotels-api responde **404 ante cualquier error** en `GetHotelByID` (`hotels_controller.go:75`) y el cliente HTTP de search no tipifica el 404 → la lógica retry/DLQ del 06 no puede decidir "reintentar vs descartar". Tratarlo como **pre-requisito** del manual-ack | **06** |
| RV15 | `GET /search` no clampa `offset`/`limit` (negativos y sin tope); post-E2 un `rows` negativo será 500 y uno gigante un DoS. hotels-api ya clampa — copiar ese patrón | **06** |
| RV16 | El middleware RequestID de search-api setea el id en el context de **Gin** pero no en el `context.Context` del request (`requestid.go:29` — falta `c.Request = c.Request.WithContext(utils.WithRequestID(...))`); `RequestIDFromContext` devuelve `""` en todo el path HTTP | **06** (o al tocar ese archivo) |
| RV17 | `IsConnected` de search solo mira la **conexión**: si el *channel* AMQP muere, el consumer termina en silencio y `/readyz` sigue ok. La reconexión (E5) debe dejar el check reflejando "consumer vivo" | **06** |
| RV18 | `hotels_solr.go:184`: `Delete` reporta "failed to index hotel" (copy-paste) | **06** |
| RV19 | Validaciones de reserva devuelven **500 en vez de 400** (`hotels_service.go:258-269`: check-in pasado, checkout<=checkin post-normalización, num_guests) — el controller solo mapea `ErrNoAvailability`. Mismo patrón: reservar sobre hotel inexistente da 500 en vez de 404 | **07** (A1/A6: errores tipados + semántica HTTP) |
| RV20 | Respuestas de reserva serializan `check_in` como RFC3339-UTC y el frontend lo renderiza en hora local → **fechas corridas un día** en TZ al oeste de UTC (`helpers.js:28-34`, estados/canCancel un día antes). Decidir formato date-only en la respuesta o `slice(0,10)` en el frontend | **07** (contrato) |
| RV21 | Tipo de `check_in_time`/`check_out_time` mal modelado como `time.Time`: **el form admin de hoteles no puede crear ni editar** (manda "HH:mm" → 400 siempre; `HotelForm.jsx:69,136`) y `HotelDetail` muestra el RFC3339 crudo. Decidir el tipo en el contrato y adaptar UI | **07** (decisión de contrato) + UI en **13** |
| RV22 | Paginación del Search del frontend rota por construcción (`Search.jsx:61` calcula totalPages sobre la página actual) + `X-Total-Count` no está en `Access-Control-Expose-Headers` | **07** (envelope con total) + UI en **13** |
| RV23 | `releaseNights` es best-effort y la idempotencia del segundo DELETE **impide reintentar la liberación** → noches bloqueadas para siempre ante un fallo post-cancel (`hotels_mongo.go:446-448`); mismo leak en claim ambiguo por timeout | **08** (retries/deadlines; C12 adyacente) |
| RV24 | `POST /hotels/availability` lanza una goroutine + 2 queries por cada ID del body sin límite (`hotels_mongo.go:541-553`), y un solo ID malo tumba el mapa entero con 500 | **08** (R4 — es literalmente el bulkhead del fan-out) |
| RV25 | El retry-loop del publisher Rabbit re-publica sobre el canal muerto: `ensureConnection` interno solo corre `if channel == nil` (`queue_rabbit.go:236`) — debería ser `!IsConnected()` | **08** (C14) |
| RV26 | Puerto de monitoreo `8090:8090` publicado en 0.0.0.0 con `stub_status allow all` → `127.0.0.1:8090:8090` | **10** |
| RV27 | CORS **duplicado** gateway+servicio: nginx agrega ACAO por location y users/hotels-api también → con el frontend fuera del proxy de Vite el browser bloquea por header múltiple. Además `CORS_ALLOWED_ORIGINS="*"` refleja cualquier Origin con credentials | **11** (C1/CQ4 — sumar este modo de falla) |
| RV28 | El `Delete` de Memcached depende de un `Get(idKey)` para descubrir la key por username: si la key por ID fue evictada (LRU), la de username queda huérfana **para siempre** → el TTL de C7 es la mitigación; considerar borrar por username también | **11** (C7) |
| RV29 | El hash bcrypt completo viaja serializado a Memcached (DAO sin tags) — documentar el tradeoff o cachear datos públicos | **11** (o docs en 12) |
| RV30 | Menores: `fmt.Printf` suelto en `hotels_mongo.go:527`; `%v` en vez de `%w` en 5 wraps del service; `AssertNotCalled` con aridad incompleta (vacuos) en tests de users; `TestRabbitQueuePublishWithoutChannel` mete ~15s de backoff en la suite; dead code en search (`queue_mock.go`, `search_mock.go`); `Index`/`Update` de Solr idénticos | **11** (limpieza) / **06** los de search |
| RV31 | UX/frontend menores: `Promise.all` del Dashboard acopla users a search; date-picker con mínimos en UTC; sesión zombie sin validar `exp`; admin puede auto-borrarse; JSDoc de `RegisterRequest` documenta `tipo` | **13** |

## Fortalezas confirmadas por la review (no tocar)

`claimNight` + suite de solapamiento compartida, middleware JWT y sus tests negativos, seed
race-safe del admin, platform-contracts con type-alias, health checks paralelos, compose
encadenado, migraciones con receta de verificación.
