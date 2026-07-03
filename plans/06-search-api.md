# Plan 06 — Endurecer search-api (event-driven robusto)

> **Alcance:** E1, E2, E3, E4, E5, E6, DB6, DB7
> **Base:** playbook 7.6 de `plantofinish.md` + ítems Solr afines (E6, DB6, DB7) + bump de CVEs (cierra la excepción de SD1 del plan 02)
> **Prerequisitos:** plan 02 (módulo de contratos), plan 05 recomendado (`/readyz` a extender aquí)
> **Esfuerzo:** ~1 día (L)

## Contexto

`search-api` es el eslabón más débil y "event-driven/CQRS-lite" es un titular del proyecto: el consumidor usa `autoAck=true` y traga errores → pérdida silenciosa de eventos (E1, `queue_rabbit.go:53`, `search_service.go:73,82-84,111-131`); la query Solr se arma por `fmt.Sprintf` sin escapar (E2, `hotels_solr.go:177`); no hay backfill (E3); el cliente HTTP no tiene timeout (E4, `hotels_http.go:31-32`); el consumer no reconecta y `/health` miente (E5); `getTimeField` nunca parsea fechas (E6, `hotels_solr.go:240-245`); Solr recibe `Commit()` duro por documento (DB6, `hotels_solr.go:83,134,166`) y el schema tiene tipos inapropiados (DB7, `schema.xml:5-19`).

## Correcciones de la Sección 7.0 que aplican

- **E1**: re-declarar la cola `hotels-news` con args de dead-letter nuevos da **406 (PRECONDITION_FAILED)** contra la declaración sin-args del productor → coordinar los args de declaración entre productor (hotels-api) y consumidor, o declarar la DLX/DLQ aparte. `docker compose down` limpia la cola efímera para poder re-declarar.

## Pasos

### 1. Manual ack + retry + DLQ (E1)

- `autoAck=false` + `ch.Qos(1, 0, false)`.
- `HandleHotelNew` pasa a devolver `error` (hoy solo loguea).
- Loop del consumer:

```go
for msg := range msgs {
  if err := service.HandleHotelNew(ctx, msg.Body); err != nil {
    if msg.Redelivered { _ = msg.Nack(false, false) }  // 2º fallo → DLQ
    else               { _ = msg.Nack(false, true)  }  // 1er fallo → requeue
    continue
  }
  _ = msg.Ack(false)
}
```

- Declarar exchange DLX + cola `hotels-news-dlq`, y `hotels-news` con `x-dead-letter-exchange` — **actualizando la declaración del productor en hotels-api con los mismos args** (gotcha 406 de arriba).

### 2. Query Solr segura (E2)

- Pasar a `edismax` con la query del usuario como parámetro (no interpolada):

```json
{"query": "{!edismax qf='name description' v=$qq}",
 "params": {"qq": "<input del usuario>", "rows": limit, "start": offset}}
```

- Escapar metacaracteres Lucene del input (`+ - && || ! ( ) { } [ ] ^ " ~ * ? : \ /`).
- `q` vacío → `*:*` (hoy da 500).
- `rows`/`start` como parámetros reales (hoy la paginación se ignora).
- Test unitario de construcción de query: multi-palabra, vacío, con metacaracteres.

### 3. Backfill + reindex (E3)

1. Primero en **hotels-api**: `GET /hotels?limit&offset` → `{data, total}` (handler + service + repo + ruta). No existe hoy (`hotels_http.go` solo tiene `GetHotelByID`).
2. En search-api: rutina de **backfill al arranque** (goroutine que pagina `GET /hotels` e indexa todo — idempotente porque `id` es uniqueKey de Solr).
3. `POST /reindex` (admin) que dispara el mismo backfill on-demand.

### 4. Cliente HTTP con timeout + context (E4)

