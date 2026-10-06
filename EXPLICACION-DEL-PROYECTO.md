# Explicación completa del proyecto — de principio a fin

> **Nota de versión:** este documento de estudio conserva la explicación de los 13 planes originales. Para el estado actual (transacciones Mongo, búsqueda sin caché del gateway y reconciliación periódica), consultar [README.md](README.md), [ARCHITECTURE.md](docs/ARCHITECTURE.md) y [CIERRE.md](docs/CIERRE.md).

> Documento de estudio personal. Explica **qué hace cada pieza de la plataforma, cómo funciona por dentro y qué se construyó a lo largo de los 13 planes**, con los términos técnicos correctos pero explicando cada uno en simple. Está pensado para que puedas defender el proyecto en una entrevista sabiendo exactamente qué hay detrás de cada palabra.

---

## Índice

1. [Qué es este proyecto, en una frase](#1-qué-es-este-proyecto-en-una-frase)
2. [El mapa general: quién habla con quién](#2-el-mapa-general-quién-habla-con-quién)
3. [El viaje de un request: qué pasa cuando alguien reserva](#3-el-viaje-de-un-request-qué-pasa-cuando-alguien-reserva)
4. [La puerta de entrada: nginx como API Gateway](#4-la-puerta-de-entrada-nginx-como-api-gateway)
5. [users-api: identidad y login](#5-users-api-identidad-y-login)
6. [hotels-api: catálogo, reservas y el anti-overbooking](#6-hotels-api-catálogo-reservas-y-el-anti-overbooking)
7. [search-api: la búsqueda (CQRS-lite)](#7-search-api-la-búsqueda-cqrs-lite)
8. [platform-contracts: el contrato compartido](#8-platform-contracts-el-contrato-compartido)
9. [El frontend: la SPA de React](#9-el-frontend-la-spa-de-react)
10. [El contrato de API: reglas comunes de los tres servicios](#10-el-contrato-de-api-reglas-comunes-de-los-tres-servicios)
11. [Infraestructura: Docker Compose, seeds y Kubernetes](#11-infraestructura-docker-compose-seeds-y-kubernetes)
12. [CI/CD: la fábrica que valida todo](#12-cicd-la-fábrica-que-valida-todo)
13. [Testing: la pirámide completa](#13-testing-la-pirámide-completa)
14. [Seguridad: resumen transversal](#14-seguridad-resumen-transversal)
15. [Los 13 planes: qué se hizo y en qué orden](#15-los-13-planes-qué-se-hizo-y-en-qué-orden)
16. [Los números del proyecto](#16-los-números-del-proyecto)
17. [Cómo contarlo en una entrevista](#17-cómo-contarlo-en-una-entrevista)
18. [Glosario rápido](#18-glosario-rápido)

---

## 1. Qué es este proyecto, en una frase

Una **plataforma de búsqueda y reserva de hoteles** construida como **tres microservicios en Go** (usuarios, hoteles, búsqueda), cada uno con su propia base de datos (MySQL, MongoDB, Solr), sincronizados por **eventos de RabbitMQ**, detrás de un **API gateway nginx** con TLS y balanceo de carga, con una **SPA de React 19** encima — todo orquestado con Docker Compose (y alternativamente Kubernetes), testeado en todos los niveles y con CI en GitHub Actions.

> **En simple:** en vez de un solo programa gigante que hace todo (un "monolito"), el sistema está partido en tres programas chicos e independientes, cada uno responsable de una sola cosa. Se hablan entre ellos por HTTP (pedidos directos) y por mensajes en una cola (avisos asíncronos). Si uno se cae, los otros siguen funcionando.

---

## 2. El mapa general: quién habla con quién

```
                        Navegador (React SPA :5173)
                                   │  HTTPS
                                   ▼
                    ┌──────────────────────────────┐
                    │   nginx — API Gateway :443    │
                    │  TLS · routing · rate limit   │
                    │  load balancing · cache       │
                    └──────┬───────┬───────┬───────┘
                           │       │       │
              ┌────────────┘       │       └────────────┐
              ▼                    ▼                    ▼
      users-api ×3           hotels-api            search-api
      (identidad)        (catálogo+reservas)       (búsqueda)
              │                    │                    │
        MySQL + Memcached       MongoDB               Solr
                                   │                    ▲
                                   │  eventos           │ consume
                                   └──► RabbitMQ ───────┘
                                       (hotels-news + DLQ)
```

Las piezas y su rol:

| Pieza | Qué es | Qué hace acá |
|---|---|---|
| **nginx** | Servidor web usado como *API gateway* | La única puerta de entrada: termina TLS (HTTPS), enruta cada URL al servicio correcto, balancea las 3 réplicas de users-api, limita la cantidad de requests y cachea las búsquedas |
| **users-api** (Go, ×3 réplicas) | Servicio de identidad | Registro, login, emite los JWT que los otros servicios validan |
| **hotels-api** (Go) | Servicio de dominio principal | CRUD de hoteles (solo admin), reservas con garantía de no-overbooking, disponibilidad, panel de estado |
| **search-api** (Go) | Servicio de lectura | Mantiene un índice de búsqueda en Solr escuchando eventos y sirve `GET /search` |
| **MySQL 8** | Base de datos relacional | Los usuarios (tabla `users`) |
| **MongoDB 6** | Base de datos de documentos | Hoteles, reservas, inventario por noche, claves de idempotencia |
| **Solr 9** | Motor de búsqueda full-text | El índice de hoteles que responde las búsquedas por texto |
| **Memcached** | Caché en memoria compartida | Segundo nivel de caché de users-api, compartido entre las 3 réplicas |
| **RabbitMQ 3** | *Message broker* (cola de mensajes) | Transporta los eventos "se creó/modificó/borró un hotel" de hotels-api a search-api |
| **React SPA** | Aplicación de una sola página en el navegador | La interfaz: búsqueda, detalle, reserva, historial, panel admin |

> **¿Por qué cada servicio tiene SU base de datos?** Es la regla de oro de microservicios (*database-per-service*): si dos servicios compartieran una base, un cambio de esquema en uno rompería al otro y dejarían de ser independientes. Acá la única forma de comunicarse es por HTTP o por eventos — nunca leyendo la base ajena.

> **¿Por qué bases DISTINTAS (MySQL, MongoDB, Solr)?** Se llama *persistencia políglota*: cada problema con la herramienta que mejor le calza. Usuarios son datos tabulares con unicidad estricta → relacional (MySQL). Hoteles son documentos con listas anidadas (amenities, imágenes) → documental (MongoDB). Búsqueda por texto con relevancia → un motor de búsqueda de verdad (Solr), no un `LIKE '%...%'`.

---

## 3. El viaje de un request: qué pasa cuando alguien reserva

Seguí este flujo una vez y entendés el 80% del sistema:

1. **El usuario se loguea.** La SPA hace `POST /api/v1/login` → nginx lo enruta a una de las 3 réplicas de users-api (la que tenga menos conexiones activas). users-api busca al usuario (primero en sus cachés, después en MySQL), compara la contraseña con **bcrypt** (una función de hashing lenta a propósito, para que robar la base no revele contraseñas) y devuelve un **JWT**: un token firmado que dice "soy el usuario 42, rol cliente, válido 24 horas".
2. **El usuario busca "palermo".** La SPA hace `GET /api/v1/search?q=palermo`. nginx primero mira su **caché de búsquedas** (guarda cada respuesta 5 minutos): si la tiene, responde sin molestar a nadie. Si no, se lo pide a search-api, que consulta el índice de **Solr** y devuelve los hoteles que matchean, ordenados por relevancia (o por precio/rating si se pidió).
3. **El usuario abre un hotel y reserva 2 noches.** La SPA hace `POST /api/v1/reservations` con el JWT en el header `Authorization` y una **Idempotency-Key** (un UUID generado por el navegador que identifica ESTE intento de reserva). hotels-api:
   - valida el JWT (firma, emisor, audiencia, expiración) y saca el `user_id` **del token**, nunca del body;
   - reclama cada noche del rango de forma **atómica** en MongoDB (la parte más importante del sistema — explicada en detalle en la sección 6);
   - si todas las noches se aseguraron, inserta la reserva con el precio calculado **en centavos enteros** y responde `201 Created` con un header `Location`.
4. **Si el usuario reintenta por un timeout**, la misma Idempotency-Key hace que hotels-api devuelva la respuesta ya guardada (con el header `Idempotency-Replayed: true`) en vez de crear una segunda reserva.
5. **Mientras tanto, si un admin creó un hotel**, hotels-api publicó un evento `{operation: "CREATE", hotel_id}` a la cola `hotels-news` de RabbitMQ. search-api lo consume, le pide el hotel completo a hotels-api por HTTP y lo indexa en Solr. En ~1 segundo, la próxima búsqueda ya lo encuentra. Esto es **consistencia eventual**: el índice no se actualiza en el mismo instante que la base, pero converge solo.

---

## 4. La puerta de entrada: nginx como API Gateway

**Archivo:** `nginx.conf` + `nginx/snippets/security-headers.conf` + `nginx/certs/`

> **¿Qué es un API gateway?** Un único punto de entrada delante de todos los servicios. Los clientes nunca hablan con un microservicio directo: todo pasa por acá. Ventaja: las preocupaciones transversales (TLS, límites, logs, headers de seguridad) se resuelven UNA vez, en un solo lugar.

Qué hace exactamente:

- **TLS (HTTPS).** El puerto 443 termina TLS 1.2/1.3 con un certificado local autofirmado (en producción se cambia el archivo del certificado y nada más). El puerto 80 no sirve nada: responde `301` redirigiendo a HTTPS. Se manda el header **HSTS** ("navegador: hablame solo por HTTPS") — con certificado autofirmado los navegadores lo ignoran por RFC, pero la configuración de producción queda demostrada.
- **Routing.** Cada prefijo de URL va a su upstream: `/api/v1/users` y `/api/v1/login` → users-api; `/api/v1/hotels`, `/api/v1/reservations`, `/api/v1/admin` → hotels-api; `/api/v1/search` y `/api/v1/reindex` → search-api. Detalle fino: `/api/v1/users/:id/reservations` va a **hotels-api** (las reservas viven ahí), así que hay regex específicas declaradas ANTES que las rutas genéricas de users.
- **Load balancing.** El upstream de users-api tiene 3 servidores con algoritmo **`least_conn`** (cada request va a la réplica con menos conexiones activas) y *failover* pasivo: una réplica que falla 3 veces queda apartada 30 segundos.
- **Rate limiting.** 10 requests/segundo por IP para la API general, y **5 por minuto en `/login`** (mitiga fuerza bruta de contraseñas). Excederse devuelve **`429 Too Many Requests`** con el mismo envelope JSON de error que usan los servicios — no un HTML de nginx ni un 503 que parecería un servidor caído.
- **Caché de búsquedas.** `GET /api/v1/search` se cachea 5 minutos por URI completa (`proxy_cache`), con `X-Cache-Status: HIT/MISS` visible y "servir la versión vieja si el upstream falla". Es coherente: el índice de Solr ya es eventualmente consistente, así que este caché no agrega ninguna clase nueva de desactualización.
- **Un solo formato de error.** Los `404`, `429` y `502` que genera nginx mismo salen como `{"error": {"code", "message", "trace_id"}}` — el cliente ve exactamente la misma forma de error sin importar quién lo produjo.
- **Request-ID de punta a punta.** nginx propaga el header `X-Request-ID` entrante (o genera uno) hacia los servicios; cada servicio lo loguea y lo devuelve como `trace_id` en los errores. Resultado: con un solo identificador, un `grep` correlaciona el error que vio el usuario con cada línea de log que tocó, a través de todos los servicios.
- **Headers de seguridad con el fix de herencia.** nginx tiene un gotcha famoso: cualquier `add_header` dentro de una `location` **descarta** todos los heredados del bloque `server`. La solución del repo: los headers viven en un snippet que se re-incluye en cada location que agrega headers propios.
- **Monitoreo solo local.** El puerto 8090 (`stub_status`, `/status`) escucha solo en `127.0.0.1` — las métricas del gateway no se exponen al mundo.

---

## 5. users-api: identidad y login

**Qué hace en una frase:** registra usuarios, valida logins contra MySQL a través de una caché de dos niveles, y emite los JWT que consumen los tres servicios. Corre en **3 réplicas** detrás del balanceador.

### 5.1 La estructura interna (igual en los tres servicios)

Cada servicio Go tiene la misma arquitectura en capas, armada en `cmd/main.go` (el *composition root*: el lugar donde se construye e inyecta todo):

```
cmd/main.go            ← construye config → repos → service → controller → rutas
internal/
  config/              ← lee variables de entorno (12-factor)
  controllers/         ← handlers HTTP (Gin): parsean, deciden códigos de estado
  services/            ← la lógica de negocio
  repositories/        ← acceso a datos (MySQL / caché / Memcached / mock)
  dao/ y domain/       ← modelos de persistencia vs. modelos del contrato API
  middlewares/         ← JWT, roles, request-ID, timeout
  tokenizers/          ← minteo del JWT
  apperr/              ← el envelope de error estándar
```

> **¿Por qué capas e interfaces?** El service depende de una **interfaz** `Repository`, no de MySQL concreto. `main.go` inyecta la implementación real; los tests inyectan un mock. Por eso todos los tests unitarios corren **sin ninguna base de datos**.

### 5.2 Registro y login

- **Registro** (`POST /api/v1/users`, público): valida username 3–50 y password 8–72 caracteres. El campo `tipo` del body **se ignora siempre**: el registro público solo crea rol `cliente` — un atacante no puede autoproclamarse admin (hallazgo S1 del audit, corregido en el plan 01). Responde `201` con `Location: /api/v1/users/42`.
- **El único admin** sale de un *seed* al arranque: users-api lee `ADMIN_USERNAME`/`ADMIN_PASSWORD` del entorno y crea el admin si no existe. Es **idempotente entre réplicas**: las 3 réplicas lo intentan, el índice único de username hace que solo una gane y las otras reciban "ya existe" — la coordinación la resuelve la base, sin locks.
- **Contraseñas con bcrypt**: costo configurable (default 10) con un *clamp* defensivo — un `BCRYPT_COST=32` por error de configuración no cuelga el servicio, se degrada al default con un warning. Passwords >72 bytes se rechazan explícitamente (bcrypt solo mira los primeros 72).
- **Login timing-safe**: si el usuario NO existe, igual se ejecuta una comparación bcrypt contra un hash dummy, para que el camino "no existe" tarde lo mismo que "existe con password mala" — así el tiempo de respuesta no filtra qué usuarios existen.
- **Infraestructura caída ≠ credenciales inválidas**: si MySQL está caído durante un login, la respuesta es `5xx`, no un `401` mentiroso. Un matiz que casi ningún CRUD de portfolio distingue.

### 5.3 El JWT (el pasaporte del sistema)

> **¿Qué es un JWT?** Un *JSON Web Token*: un bloque de datos firmado criptográficamente. Quien tiene la clave secreta puede verificar que nadie lo alteró — sin ir a la base de datos en cada request. Es lo que hace que la autenticación sea *stateless* (sin sesión en el servidor).

El token que emite `internal/tokenizers/tokenizers_jwt.go` (firma **HS256** con el `JWT_SECRET` compartido):

```
user_id: 42                                  ← identidad
username: "demo"
tipo: "cliente" | "administrador"            ← rol para RBAC
iss: "users-api"                             ← quién lo emitió
aud: ["users-api","hotels-api","search-api"] ← para quién es válido
iat / nbf / exp                              ← emitido / no-antes-de / expira (24 h)
```

Cada servicio tiene su propio middleware `auth.go` que valida: firma **con el algoritmo fijado en HMAC** (rechaza el ataque clásico `alg: none` y la confusión con RS256), emisor, **su propia audiencia** (un token robado para otro contexto no sirve acá), y expiración con 30 s de tolerancia de reloj. Encima, guards de rol: `AdminOnly()`, `LoggedUserOnly()`, `OwnerOrAdmin()`.

**Y lo más importante:** los tres servicios **se niegan a arrancar** (`os.Exit(1)`) si `JWT_SECRET` está vacío o sigue siendo el placeholder del código. Una plataforma cuya seguridad depende de un secreto no puede bootear en un estado forjable.

### 5.4 La caché de tres niveles (read-through)

```
GET usuario ─► L1: ccache (memoria del propio proceso, TTL 30 s)   ── hit → devolver
                 miss ▼
               L2: Memcached (compartido por las 3 réplicas, TTL 5 min) ── hit → copiar a L1 → devolver
                 miss ▼
               MySQL (la fuente de verdad)  ── poblar L1 y L2 → devolver
```

> **¿Por qué DOS niveles?** L1 es privado de cada proceso: con 3 réplicas, un usuario "caliente" falla 2 de cada 3 veces (le tocó otra réplica). L2 es compartido: apenas UNA réplica lo trajo de MySQL, las otras dos lo encuentran en Memcached. L1 ahorra el viaje de red; L2 ahorra el viaje a la base.

Detalles con intención: cada usuario ocupa **dos claves** por nivel (`user:id:42` y `user:username:demo`, porque se busca por ambas); el DELETE primero trae el registro completo para poder borrar las dos claves de forma determinística; y el trade-off conocido está **documentado en el código**: con L1 por réplica, un usuario borrado puede loguearse hasta 30 s desde otra réplica — aceptado para la demo, con la solución de producción escrita (invalidación pub/sub o tokens de vida corta).

### 5.5 MySQL bien usado

- **Migraciones versionadas** con golang-migrate (archivos SQL numerados `0001_create_users`, `0002_seed_demo_users`), ejecutadas por un contenedor **one-shot** `migrate` antes de que arranquen las réplicas. Una sola instancia migra a propósito: tres migrando a la vez pelean por el lock y pueden dejar el esquema "sucio".
- **Pool de conexiones** con límites (25 abiertas/idle, vida 5 min), **timeouts del driver** (5 s), y **ping al arranque** que hace fallar rápido si MySQL es inalcanzable.
- **`context` propagado hasta la query**: cada request tiene un deadline de 5 s; si se agota, el cliente recibe `503 timeout` en vez de colgarse.
- **Errores tipados, nunca por substring**: el duplicado de username se detecta por el código de error MySQL 1062, no parseando el texto del mensaje. `DELETE` de un usuario inexistente → `404` real (vía `RowsAffected == 0`).

---

## 6. hotels-api: catálogo, reservas y el anti-overbooking

**Qué hace en una frase:** gestiona el catálogo de hoteles (CRUD solo admin) y las reservas con **garantía atómica de que nunca se sobrevende una noche**, publica eventos a RabbitMQ para que search-api se sincronice, y expone disponibilidad y un panel de estado de la plataforma.

### 6.1 El modelo de dominio

- **Hotel**: nombre, descripción, dirección, precio por noche, rating, **`available_rooms`** (la capacidad — renombrado del typo histórico `avaiable_rooms` en el plan 11, como un cambio coordinado atómico que tocó BSON, Solr, frontend, seeds y tests golden a la vez), horarios de check-in/out como strings `"HH:mm"`, amenities e imágenes.
- **Reservation**: hotel, usuario, fechas, `status` (`confirmed` | `cancelled`), cantidad de habitaciones y huéspedes, **`total_price` en centavos enteros** (`int64`) y `created_at`/`cancelled_at`.

> **¿Por qué dinero en centavos enteros?** Los floats no representan exacto los decimales (0.1 + 0.2 ≠ 0.3): sumar plata en float acumula errores de redondeo. La regla profesional es enteros de la unidad mínima: $451.50 se guarda como `45150`. El cálculo redondea el precio a centavos ANTES de multiplicar: `round(precio×100) × noches × habitaciones`.

> **¿Por qué las fechas de estadía son `"2026-09-02"` sin hora ni zona?** Porque una fecha de check-in es una **fecha civil**, no un instante. Serializarla como timestamp UTC corría las fechas un día para usuarios al oeste de UTC (bug real encontrado y corregido). Los timestamps de auditoría sí van en RFC3339 completo.

### 6.2 El anti-overbooking: la pieza central del proyecto

**El problema:** dos personas piden la última habitación de la misma noche AL MISMO TIEMPO. El enfoque ingenuo — "consulto cuántas reservas hay, y si hay lugar, inserto" — tiene una **carrera TOCTOU** (*time-of-check to time-of-use*): ambos requests pasan el chequeo antes de que el otro inserte, y ambos insertan. Dos inserts no chocan entre sí, así que la base no te salva.

**La solución:** materializar la restricción como **contadores de inventario por hotel-noche** en una colección propia (`reservation_inventory`), con un **índice único sobre `{hotel_id, date}`**, y reclamar cada noche con **una sola operación atómica** de MongoDB:

```go
// hotels-api/internal/repositories/hotels/hotels_mongo.go — claimNight
filter := {hotel_id, date, booked <= capacity - rooms}   // ¿entra lo que pido?
update := {$inc: {booked: rooms},                        // reclamo
           $setOnInsert: {capacity}}                     // si el doc no existía, nace con su capacidad
collection.FindOneAndUpdate(filter, update, upsert: true)
```

La magia: **el filtro ES el chequeo de cupo y el update ES el reclamo, en una sola operación que MongoDB ejecuta atómicamente a nivel documento.** No hay ventana entre "verificar" y "reservar". Si dos requests compiten por la última habitación, Mongo los serializa: uno matchea el filtro e incrementa; el otro ya no matchea (`booked` quedó al tope) y recibe "no hay lugar".

Los detalles finos que muestran profundidad:

- **La carrera del upsert**: si dos requests crean el contador de la misma noche a la vez, uno recibe error de clave duplicada (gracias al índice único); se reintenta UNA vez sin upsert y el filtro decide: matchea → reclamada; no matchea → noche llena.
- **Compensación (patrón saga)**: una reserva de 3 noches reclama noche por noche. Si la tercera falla, las dos ya reclamadas se **liberan** y el cliente recibe `409 no_availability`. La liberación corre sobre un `context.WithoutCancel` (si el cliente cortó la conexión a mitad del request, la compensación corre igual) con 3 reintentos y backoff.
- **Sesgo conservador documentado**: si aun así una liberación falla, se loguea "requiere reconciliación manual" — porque liberar de más habilitaría overbooking, que es peor que una noche temporalmente bloqueada.
- **Demostrado con un test real**: `TestMongo_ConcurrentClaimLastRoom` lanza 20 goroutines contra la última habitación en un MongoDB de verdad (testcontainers) y verifica: exactamente 1 gana, 19 reciben `no_availability`, y jamás existe un contador con `booked > capacity`.

> **¿Por qué no una transacción?** Las transacciones multi-documento de Mongo exigen un replica set, y de todas formas dos inserts independientes no generan conflicto entre sí — el problema no se resuelve con transacciones sino materializando la restricción en un documento donde los dos requests SÍ chocan. Bonus: este diseño funciona en un Mongo de un solo nodo y se traslada sin cambios a un replica set en producción.

### 6.3 Idempotencia: reintentar sin duplicar

> **¿Qué es idempotencia?** Que ejecutar la misma operación dos veces tenga el mismo efecto que una. Crítico en pagos y reservas: si el cliente hace `POST /reservations`, se corta la red, y reintenta… sin protección crearía DOS reservas.

El mecanismo (`internal/middlewares/idempotency.go`):

- El cliente manda un header **`Idempotency-Key`** (un UUID). La clave se guarda en Mongo compuesta como `(key, user_id)` con índice único — dos usuarios distintos pueden usar la misma key sin pisarse.
- **Primer request**: se ejecuta normal y se persiste el resultado (status + body).
- **Reintento**: devuelve la respuesta guardada **byte a byte** con el header `Idempotency-Replayed: true`, sin re-ejecutar nada.
- **Reintento mientras el original sigue en vuelo**: `409 request_in_flight` (la detección es atómica: el segundo insert choca contra el índice único).
- **Si el resultado fue un 5xx**: la clave se libera, para que el cliente pueda reintentar de verdad (no quedar 24 h bloqueado por un error transitorio). Los 4xx sí se persisten y replayean.
- Las claves expiran solas a las 24 h con un **índice TTL** de Mongo (Mongo borra los documentos vencidos, sin cron propio).

**La distinción clave para la entrevista:** el claim atómico de inventario protege contra carreras **entre usuarios distintos**; la Idempotency-Key deduplica reintentos **del mismo cliente**. Son dos problemas diferentes y hacen falta los dos.

### 6.4 Cancelación: soft-delete auditable

`DELETE /api/v1/reservations/:id` — solo el **dueño** puede cancelar (ni siquiera el admin cancela reservas ajenas; decisión de contrato documentada):

- Es un **soft-delete**: un `FindOneAndUpdate` con filtro `{_id, status: "confirmed"}` cambia el estado a `cancelled` y estampa `cancelled_at`. El documento nunca se borra: queda en el historial del usuario como registro auditable.
- El filtro por `status: "confirmed"` la hace **idempotente**: un segundo cancel no matchea, se devuelve la reserva ya cancelada y — crucial — **no se liberan las noches dos veces** (eso duplicaría cupo fantasma).
- Las noches reclamadas se devuelven al inventario (misma rutina de liberación con reintentos), y el cupo queda reservable de nuevo (testeado).

### 6.5 Eventos, caché y el resto

- **Eventos RabbitMQ**: cada CREATE/UPDATE/DELETE de hotel publica `{operation, hotel_id}` a `hotels-news` (si el publish falla, la operación falla — search-api depende del evento). Las reservas publican a una cola separada `reservations-news` en modo *best-effort* (sin consumidor aún; punto de extensión deliberado). El **productor se auto-repara**: reconexión con backoff exponencial, listener de `NotifyClose`, 3 reintentos por publish, mensajes `Persistent` (sobreviven un reinicio del broker).
- **Caché cache-aside** (ccache en memoria, TTL 30 s): se lee la caché primero, en miss se va a Mongo y se puebla. Regla de oro de las listas: las listas denormalizadas (`reservations:user:42`) solo se escriben **enteras** y cualquier escritura de reserva las **invalida** — nunca se editan por ítem (editar por ítem es la fuente clásica de listas incoherentes). Todo fallo de caché es *best-effort*: se loguea y la operación sigue — la base es la verdad.
- **Disponibilidad** (`POST /hotels/availability`): chequea N hoteles en paralelo con un **bulkhead** (semáforo de 8 goroutines máximo — un batch de 500 hoteles no explota en 500 conexiones). Respuesta parcial con sesgo conservador: un hotel que no se pudo verificar se reporta "no disponible", nunca se ofrece lo que no se confirmó.
- **Panel de microservicios** (`GET /admin/microservices`, solo admin): sondea el `/readyz` de cada instancia de cada servicio **en paralelo** con timeout de 2 s y reporta estado y latencia **medidos**. Reemplazó a un panel anterior 100% falso (uptimes inventados, botones de scale/restart que no hacían nada) — se eliminó en vez de maquillarlo.

---

## 7. search-api: la búsqueda (CQRS-lite)

**Qué hace en una frase:** es el lado de **lectura** del sistema — consume los eventos de `hotels-news`, mantiene un índice de búsqueda en Solr, y sirve `GET /search`.

> **¿Qué es CQRS?** *Command Query Responsibility Segregation*: separar el modelo de escritura del de lectura. Acá: MongoDB (vía hotels-api) es la **fuente de verdad** donde se escribe; Solr es una **vista derivada** optimizada para buscar. Se sincronizan por eventos, con consistencia eventual. El "lite" es porque no hay event sourcing ni frameworks — solo la idea esencial.

### 7.1 El pipeline de eventos, con red de seguridad completa

1. Llega un evento `{operation, hotel_id}` por RabbitMQ. Es un **evento "flaco"** (solo el ID): el consumidor le pide el documento completo a hotels-api por HTTP. ¿Por qué? Porque si el evento cargara el hotel entero, un evento viejo reordenado indexaría datos obsoletos — yendo siempre a la fuente de verdad, eso es imposible.
2. **Ack manual**: el mensaje se confirma al broker SOLO si el indexado terminó bien. Con `autoAck` (lo que había antes del plan 06), un crash a mitad de procesamiento perdía el evento en silencio.
3. **Política de fallos explícita**: primer fallo → el mensaje se reencola (1 reintento); segundo fallo → va a la **DLQ** (*dead-letter queue*, la cola de mensajes muertos `hotels-news-dlq`) para inspección y replay manual. Un mensaje con JSON inválido ("mensaje veneno") va a la DLQ **directo** — reencolarlo jamás lo arreglaría.
4. **404 vs 5xx tipados**: si hotels-api responde 404, el hotel fue borrado → se limpia el documento del índice y se descarta el evento (correcto). Un 5xx es transitorio → reintento/DLQ. La distinción viene de errores tipados (`errors.Is`), no de parsear strings.
5. **Consumer inmortal**: corre en su propia goroutine con loop de reconexión infinito y backoff; el servidor HTTP arranca y sirve desde el primer segundo aunque RabbitMQ esté caído. Y el `/readyz` es honesto: reporta "degraded" si el **loop de consumo** murió, aunque la conexión TCP siga viva.

### 7.2 El circuit breaker en el camino search → hotels

> **¿Qué es un circuit breaker?** Como la llave térmica de tu casa: si una dependencia falla repetidamente, el "circuito se abre" y los llamados fallan al instante durante un tiempo, en vez de seguir martillando un servicio caído y acumular goroutines colgadas esperando timeouts.

El cliente HTTP a hotels-api (`sony/gobreaker`): timeout de 5 s por request, 3 reintentos con backoff y *jitter* (ruido aleatorio para desincronizar réplicas), y el breaker abre tras 5 fallos consecutivos, queda abierto 30 s, y prueba con 3 requests en modo *half-open* antes de cerrar. Semántica fina: un **404 es una respuesta, no un fallo** — no abre el breaker ni se reintenta. Todo testeado, incluida la transición half-open→closed.

### 7.3 La búsqueda segura (anti-inyección Lucene)

El input del usuario **jamás se interpola** en la query de Solr. Dos capas:

1. La query es el literal fijo `{!edismax qf='name description' v=$qq}` y el texto del usuario viaja **solo como parámetro** `qq` (parámetro dereferenciado) — el equivalente Solr de los *prepared statements* de SQL.
2. Además se escapan los metacaracteres de Lucene (`+ - && || ! ( ) { } [ ] ^ " ~ * ? : /`).

El `sort` se valida contra una **whitelist cerrada** (`relevance`, `price_asc`, `price_desc`, `rating_desc`); cualquier otro valor → `400 invalid_sort` sin tocar el service. Cada orden lleva un desempate `id asc` para que la **paginación sea estable** (dos hoteles con el mismo precio no se reordenan entre páginas). El `limit` se clampea a 100 (un `rows=999999` sería un DoS a Solr).

### 7.4 El índice es descartable (y eso es una feature)

Solr es una **vista derivada**: si se pierde, se reconstruye desde la fuente de verdad. Al arrancar, search-api hace **backfill** — pagina el `GET /hotels` de hotels-api e indexa todo (con 5 reintentos y backoff para tolerar arranques en frío) — y `POST /api/v1/reindex` (solo admin) dispara lo mismo bajo demanda. Indexar es idempotente (el `id` es la `uniqueKey` de Solr: reprocesar un evento pisa el documento, no lo duplica). **Un evento perdido degrada a "índice desactualizado hasta el próximo backfill", nunca a pérdida de datos.**

Solr también está afinado: nada de hard-commit por documento (lo más caro que se le puede pedir); en su lugar, *soft-commit* automático cada 1 s (visibilidad casi en tiempo real) y *hard-commit* cada 15 s sin abrir searcher (durabilidad barata). El schema usa tipos con intención: `string` exacto para teléfonos/emails, `text_general` tokenizado para lo buscable, `pfloat/pint` con docValues para poder ordenar.

---

## 8. platform-contracts: el contrato compartido

**El problema que resuelve:** el struct `Hotel` estaba duplicado 4 veces (dominio de hotels-api, dominio de search-api, DAO de Solr, evento). Cambiar un tag JSON en uno solo rompía la deserialización **silenciosamente en runtime**.

**La solución:** un módulo Go chico que es la única fuente de verdad de los tipos que cruzan servicios:

- `Hotel` (la representación wire) y `HotelNew` (el evento `{operation, hotel_id}`). hotels-api y search-api los consumen por *type alias* (`type Hotel = contracts.Hotel`) — el drift ahora **no compila**.
- El **middleware CORS único** de la plataforma (allowlist por variable de entorno; nunca emite el combo inválido `Access-Control-Allow-Origin: *` + credentials; los tres servicios lo montan y nginx no duplica headers CORS en rutas proxiadas — la duplicación hace que el navegador rechace la respuesta).
- El **contract test golden**: `TestHotelNewWireFormat` fija el JSON del evento contra un archivo golden en las dos direcciones — el `Marshal` del productor debe producirlo byte a byte, y el `Unmarshal` del consumidor debe poblar todos los campos. Si alguien toca un tag, **CI falla antes de que el cambio rompa RabbitMQ en producción**. El golden solo se actualiza deliberadamente como parte de un cambio de contrato coordinado.

---

## 9. El frontend: la SPA de React

**Stack:** React 19 + Vite 7 + MUI 7 + react-router 8 + **TanStack Query 5** + Axios + react-hook-form. JavaScript con JSDoc (TypeScript quedó como opcional diferido documentado).

> **¿Qué es una SPA?** *Single-Page Application*: el navegador carga la app una vez y después navega sin recargar la página; los datos van y vienen por la API. Esta SPA consume **el mismo contrato público `/api/v1` que cualquier otro cliente** — no tiene ningún acceso privilegiado.

### 9.1 Estado del servidor con TanStack Query

Todo dato remoto (búsquedas, reservas, listados admin) vive en TanStack Query, que maneja caché, reintentos, invalidación y estados de carga. Configuración con intención: reintenta 1 vez y **nunca los errores 4xx** (un 403 no se arregla reintentando), `staleTime` 30 s, y las búsquedas usan `keepPreviousData` (al paginar, la página anterior queda visible con una barrita de progreso — sin parpadeo).

### 9.2 Sesión sin redirects bruscos

- El token vive en `localStorage`, pero al arrancar la app **valida localmente** los claims (`exp`, `iss`, `aud`) antes de aceptar la sesión — mata la "sesión zombie" (token vencido navegando como logueado hasta el primer 401). La firma la verifica el backend; el navegador solo se protege de mostrarse logueado en falso.
- Cuando la sesión expira en medio del uso (un 401 con token presente), un evento interno dispara una **navegación SPA** a `/login` que preserva a dónde ibas (`state.from`) y muestra "tu sesión expiró" — nunca un `window.location` que pierde todo.
- Guards de ruta: anónimo → login con retorno; cliente en zona admin → **pantalla 403 explícita**, no un redirect mudo.

### 9.3 Las pantallas

| Pantalla | Qué demuestra |
|---|---|
| **Búsqueda** (`/search`) | La **URL es la fuente de verdad** (`q`, `page`, `sort` viven en la query string): copiar el link reproduce la pantalla, Back/Forward restauran todo. El sort espeja la whitelist del backend; el orden lo aplica Solr sobre el índice completo, no un sort local de la página visible |
| **Detalle + reserva** (`/hotels/:id`) | El flujo de booking: fechas civiles DST-safe, total **en centavos** calculado espejo exacto del backend (coincide al centavo con lo persistido), **Idempotency-Key por intento** — reintentar el mismo payload conserva la key (el backend dedupea); cambiar fechas la rota sola. Errores con copy propio: "no_availability" explica que no se cobró nada |
| **Historial** (`/reservations`) | Tabs Upcoming/Past/Cancelled/All; las canceladas **siempre visibles** (auditabilidad). Cancelar pide confirmación y actualiza por refetch (invalidación de query), no editando el estado a mano |
| **Admin** (`/admin`) | CRUD de hoteles con validación del contrato estricto, panel de salud **read-only** con latencias reales, y **self-delete bloqueado** (no podés borrar tu propia cuenta admin logueado, con el motivo visible) |

### 9.4 Accesibilidad y performance (con evidencia)

- **A11y AA**: skip-link, manejo de foco al navegar (el foco se mueve al `<main>` y se anuncia el título), focus trap en dialogs con retorno al trigger, live regions para errores y counts, contraste recalibrado a AA (el dorado pasó de 3:1 a 5.6:1), `prefers-reduced-motion` respetado, targets táctiles de 44 px. Verificado en 3 niveles: axe en tests unitarios por ruta, axe en Playwright contra el stack real (desktop + mobile), y viewports de 320/390 px sin overflow horizontal.
- **Code splitting**: todas las rutas salvo Home cargan *lazy* (el bundle del admin jamás llega a un visitante anónimo), vendors partidos a mano, prefetch del detalle al hacer hover sobre una card. Resultado medido: de un bundle único de **734 kB** a **192 kB gzip iniciales**. **Lighthouse: 94/100/100/100 mobile, 98/100/100/100 desktop** (reportes guardados en `frontend/lighthouse/`).

---

## 10. El contrato de API: reglas comunes de los tres servicios

Decisiones que aplican a TODA la plataforma (y que los tests verifican):

- **Versionado por URI**: todo bajo `/api/v1`. Un cambio incompatible saldría como `/api/v2` conviviendo con v1. Los health endpoints quedan sin versionar (pertenecen al proceso, no al contrato).
- **Un envelope de éxito**: `{"data": ...}` para un recurso; `{"data": [...], "meta": {"total","limit","offset"}}` para listas. Paginación idéntica en todos lados (`limit` default 20, tope 100).
- **Un envelope de error** — producido por los servicios **y por nginx**:
  ```json
  {"error": {"code": "no_availability", "message": "no availability for the requested dates", "trace_id": "9860c40e..."}}
  ```
  `code` es estable y para máquinas (el frontend hace switch sobre él); `message` es para humanos; la **causa real va solo al log**, correlacionada por `trace_id` — un handler jamás filtra internals de un driver al cliente.
- **Semántica HTTP correcta**: `POST` → `201` + `Location`; `PUT` → devuelve la representación actualizada; `DELETE` → `204` sin body; conflictos → `409`.
- **JSON-only explícito**: un `Accept` que no admite JSON recibe `406`, no una negociación fingida.
- **`user_id` es string** en todo lo que cruza servicios (users-api guarda su PK int64 como detalle interno y serializa como string).
- **Fechas civiles y dinero en centavos** (explicados en la sección 6).

Todo documentado en **OpenAPI 3.0 por servicio** (`docs/openapi/users.yaml`, `hotels.yaml`, `search.yaml`, validados con Redocly) y ejercitado por las **colecciones Bruno** (30 requests con asserts automáticos, incluyendo el replay idempotente y el 400 de sort inválido).

---

## 11. Infraestructura: Docker Compose, seeds y Kubernetes

### 11.1 Docker Compose: 12 contenedores con arranque determinístico

`docker compose up -d --build` levanta: nginx, MySQL, Memcached, MongoDB, RabbitMQ, Solr, users-api ×3, hotels-api, search-api, y el one-shot `migrate`. El **orden de arranque está encadenado con healthchecks reales**, no con sleeps:

```
mysql (healthy) ──► migrate (termina OK) ──► users-api ×3 (healthy) ──┐
mongo + rabbitmq (healthy) ──► hotels-api (healthy) ─────────────────┼──► nginx
solr + rabbitmq (healthy) ──► search-api (healthy) ──────────────────┘
```

nginx no arranca hasta que **todos** los upstreams pasan su `/readyz` — eso elimina los `502` de arranque en frío. Detalles que muestran oficio:

- Los 5 secretos usan la sintaxis `${VAR:?mensaje}`: sin `.env`, compose **falla ruidosamente** con instrucciones, en vez de arrancar con defaults inseguros.
- Todos los servicios llevan **límites de CPU/memoria** (el stack entero cabe en ~4 GB; Solr con 1.5 GB porque con menos hace OOM).
- El frontend buildeado es un contenedor **opcional** detrás de un profile (`--profile frontend`).
- Cada contenedor Go se buildea con **Dockerfile multi-stage**: la etapa de build compila un binario estático (`CGO_ENABLED=0`); la imagen final es Alpine con el binario y corre como **usuario no-root con UID numérico fijo** (compatible con el `runAsNonRoot` de Kubernetes). Las imágenes base están **pineadas por digest** (supply chain: nadie puede cambiar la imagen debajo del mismo tag).

### 11.2 Datos iniciales (seeds)

- **MySQL**: la migración `0002` siembra el cliente demo `demo`/`DemoCliente123` (idempotente). El admin sale SOLO del `.env`.
- **MongoDB**: `mongo-init.js` (solo corre en volumen virgen) siembra 5 hoteles argentinos de demo y crea los índices — incluido el **índice único de inventario** del que depende el no-overbooking.
- **Solr**: el core `hotels` se precrea al arrancar el contenedor con el schema del repo.
- `docker compose down -v && up` **resetea el mundo** — reproducibilidad total.

### 11.3 Kubernetes (la alternativa cloud-native)

La carpeta `k8s/` corre la misma plataforma en Kubernetes (probado con kind): **Deployments** con probes de liveness (`/livez`) y readiness (`/readyz`) reales, users-api con 3 réplicas y un **HPA** (autoscaler horizontal: entre 2 y 6 réplicas según CPU), un Service reemplazando la lista estática de upstreams de nginx (escalar = `kubectl scale`, sin tocar YAML), `preStop` + graceful shutdown para **rolling deploys sin perder requests**, imágenes por tag inmutable de GHCR (`sha-...`, nunca `latest`), y securityContext completo (no-root, filesystem read-only, capabilities dropped). Los datastores van como StatefulSets "solo-dev" — el README es explícito sobre qué sería un servicio gestionado en producción.

> **El puente entre los dos mundos:** el graceful shutdown (SIGTERM → drenar requests en vuelo hasta 10 s → cerrar colas y conexiones) se implementó ANTES que k8s a propósito (plan 08 antes del 09): sin él, cada rolling deploy cortaría requests a la mitad.

---

## 12. CI/CD: la fábrica que valida todo

Un workflow de GitHub Actions (`.github/workflows/ci.yml`) con 5 jobs:

| Job | Cuándo | Qué valida |
|---|---|---|
| **go** | cada push y PR | Matriz de los 4 módulos: formato (`gofmt`), análisis estático (`go vet` + golangci-lint), **tests con detector de carreras (`-race`)** y cobertura, y **govulncheck** (escanea CVEs reales en las dependencias) |
| **integration** | solo PRs | Los tests de integración de hotels-api con **testcontainers** (levanta un MongoDB real y prueba el claim concurrente de verdad) |
| **frontend** | cada push y PR | Lint, suite Vitest con umbrales de cobertura — corrida a propósito con `TZ=America/Los_Angeles` (una zona al oeste de UTC) como regresión del bug de fechas civiles —, build de producción y `npm audit` |
| **frontend-e2e** | solo PRs | Levanta el **stack Docker completo** (con TLS y el SPA buildeado) y corre Playwright contra él |
| **docker** | cada push (necesita `go` verde) | Buildea las 3 imágenes y las pasa por **Trivy** como gate (vulnerabilidades HIGH/CRITICAL con fix disponible = build rojo); solo en push a `main` o tags publica imágenes **multi-arch (amd64+arm64) a GHCR con tags inmutables** |

> **¿Por qué importa `-race`?** El detector de carreras de Go instrumenta el binario y detecta accesos concurrentes sin sincronizar en tiempo de ejecución. En un sistema con goroutines por todos lados (fan-out de disponibilidad, consumer de RabbitMQ, health checks paralelos), correr TODA la suite bajo `-race` es la diferencia entre "compila" y "es concurrentemente correcto".

---

## 13. Testing: la pirámide completa

Cada capa responde una pregunta distinta:

| Capa | Herramienta | Qué demuestra |
|---|---|---|
| Unit de services | mocks de repos y colas, `-race` | La lógica de negocio: cache-aside, reglas de fechas, precio en centavos, publicación de eventos |
| Controllers | Gin + `httptest` con **JWTs reales firmados** | Routing, binding (400), autenticación/autorización (401/403), códigos y envelopes. Sin atajos: los tests pasan por el middleware real |
| Integración | **testcontainers** (MongoDB real) | El claim atómico bajo concurrencia real: 20 goroutines, 1 gana, inventario nunca sobrevendido |
| Contract | golden files | El formato wire del evento no puede cambiar sin romper CI |
| Frontend unit | **Vitest + MSW** (131 tests) | La UI contra el contrato mockeado al detalle (envelopes, errores con trace_id, paginación); ningún test puede tocar red sin declararlo |
| E2E | **Playwright** contra el stack Docker real (14 escenarios × desktop y mobile) | Los recorridos completos por nginx TLS: búsqueda anónima, booking con total verificado al centavo, admin creando un hotel y **viéndolo aparecer en la búsqueda vía RabbitMQ→Solr**, y **degradación**: se apaga search-api en vivo y se verifica que la UI muestra el error con `trace_id` y se recupera — no una pantalla blanca |
| API smoke | colecciones **Bruno** (30 requests con asserts) | Todo el contrato ejecutable contra el stack, incluyendo replay idempotente y sort inválido |
| Gateway | `test_load_balancer.sh` | Distribución real entre las 3 réplicas, redirect 301, cache MISS→HIT, y que el rate-limit devuelve 429 JSON (nunca 503) |

---

## 14. Seguridad: resumen transversal

- **Escalada de privilegios cerrada**: el registro público solo crea `cliente`; el único admin nace del seed del `.env`.
- **JWT endurecido**: algoritmo fijado (anti `alg:none`), `iss`/`aud` por servicio, expiración, y los tres servicios se niegan a arrancar con el secreto placeholder.
- **Autorización en el servidor**: ownership (solo tus reservas, solo tu cuenta) validado en handlers contra el token — jamás confiando en lo que dice el body o el frontend.
- **PII protegida**: listar las reservas de un hotel (expone `user_id` de otros huéspedes) es solo-admin.
- **Contraseñas**: bcrypt, política de longitud, login timing-safe, rate-limit de 5/min en `/login` en el gateway.
- **Inyección**: Lucene neutralizada con parámetros dereferenciados + escapado; MySQL vía GORM parametrizado; errores tipados sin filtrar internals.
- **Secretos fuera del repo**: `.env` git-ignorado, compose falla sin él, k8s con Secrets.
- **Supply chain**: govulncheck + npm audit + Trivy como gates de CI, imágenes base pineadas por digest, contenedores no-root, tags inmutables.
- **Transporte**: TLS en el gateway, HSTS, security headers en toda ruta, CORS con allowlist y sin el combo inválido `*`+credentials.

---

## 15. Los 13 planes: qué se hizo y en qué orden

El proyecto partió de una auditoría maestra (`plantofinish.md`) que catalogó ~100 hallazgos con ID (S1, D1, C11…), y se ejecutó como 13 planes autocontenidos (`plans/`), cada uno cerrado con su bloque de verificación. Este fue el recorrido:

| # | Plan | Qué cambió (el antes → después) |
|---|---|---|
| **01** | **Seguridad y auth** | Antes: users-api sin NINGÚN middleware de auth, registro que aceptaba rol admin, secreto JWT hardcodeado. Después: middleware JWT completo con `iss`/`aud`, registro sin escalada, secretos a `.env` con fail-fast, política de contraseñas, y toda la matriz negativa de auth testeada |
| **02** | **CI y contratos** | Nació la red de seguridad: `go.work`, el módulo `platform-contracts` (mató los structs duplicados 4×), golangci-lint, GitHub Actions (vet/`-race`/govulncheck), el contract test golden, Makefile, LICENSE, .gitignore |
| **03** | **Persistencia y seed** | Migraciones versionadas con contenedor one-shot, índices, pools y timeouts de driver, `context` propagado en users-api, paginación en la DB, y los datos demo (5 hoteles + usuario `demo`) |
| **04** | **Dominio de reservas** | El corazón: el inventario atómico por hotel-noche (antes `CreateReservation` sobrevendía incondicionalmente), `Reservation` rica (estado, cantidad, dinero en centavos), fechas canónicas, caché best-effort, tests de repositorio |
| **05** | **Observabilidad** | `slog` JSON estructurado, request-ID de punta a punta, `/livez` + `/readyz` reales, Gin en release mode, healthchecks del compose |
| **06** | **search-api endurecido** | Antes: autoAck que perdía eventos, inyección Lucene, sin backfill, cliente HTTP sin timeout, consumer que moría sin reconexión. Después: todo lo descrito en la sección 7 |
| **07** | **Contratos de API** | El envelope de error estándar, `/api/v1`, idempotencia con `Idempotency-Key`, paginación/envelopes/semántica HTTP unificados, `user_id` string, content negotiation |
| **08** | **Resiliencia de runtime** | Graceful shutdown/SIGTERM en los tres servicios (prerrequisito de k8s), deadlines de Mongo/Solr/consumer, bulkhead del fan-out, circuit breaker, robustez del productor RabbitMQ |
| **09** | **Cloud-native / k8s** | Manifests completos (Deployments, Services, HPA, probes, limits), Dockerfiles multi-stage no-root, `.dockerignore`, Trivy y tags inmutables a GHCR |
| **10** | **Gateway TLS** | TLS + redirect + HSTS, el fix de herencia de `add_header`, la caché real de `/search` (la anterior era un no-op), y 429 (no 503) en rate-limit |
| **11** | **Consistencia y limpieza** | CORS unificado en platform-contracts, el panel de microservicios real (el fake se borró), PII admin-only, errores tipados, TTL en Memcached, 404 en deletes, **el rename atómico `avaiable_rooms`→`available_rooms`** (contratos+BSON+Solr+frontend+seeds+goldens en un solo cambio coordinado), mocks fuera del binario |
| **13** | **Frontend portfolio-grade** | (Antes del 12 a propósito: las capturas debían mostrar el producto final.) Todo lo de la sección 9: contrato único, búsqueda con URL, booking idempotente, historial auditable, admin, a11y AA, 131 tests + E2E |
| **12** | **Documentación y cierre** | README nuevo con demo.gif real, `docs/ARCHITECTURE.md` en inglés (con la tabla de trade-offs), OpenAPI validada por servicio, colecciones Bruno rehechas y verificadas en verde, quickstart probado en frío, docs viejos retirados |

Entre medio, una review externa dejó 31 hallazgos adicionales (`RV1`–`RV31`) que se fueron absorbiendo dentro de los planes que tocaban ese código — varios de los detalles más finos del sistema (login timing-safe, listas de caché solo-enteras, el readyz que detecta el consumer muerto, la validación local de sesión del frontend) vienen de ahí.

---

## 16. Los números del proyecto

- **3 microservicios Go + 1 módulo de contratos** (workspace `go.work`), **12 contenedores** orquestados (11 long-running + 1 one-shot), **3 réplicas** balanceadas de users-api.
- **4 almacenes de datos** (MySQL, MongoDB, Solr, Memcached) + **1 broker** (RabbitMQ con DLQ).
- **Tests Go** en todas las capas bajo `-race`, incluyendo integración con testcontainers y el test de concurrencia de 20 goroutines por la última habitación.
- **Frontend**: 131 tests Vitest+MSW en 22 archivos (~85% de cobertura de statements), 14 escenarios E2E Playwright en desktop y mobile contra el stack real (incluida degradación con servicios apagados en vivo).
- **Lighthouse**: 94/100/100/100 mobile, 98/100/100/100 desktop. Bundle inicial: **192 kB gzip** (desde 734 kB).
- **30 requests Bruno** con asserts, todas en verde contra el stack.
- **CI**: 5 jobs — lint/tests/govulncheck por módulo, integración, frontend, E2E, build de imágenes con gate de Trivy y push multi-arch a GHCR.
- **~100 hallazgos de auditoría** cerrados a través de 13 planes + 31 fixes de review externa.

---

## 17. Cómo contarlo en una entrevista

Las cinco historias que más rinden, en orden:

1. **"¿Cómo evitás el overbooking?"** → TOCTOU, por qué check-then-insert no alcanza, el `findOneAndUpdate` donde el filtro es el chequeo y el update es el reclamo, el índice único, la compensación con sesgo conservador, y el test de 20 goroutines que lo demuestra. Cerrá con: "y no necesita transacciones, así que funciona igual en un Mongo de un nodo que en un replica set".
2. **"¿Qué pasa si se cae X?"** → RabbitMQ caído: el productor reconecta solo y el índice se rehace con backfill (evento perdido = staleness acotada, nunca pérdida de datos). hotels-api caído: el circuit breaker corta, los eventos van a retry/DLQ. Solr caído: `/readyz` degrada y el gateway sirve el caché stale. Una réplica de users caída: `least_conn` + failover pasivo.
3. **"¿Cómo funciona la auth entre servicios?"** → JWT HS256 con `iss`/`aud` por servicio, validación independiente en cada uno, RBAC + ownership server-side, fail-fast del secreto, y el trade-off documentado de la revocación (24 h sin denylist → refresh tokens en producción).
4. **"¿Consistencia fuerte o eventual?"** → Las dos, cada una donde corresponde: el inventario de reservas es fuertemente consistente (atómico); el índice de búsqueda es eventual (eventos + backfill); y el caché del gateway es coherente con eso. Saber CUÁL usar dónde es la respuesta.
5. **"¿Cómo sabés que funciona?"** → La pirámide completa: `-race` en unit, testcontainers en integración, golden en contrato, MSW en frontend, Playwright con caos parcial en E2E, Bruno como smoke ejecutable, y CI que no publica una imagen sin pasar Trivy.

---

## 18. Glosario rápido

| Término | En simple |
|---|---|
| **Microservicio** | Programa chico e independiente responsable de un solo dominio, con su propia base de datos |
| **API Gateway** | El único punto de entrada que enruta, protege y balancea el tráfico hacia los servicios |
| **JWT** | Token firmado que prueba identidad y rol sin consultar la base en cada request |
| **bcrypt** | Función de hashing de contraseñas deliberadamente lenta (frena la fuerza bruta) |
| **RBAC** | Control de acceso por roles (cliente vs administrador) |
| **TOCTOU** | Carrera entre "chequear" y "usar": el estado cambió entre medio |
| **Operación atómica** | Se ejecuta entera o no se ejecuta — sin estado intermedio visible |
| **Saga / compensación** | Deshacer los pasos ya hechos cuando un paso posterior falla |
| **Idempotencia** | Repetir la operación no duplica el efecto |
| **CQRS** | Separar el modelo de escritura (Mongo) del de lectura (Solr) |
| **Consistencia eventual** | Las vistas derivadas se actualizan "enseguida pero no ya"; convergen solas |
| **Message broker / cola** | Intermediario que guarda mensajes hasta que el consumidor los procesa |
| **DLQ** | Cola de mensajes muertos: donde van los mensajes que fallaron para inspección |
| **Ack manual** | Confirmar el mensaje al broker solo después de procesarlo bien |
| **Cache-aside / read-through** | Patrones de caché: mirar la caché primero, poblarla al fallar |
| **TTL** | Tiempo de vida de una entrada (de caché o de un documento) antes de expirar sola |
| **Circuit breaker** | "Llave térmica" que corta llamados a una dependencia que viene fallando |
| **Bulkhead** | Tope de concurrencia que evita que una operación monopolice recursos |
| **Backoff (+ jitter)** | Esperar cada vez más entre reintentos (+ ruido aleatorio para no sincronizarse) |
| **Graceful shutdown** | Al recibir la señal de apagado, terminar los requests en curso antes de morir |
| **Liveness / readiness** | "¿El proceso vive?" vs "¿puede atender tráfico (sus dependencias responden)?" |
| **Healthcheck** | Chequeo periódico que decide si un contenedor está sano |
| **HPA** | Autoscaler de Kubernetes: agrega/quita réplicas según carga |
| **Multi-stage build** | Dockerfile en dos etapas: compilar en una imagen grande, correr en una mínima |
| **Supply chain** | La seguridad de todo lo que entra al build: dependencias, imágenes base, escaneos |
| **Golden test** | Test que compara contra un archivo de referencia byte a byte (fija un contrato) |
| **testcontainers** | Librería que levanta contenedores reales (un Mongo de verdad) dentro de un test |
| **MSW** | Mock Service Worker: intercepta las llamadas HTTP del frontend en los tests |
| **Envelope** | La forma fija que envuelve toda respuesta (`{"data":...}` / `{"error":...}`) |
| **trace_id / request-ID** | Identificador que viaja por todos los servicios y permite correlacionar logs |
| **12-factor** | Metodología de apps cloud: config por variables de entorno, procesos sin estado, logs a stdout |
