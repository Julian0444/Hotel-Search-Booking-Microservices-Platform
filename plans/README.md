# Plans — índice, orden y trazabilidad

> Fuente de verdad de **qué ID vive en qué plan**. La referencia maestra de contexto es `../plantofinish.md` (intacta).
> Cada plan es autocontenido: se ejecuta en una sesión nueva sin leer nada más.

## Cómo usar

1. Abrí una sesión nueva y decí "empecemos con el NN". **Antes, leé la última sección de [`HANDOFF.md`](HANDOFF.md)** — ahí está el estado real (qué se hizo, qué está sin commitear, advertencias). Al terminar la sesión, agregale una sección nueva fechada (es un log acumulativo).
2. Lo primero del archivo es su línea **Alcance:** — verificá de un vistazo qué IDs cubre y cruzá contra la tabla de abajo (ningún ID aparece en dos planes).
3. **Antes de implementar, validá los snippets contra el código actual** (nombres de campos de config, nombres de servicios del compose, firmas): los planes fueron verificados contra el código de 2026-06-30, pero los planes anteriores pueden haber movido cosas.
4. Al terminar un plan: corré su bloque **Verificar**. El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git (nada de add/commit/push/merge — git solo lectura).
5. Respetá el orden salvo que el plan diga que es independiente — las dependencias están pensadas para no rehacer trabajo.

## Orden de ejecución y estado

| ✔ | Plan | Alcance en una línea | Depende de |
|---|------|----------------------|------------|
| [x] | [01 — Seguridad: auth + secretos](01-seguridad-auth.md) | Middleware JWT en users-api, registro sin escalada, `JWT_SECRET` obligatorio, secretos a `.env`, validación de entrada + password policy, claims `iss/aud`, tests negativos de auth | — |
| [x] | [02 — CI/CD, tooling y módulo de contratos](02-ci-tooling-contratos.md) | `go.work`, `platform-contracts`, golangci-lint v2, GitHub Actions (vet/test `-race`/govulncheck/npm audit), contract test golden, testcontainers skeleton, script e2e endurecido, Makefile/LICENSE/.gitignore | — (hacerlo temprano: red de seguridad) |
| [x] | [03 — Persistencia + seed](03-persistencia-seed.md) | Migraciones versionadas, índices, pool/timeouts de driver, refactor `context` en users-api (incluye R1), paginación en DB, seed de hoteles + usuarios demo | 02 (recomendado) |
| [x] | [04 — Dominio: no-overbooking + reserva rica](04-dominio-reservas.md) | Inventario atómico por hotel-noche, `Reservation` con estado/cantidad/dinero, evento `ReservationNew`, caché best-effort en escrituras y lecturas, fechas canónicas, tests de repositorio | 03 |
| [x] | [05 — Observabilidad](05-observabilidad.md) | `slog` JSON, request-ID end-to-end, `/livez` + `/readyz` reales, Gin release mode, healthchecks del compose | 03 (recomendado) |
| [x] | [06 — Endurecer search-api](06-search-api.md) | Manual ack + DLQ, query Solr segura, backfill + `/reindex`, timeouts HTTP, reconexión del consumer, `getTimeField`, commit/schema Solr, bump de CVEs | 02, 05 |
| [x] | [07 — Contratos de API](07-contratos-api.md) | Envelope de error estándar, `/api/v1`, idempotencia, paginación/envelopes/semántica HTTP consistentes, tipo de `user_id`, content negotiation | 04, 06 |
| [x] | [08 — Resiliencia de runtime](08-resiliencia-runtime.md) | **Graceful shutdown/SIGTERM (prereq de k8s)**, deadlines Mongo/Solr/consumer, bulkhead del fan-out, circuit breaker, robustez fina del productor RabbitMQ | 04, 06 · **antes del 09** |
| [x] | [09 — Cloud-native / k8s](09-cloud-native-k8s.md) | k8s con Deployments+Service+HPA+probes, Dockerfiles multi-stage/no-root, `.dockerignore`, resource limits, tags inmutables/Trivy/GHCR | 05 (`/readyz`), 08 (SIGTERM), 02 |
| [ ] | [10 — Gateway nginx: TLS + hardening](10-nginx-gateway.md) | TLS + redirect + HSTS, fix de herencia de `add_header`, cache real de `/search`, 429 en rate-limit | independiente (ideal tras 07 para no re-tocar locations) |
| [ ] | [11 — Consistencia y limpieza de código](11-consistencia-limpieza.md) | CORS válido, panel microservices, endpoint con PII, `PORT`, errores tipados, TTL L2, 404 en delete, **rename `AvaiableRooms` (cambio coordinado atómico)**, mocks fuera del binario, layering, perfil frontend | 02 y 07 (obligatorios para C11) |
| [ ] | [13 — Frontend portfolio-grade](13-stretch-dominio-frontend.md) | **Núcleo:** contratos/UI, búsqueda, booking, historial, admin, a11y, tests y lazy loading. **Menú opcional:** User rico/TS; reviews, pagos y room-types se difieren | 07 y 11 · **antes del 12** |
| [ ] | [12 — Documentación y presentación](12-documentacion-portfolio.md) | README nuevo, capturas/GIF/diagrama, OpenAPI, traducción de `ProyectoBackend.md`, errores fácticos, badges, prep de entrevista | 01–11 + núcleo de 13 (retrata el producto final) |

