# Plan to Finish — Hotel Search & Booking Microservices Platform

> Plan propuesto para dejar el proyecto **portfolio-ready** para búsqueda de puestos **Backend**.
> Basado en una auditoría exhaustiva del código (los 3 servicios Go, frontend, nginx, docker-compose y docs),
> con **verificación adversarial** de cada hallazgo crítico contra el código real.
>
> **Fecha:** 2026-06-30 · **Autor del plan:** revisión automatizada + síntesis · **Estado:** propuesta (no se modificó código)

---

## 0. Veredicto ejecutivo

**¿Está terminado?** Funcionalmente, casi. El stack levanta con `docker compose up`, el frontend recorre el flujo completo (registro → login → búsqueda → reserva → panel admin), los 3 servicios Go **compilan, pasan `go vet` limpio y sus tests pasan**, y el frontend buildea. La **arquitectura y la documentación escrita son notablemente buenas** para un proyecto que arrancó como trabajo de facultad.

**¿Está listo para portfolio? Todavía no.** Hay un conjunto acotado de defectos que un revisor backend detecta en los primeros 10 minutos y que, sin arreglar, **le restan credibilidad a todo lo demás**:

1. **Seguridad (bloqueante):** cualquiera puede auto-registrarse como `administrador` y `users-api` no tiene *ningún* middleware de autenticación (se puede listar y **borrar cualquier usuario sin token**).
2. **Correctitud del dominio:** el corazón de un "booking" — no permitir overbooking — **no está implementado**; y hay bugs de caché que hacen que respuestas correctas dependan del estado de la caché.
3. **Robustez del servicio event-driven (`search-api`):** el consumidor pierde eventos silenciosamente y la query a Solr se arma por concatenación de strings.
4. **Presentación:** faltan capturas, diagrama renderizado, OpenAPI, LICENSE, CI, y hay secretos en texto plano + docs contradictorias.

**Estimación:** ~**5–8 días de trabajo enfocado** para pasar de "funciona pero tiene agujeros que un senior detecta" a "pieza de portfolio sólida y defendible en entrevista". El plan de abajo está priorizado para que, si solo tenés un fin de semana, arregles primero lo que más pesa (Fase 1 + 2).

> **Metodología del veredicto:** cada hallazgo marcado como *Verificado ✅* fue confirmado leyendo el código real y trazando la cadena completa (ruta → controller → service → repo). Ninguno de los hallazgos críticos resultó falso; algunas severidades se ajustaron a la baja porque es un proyecto self-hosted de portfolio (no una producción real), pero siguen siendo señales que un revisor va a marcar.

---

## 1. Lo que está muy bien (tus argumentos de venta)

No todo es arreglar. Esto es lo que **conviene destacar** en el README y en la entrevista — está genuinamente por encima del promedio de un portfolio junior/junior-semi:

- **Arquitectura en capas idiomática con Dependency Inversion** en los 3 servicios: `controller → service → repository (interfaces)`, con implementaciones intercambiables (Mongo/Solr/MySQL/cache/mock) inyectadas en `cmd/main.go` como *composition root*. Es exactamente la estructura que un entrevistador quiere ver.
- **JWT con defensa contra *algorithm confusion*:** el middleware de `hotels-api` rechaza explícitamente cualquier método de firma que no sea HMAC (`auth.go:37-43`). Buen detalle de seguridad.
- **RBAC + ownership checks reales** en `hotels-api`: un usuario solo puede crear/cancelar/ver **sus** reservas; admin es la excepción (`hotels_controller.go:163-168, 216-221, 281-286`). La pregunta clásica "¿puede A cancelar la reserva de B?" está bien resuelta (No).
- **Nunca se exponen passwords:** modelos `domain` vs `dao` separados, `toUser()` elimina el hash, y el login colapsa "usuario inexistente" y "password incorrecto" en un mismo 401 (anti-enumeración).
- **Caché multi-nivel L1→L2→DB** en `users-api` con back-fill en miss y escritura best-effort — buen tema de conversación sobre estrategias de caché.
- **Productor RabbitMQ auto-reparable** en `hotels-api`: backoff exponencial, `sync.RWMutex`, reconexión vía `NotifyClose`. Más sofisticado que el código típico de portfolio.
- **API Gateway con Nginx de calidad**: `least_conn` sobre 3 réplicas con failover (`max_fails`/`fail_timeout`), `keepalive`, rate limiting diferenciado (login 5r/m vs API 10r/s), security headers, CORS por `map`, logs JSON y server de monitoreo en `:8090`. **Este es el activo estrella para entrevistas.**
- **docker-compose con healthchecks reales** y `depends_on: condition: service_healthy` en las DBs — demuestra que entendés el orden de arranque, no solo `depends_on` por nombre.
- **`users-api/Dockerfile` multi-stage de manual**: binario estático `CGO_ENABLED=0`, `-ldflags "-s -w"`, imagen final `alpine` mínima, usuario no-root.
- **`ProyectoBackend.md`** es un documento de arquitectura genuinamente bueno (explica el *por qué* de cada patrón). Es tu mejor material — hay que traducirlo y ponerlo al frente.
- **Tests con `testify/mock` + `httptest`** en las capas service/controller, con casos 401/403/400/200 y tokens JWT reales.

---

## 2. Lo que falta / hallazgos (priorizado por severidad)

Leyenda de severidad tras verificación · 🔴 crítico · 🟠 alto · 🟡 medio · ⚪ bajo · Esfuerzo: **S** (<2h) · **M** (medio día) · **L** (1–2 días)

### 🔴 Bloqueantes de seguridad — arreglar SÍ o SÍ antes de mostrar

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| S1 | **Escalada de privilegios anónima**: cualquiera puede registrarse como `administrador`. `POST /users` no tiene auth, el service acepta `tipo` del body verbatim, y el dropdown del Register ofrece "Administrator". El JWT resultante desbloquea todo el `/admin/*` de `hotels-api` (incluido scale/restart/logs). *Verificado ✅ (cadena completa trazada)* | `users_service.go:95-101`, `users_controller.go:86`, `frontend/src/pages/Register.jsx:255`, `hotels-api/.../auth.go:92` | Forzar `tipo=cliente` en el registro público. Crear admins solo vía seed/migración o endpoint protegido con `AdminOnly`. Quitar la opción "Administrator" del Register. | 🔴 · M |
| S2 | **`users-api` no tiene NINGÚN middleware de auth**: `GET /users` (lista todo), `GET /users/:id` y **`DELETE /users/:id`** están 100% abiertos. Cualquiera enumera usuarios y borra cuentas (incl. admins) sin token. *Verificado ✅* | `users-api/cmd/main.go:58-64` (solo `CorsMiddleware`); `users_controller.go:107-124` (Delete sin authz) | Portar el middleware JWT de `hotels-api` a `users-api`. Proteger: `GET /users` → admin; `GET /users/:id` y `DELETE /users/:id` → owner **o** admin. Paginar `GetAll`. | 🔴 · M |

> S1 y S2 son en realidad **un mismo problema** (`users-api` nunca implementó el lado *verify* del JWT, solo el *issue*). Resolverlos juntos: crear `users-api/internal/middlewares/auth.go` (copiar/adaptar el de `hotels-api`) y aplicar grupos de rutas en `main.go`. Es el trabajo de mayor impacto de todo el plan.

### 🟠 Correctitud del dominio — el "booking" tiene que ser correcto

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| D1 | **Overbooking incondicional**: `CreateReservation` **no llama a `GetAvailability` ni verifica capacidad**. La lógica de disponibilidad existe pero es un endpoint aparte, solo de lectura. No hay transacción, ni índice único, ni lock. Un revisor pregunta "¿cómo evitás doble reserva?" y hoy la respuesta es "no lo evito". *Verificado ✅* | `hotels_service.go:205-225`, `hotels_controller.go:135-183`, `hotels_mongo.go:207` (InsertOne pelado) | En `CreateReservation`: validar `hotel existe`, `CheckOut > CheckIn`, fechas no pasadas; recomputar ocupación por noche y rechazar si está lleno; hacerlo atómico con **transacción Mongo** o índice/contador único por hotel-noche. Derivar `HotelName` del hotel, no del body. | 🟠 · L |
| D2 | **`Update` de hotel devuelve 500 y NO publica el evento** si el hotel expiró de la caché (TTL 30s). Escribe en Mongo OK, pero `cacheRepository.Update` falla si la key no está → el service retorna error **antes** del `Publish("UPDATE")`. Resultado: la API responde 500 en un PUT exitoso y `search-api` queda desincronizado. *Verificado ✅* | `hotels_service.go:150-166`, `hotels_cache.go:201-207`, `config.go:22` | Tratar la caché como best-effort en escrituras: en `Update`/`Delete`, loguear-y-continuar ante error de caché (o hacer upsert), y **publicar el evento siempre**. Nunca fallar una escritura de DB exitosa por un miss de caché. | 🟠 · S |
| D3 | **La caché reporta hoteles libres como "no disponibles"**: `Cache.IsHotelAvailable` devuelve `false` cuando la lista de reservas no está cacheada. Como ver un hotel cachea el hotel (pero no sus reservas), tras cualquier visita el hotel aparece **no disponible** hasta 30s. Falso negativo en la ruta central de búsqueda/reserva. *Verificado ✅* | `hotels_cache.go:456-460, 388-394`; diverge de `hotels_mongo.go:481` | Una lista de reservas ausente = **cero reservas = disponible**, no lo contrario. O caer a Mongo ante miss de la lista. Añadir tests de repositorio que fijen el comportamiento. | 🟠 · M |
| D4 | **Semántica de fechas divergente Mongo vs Caché**: Mongo cuenta el día de checkout como ocupado; caché/mock lo excluyen. La misma consulta da resultados distintos según el estado de caché. Mongo tampoco valida `checkOut > checkIn`. *Verificado (auditoría)* | `hotels_mongo.go:399-414` vs `hotels_cache.go:474-489` | Elegir un modelo canónico (por noche, checkout excluido), implementarlo idéntico en ambos repos, y una suite de tests de rango de fechas que corra contra **las dos** implementaciones. | 🟡 · M |

### 🟠 Robustez del servicio event-driven (`search-api`)

`search-api` es el eslabón más débil de los tres. Como el "event-driven / CQRS-lite" es un titular del proyecto, conviene endurecerlo.

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| E1 | **Pérdida silenciosa de eventos**: el consumidor usa `autoAck=true` y el handler **traga todos los errores** (solo loguea). Si Solr o `hotels-api` fallan, o el JSON no parsea, el evento se pierde para siempre y el índice se desincroniza. No hay nack/retry/DLQ. *Verificado ✅* | `queue_rabbit.go:53` (autoAck), `search_service.go:73,82-84,111-131` | Pasar a **manual ack** (`autoAck=false`); que `HandleHotelNew` retorne `error`; `Ack` en éxito, `Nack(requeue)` en fallo transitorio con reintentos acotados; **dead-letter queue** para mensajes envenenados. | 🟠 · M |
| E2 | **Inyección Lucene + búsqueda rota**: la query se arma con `fmt.Sprintf("q=(name:%s OR description:%s)...")` sin escapar ni codificar. Una búsqueda multi-palabra ("hotel spa") queda malformada, `q` vacío da 500, y se pueden inyectar cláusulas. Además la paginación se ignora (el string entero va como `query` JSON a Solr). *Verificado ✅* | `hotels_solr.go:177` | Usar el query builder de `solr-go` / `edismax` con `qf` sobre `name,description`, escapar metacaracteres Lucene, manejar `q` vacío (`*:*`), y pasar `rows`/`start` como parámetros reales. Test de construcción de query. | 🟠 · M |
| E3 | **Sin backfill ni reconciliación**: al arrancar solo se levanta el consumidor. Si el volumen de Solr está fresco, o se crearon hoteles mientras `search-api`/RabbitMQ estaban caídos (o se perdió un evento por E1), esos hoteles **nunca** aparecen y no hay forma de recuperarlos. *Verificado ✅* | `search-api/cmd/main.go:48-51`; `hotels_http.go` solo tiene `GetHotelByID` | Endpoint `list-all` en `hotels-api` + rutina de backfill al arranque en `search-api` + job periódico o `POST /reindex` admin. | 🟡 · M |
| E4 | **Cliente HTTP a `hotels-api` sin timeout** y que ignora el `context`: `http.Get` con `DefaultClient`. Como el consumidor procesa serial, un `hotels-api` colgado **bloquea todo el pipeline** de indexación. *Verificado ✅* | `hotels_http.go:31-32` | `http.Client{Timeout: ...}` + `http.NewRequestWithContext(ctx, ...)` + reintentos acotados en 5xx/conexión. | 🟡 · S |
| E5 | **Consumidor frágil**: error de `QueueDeclare` sin chequear, `log.Fatalf` si RabbitMQ no está listo al arrancar, y **sin reconexión** (a diferencia del productor de `hotels-api`). El `/health` siempre dice "ok" aunque la indexación esté muerta. *Verificado ✅* | `queue_rabbit.go:32,37,40,64-75`; `main.go:63-69` | Reusar el patrón `connectWithRetry` del productor; detectar cierre de canal y reconectar; que `/health` refleje conectividad real a Rabbit/Solr. | 🟡 · M |
| E6 | **`getTimeField` nunca puebla check-in/out**: hace `doc[field].(time.Time)`, pero Solr devuelve strings → el assert siempre falla y ambos tiempos salen en cero en cada resultado. *Verificado (auditoría)* | `hotels_solr.go:240-245` | `time.Parse(time.RFC3339, ...)`. Test de parseo de documento Solr. | 🟡 · S |