`hotels_http.go:31-32`: reemplazar `http.Get`/`DefaultClient` por **un** `http.Client{Timeout: 5 * time.Second}` reutilizado + `http.NewRequestWithContext(ctx, ...)` (propagando el `X-Request-ID` del plan 05). Reintentos acotados (2-3) en 5xx/error de conexión.

### 5. Reconexión del consumer + readiness real (E5)

- Espejar el patrón `connectWithRetry` del **productor de hotels-api** (backoff exponencial + `NotifyClose` → reconectar y re-ejecutar `Consume`).
- Chequear el error de `QueueDeclare` (hoy ignorado, `queue_rabbit.go:32`); no `log.Fatalf` si RabbitMQ tarda en el arranque — reintentar.
- Extender el `/readyz` del plan 05: `Solr.Ping` + `IsConnected()` de RabbitMQ (si el plan 05 no corrió aún, crear `/readyz` acá con ese contrato).

### 6. Fix `getTimeField` (E6)

`hotels_solr.go:240-245`: Solr devuelve strings, el assert `doc[field].(time.Time)` siempre falla → `time.Parse(time.RFC3339, s)`. Test de parseo con un documento Solr real en `testdata/` (fixture del plan 02/T6).

### 7. Commit y schema de Solr (DB6, DB7)

- **DB6**: quitar el `Commit()` explícito por documento (`hotels_solr.go:83,134,166`) — el `autoCommit`/`autoSoftCommit` ya configurado (`solrconfig.xml:21-27`) se encarga; si hace falta visibilidad inmediata usar `commitWithin`.
- **DB7** (`schema.xml:5-19`): `phone`/`email` como `string` (match exacto, no `text_general`); `copyField` de `name`/`description` a un catch-all `_text_`; `docValues` en strings facetables. Requiere recrear el core o reindexar (usar el `POST /reindex` del paso 3).

### 8. Bump de dependencias con CVEs (cierra SD1)

- `go get -u golang.org/x/net github.com/gin-gonic/gin && go mod tidy` en search-api (12 CVEs, p.ej. `x/net@0.10.0` → GO-2023-2102 rapid-reset).
- Verificar `GOWORK=off govulncheck ./...` limpio y **quitar el `continue-on-error`** de la leg search-api en `.github/workflows/ci.yml` (lo dejó el plan 02).

## Verificar

```bash
docker compose down && docker compose up -d --build   # re-declara colas con los args nuevos

# E1: matar Solr, crear un hotel → mensaje reintenta y termina en DLQ; revivir Solr + reindex → consistente
docker compose stop solr
curl -X POST localhost/hotels -H "Authorization: Bearer $ATOKEN" -d '{...}'
docker compose exec rabbitmq rabbitmqctl list_queues name messages   # hotels-news-dlq > 0
docker compose start solr && curl -X POST localhost/reindex -H "Authorization: Bearer $ATOKEN"

# E2: búsquedas que hoy fallan
curl -s 'localhost/search?q=hotel%20spa&offset=0&limit=10' | jq '.[:2]'   # multi-palabra funciona
curl -s 'localhost/search?q=&offset=0&limit=10'                            # vacío → resultados, no 500
curl -s 'localhost/search?q=%22(malicious%3A*%22&offset=0&limit=10'        # metacaracteres → 200 sin inyección

# E3: crear un hotel con search-api caído → al levantar, aparece por backfill
# E6: los resultados traen check-in/out reales (no 0001-01-01)
# E4/E5: docker compose stop hotels-api → el consumer no se cuelga (timeout 5s); stop/start rabbitmq → reconecta solo

cd search-api && go test ./... && GOWORK=off govulncheck ./...   # limpio
```

## Al terminar

Tildá el plan 06 en `plans/README.md`. Commit sugerido: `search-api: manual ack+DLQ, safe edismax queries, backfill/reindex, http timeouts, consumer reconnect, solr schema/commit, CVE bumps (E1-E6,DB6,DB7)`.