### Fixes de la review externa (2026-07-11)

Una review general post-planes 01–05 dejó hallazgos con IDs `RV1`–`RV31`, triageados en
[`fixes/README.md`](fixes/README.md): los huérfanos tienen planes propios `F1`–`F4` en `fixes/`
(F4 conviene **antes de que el CI corra por primera vez**; F1 antes de los planes 06/07), y el resto se ejecuta
**dentro** del plan pendiente que ya tocaba ese código — al empezar los planes 06, 07, 08, 10,
11 o 13, revisar su fila en esa tabla y sumar los `RV*` correspondientes al alcance.

### Grafo de dependencias (resumen)

```
01 ───────────────────────────────────┐
02 ─→ 03 ─→ 04 ─→ 07 ─┐               │
02 ─────────→ 06 ─→ 08 ─→ 09 ─────────┤
      03 ─→ 05 ─→ 06   05 ─→ 09       ├─→ 13 (frontend) ─→ 12
                  07 ─→ 10, 11 ────────┘
```

## Tabla de trazabilidad ID → plan (fuente de verdad)

Todos los IDs de las Secciones 2, 4 y 6 de `plantofinish.md`. **96 IDs + 4 bloques de la Sección 4; cada uno en exactamente un plan.**

| ID | Hallazgo (resumen) | Plan |
|----|--------------------|:----:|
| S1 | Escalada de privilegios: registro como `administrador` | 01 |
| S2 | `users-api` sin ningún middleware de auth | 01 |
| D1 | Overbooking incondicional en `CreateReservation` | 04 |
| D2 | `Update` devuelve 500 y no publica evento en cache miss | 04 |
| D3 | Caché reporta hoteles libres como "no disponibles" | 04 |
| D4 | Semántica de fechas divergente Mongo vs caché | 04 |
| E1 | Pérdida silenciosa de eventos (autoAck + errores tragados) | 06 |
| E2 | Inyección Lucene / búsqueda multi-palabra rota | 06 |
| E3 | Sin backfill ni reconciliación de Solr | 06 |
| E4 | Cliente HTTP a hotels-api sin timeout ni context | 06 |
| E5 | Consumidor frágil: sin reconexión, health mentiroso | 06 |
| E6 | `getTimeField` nunca puebla check-in/out | 06 |
| C1 | CORS inválido (`*` + `AllowCredentials:true`) | 11 |
| C2 | Panel "microservices admin" 100% falso | 11 |
| C3 | `GET /hotels/:id/reservations` público con PII | 11 |
| C4 | Secreto JWT por defecto peligroso | 01 |
| C5 | `hotels-api` ignora `PORT` | 11 |
| C6 | Mapeo de errores por substring | 11 |
| C7 | L2 memcached sin expiración | 11 |
| C8 | Validación de entrada pobre | 01 |
| C9 | `DELETE` devuelve 200 para usuarios inexistentes | 11 |
| C10 | Sin tests de la capa repositories | 04 |
| C11 | Rename `AvaiableRooms` → cambio coordinado atómico | 11 |
| C12 | Sin graceful shutdown / SIGTERM ni ping al arranque | 08 |
| C13 | Mocks compilados en el binario de producción | 11 |
| C14 | Robustez fina del productor RabbitMQ | 08 |
| I1 | Secretos en texto plano commiteados → `.env` | 01 |
| I2 | 2 de 3 Dockerfiles no multi-stage / no-root | 09 |
| I3 | `proxy_cache_valid` es un no-op | 10 |
| I4 | Sin CI | 02 |
| I5 | Servicios Go sin healthcheck de contenedor | 05 |
| I6 | Deriva de versión de Go + module path de search-api | 02 |
| I7 | Rate-limit devuelve 503 en vez de 429 | 10 |
| I8 | Frontend comentado del compose / conteo de servicios | 11 |
| I9 | Falta `.gitignore` raíz, `LICENSE`, `Makefile` | 02 |
| P1 | README anuncia endpoint inseguro y destructivo | 12 |
| P2 | Docs contradictorias (Bruno vs README vs código) | 12 |
| P3 | Sin activos visuales ni API reference (OpenAPI) | 12 |
| P4 | Borrar `PLAN.md` / `RULES.md` | 12 |
| P5 | Mejor doc solo en español → traducir | 12 |
| P6 | Errores fácticos verificables en docs | 12 |
| P7 | Seed data + credenciales demo | **03** (playbook 7.3; el README solo lo referencia) |
| P8 | Placeholders sin rellenar + lib amqp deprecada | 12 |
| P9 | LICENSE/badges/sección "production roadmap" | 12 |
| §4.1 | Reestructurar el README | 12 |
| §4.2 | Activos visuales (capturas, GIF, diagrama, OpenAPI) | 12 |
| §4.3 | Traducir y consolidar docs | 12 |
| §4.4 | Prep de entrevista | 12 |
| O1 | Sin correlación de requests end-to-end | 05 |
| O2 | Sin logging estructurado / por niveles | 05 |
| O3 | `/health` es solo liveness | 05 |
| O4 | Gin en modo debug en contenedores | 05 |
| DB1 | Sin migraciones versionadas | 03 |
| DB2 | Cero índices secundarios | 03 |
| DB3 | Sin pool de conexiones ni timeouts de driver | 03 |
| DB4 | `users-api` nunca propaga `context` a la DB | 03 |
| DB5 | Sin paginación a nivel DB | 03 |
| DB6 | Solr con `Commit()` duro por documento | **06** (afinidad Solr) |
| DB7 | Schema Solr con tipos inapropiados | **06** (afinidad Solr) |
| A1 | Sin envelope de error estándar + fuga de internos | 07 |
| A2 | Sin versionado de API | 07 |
| A3 | Sin idempotencia en POST inseguros | 07 |
| A4 | Paginación inconsistente | 07 |
| A5 | Envelopes de respuesta inconsistentes | 07 |
| A6 | Semántica HTTP floja (201/204/Location) | 07 |
| A7 | `user_id` cambia de tipo entre servicios | 07 |
| A8 | Sin content negotiation | 07 |
| R1 | `users-api` sin timeout en ningún nivel | **03** (fusionado con DB4 según 7.0) |
| R2 | Sin deadlines Mongo/Solr; consumer con `context.Background()` | 08 |
| R3 | Caché como dependencia dura en lectura | **04** (mismo patrón/archivos que D2) |
| R4 | Fan-out sin bulkhead + fail-hard | 08 |
| R5 | Sin circuit breaker ni retry client-side | 08 |
| T1 | Sin tests de integración (testcontainers) | 02 |
| T2 | Sin contract test productor↔consumidor | 02 |
| T3 | Sin e2e; `test_load_balancer.sh` no afirma nada | 02 |
| T4 | Sin coverage gate ni `-race` | 02 |
| T5 | Rutas negativas de auth y minter JWT sin test | **01** (verifica lo que 01 construye) |
| T6 | Sin fixtures/golden files ni benchmarks | 02 |
| CN1 | Sin Kubernetes/Helm (el titular es orquestación) | 09 |
| CN2 | Sin `.dockerignore` | 09 |
| CN3 | Sin límites de CPU/memoria | 09 |
| CN4 | Sin estrategia de tags / supply-chain | 09 |
| SD1 | Sin escaneo de dependencias (CVEs reales) | 02 |
| SD2 | Sin política de contraseña | 01 |
| SD3 | JWT sin `aud`/`iss`/`nbf` | 01 |
| SD4 | Sin TLS/HTTPS | 10 |
| SD5 | Security headers de nginx descartados por herencia | 10 |
| CQ1 | Sin `golangci-lint`; `gofmt` sucio en 7 archivos | 02 |
| CQ2 | Sin módulo compartido (structs duplicados 4×) | 02 |
| CQ3 | Sin `go.work` | 02 |
| CQ4 | Layering/paquetes inconsistentes + 3 CORS divergentes | 11 |
| DM1 | `Reservation` sin estado/lifecycle | 04 |
| DM2 | `Reservation` sin cantidad ni dinero | 04 |
| DM3 | Sin entidad `Review` (rating manual) | **13 (menú diferido)** |
| DM4 | Sin concepto de pago/total | **13 (menú diferido)** |
| DM5 | Struct `ReservationNew` muerto (nunca se publica) | 04 |
| DM6 | Sin modelado de `Room`/room-type | **13 (menú diferido)** |
| DM7 | `User` anémico (sin email/nombre/rol tipado) | **13 (menú opcional)** |
| FE1 | Frontend sin TypeScript | **13 (menú opcional)** |
| FE2 | Sin code-splitting/lazy por ruta | **13 (núcleo)** |