### 🟡 Consistencia, robustez y calidad de código

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| C1 | **CORS inválido**: `hotels-api` y `search-api` usan `AllowOrigins:["*"]` **con** `AllowCredentials:true` — combinación que el browser rechaza. Como la auth es por header Bearer (no cookies), lo correcto es `*` + `credentials:false`, o allowlist + `credentials:true`. | `hotels-api/cmd/main.go:60-67`, `search-api/.../middlewares.go:9-10` | Elegir una de las dos configuraciones válidas y unificar con el patrón `CORS_ALLOWED_ORIGINS` que ya usa `users-api`. | 🟡 · S |
| C2 | **Panel "microservices admin" es 100% falso**: uptimes hardcodeados, health por hash de caracteres, logs mock, scale/restart que no hacen nada. Puertos internamente inconsistentes. Está en rutas admin reales. *Verificado (auditoría)* | `microservices_controller.go:43-274` | **Decidir:** (a) hacerlo real y read-only (health = `GET /health` a cada servicio) y quitar scale/restart, **o** (b) etiquetarlo claramente como demo/mock en la respuesta y el README, **o** (c) removerlo. Recomendado: (a) para status, quitar acciones de escritura. | 🟡 · M |
| C3 | **`GET /hotels/:id/reservations` es público** y devuelve `user_id` de todos: cualquiera enumera quién reservó qué. | `hotels-api/cmd/main.go:71`, `hotels_controller.go:237-252` | Mover a auth (admin), o quitar `user_id`/PII de la respuesta pública. | 🟡 · S |
| C4 | **Secreto JWT por defecto peligroso**: default literal `your-secret-key-change-in-production`. Si `JWT_SECRET` no está seteado, se firma con una key pública conocida → cualquiera forja tokens admin. | `users-api/config.go:25`, `hotels-api/config.go:32` | `log.Fatal` al arranque si `JWT_SECRET` está vacío o es el placeholder. Nunca commitear un default usable. | 🟡 · S |
| C5 | **`hotels-api` ignora `PORT`**: `router.Run(":8081")` hardcodeado; `config.Port` es código muerto. | `hotels-api/cmd/main.go:108` vs `config.go:35` | `router.Run(":" + config.Port)`. | ⚪ · S |
| C6 | **Mapeo de errores por substring**: `strings.Contains(err.Error(), "Duplicate")` para decidir 400/409/500. Frágil ante cambios del driver y filtra texto interno al cliente. | `users_controller.go:89-97` | Errores centinela tipados (`ErrUsernameTaken`, `ErrValidation`) + `errors.Is`. Detectar el 1062 de MySQL de forma tipada. | 🟡 · M |
| C7 | **L2 (memcached) sin expiración**: los items se escriben sin TTL → viven para siempre. Con el `Delete` best-effort que traga errores, una cuenta borrada puede seguir logueando desde L2. | `users_memcached.go:92,98,128-140` | Setear `Expiration` explícito en los items de memcached. | 🟡 · S |
| C8 | **Validación de entrada pobre**: sin longitud/complejidad de password, sin límites de longitud de username, sin email. Reservas sin tags de binding (`CheckOut<=CheckIn` no se rechaza). | `users_service.go:87-92`, `reservations.go:5-12` | Tags `binding` de gin (min/max/required) + validación en service, mensajes 400 claros. | 🟡 · S |
| C9 | **`DELETE` devuelve 200 para usuarios inexistentes** (gorm no chequea `RowsAffected`). Inconsistente con el 404 de `GetByID`. | `users_mysql.go:105-110` | Chequear `RowsAffected==0` → `ErrUserNotFound` → 404. | ⚪ · S |
| C10 | **Tests que validan mocks divergentes del código real**: no hay tests de la capa `repositories` (Mongo/Solr/ccache) — justo donde viven D1–D4 y E2/E6. La suite está verde mientras esos bugs viven en código no testeado (falsa confianza). | `find` → sin `*_test.go` en `internal/repositories` | Tests de repositorio: date-overlap contra ambas implementaciones; parseo de doc Solr; (opcional) integración con testcontainers Mongo/Solr. | 🟡 · M |
| C11 | **Tipo `AvaiableRooms` mal escrito** en todo el contrato JSON/BSON (y propagado al frontend). Cuanto antes se renombre, más barato. | `hotels_domain.go:17`, `hotels_dao.go:17`, frontend | Rename coordinado a `available_rooms` en backend + frontend. | ⚪ · S |
| C12 | **Sin graceful shutdown** ni `Ping` a Mongo al arranque: el proceso "arranca ok" aunque la DB no responda; no drena requests ante SIGTERM ni cierra RabbitMQ. | `hotels-api/cmd/main.go:107-110`, `hotels_mongo.go:45-56` | `http.Server` + `Shutdown(ctx)` por señal, `defer queue.Close()`, `Ping` con timeout al arranque. | 🟡 · M |
| C13 | **Mocks compilados en el binario de producción**: `*_mock.go` (no `_test.go`) importan `testify` dentro de paquetes de producción. | `users_mock.go`, `tokenizers_mock.go` | Renombrar a `*_test.go` o mover a subpaquete `testonly/`. | ⚪ · S |
| C14 | **Detalles de robustez del productor RabbitMQ**: entre reintentos de `Publish` no reconecta (los 3 intentos pegan al mismo canal muerto); `$unionWith: {coll: nil}` es sintaxis no estándar que podría fallar en runtime; `defer cursor.Close` dentro de un loop por día. | `hotels-api/.../queue_rabbit.go:203-247`, `hotels_mongo.go:430-462` | Forzar `ensureConnection` entre reintentos; simplificar el `$unionWith` (default 0 en Go); cerrar el cursor por iteración. | ⚪ · M |

### 🔵 DevOps / Infra

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| I1 | **Secretos en texto plano commiteados**: `root/root` en todas las DBs y `JWT_SECRET` compartido en el `docker-compose.yml` versionado. Sin `.env` ni `.env.example`. Es lo primero que un revisor grepea. *Verificado ✅* | `docker-compose.yml:37,65-66,85-86,139,165,193,235` | Mover a `.env` (gitignored) vía `${VAR}`; commitear `.env.example` con placeholders; `.gitignore` raíz. Nota en README de que son creds de demo. | 🟡 · S |
| I2 | **2 de 3 Dockerfiles NO son multi-stage/no-root**: `hotels-api` y `search-api` son single-stage sobre `golang:*-alpine` (envían el toolchain completo, corren como root, hacen `go mod tidy` en build). **Contradice** a `ProyectoBackend.md` que vende "multi-stage + no-root". *Verificado (auditoría)* | `hotels-api/dockerfile`, `search-api/Dockerfile` | Replicar el patrón multi-stage de `users-api` en los otros dos; `go mod download` (no tidy); `.dockerignore` por servicio. | 🟡 · M |
| I3 | **`proxy_cache_valid` es un no-op**: `/search` declara cache de 5 min pero no hay `proxy_cache_path` ni `proxy_cache <zone>`. El comentario documenta algo que no ocurre. Un revisor con nginx lo ve en 5 segundos. *Verificado (auditoría)* | `nginx.conf:418-419` | Definir `proxy_cache_path ... keys_zone=search_cache:10m` + `proxy_cache search_cache; proxy_cache_key ...` en `/search`, **o** borrar la directiva y el comentario. | 🟡 · S |
| I4 | **Sin CI**: no hay `.github/workflows`. Los tests ya existen — un pipeline `go vet` + `go test` + `npm build` con badge es de los cambios de mayor ROI para un portfolio backend. | (ausencia) | GitHub Actions con matrix sobre los 3 módulos + build del frontend + badge en README. | 🟡 · M |
| I5 | **Servicios Go sin healthcheck de contenedor** y nginx depende de ellos por `service_started` → el gateway puede dar 502 antes de que las apps estén listas. | `docker-compose.yml:145-152,237-276` | Healthcheck (`wget --spider /health`) en cada servicio Go; nginx `depends_on: condition: service_healthy` para el tier de apps. | ⚪ · S |
| I6 | **Deriva de versión de Go**: `search-api` en `1.22`; `users/hotels` en `1.23.0` + `toolchain 1.24.11`. Y module path de `search-api` es `search-api` pelado vs `github.com/Julian0444/...` en los otros. | `*/go.mod` | Unificar versión de Go y renombrar el módulo de `search-api` al path canónico. (Opcional `go.work`.) | ⚪ · S |
| I7 | **Rate-limit devuelve 503 en vez de 429** (falta `limit_req_status`/`limit_conn_status`), colisiona con "upstream caído". | `nginx.conf:88-90` | `limit_req_status 429; limit_conn_status 429;`. | ⚪ · S |
| I8 | **Frontend comentado del compose** pero el README dice "Full orchestration (10 services)". Y el Dockerfile del frontend queda huérfano. | `docker-compose.yml:281-294`, `README.md:66` | `docker compose --profile frontend up` opcional, y reconciliar el conteo de servicios. | ⚪ · S |
| I9 | **Falta `.gitignore` raíz, `LICENSE`, `Makefile`**. (Nada basura commiteado aún — `node_modules` está fuera — pero son señales baratas de repo pulido.) | (ausencia) | `.gitignore` raíz, `LICENSE` (MIT), `Makefile` con `build/test/lint/up/down`. | ⚪ · S |

### 📄 Documentación / Presentación

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| P1 | **El README anuncia un endpoint inseguro y destructivo**: la tabla lista `GET /users` y `DELETE /users/:id` con Auth "—". Documentarlo con honestidad no lo hace aceptable — un revisor lee "DELETE /users/:id \| — \| Delete user" y deja de leer. *Verificado ✅* | `README.md:202-204` | Se resuelve con S2 (protegerlos) + actualizar la tabla a Auth = Admin. | 🟠 · (con S2) |
| P2 | **Docs contradictorias**: la colección Bruno dice que `/users` requiere `bearer`; README y código dicen que no. Un revisor concluye que no conocés la postura de seguridad de tu propio sistema. *Verificado ✅ (severidad baja, pero mala señal)* | `Bruno .../Delete User.bru` vs `README.md:202` vs `main.go` | Una sola verdad (protegerlos) y que README + `users.md` + Bruno coincidan con el código. | ⚪ · S |
| P3 | **Sin activos visuales ni API reference**: la única imagen del repo es `vite.svg`. No hay capturas/GIF del SPA, ni diagrama renderizado (solo ASCII), ni OpenAPI/Swagger. Para un proyecto full-stack, cero visuales es una oportunidad perdida enorme. *Verificado ✅* | `git ls-files` (solo `vite.svg`) | `docs/` con 3–4 capturas + un GIF del flujo búsqueda→reserva→admin; diagrama renderizado (PNG/SVG); **OpenAPI** por servicio (o `swaggo`) servido en `/swagger`. | 🟠 · L |
| P4 | **Archivos de scaffolding "para la IA" commiteados**: `hotels-api/PLAN.md` ("PLAN DE TRABAJO PARA LA IA") y `RULES.md` ("no negociables para IA"). Leen como repo auto-generado y encima revelan que el panel de microservicios es un mock. *Verificado (auditoría)* | `hotels-api/PLAN.md`, `RULES.md` | **Borrarlos** del repo (guardarlos local o en un CONTRIBUTING privado). Plegar lo útil en `ProyectoBackend.md` con audiencia humana. | 🟡 · S |
| P5 | **Mejor doc en español solamente**: `ProyectoBackend.md` (tu mejor material), `LOAD_BALANCER.md`, `users.md` están en español; README en inglés. Para búsqueda internacional, un revisor angloparlante no puede leer lo mejor que tenés. | (idiomas mezclados) | Estandarizar en **inglés** para búsqueda internacional (o `.md` + `.es.md`). Traducir al menos `ProyectoBackend.md`. | 🟡 · M |
| P6 | **Errores fácticos verificables en docs**: dice "10 containers" pero hay **11**; `LOAD_BALANCER.md` documenta `GET /hotels` y `/health/all` que **no existen**; el diagrama pone Users API y Search API ambos en "Port 8082". | `README.md:176`, `ProyectoBackend.md:49`, `LOAD_BALANCER.md:68,80` | Corregir a 11; quitar rutas fantasma; aclarar puertos por servicio. | 🟡 · S |
| P7 | **Sin credenciales de demo ni seed data**: seguir el quickstart deja una app vacía y sin forma obvia de obtener un admin. La primera impresión (la demo) falla. | `README.md:161-192` (sin seed) | Script de seed (hoteles demo + 1 admin + 1 cliente) y bloque "Demo credentials" + walkthrough "Try it" de 3 líneas. | 🟡 · M |
| P8 | **Placeholders sin rellenar y lib deprecada**: `README_HOTELS.md` tiene `[Add your repo URL]`, `Last Updated: January 2026`, y publicita `streadway/amqp` (archivada en 2021). | `README_HOTELS.md:600,622-627` | Rellenar links reales, quitar "Last Updated" manual, migrar a `rabbitmq/amqp091-go`. | ⚪ · S |
| P9 | **Falta LICENSE, badges, sección "cómo lo llevaría a producción"** consolidada (hoy dispersa entre `LOAD_BALANCER.md` y `README_HOTELS.md`). | (ausencia) | `LICENSE` MIT, badges (CI/Go/license), y una sección "Production roadmap / trade-offs" en el README. | ⚪ · S |

---

## 3. Plan de ejecución por fases

Ordenado por **impacto / credibilidad**. Cada fase deja el repo en un estado coherente y "mostrable".

### Fase 1 — Cerrar los agujeros de seguridad (≈1 día) 🔴
*El trabajo de mayor impacto. Sin esto, nada más importa para un rol backend.*
- [ ] Crear `users-api/internal/middlewares/auth.go` (portar de `hotels-api`) y aplicar grupos de rutas en `main.go`.
- [ ] Proteger `GET /users` (admin), `GET /users/:id` y `DELETE /users/:id` (owner o admin). Paginar `GetAll`. **(S2, P1, P2)**
- [ ] Forzar `tipo=cliente` en registro público; quitar "Administrator" del `Register.jsx`; seed del primer admin. **(S1)**
- [ ] `log.Fatal` si `JWT_SECRET` está vacío/placeholder. **(C4)**
- [ ] Mover secretos a `.env` + `.env.example` + `.gitignore` raíz. **(I1)**
- [ ] Actualizar README/Bruno para reflejar la nueva postura de auth.
- **Criterio de aceptación:** un anónimo no puede listar/borrar usuarios ni crear un admin; `go test ./...` verde en los 3 servicios.

### Fase 2 — Correctitud del dominio (≈1.5 días) 🟠
*Que el "booking" sea realmente un booking.*
- [ ] `CreateReservation`: validar hotel/fechas + chequeo de capacidad **atómico** (transacción Mongo o índice único). **(D1)**
- [ ] `Update`/`Delete`: caché best-effort + publicar evento siempre. **(D2)**
- [ ] Arreglar el falso negativo de disponibilidad en caché + unificar semántica de fechas. **(D3, D4)**
- [ ] Tests de repositorio de disponibilidad (contra Mongo y caché). **(C10)**
- **Criterio de aceptación:** no se puede sobre-reservar; la disponibilidad es determinista independientemente del estado de caché; hay tests que lo prueban.

### Fase 3 — Endurecer `search-api` (≈1 día) 🟠
- [ ] Manual ack + `error` en el handler + retry/DLQ. **(E1)**
- [ ] Query builder / edismax con escape + `q` vacío + paginación real. **(E2)**
- [ ] Timeout + context en el cliente HTTP. **(E4)**
- [ ] Reconexión del consumidor + `/health` real. **(E5)**
- [ ] Backfill al arranque + `POST /reindex` admin (requiere `list-all` en hotels-api). **(E3)**
- [ ] Fix `getTimeField` (parseo RFC3339) + test. **(E6)**
- **Criterio de aceptación:** matar Solr/RabbitMQ y recuperarse sin perder consistencia; búsquedas multi-palabra funcionan.

### Fase 4 — Consistencia, DevOps y CI (≈1 día) 🟡🔵
- [ ] CORS válido y unificado. **(C1)** · `hotels-api` respeta `PORT`. **(C5)**
- [ ] Dockerfiles multi-stage/no-root en los 3; unificar Go + module path de search. **(I2, I6)**
- [ ] Healthchecks de apps + `service_healthy` en nginx. **(I5)** · `429` en rate-limit. **(I7)** · nginx cache real o quitar la directiva. **(I3)**
- [ ] **GitHub Actions CI** (vet + test + build) con badge. **(I4)**
- [ ] `Makefile`, `LICENSE` (MIT), `.gitignore` raíz. **(I9)**
- [ ] Decidir el destino del panel "microservices" (real read-only / rotular como demo / quitar). **(C2)**
- [ ] Limpieza: TTL en L2, errores tipados, graceful shutdown, rename `AvaiableRooms`, mocks fuera del binario. **(C6, C7, C9, C11, C12, C13, C14)**

### Fase 5 — Documentación y presentación de portfolio (≈1–1.5 días) 📄
*Ver la sección 4 en detalle.* Traducir `ProyectoBackend.md`, README nuevo, diagrama renderizado, capturas/GIF, OpenAPI, seed + credenciales demo, borrar `PLAN.md`/`RULES.md`, corregir errores fácticos, sección "cómo lo productionizaría".

### Fase 6 — Stretch (opcional, para diferenciarte) ⭐
- [ ] Observabilidad: `/metrics` Prometheus + un dashboard Grafana (o structured logging con `slog`).
- [ ] `refresh token` / logout / revocación (hoy los JWT de 24h no se pueden invalidar).
- [ ] Trazas distribuidas (OpenTelemetry) a través del gateway y RabbitMQ.
- [ ] Deploy de una demo en vivo (Fly.io/Render) enlazada desde el README.
- [ ] Pestaña "Infraestructura" en el admin que muestre el status real de los microservicios (aprovecha el load balancing que ya tenés).

> **Si solo tenés un fin de semana:** hacé **Fase 1 + Fase 2 + el CI de Fase 4 + un README decente con 3 capturas**. Eso solo ya te lleva de "tiene agujeros" a "defendible".

---

## 4. Plan de documentación para portfolio

El objetivo: que un hiring manager, en un **skim de 5 minutos**, entienda qué construiste, vea que funciona, y quiera hablar con vos. Orden de lectura pensado: *hook visual → arquitectura → cómo correrlo → profundidad técnica*.