### Notas de asignación (decisiones deliberadas)

- **P7 → 03** y no al 12: su implementación real vive en el playbook 7.3 (migración 0002 + `mongo-init.js`); el plan 12 solo documenta las credenciales demo en el README.
- **R1 → 03**: la corrección 7.0 lo fusiona con el refactor de `context` (DB4) — mismas firmas, mismo trabajo.
- **R3 → 04**: mismo patrón "caché best-effort" y mismos archivos (`hotels_service.go`) que D2/D3.
- **T5 → 01**: los tests negativos de auth verifican exactamente lo que el plan 01 construye.
- **DB6/DB7 → 06**: todo el trabajo Solr en una sola sesión.
- **C12 → 08 (antes del 09/k8s)**: el drain limpio de pods en rolling deploys depende de manejar SIGTERM; por eso resiliencia va antes de cloud-native.
- **C11 → 11, después de 02 y 07**: el módulo de contratos (02) mantiene el typo a propósito; el rename es un cambio coordinado atómico (tags JSON/BSON, Solr, frontend, golden test) que se hace con `/api/v1` ya en su lugar.
- **13 antes de 12**: el núcleo frontend es parte del producto que el portfolio documenta; las capturas/GIF del plan 12 deben mostrar esa versión final. La checkbox de 13 exige solo su núcleo; DM3/DM4/DM6, DM7 y FE1 conservan checkboxes internas opcionales.