### 4.1 Reestructurar el `README.md` (el que más se lee)
Orden recomendado:
1. **Título + una línea + badges** (CI passing, Go version, License).
2. **GIF/captura del app funcionando** (búsqueda → reserva → admin). Esto es lo que engancha.
3. **Qué demuestra este proyecto** (3–5 bullets de conceptos: microservicios, API gateway + LB, event-driven/CQRS-lite, caché multi-nivel, JWT/RBAC distribuido). Robado de tu `ProyectoBackend.md`.
4. **Diagrama de arquitectura renderizado** (PNG/SVG, no ASCII). Exportá tu diagrama actual con [excalidraw](https://excalidraw.com) o Mermaid.
5. **Stack** (tabla, ya la tenés).
6. **Quickstart que funcione de verdad**: `cp .env.example .env` → `docker compose up -d --build` → `make seed` → "abrí http://localhost:5173" → **credenciales demo** (admin + cliente).
7. **Tabla de endpoints** (corregida: auth real por endpoint) + link a la **spec OpenAPI**.
8. **Testing** (comandos + mención de cobertura).
9. **Decisiones de arquitectura / trade-offs** (link a `ProyectoBackend.md` en inglés).
10. **"Cómo lo llevaría a producción"** (consolidar: secrets manager, observabilidad, HPA/K8s, CI/CD, outbox para RabbitMQ). Esto es lo que un senior busca para medir seniority.

### 4.2 Activos a crear (`docs/` o `assets/`)
- [ ] **Diagrama** de arquitectura renderizado (incluí el flujo de un evento CREATE hotel → Solr).
- [ ] **3–4 capturas**: Home/Search, HotelDetail, MyReservations, Admin Dashboard.
- [ ] **1 GIF** del flujo end-to-end (login → buscar → reservar).
- [ ] **OpenAPI/Swagger** por servicio (anotaciones `swaggo` o `openapi.yaml` a mano) — un revisor backend lo prefiere sobre tablas markdown.
- [ ] (Opcional) diagrama de secuencia del flujo event-driven.

### 4.3 Traducir y consolidar
- [ ] Traducir `ProyectoBackend.md` a inglés (es tu mejor pieza). **(P5)**
- [ ] **Borrar** `PLAN.md` y `RULES.md`. **(P4)**
- [ ] Corregir errores fácticos (11 containers, rutas fantasma, puertos). **(P6)**
- [ ] Rellenar placeholders, quitar "Last Updated", nota migración `amqp091-go`. **(P8)**
- [ ] `LICENSE` (MIT) + `CONTRIBUTING` breve (opcional). **(P9)**

### 4.4 Prep de entrevista (guardá esto para vos)
Preguntas que un entrevistador **va a hacer** con este proyecto — tené las respuestas listas:
- *"¿Cómo evitás doble reserva?"* → (tras Fase 2) transacción/índice único + explicar el TOCTOU.
- *"¿Qué pasa si Solr o RabbitMQ se caen cuando llega un evento?"* → (tras Fase 3) manual ack + retry + DLQ + backfill/reconciliación.
- *"¿Por qué caché de 2 niveles?"* → L1 local por réplica vs L2 compartida entre las 3 réplicas del users-api.
- *"¿Cómo asegurás la autorización entre servicios?"* → users-api emite el `tipo`, hotels-api lo valida (HMAC-pinned); explicá el *blast radius* de un fallo de auth aquí (por eso Fase 1 importaba).
- *"¿Consistencia fuerte o eventual?"* → CQRS-lite: Mongo es la fuente de verdad (fuerte), Solr es proyección (eventual); explicá el límite de consistencia.

---

## 5. Checklist "portfolio-ready" (definición de terminado)

**Seguridad**
- [ ] Ningún endpoint destructivo o de PII sin auth (S2, C3).
- [ ] Imposible auto-registrarse como admin (S1).
- [ ] `JWT_SECRET` obligatorio; secretos fuera de git (C4, I1).

**Correctitud**
- [ ] No se puede sobre-reservar; disponibilidad determinista (D1–D4).
- [ ] `search-api` no pierde eventos y se recupera de caídas (E1, E3, E5).
- [ ] Búsquedas multi-palabra/vacías funcionan; sin inyección (E2).

**Calidad / DevOps**
- [ ] `go vet` + `go test` verdes en CI, con badge (I4).
- [ ] 3 Dockerfiles multi-stage/no-root; healthchecks (I2, I5).
- [ ] Versión de Go y module paths unificados (I6).

**Presentación**
- [ ] README con GIF + diagrama renderizado + quickstart que funciona + credenciales demo (P3, P7).
- [ ] OpenAPI publicada (P3).
- [ ] Mejor doc en inglés; `PLAN.md`/`RULES.md` borrados; errores fácticos corregidos (P4, P5, P6).
- [ ] `LICENSE` presente (P9).

---

## 6. Segunda pasada — Profundidad y madurez de Backend Engineering (hallazgos nuevos)

> Esta sección se agregó en una **segunda auditoría** (9 agentes en paralelo, una por dimensión de madurez), enfocada específicamente en tu pregunta: *"la arquitectura ya es compleja, ¿necesita más?"*. Todos los hallazgos de abajo son **nuevos** (no repiten la Sección 2) y están **verificados contra el código** con `archivo:línea` o con el `grep`/comando que confirma la ausencia.

### 6.0 Respuesta directa a tu pregunta

**No necesitás MÁS arquitectura. Necesitás más PROFUNDIDAD en la que ya tenés.**

La amplitud (3 microservicios, poliglot persistence, gateway con LB, event-driven, caché multinivel) ya es **más que suficiente** — de hecho es ambiciosa para un portfolio. El riesgo hoy no es "le falta un servicio más", es que **cada capa está resuelta al 60–70%**: la infraestructura existe pero no se ve "operada como un servicio real". Un entrevistador senior no se impresiona con más cajas en el diagrama; se impresiona con **una caja hecha de punta a punta** (con migraciones, índices, timeouts, tests de integración, observabilidad y un dominio que modele el negocio de verdad).

Dos conclusiones concretas que salieron de esta pasada:
1. **El dominio es anémico** — una `Reservation` es un CRUD de 6 campos sin estado, sin cantidad de habitaciones, sin monto, sin eventos. Para un "booking platform" esto es lo primero que un entrevistador va a picar. **Es tu mayor oportunidad de subir el nivel.**
2. **El titular del proyecto es "orquestación / load balancing / escalado horizontal" pero no hay Kubernetes.** Escalar hoy = copiar y pegar YAML + editar nginx a mano. Ese gap contradice directamente el argumento de venta principal.

Sobre el **frontend**: para un rol de *backend*, ya está bien (completo, funcional, con service layer y validación). No inviertas ahí más allá de lo listado en 6.9 — es de baja prioridad.

### 6.1 Resumen — dónde está la aguja por dimensión

| Dimensión | Madurez | Gap titular |
|-----------|:------:|-------------|
| Observabilidad | 🔴 | Sin correlation-ID/logs estructurados/readiness real; Gin en debug en los contenedores |
| Base de datos | 🔴 | Sin migraciones versionadas, **cero índices**, sin pool/timeouts, sin paginación en DB |
| Modelado de dominio | 🔴 | `Reservation` anémica: sin estado/lifecycle, sin cantidad, sin dinero, sin eventos |
| Cloud-native | 🔴 | Sin k8s/Helm (el titular es orquestación); todo `:latest`, sin `.dockerignore`, sin resource limits |
| Diseño de API | 🟡 | Sin versionado, sin envelope de error estándar, sin idempotencia, paginación inconsistente |
| Resiliencia | 🟡 | Sin timeouts/deadlines en la app, sin circuit breaker, sin bulkhead, caché como dependencia dura |
| Testing | 🟡 | Buenos unit tests, pero **cero** integración/contract/e2e; sin `-race` ni coverage gate |
| Seguridad (2ª capa) | 🟡 | Sin escaneo de deps (search-api: 12 CVEs; frontend: 16 npm audit), sin TLS, JWT sin `aud/iss` |
| Calidad / tooling | 🟡 | Sin `golangci-lint`/`go.work`, struct `Hotel` duplicado 4×, contrato de evento duplicado |

### 6.2 Observabilidad & operabilidad
*Hoy los 3 servicios usan `gin.Default()` + `log` stdlib. Se ve como código que nunca se operó como servicio.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| O1 | **Sin correlación de requests end-to-end**: nginx genera `X-Request-ID` pero **ningún servicio Go lo lee, loguea ni propaga** (el hop interno `http.Get` tampoco lo reenvía), y nginx ni siquiera lo pone en su `log_format`. "¿Cómo trazás un request entre servicios?" → hoy no podés. | `nginx.conf:130` (setea header); grep de RequestID/correlation en `*.go` = 0; `hotels_http.go:32` no reenvía headers | Middleware de request-ID por servicio (leer o generar UUID → context → response), reenviarlo en la llamada interna, y agregar `$request_id` al `log_format` + usar `json_combined`. | 🟠 · M |
| O2 | **Sin logging estructurado/por niveles**: niveles simulados como prefijos de string (`log.Printf("warn: ...")`) y mensajes mezclados inglés/español. No se puede filtrar ni ingerir en un agregador. | sin `slog`/`zap`/`zerolog` en go.mod; `users_service.go:219`, `hotels-api/cmd/main.go:109` | Adoptar `log/slog` (stdlib, JSON handler) una vez al arranque; adjuntar `request_id` + nombre de servicio; estandarizar en inglés. | 🟡 · M |
| O3 | **`/health` es solo liveness**: devuelve 200 aunque MySQL/Mongo/RabbitMQ estén caídos. Sin distinción liveness/readiness → el orquestador enruta tráfico a instancias rotas. | `hotels-api/cmd/main.go:99-105`, `users-api/cmd/main.go:68-74` (200 hardcodeado, sin Ping) | `/livez` (barato) + `/readyz` (pinguea dependencias, 503 con mapa de estado). Enganchar el healthcheck del compose/k8s a `/readyz`. | 🟡 · M |
| O4 | **Gin corre en modo debug en los contenedores**: imprime el dump de rutas y el warning en cada boot; overhead por request. Señal clara de "nunca corrió en prod". | sin `gin.SetMode`/`GIN_MODE` en todo el repo | `gin.SetMode(gin.ReleaseMode)` al arranque o `GIN_MODE=release` en cada Dockerfile. | 🟡 · S |

### 6.3 Base de datos & persistencia
*La dimensión que un entrevistador backend pica más fuerte, y una de las más flojas hoy.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| DB1 | **Sin migraciones versionadas**: el esquema se crea con `AutoMigrate` (nunca dropea/renombra columnas, no hay rollback ni historial). Red flag inmediato. | `users_mysql.go:32-36, 50-54`; no hay dir de migraciones | `golang-migrate`/`goose` con archivos up/down numerados en el repo; `AutoMigrate` solo para dev local. Documentar el flujo. | 🟠 · M |
| DB2 | **Cero índices secundarios**: reservas se consultan por `hotel_id`/`user_id` sin índice, y `IsHotelAvailable` hace **una aggregation por noche** de la estadía → `O(días × documentos)`, todo full scan. | grep `CreateIndex`/`IndexModel` = 0; `hotels_mongo.go:271/287/303` (Find sin índice), `:399-479` (loop por día) | Índice compuesto Mongo `{hotel_id, check_in, check_out}` + `{user_id}`; colapsar el loop por noche en una sola aggregation por rango. | 🟠 · M |
| DB3 | **Sin pool de conexiones ni timeouts de driver**: GORM con `sql.DB` por defecto (conexiones ~ilimitadas, sin lifetime); cliente Mongo sin `MaxPoolSize`/`ServerSelectionTimeout` y conecta con `context.Background()`. | grep `SetMaxOpenConns`/`SetMaxPoolSize` = 0; `hotels_mongo.go:50-53` | `db.DB()` → `SetMaxOpenConns/MaxIdleConns/ConnMaxLifetime`; en Mongo `SetMaxPoolSize` + `SetServerSelectionTimeout` + `context.WithTimeout` al conectar. | 🟡 · S |
| DB4 | **`users-api` nunca propaga `context` a la DB**: los métodos del repo no reciben `ctx`; ninguna llamada GORM usa `.WithContext(ctx)`. Un query lento no se cancela al desconectarse el cliente. (Distinto del gap ya conocido del cliente HTTP de search.) | grep `WithContext` en `users-api` = 0; `users_mysql.go:61,69,80,91,98,105` | Enhebrar `ctx` desde el handler → service → repo y usar `db.WithContext(ctx)` con timeout por query. | 🟡 · M |
| DB5 | **Sin paginación a nivel DB**: `GET /users` materializa toda la tabla; las listas de reservas cargan el resultado completo en memoria. Crecen sin límite. | `users_mysql.go:61-66` (Find sin Limit/Offset); `hotels_mongo.go:271/287/303` | `db.Limit().Offset()` / `options.Find().SetLimit().SetSkip()` + devolver `total`. | 🟡 · M |
| DB6 | **Solr hace `Commit()` duro por cada documento**: fuerza un nuevo searcher y flush de segmento en cada write, matando el throughput y anulando el `autoCommit`/`autoSoftCommit` ya configurado. | `hotels_solr.go:83,134,166` vs `solrconfig.xml:21-27` | Quitar el commit explícito y confiar en `autoSoftCommit`; usar `commitWithin` si hace falta visibilidad inmediata. | 🟡 · S |
| DB7 | **Schema Solr con tipos inapropiados y sin `copyField`**: `phone`/`email`/`name` como `text_general` (no hay match exacto); sin catch-all `_text_`; strings sin `docValues`. | `schema.xml:5-19`; sin `<copyField>` | `string` para phone/email, `copyField` a un `_text_`, `docValues` en campos facetables. | ⚪ · S |

### 6.4 Diseño de API & contratos
*Superficie HTTP funcional pero "hecha a mano por handler", sin disciplina de contrato.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| A1 | **Sin envelope de error estándar** + fuga de errores internos: `hotels`/`search` devuelven `fmt.Sprintf("...: %s", err.Error())` (filtra texto de Mongo/Solr al cliente); `users` devuelve genéricos; nginx usa otra forma. Ningún error lleva `code`/`trace_id`. | `hotels_controller.go:49,73,104,174`; `search_controller.go:39,48,56`; `nginx.conf:426` | Un tipo/middleware de error por servicio con envelope estable (idealmente `application/problem+json`); nunca `err.Error()` al body. | 🟠 · M |
| A2 | **Sin versionado de API**: rutas sin `/v1`. Corregir el typo `AvaiableRooms` rompe a todo consumidor en silencio. | rutas en los 3 `main.go`; grep `/v1` = 0 | Prefijo `/api/v1` (RouterGroup de gin) + ruteo en nginx; documentar la estrategia. | 🟡 · M |
| A3 | **Sin idempotencia en POST inseguros** (`/reservations`, `/hotels`): un doble-click crea reserva duplicada. El `X-Request-ID` que setea nginx ni se usa para dedup. (Complementa D1: esto es la capa de contrato `Idempotency-Key`.) | `hotels_controller.go:135-183, 59-82`; grep `idempoten` = 0 | Aceptar header `Idempotency-Key` en POST, persistir la primera respuesta por esa clave y devolverla en replays. | 🟡 · M |
| A4 | **Paginación inconsistente**: `/search` **exige** `offset`+`limit` (400 con fuga `strconv.Atoi: parsing ""` si faltan) mientras las demás listas son ilimitadas. | `search_controller.go:35-50` vs `users_controller.go:39-49`, `hotels_controller.go:242-298` | Una sola convención (`?page&size` o `?limit&offset`) con defaults y clamp en todos los list endpoints; envelope con `total`. | 🟡 · M |
| A5 | **Envelopes de respuesta inconsistentes**: mezcla de arrays desnudos, mapas, objetos y wrappers ad-hoc (`{"id":..}` vs `{"message":..}`). | `hotels_controller.go:110,131,232` vs `:79,180`; `search_controller.go:62` | Un envelope único (`{"data":.., "meta":..}`); nunca arrays/maps top-level. | 🟡 · M |
| A6 | **Semántica HTTP floja**: 201 sin header `Location`; `PUT`/`DELETE` devuelven 200 + `{message:id}` en vez de 204; `204` nunca se usa. | `hotels_controller.go:79,110,131,180`; grep `StatusNoContent` = 0 | 201 + `Location` en create; 204 en delete/put sin body; PUT devuelve la representación. | 🟡 · S |
| A7 | **`user_id` cambia de tipo entre servicios**: `int64` en users-api, `string` en hotels-api. Un generador de SDK/OpenAPI lo marca al instante. | `users_domain.go:13,21` vs `reservations.go:9` | Unificar el tipo de `user_id` en toda la plataforma y fijarlo en un contrato compartido. | ⚪ · S |
| A8 | **Sin content negotiation**: todo `ctx.JSON` sin mirar `Accept`. | grep `Negotiate`/`MIMEJSON` = 0 | Documentar JSON-only o usar `gin.Negotiate`; al menos `Content-Type` correcto en errores. | ⚪ · S |

### 6.5 Resiliencia & tolerancia a fallos
*Hay scaffolding real (LB, reconexión de RabbitMQ), pero la capa de aplicación casi no tiene tolerancia a fallos por request.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| R1 | **`users-api` no tiene timeout en NINGÚN nivel** (ctx/GORM/driver). El login vive acá → si MySQL se cuelga, **todas las réplicas se cuelgan para siempre**. | interfaz sin `ctx` (`users_service.go:16-21`); DSN `users_mysql.go:40` sin timeouts | `ctx` en la interfaz + `WithContext(ctx)` + `readTimeout/writeTimeout` en DSN + deadline por request. Fallar rápido con 503. | 🟠 · M |
| R2 | **Sin deadlines en Mongo/Solr; el consumer usa `context.Background()`**: la ruta de indexación (no pasa por nginx) no tiene nada que la acote. | `hotels_mongo.go:50` (sin ServerSelection/Socket timeout); `search_service.go:81,111,119,129` | `context.WithTimeout` (2–5s) en cada llamada saliente; `SetServerSelectionTimeout`/`SetSocketTimeout`; context con deadline en el consumer. | 🟡 · M |
| R3 | **Caché como dependencia dura en lectura**: `GetHotelByID` y 3 getters de reservas devuelven **error** si falla el `cacheRepository.Create` *después* de que la DB ya respondió OK → convierte lecturas sanas en 500. Incoherente con `GetReservationByID` que sí loguea y sigue. | `hotels_service.go:57-58, 283, 316, 349` vs `:237-240` | Poblar la caché en lectura siempre best-effort (log-and-continue). Una escritura de caché nunca debe fallar una lectura ya resuelta. | 🟡 · S |
| R4 | **Fan-out sin bulkhead + fail-hard**: `GetAvailability` lanza 1 goroutine por hotel **sin límite** (× 1 aggregation por noche) contra un pool default ~100 → riesgo de auto-DoS; y aborta todo el batch si un solo hotel falla. | `hotels_mongo.go:342-351, 357-359` | Acotar con `errgroup.SetLimit`/semáforo; colapsar el N+1 por noche; devolver disponibilidad parcial en vez de abortar. | 🟡 · M |
| R5 | **Sin circuit breaker ni retry client-side** en llamadas HTTP entre servicios ni en DB. Un `hotels-api` lento/inestable se martilla en cada evento sin fast-fail. | grep `breaker`/`gobreaker` = 0; único retry es la conexión de RabbitMQ | Circuit breaker (`sony/gobreaker`) alrededor del repo HTTP inter-servicio + retry con jitter para errores transitorios. | ⚪ · M |

### 6.6 Testing (profundidad)
*Buenos unit tests, pero la historia se corta justo donde el entrevistador empieza a preguntar.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| T1 | **Sin tests de integración contra dependencias reales** (testcontainers/dockertest): el punto de venta es Mongo+Solr+MySQL+RabbitMQ juntos, pero ningún test toca un datastore real. El código más propenso a bugs (query de fechas Mongo, query Lucene, RabbitMQ) solo corre en prod. | grep `testcontainers`/`dockertest` = 0; sin `*_test.go` en repos/dao | Suite `//go:build integration` con testcontainers-go: repos reales, round-trip de indexado Solr y publish→consume RabbitMQ. Job de CI aparte. | 🟠 · L |
| T2 | **Sin contract test productor↔consumidor**: el evento `HotelNew` está duplicado como 2 structs independientes; renombrar un tag rompe la indexación en runtime **sin fallar ningún test**. | `hotels-api/.../hotels_domain.go:24` == `search-api/.../hotels_domain.go:24`; sin pact | Contract test: serializar el `HotelNew` del productor y afirmar que el consumidor lo deserializa (golden JSON), o extraer a módulo compartido. | 🟠 · M |
| T3 | **Sin e2e; `test_load_balancer.sh` no afirma nada**: no tiene `set -e` ni `exit 1`, siempre sale 0 → no puede vivir en CI ni atrapar regresiones. | `test_load_balancer.sh:255-300`; sin `httptest.NewServer` en tests | Endurecerlo (`set -euo pipefail`, acumular fallos, exit ≠0) o un test e2e con `docker compose up` que afirme register→login→search→book. | 🟡 · M |
| T4 | **Sin coverage gate ni `-race`**: hay goroutines (fan-out, consumer, mock con mutex) pero los tests nunca corren con `-race`. | README `go test ./... -v`; sin `-race`/`-cover`/codecov | `go test -race -coverprofile=... ./...` por módulo en CI + badge de coverage. | 🟡 · S |
| T5 | **Rutas negativas de auth y el minter de JWT sin test**: `GenerateToken` solo se ejercita vía mock; el middleware no tiene casos de token expirado/firma alterada/`alg=none`/header malformado. | sin test en `tokenizers/`; `auth.go` sin test directo | Tests de `GenerateToken` (round-trip de claims) + middleware negativo (exp vencido, key errónea, `alg=none`, header malo → 401). | 🟡 · S |
| T6 | **Sin fixtures/golden files ni benchmarks**: todo dato de test es inline. | grep `func Benchmark`/`Fuzz` = 0; sin `testdata/` | Un `testdata/` reutilizable + 1–2 `Benchmark` en hot paths (cache-aside, serialización de search). | ⚪ · S |

### 6.7 Cloud-native & deployment
*Solo docker-compose, single-node. El titular del proyecto es orquestación — y no hay orquestador.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| CN1 | **Sin Kubernetes/Helm**: "escalar" = duplicar a mano un servicio en compose **y** editar el upstream de nginx **y** reiniciar. Contradice el argumento de venta principal (LB/escalado horizontal). **El agregado de mayor impacto de todo el repo.** | sin dir k8s/helm; `LOAD_BALANCER.md:120-146` ("Duplicar... Actualizar nginx.conf... Reiniciar"); `docker-compose.yml:125-208` (3 bloques copy-paste) | `k8s/` (o Helm) con Deployments+Services para los 3 APIs, StatefulSets/managed para las DBs, **HPA** en users-api, y probes readiness/liveness. Reemplazar las 3 réplicas copy-paste + upstreams estáticos por un Deployment escalado con `kubectl scale`/HPA. | 🟠 · L |
| CN2 | **Sin `.dockerignore`**: `COPY . .` mete `PLAN.md`, `RULES.md`, seeds, tests Bruno en el contexto; en el frontend el `node_modules` del host puede sombrear el `npm ci` del contenedor. | `find .dockerignore` = 0; `hotels-api/dockerfile`, `search-api/Dockerfile`, `frontend/Dockerfile` | `.dockerignore` por servicio (node_modules, dist, `*.md`, `.git`, Bruno, seeds). | 🟡 · S |
| CN3 | **Sin límites de CPU/memoria en compose**: no se puede demostrar failover bajo carga ni "fair sharing"; un contenedor puede ahogar al host. Justo lo que el entrevistador pregunta sobre el claim de escalado. | grep `deploy:`/`limits:`/`mem_limit` en compose = 0 | `deploy.resources.limits/reservations` (o `mem_limit`/`cpus`) en los 3 APIs y las DBs; llevar lo mismo a k8s como requests/limits. | 🟡 · S |
| CN4 | **Sin estrategia de tags de imagen ni CD/supply-chain**: todo `:latest`, bases con tags flotantes, sin scan (Trivy), sin push a registry, sin multi-arch. | `docker-compose.yml:129,155,183,217,253`; sin `@sha256` | Tag por git-SHA + semver, pin de base por digest, step de Trivy, buildx multi-arch, push a GHCR; k8s referencia el tag inmutable. | 🟡 · M |

### 6.8 Seguridad (2ª capa, más allá de authN/authZ)
*Los fundamentos son decentes (bcrypt bien, GORM parametrizado, Mongo con `bson.M`, login anti-enumeración). Falta la "segunda capa".*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| SD1 | **Sin escaneo de dependencias — y hay CVEs reales**: `govulncheck` en search-api reporta **12 vulnerabilidades** (p.ej. `x/net@0.10.0` → HTTP/2 rapid-reset GO-2023-2102); `npm audit` en frontend reporta **16** (7 high, path-traversal en rollup/vite). | corrido en vivo; sin `.github/workflows` | Job de CI con `govulncheck ./...` por módulo + `npm audit --audit-level=high`; bump de gin/x/net en search-api; `npm audit fix`; Dependabot/renovate. | 🟠 · M |
| SD2 | **Sin política de contraseña** (complementa C8): registrar con password `"1"` funciona; además bcrypt trunca en silencio a 72 bytes sin guarda. | `users_service.go:86-92, 247-258` | Mínimo de longitud (≥8) + rechazo de inputs largos (o pre-hash SHA-256 para el límite de 72B). | 🟡 · S |
| SD3 | **JWT sin `aud`/`iss`/`nbf` y secreto único compartido**: un token de un servicio lo acepta cualquier otro sin scoping de audiencia. | `tokenizers_jwt.go:38-44`; `auth.go:37-48`; `docker-compose.yml:139,165,193,235` | Emitir y validar `iss`/`aud` (y `nbf`) con `jwt.WithIssuer`/`WithAudience` + leeway, para scopear tokens por servicio. | 🟡 · S |
| SD4 | **Sin TLS/HTTPS**: credenciales de login y JWTs viajan en texto plano (`listen 80`). | `nginx.conf:111,435`; grep `ssl`/`443` = 0 | Terminar TLS en nginx (self-signed local, cert real en prod) + redirect HTTP→HTTPS + HSTS; documentarlo en "productionize". | 🟡 · M |
| SD5 | **Los security headers de nginx se descartan en las respuestas de API**: por las reglas de herencia de `add_header`, definirlos a nivel `server` y luego tener `add_header` (CORS) en cada `location` hace que los headers de seguridad **nunca lleguen** a ninguna ruta de API. Gotcha real de nginx. | `nginx.conf:115-118` (server) vs `add_header` en cada location `:164-269+` | Mover los security headers a un snippet re-incluido en cada location; verificar con `curl -I` contra una ruta de API real. | ⚪ · S |

### 6.9 Calidad de código, tooling y arquitectura
| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| CQ1 | **Sin `golangci-lint` y `gofmt` sucio en 7 archivos** (no 2): no hay linter config, la señal #1 de disciplina en Go. | `find *golangci*` = 0; `gofmt -l` lista 7 archivos en los 3 módulos | `.golangci.yml` raíz (gofmt/goimports/govet/staticcheck/errcheck/ineffassign/unused) + `gofmt -w` + engancharlo al CI. | 🟠 · S |
| CQ2 | **Sin módulo compartido: `Hotel` duplicado 4× y el contrato de evento `HotelNew` duplicado en ambos extremos** del pipeline. Renombrar un campo en un lado rompe RabbitMQ **sin error de compilación**. | `hotels_domain.go:5-22`/`dao:5-22` × ambos servicios; `HotelNew` en `hotels_domain.go:24-27` × 2 | Módulo Go compartido (`platform-contracts`) con el tipo `Hotel` y el evento `HotelNew` como única fuente de verdad; ambos servicios lo importan. | 🟠 · L |
| CQ3 | **Sin `go.work`** para el monorepo de 3 módulos: no se puede `go build/test ./...` en una sola pasada. | `find go.work*` = 0 | `go.work` raíz con `use ./hotels-api ./search-api ./users-api` (+ el módulo compartido). | 🟡 · S |
| CQ4 | **Layering/paquetes inconsistentes entre servicios** + **3 implementaciones divergentes de CORS**: `hotels-api` tiene `internal/services/` plano y `package middleware` en `internal/middlewares/` (nombre≠dir); los otros anidan por dominio y ponen CORS en `internal/utils`. | `hotels_service.go:1`, `auth.go:2` vs `search/`/`users/`; CORS en `main.go:60-67` vs 2 `utils.CorsMiddleware` distintos | Elegir una convención (`internal/services/<dominio>`, un paquete de middleware) y converger CORS en el módulo compartido. | 🟡 · M |

### 6.10 Modelado de dominio & Frontend
*El dominio anémico es, para un rol backend, la dimensión más limitante — y la de mayor upside.*

| # | Hallazgo | Evidencia | Fix | Sev · Esf |
|---|----------|-----------|-----|-----------|
| DM1 | **`Reservation` sin estado/lifecycle**: cancelar es un `DeleteOne` (hard delete) → no hay historial, ni `CancelledAt`, ni máquina de estados. Es CRUD, no diseño de dominio. "Modelá el ciclo de vida de la reserva" es *la* primera pregunta de dominio. | `reservations.go:5-12` (sin `Status`); `hotels_mongo.go:248-266` (DeleteOne) | Campo `Status` con enum + transiciones (pending/confirmed/checked-in/completed/cancelled); cancelación = cambio de estado (soft) + `CancelledAt`; `CreatedAt`/`UpdatedAt`. | 🟠 · M |
| DM2 | **`Reservation` sin cantidad ni dinero**: no guarda nº de habitaciones/huéspedes (la disponibilidad asume 1 hab/reserva) ni precio/total/moneda → no se puede emitir confirmación/factura, y re-precificar un hotel reescribe el valor de reservas históricas. | `reservations.go:5-12`, `dao:24-31`; `PricePerNight float64` solo en Hotel | Agregar `NumRooms`/`NumGuests` y `TotalPrice`+`Currency` snapshoteados al crear (dinero como **entero en unidades mínimas**, no `float64`); validar cantidad contra `AvailableRooms`. | 🟠 · M |
| DM3 | **`Hotel.Rating` existe pero no hay entidad `Review`**: el rating es un número que tipea el admin, sin fuente de verdad. Un `Review` (crear → recomputar promedio) es una forma linda de mostrar un 2º agregado + read model derivado. | `hotels_domain.go:16`; `hotels_service.go:98,141`; grep `review` = 0 | Entidad `Review` (user_id, hotel_id, score, comment, created_at) con endpoints; derivar `Hotel.Rating` como promedio. | 🟡 · L |
| DM4 | **Sin concepto de pago/total de booking**: las reservas son gratis. Aun stubeado (`authorize→capture→refund`) habilita hablar de idempotencia, manejo de dinero y saga/compensación en cancelación. | grep `payment`/`refund`/`invoice` = 0 | Flujo de pago/total stubeado atado al estado de reserva (`PENDING_PAYMENT→CONFIRMED`, refund-on-cancel), montos en unidades mínimas. | 🟡 · M |
| DM5 | **Struct `ReservationNew` muerto**: se definió el evento pero **nunca se publica** — solo el CRUD de hotel emite eventos. En una plataforma event-driven, los eventos de reserva son justo lo que emitirías (confirmaciones, sync de disponibilidad a search, analytics). | `reservations.go:14-17` (definido); publicadores = solo `HotelNew` (`hotels_service.go:117,161,195`) | Publicar `ReservationNew` (CREATED/CANCELLED) desde el service de reservas, o borrar el struct muerto. Da a search updates de disponibilidad en tiempo real. | 🟡 · S |
| DM6 | **Sin modelado de `Room`/room-type**: un hotel es un solo `PricePerNight` + un solo `AvaiableRooms`. Room types (single/double/suite) con capacidad y precio propios abren la lógica de disponibilidad/pricing. | `hotels_domain.go:15-17`; grep `room_type` = 0 | Modelar `RoomType` (nombre, capacidad, precio, inventario) como colección del Hotel y clavar disponibilidad/reservas por tipo. | 🟡 · L |
| DM7 | **`User` anémico**: solo id/username/password/tipo → no se puede mandar un email de confirmación ni guardar nombre. `Tipo` como string libre es *primitive obsession*. | `users_dao.go:3-8`, `users_domain.go:12-16`; grep `email`/`created_at` = 0 | Agregar `Email` (validado), display name y `CreatedAt`; rol como constante tipada. | 🟡 · M |
| FE1 | **Frontend en JSX plano, sin TypeScript** (usa JSDoc + `@types/react` pero sin `typescript` ni `tsconfig`). Señal de madurez reconocible, pero **baja prioridad para un rol backend**. | sin `tsconfig*`; sin `typescript` en package.json | Opcional: migrar a TS. Para portfolio backend, dejarlo con JSDoc está OK. | ⚪ · L |
| FE2 | **Sin code-splitting/lazy por ruta**: todas las páginas (incl. admin) se importan estáticas → un solo chunk. Barato pero señal casi nula para backend. | `App.jsx:16-22`; grep `React.lazy`/`Suspense` = 0 | Si tocás el front: `React.lazy` + `Suspense` (al menos el bundle admin). Si no, saltear. | ⚪ · S |

### 6.11 Roadmap actualizado (integrando la 2ª pasada)

Las Fases 1–3 de la Sección 3 **no cambian** (siguen siendo lo primero: seguridad + correctitud + robustez de search). Estas fases nuevas se agregan después, ordenadas por ROI para un rol backend:

- **Fase 7 — Persistencia & Observabilidad (≈1.5 días) 🔴** — el mayor salto de "parece código de estudiante" a "servicio operado": migraciones versionadas (DB1) + índices (DB2) + pool/timeouts (DB3, R1, R2) + `context` a la DB (DB4) + paginación (DB5); logs estructurados con `slog` + correlation-ID (O1, O2) + `/readyz` real (O3) + Gin release mode (O4).
- **Fase 8 — Contratos de API & Resiliencia (≈1 día) 🟡** — envelope de error estándar (A1) + versionado `/api/v1` (A2) + idempotencia en POST (A3) + paginación/envelope consistentes (A4, A5, A6); caché best-effort en lectura (R3) + bulkhead en el fan-out (R4) + circuit breaker (R5).
- **Fase 9 — Testing de integración & Cloud-native (≈2 días) 🔴** — testcontainers (T1) + contract test del evento (T2) + `-race`/coverage en CI (T4) + tests negativos de auth (T5); **k8s/Helm con HPA** (CN1) + `.dockerignore` (CN2) + resource limits (CN3) + tags/scan de imagen (CN4); `govulncheck`+`npm audit`+Dependabot (SD1) + `golangci-lint`+`go.work` (CQ1, CQ3).
- **Fase 10 — Dominio más rico (≈2 días) ⭐ — el mayor diferenciador** — estado/lifecycle de `Reservation` (DM1) + cantidad+dinero (DM2) + publicar `ReservationNew` (DM5); módulo de contratos compartido (CQ2). Opcional pero muy vendible: `Review` (DM3), pago stub (DM4), room-types (DM6), `User` con email (DM7).

### 6.12 Top 10 de mayor impacto (combinando ambas pasadas)

Si querés una sola lista priorizada de "qué mueve más la aguja para un rol backend":

1. **Cerrar la seguridad** (S1, S2) — *bloqueante*.
2. **Enriquecer el dominio de `Reservation`** (DM1, DM2, DM5) — es lo que convierte "CRUD" en "diseño de sistema".
3. **No permitir overbooking, de forma atómica** (D1) — la invariante central del negocio.
4. **Persistencia real**: migraciones + índices + timeouts/pool (DB1, DB2, DB3, R1).
5. **Kubernetes/Helm con HPA** (CN1) — hace verdadero el titular de "orquestación/escalado".
6. **Tests de integración + contract test del evento** (T1, T2) — prueba que la arquitectura *funciona junta*.
7. **Observabilidad**: correlation-ID + logs estructurados + `/readyz` (O1, O2, O3).
8. **Contratos de API**: envelope de error + versionado + idempotencia (A1, A2, A3).
9. **CI con quality gates**: `go test -race`, `golangci-lint`, `govulncheck`, `npm audit` (T4, CQ1, SD1).
10. **Endurecer `search-api`** (E1, E2, E3 de la Sección 2) + módulo de contratos compartido (CQ2).

> **Nota de alcance:** esto es un *menú*, no un mandato. Hacer las Fases 1–3 + los ítems 1–6 de este top ya te deja un proyecto que **se defiende sobresalientemente** en una entrevista de backend. El resto es diferenciación incremental.

---

## 7. Guías de implementación (playbooks verificados paso a paso)

> Tercera pasada. Pregunta que la disparó: *"¿el plan es realmente implementable? Faltaba detallar más algunas soluciones."* **Tenías razón.** Convertí los 8 paquetes de trabajo de mayor valor en playbooks concretos, cada uno **leído y verificado contra el código real** (tipos, firmas, module paths, números de línea). El resultado honesto:
>
> **De 8 paquetes: 7 quedaron "implementable pero sub-especificado" y 1 tenía el fix directamente equivocado.** Ninguno era "copiar y pegar la línea del plan". Cada uno tenía dependencias ocultas, orden que importa, o un caso donde el one-liner original no compila / no funciona / rompe otra cosa.

### 7.0 Correcciones al plan (donde el fix de una línea estaba mal, bloqueado o incompleto)

Esto es lo más importante de esta pasada — los lugares donde seguir el plan literal te habría hecho perder tiempo o roto algo:

| Ítem | Lo que decía el plan | El problema real | La corrección |
|------|----------------------|------------------|---------------|
| **D1** 🔴 | "hacerlo atómico con **transacción Mongo** o índice único" | El compose corre **Mongo standalone** → `WithTransaction` **falla** ("Transaction numbers are only allowed on a replica set"). Y aunque fuera replica set, **una transacción sola NO evita overbooking**: dos `InsertOne` concurrentes en `reservations` nunca colisionan (no hay escritura en conflicto). | Usar **contador de inventario por hotel-noche** `{hotel_id,date,booked,capacity}` con **índice único `{hotel_id,date}`** y `findOneAndUpdate` atómico por noche + **compensación** si una noche está llena. Es race-safe en standalone (cada `findOneAndUpdate` es atómico a nivel de 1 documento). Sin replica set, sin transacción. |
| **DB4/R1** 🟠 | "enhebrar `context` en users-api" citando líneas de `users_mysql.go` | Es un **refactor cross-cutting** mucho más amplio: cambia la interfaz `Repository`, **los 4 implementadores** (mysql/cache/memcached/mock), los métodos del service + helpers de caché, la interfaz `Service` del controller, los handlers, y **ambos archivos de test**. Además ccache/memcached **no tienen API con ctx** → aceptan-e-ignoran. | Tratarlo como refactor de interfaz completo (lista de edición ordenada). Solo MySQL honra cancelación; documentar que L1/L2 ignoran ctx a propósito para que no parezca bug. |
| **P7** 🟡 | "script de seed en Mongo initdb que también siembre admin+cliente" | **Imposible**: los usuarios viven en **MySQL** (users-api), no en Mongo — un init de Mongo no puede escribir filas MySQL. Y un SQL en `/docker-entrypoint-initdb.d` de MySQL corre **antes** de que la app cree la tabla `users`. | Partir P7: (a) Mongo initdb siembra hoteles+índices; (b) admin+cliente demo vía **migración golang-migrate 0002** (corre después de crear la tabla) con un **hash bcrypt generado con la lib de la app** (htpasswd usa `$2y$` que la lib Go rechaza). |
| **S1** 🔴 | "forzar tipo=cliente; crear admins con endpoint `AdminOnly`" | Forzar `tipo` dentro de `service.Create` **también rompería el seed**. Y un endpoint admin bajo `/admin` es **inalcanzable**: nginx rutea `/admin/*` a **hotels-api**, no users-api. Además el test actual `TestController_Create` **codifica la vulnerabilidad** y se pondrá rojo. | Forzar `tipo="cliente"` **solo en el handler público** (`Controller.Create`), dejando `service.Create` intacto para el seed. Crear admin vía seed env-driven (cero cambios de nginx). Actualizar los 3 subtests de `users_controller_test.go`. |
| **C4** 🟡 | "`log.Fatal` si `JWT_SECRET` vacío o placeholder" | `getEnv` devuelve el default si la env var está vacía → `JWTKey` **nunca** es `""`, siempre es el placeholder literal. El guard tiene que comparar contra el **valor placeholder**, no `""`. Y el mismo default peligroso está en **hotels-api** (comparten secreto). | Guard `== "" || == "your-secret-key-change-in-production"` en **ambos** servicios. |
| **DM5** 🟡 | "publicar el evento `ReservationNew`" | El `Queue.Publish` actual está atado a la cola `hotels-news`, que **search-api consume esperando `HotelNew`**. Publicar `ReservationNew` ahí **rompe** el `json.Unmarshal` del consumidor. | Método `PublishReservation` separado sobre una **cola distinta** (`reservations-news`). |
| **E1** 🟠 | "manual ack + Nack(requeue) + DLQ" | Re-declarar la cola `hotels-news` con args de dead-letter nuevos da **406 (PRECONDITION_FAILED)** contra la declaración sin-args del productor. | Coordinar los args de declaración entre productor y consumidor, o declarar la DLX/DLQ aparte; `docker compose down` limpia la cola efímera para re-declarar. |
| **A2** 🟡 | "prefijo `/api/v1` + ruteo en nginx" | Rompe el **frontend** (base URL + proxy Vite) y hay que cambiar **tanto la location como el `proxy_pass` target** en cada bloque nginx, **incluidas las 2 regex** de reservas-por-usuario; health debe quedar **sin versionar**. | Versionar todo salvo `/health`; actualizar frontend en el mismo cambio, o (menor riesgo) versionar solo el target del `proxy_pass`. |
| **CN1** 🔴 | "probes readiness/liveness" en k8s | Los probes asumen `/readyz` y `/livez` que **no existen** (solo hay un `/health` liveness-only). CN1 **depende de O3**. Además hotels-api **ignora `PORT`** (hardcodea `:8081`) → ponerlo en el ConfigMap es no-op. | Implementar O3 primero (o apuntar ambos probes a `/health` como interino). Fijar `containerPort: 8081` para hotels-api sin depender de `PORT`. |
| **CN4** 🟡 | "pin de bases por digest + buildx multi-arch" | Pinear un digest **por-arquitectura** rompe el build multi-arch (ese digest existe para 1 arch). Y `hotels-api/dockerfile` es **minúscula** → un matrix con `file: ./<svc>/Dockerfile` falla. | Pinear el digest del **manifest-list** (`docker buildx imagetools inspect`). Renombrar `dockerfile`→`Dockerfile`. |
| **CQ1** 🟡 | ".golangci.yml con gofmt/goimports como linters" | golangci-lint es **v2** ahora: `gofmt`/`goimports` son **formatters**, no linters; el schema v1 plano **no carga**. | Schema v2: `version:"2"`, linters bajo `linters.enable`, formatters bajo `formatters.enable`. |
| **SD1** 🟠 | "govulncheck en CI como gate que pasa" | search-api tiene **12 CVEs hoy** → el gate se pone **rojo inmediatamente**; y en modo workspace `go install ...@latest` está **prohibido**. | `continue-on-error` en la leg de search-api hasta bumpear gin/x/net (E1); instalar govulncheck con `GOWORK=off`. |
| **A1** 🟠 | "reemplazar los sitios de fuga [4 líneas]" | Son **~15 sitios** en hotels-api (no 4) + search-api filtra `strconv` y texto de Solr; y falta `error_page` 5xx en nginx (upstream caído devuelve HTML default, no el envelope). | Lista completa de ~18 sitios; agregar `error_page 500 502 503 504` con body JSON que matchee el envelope. |

**Lectura:** el plan (Secciones 1–6) sigue siendo un diagnóstico correcto — pero para *ejecutar*, usá los playbooks de abajo, no las líneas de una sola frase de la Sección 2/6.

### 7.1 Auth hardening (S1, S2, C4) — verdict: sub-especificado · esfuerzo M
**Prereqs:** `golang-jwt/jwt/v5` ya es dependencia de users-api (no hace falta `go get`). Las réplicas de users-api no exponen puertos → verificar todo por el gateway `:80`.
**Pasos:**
1. **Crear `users-api/internal/middlewares/auth.go`** (`package middleware`): copiar `Authenticate` + `AdminOnly` de hotels-api **verbatim** y agregar `OwnerOrAdmin()` (nuevo — hotels-api no lo tiene):
   ```go
   func OwnerOrAdmin() gin.HandlerFunc {
     return func(c *gin.Context) {
       if v, _ := c.Get("userType"); v == "administrador" { c.Next(); return }
       if id := c.GetString("userID"); id == "" || id != c.Param("id") {
         c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you can only access your own account"}); return
       }
       c.Next()
     }
   }
   ```
   *Gotcha:* `userID` en el contexto es **string**; comparar contra `c.Param("id")` (string), no contra el int64 que parsea el controller.
2. **Forzar rol en registro público** — en `Controller.Create`, justo tras `ShouldBindJSON`: `request.Tipo = "cliente"`. **No** tocar `service.Create` (el seed lo necesita para crear admin).
3. **Config**: agregar `SeedAdminUsername`/`SeedAdminPassword` (env `ADMIN_USERNAME`/`ADMIN_PASSWORD`).
4. **`cmd/main.go`**: (a) fail-fast `if config.JWTKey == "" || config.JWTKey == "your-secret-key-change-in-production" { log.Fatal(...) }`; (b) grupos de rutas:
   ```go
   router.POST("/users", controller.Create); router.POST("/login", controller.Login) // públicos
   auth := router.Group("/", jwtMiddleware.Authenticate())
   auth.GET("/users", middleware.AdminOnly(), controller.GetAll)
   auth.GET("/users/:id", middleware.OwnerOrAdmin(), controller.GetByID)
   auth.DELETE("/users/:id", middleware.OwnerOrAdmin(), controller.Delete)
   ```
   (c) seed idempotente (swallow duplicados — las 3 réplicas corren el seed; el índice único de username hace fallar a 2).
5. **Arreglar tests** (`users_controller_test.go`): los 3 subtests de `Create` esperan ahora `Tipo:"cliente"`; renombrar "create admin→201" a "public register cannot self-assign admin".
6. **Compose + frontend**: `ADMIN_USERNAME`/`ADMIN_PASSWORD` en las 3 réplicas; quitar la opción "Administrator" de `Register.jsx`; actualizar README/Bruno.
**Verificar:** `curl -X POST localhost/users -d '{...,"tipo":"administrador"}'` → login → el JWT decodificado muestra `"tipo":"cliente"`; `curl -i localhost/users` sin token → 401; `DELETE localhost/users/1` sin token → 401; con token de otro user → 403.

### 7.2 No-overbooking atómico + dominio de reserva (D1, D4, DM1, DM2, DM5) — verdict: sub-especificado (fix D1 corregido) · esfuerzo L
**El más importante y el más mal-especificado del plan original.** Recomendación: **contador de inventario por noche** (Opción B), no transacciones.
**Pasos:**
1. **Modelos** (`hotels_dao.go`): extender `Reservation` con `Status,NumRooms,NumGuests,TotalPrice int64,Currency,CreatedAt,CancelledAt *time.Time`; nuevo `Inventory{HotelID,Date,Booked,Capacity}`; constantes `StatusConfirmed/StatusCancelled`. En el dominio agregar el sentinel `var ErrNoAvailability`.
2. **Índice único** en `NewMongo`: `CreateOne` sobre `{hotel_id:1, date:1}` con `SetUnique(true)` en la colección `reservation_inventory` (env `MONGO_COLLECTION_INVENTORY`). *Sin* cambios de connection string ni replica set.
3. **Claim atómico + compensación** en `CreateReservation` del repo:
   ```go
   func (r Mongo) claimNight(ctx, c, hotelID, day string, rooms, cap int) (bool, error) {
     filter := bson.M{"hotel_id":hotelID,"date":day,"booked":bson.M{"$lte":cap-rooms}}
     update := bson.M{"$inc":bson.M{"booked":rooms},"$setOnInsert":bson.M{"capacity":cap}}
     opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
     // reintentar en IsDuplicateKeyError; ok=false si la noche está llena
   }
   ```
   Recorrer las noches; si una falla, **liberar** (`$inc booked:-rooms`) las ya reclamadas y devolver `ErrNoAvailability`. Luego `InsertOne`; si falla, liberar todo.
   *Gotcha crítico:* usar `SetReturnDocument(options.After)` (el default `Before` devuelve `ErrNoDocuments` en un upsert-insert exitoso y parecería fallo); **no** poner `hotel_id`/`date` en `$setOnInsert` (ya están en el filtro → error "conflict").
4. **Cancel = soft** (`FindOneAndUpdate` con filtro `status:confirmed` → `cancelled`+`cancelled_at`, devuelve pre-imagen, libera noches). El filtro `status:confirmed` lo hace **idempotente** (segundo cancel no matchea → no doble-libera).
5. **`IsHotelAvailable`**: reescribir para leer el contador por noche (checkout **excluido**), **borrando** el `$unionWith{coll:nil}` frágil (arregla D4 de paso).
6. **Service**: validar `CheckOut>CheckIn`, no-pasado, `NumRooms<=capacity`; derivar `HotelName` y `TotalPrice` (int64 centavos: `round(PricePerNight*100)*noches*rooms`) del hotel; publicar `ReservationNew` en cola **separada** (`reservations-news`).
7. **Controller**: DTO con fechas **string "2006-01-02"** (resuelve la inconsistencia de formato); mapear `ErrNoAvailability` → **409**.
8. **Backfill** (`migrate-inventory.js`): setear `status/num_rooms/...` en reservas viejas y poblar el inventario desde las reservas confirmadas (join `ObjectId(r.hotel_id)`); sincronizar cache/mock para contar `NumRooms` y saltar canceladas.
**Verificar:** disparar ~20 `curl` paralelos a `/reservations` en un hotel de 1 habitación → **exactamente 1 devuelve 201, el resto 409**; `db.reservation_inventory` nunca supera `capacity`; cancelar → `status:cancelled` + noches decrementan; segundo DELETE idempotente.

### 7.3 Persistencia + seed automatizado (DB1-DB5, P7) — verdict: sub-especificado · esfuerzo L
**Correcciones clave** ya en 7.0 (P7, DB4). **Pasos:**
1. **DB1 migraciones** (`golang-migrate`): `users-api/migrations/0001_create_users.up/down.sql` que matchee **exacto** el schema de GORM (tabla `users`, `tipo ENUM default 'cliente'`, unique `username`). Correr por **CLI en CI o un one-shot** — *no* desde las 3 réplicas (contención del advisory lock → schema "dirty"). Si es programático, `go:embed` la SQL (el Dockerfile multi-stage solo copia el binario) y gate `RUN_MIGRATIONS=true` en **una** instancia. Gatear `AutoMigrate` off en compose (`AUTO_MIGRATE=false`).
2. **DB2 índices**: Mongo `EnsureIndexes` idempotente (`{hotel_id,check_in,check_out}` + `{user_id}`) llamado en el startup de hotels-api; MySQL vía la migración (el unique de username ya es el índice). *Gotcha:* los nombres de campo deben matchear los bson tags (`check_in`, no `check_in_time`).
3. **DB3 pool+ping**: GORM `db.DB()` → `SetMaxOpenConns(25)/SetMaxIdleConns(25)/SetConnMaxLifetime(5m)` + `PingContext`; Mongo `SetMaxPoolSize(50)/SetServerSelectionTimeout(5s)` + `client.Ping`.
4. **DB4 context**: refactor de interfaz completo (ver 7.0). Orden: interfaz → mysql (`db.WithContext(ctx)`) → cache/memcached (`_ context.Context`, ignoran) → mock (`m.Called(ctx,...)`) → service+helpers → controller (`ctx.Request.Context()`) → tests (`mock.Anything` extra en cada `.On`).
5. **DB5 paginación**: `Limit/Offset` + `CountAll`; devolver total en header **`X-Total-Count`** (no envelope `{results,total}`, que rompería el admin del frontend que espera un array).
6. **P7 seed**: (a) `hotels-api/seed/mongo-init.js` montado en `/docker-entrypoint-initdb.d` (empieza con `db.getSiblingDB('hotels-api')` + guard `countDocuments()===0` + crea índices) y `MONGO_INITDB_DATABASE: hotels-api` en compose; (b) admin+cliente demo vía **migración 0002** con hash bcrypt generado por la lib Go (`$2a$`, no `$2y$`); documentar `docker compose down -v` para re-seed.

### 7.4 Observabilidad (O1-O4) — verdict: sub-especificado · esfuerzo L
**Correcciones** (7.0): O1 el hop search→hotels sale del **consumer** (no hay X-Request-ID inbound → generar uno por mensaje); readyz por servicio tiene **sets de deps distintos** (hotels: Mongo+Rabbit; users: MySQL+Memcached; search: Solr+Rabbit — hotels **no** usa memcached).
**Pasos:** (1) `log/slog` JSON handler con `service` attr, una vez en cada `main.go`; (2) `gin.SetMode` release antes de `gin.Default()` (o `GIN_MODE=release` en las **5** entradas de compose); (3) middleware request-ID (leer `X-Request-ID` o generar UUID → ctx → response header → logger scoped); reenviar el header en `hotels_http.go` con `NewRequestWithContext`; agregar `$request_id` al `json_combined` de nginx y `access_log ... json_combined`; (4) `Ping`/`IsConnected` en los repos + `/livez` (200 barato) y `/readyz` (pinguea deps, **503** si alguno cae) reemplazando `/health`; wire compose healthchecks a `/readyz` con `condition: service_healthy` (dar `start_period ~90s` a search-api por Solr).

### 7.5 Contratos de API (A1, A2, A3, A6) — verdict: sub-especificado · esfuerzo L
**Correcciones** (7.0): ~18 sitios de fuga (no 4), `error_page` 5xx faltante, versionado rompe frontend, idempotency-key ≠ X-Request-ID.
**Pasos:** (1) paquete `apperr` **copiado en cada módulo** (`internal/` no cruza módulos) con envelope `{error:{code,message,trace_id}}` + `Abort` que loguea la causa vía `c.Error()` pero **nunca** la serializa; (2) reemplazar cada `fmt.Sprintf("...: %s", err.Error())` por `apperr.Abort(...)` con códigos estables (lista de ~18 líneas); alinear el 404/5xx de nginx al mismo shape; (3) `router.Group("/api/v1")` en los 3 (health fuera del grupo) + prefijar location **y** `proxy_pass` target en nginx (incluidas las 2 regex); (4) idempotencia: middleware con store Mongo, clave **`(Idempotency-Key, userID)`** con índice único (replay concurrente → `IsDuplicateKeyError` → 409) + índice TTL 24h; **compone** con el inventario de 7.2 (idempotencia dedup reintentos del mismo key; el inventario evita que 2 users distintos hagan overbooking); (5) 201+`Location`, 204 en delete, PUT devuelve la representación.

### 7.6 Endurecer search-api (E1-E5) — verdict: sub-especificado · esfuerzo L
**Pasos:** (1) **manual ack**: `autoAck=false`, `Qos(1,0,false)`, `HandleHotelNew` devuelve `error`; loop `Ack` en éxito / `Nack(false,true)` primer intento / `Nack(false,false)`→DLQ si `Redelivered`; declarar DLX+DLQ (*gotcha:* re-declarar `hotels-news` con args nuevos da 406 — coordinar con el productor); (2) **query Solr segura**: `edismax` `qf='name description'` con `$qq` en `Params`, escapar metacaracteres Lucene, `q` vacío → `*:*`, `rows/start` reales; (3) **backfill**: primero agregar `GET /hotels?limit&offset` `{data,total}` a hotels-api (handler+repo+service+ruta), luego backfill al arranque en goroutine + `POST /reindex` admin (idempotente por `id` uniqueKey de Solr); (4) `http.Client{Timeout:5s}` + `NewRequestWithContext` (reusar **un** client); (5) reconexión del consumer espejando el productor de hotels-api (re-ejecutar `Consume` tras `NotifyClose`) + `/readyz` con `Solr.Ping` + `IsConnected`.

### 7.7 CI/CD + quality gates + módulo de contratos (SD1, CQ1-3, T1, T2, T4) — verdict: **implementable-as-written** · esfuerzo L
El único paquete listo casi tal cual (con las correcciones de schema v2 y govulncheck de 7.0). **Pasos:** (1) `go.work` raíz (`use ./hotels-api ./users-api ./search-api ./platform-contracts`); (2) módulo compartido `platform-contracts` con `Hotel`+`HotelNew`, importado vía **type-alias bridge** (`type Hotel = contracts.Hotel`) para **cero cambios** en call-sites; `require`+`replace` local en cada go.mod (*gotcha:* mantener el typo `AvaiableRooms`/`hotel_id` — arreglarlo es el rename C11, aparte, o rompe RabbitMQ); (3) `.golangci.yml` **schema v2** + `gofmt -w` los 7 archivos; (4) `.github/workflows/ci.yml` matrix sobre 3 módulos (gofmt-check, vet, golangci-lint v2, `go test -race -coverprofile`, govulncheck con `GOWORK=off` y `continue-on-error` en search-api) + job frontend (`npm ci/build/audit`) + `dependabot.yml`; (5) **contract test** con golden JSON (`{"operation":"CREATE","hotel_id":"abc-123"}`) que falla si el wire format deriva; (6) skeleton testcontainers-go `//go:build integration` para availability de Mongo (usa el path real `Create→CreateReservation→IsHotelAvailable`).

### 7.8 Cloud-native / k8s (CN1-CN4) — verdict: sub-especificado · esfuerzo L
**Correcciones** (7.0): CN1 depende de O3 (`/readyz`), hotels-api ignora `PORT`, HPA necesita `resources.requests`, DNS = nombre de Service.
**Pasos:** (1) `k8s/` con `ConfigMap`+`Secret` (JWT_SECRET **único** compartido); (2) template canónico **Deployment+Service+HPA** de users-api (requests obligatorios para el HPA; probes a `/readyz`+`/livez`; el ClusterIP Service **reemplaza** las 3 réplicas copy-paste + el upstream estático de nginx → escalar = `kubectl scale` sin tocar nginx); (3) hotels-api/search-api como deltas (hotels `containerPort:8081` fijo); (4) datastores como StatefulSets dev (nota: preferir managed en prod; Mongo sigue standalone); (5) opcional Ingress espejando el ruteo de nginx (*gotcha:* las 2 sub-rutas `/users/{id}/reservations` van a hotels-api → `use-regex`); (6) `.dockerignore` por servicio (el del frontend **debe** excluir `node_modules`/`dist`); (7) `deploy.resources.limits` en compose (`limits` se enforce, `reservations.cpus` es best-effort; Solr ≥1G o OOM); (8) tags inmutables git-SHA+semver, digest del **manifest-list**, Trivy gate, buildx multi-arch, push a GHCR (*gotcha:* renombrar `hotels-api/dockerfile`→`Dockerfile`; convertir hotels/search a multi-stage no-root o Trivy marca el toolchain).

### 7.9 Orden de implementación recomendado (con dependencias entre paquetes)

Las dependencias entre paquetes importan — este orden evita rehacer trabajo:

1. **7.1 Auth** (independiente, bloqueante de seguridad) → primero.
2. **7.7 CI + go.work + módulo compartido** → temprano: te da `go test -race`/lint como red de seguridad **antes** de los refactors grandes, y el módulo de contratos que 7.6 aprovecha.
3. **7.3 Persistencia** (migraciones/índices/pool/**ctx**) → antes de tocar la lógica de reservas; el refactor de `context` (DB4) toca las mismas firmas que 7.2.
4. **7.2 Overbooking + dominio** → sobre la base de persistencia (usa el índice único de 7.3).
5. **7.4 Observabilidad** (`/readyz`) → **prerequisito de 7.8** (los probes de k8s lo necesitan).
6. **7.6 search-api** → usa el `GET /hotels` paginado y el módulo de contratos.
7. **7.5 Contratos de API** (versionado/idempotencia/envelope) → tras estabilizar rutas; idempotencia compone con el inventario de 7.2.
8. **7.8 Cloud-native/k8s** → último: consume `/readyz` (7.4) y las imágenes multi-stage.

> **Nota metodológica honesta:** estos playbooks los verificó una pasada multi-agente leyendo el código, pero **no ejecuté los cambios**. Los snippets son correctos respecto de las firmas/tipos actuales, pero al implementar pueden aparecer detalles (imports, un test extra) — por eso cada paquete trae su bloque **Verificar** con comandos concretos. Implementá **de a un paquete**, corré su verificación, y recién ahí seguí con el próximo.

---

## Apéndice — Cómo se produjo este plan

**1ª pasada (Secciones 1–5):** auditoría multi-agente sobre el código real: 6 revisores en paralelo (users-api, hotels-api, search-api, frontend, infra-devops, docs) + un runner que ejecutó `go build/vet/test` y `npm build` de verdad, seguido de una fase de **verificación adversarial** que confirmó cada hallazgo crítico/alto trazando la cadena completa en el código.

**2ª pasada (Sección 6):** auditoría de **profundidad/madurez de backend engineering** con 9 agentes en paralelo (observabilidad, base de datos, diseño de API, resiliencia, testing, cloud-native, seguridad-2ª-capa, calidad/tooling, dominio+frontend), cada uno con la lista de hallazgos ya conocidos para reportar **solo cosas nuevas**. Incluyó ejecución real de `govulncheck` (search-api: 12 CVEs) y `npm audit` (frontend: 16), más `gofmt -l`, y greps que confirman ausencias.

**3ª pasada (Sección 7):** auditoría de **implementabilidad** con 8 agentes (uno por paquete de trabajo de alto valor), cada uno leyendo el código real y escribiendo un playbook paso a paso con snippets, prerequisitos, orden y una sección de verificación. Su hallazgo central: 7 de 8 fixes del plan estaban sub-especificados y 1 (D1, transacción Mongo) estaba directamente equivocado para este deployment — todas las correcciones están en la Sección 7.0. (3 de los 8 agentes se re-ejecutaron por un corte de conexión en el primer intento.)

Referencias `archivo:línea` incluidas para que puedas ir directo al punto. No se modificó ningún archivo de código — solo se creó/actualizó este documento.
