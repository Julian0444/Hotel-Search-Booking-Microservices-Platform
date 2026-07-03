# Plan 05 — Observabilidad y operabilidad

> **Alcance:** O1, O2, O3, O4, I5
> **Base:** playbook 7.4 de `plantofinish.md` + I5 (healthchecks del compose, que se enganchan al `/readyz` nuevo)
> **Prerequisitos:** plan 03 recomendado (los pings de driver ya existen). **Es prerequisito del plan 09 (k8s)** — los probes necesitan `/readyz`/`/livez`.
> **Esfuerzo:** ~1 día (L)

## Contexto

Los 3 servicios usan `gin.Default()` + `log` stdlib: nginx genera `X-Request-ID` pero ningún servicio lo lee/propaga (O1, `nginx.conf:130`), niveles de log simulados como prefijos de string (O2), `/health` devuelve 200 aunque MySQL/Mongo/RabbitMQ estén caídos (O3, `hotels-api/cmd/main.go:99-105`, `users-api/cmd/main.go:68-74`), Gin corre en modo debug en los contenedores (O4), y los servicios Go no tienen healthcheck de contenedor — nginx depende de ellos por `service_started` y puede dar 502 en el arranque (I5, `docker-compose.yml:145-152,237-276`).

## Correcciones de la Sección 7.0 que aplican

- **O1**: el hop search→hotels sale del **consumer** (no hay request HTTP inbound) → no hay `X-Request-ID` que leer: **generar un ID por mensaje** consumido y propagarlo en la llamada HTTP.
- **O3**: los sets de dependencias del readyz son **distintos por servicio**: hotels: Mongo+RabbitMQ; users: MySQL+Memcached; search: Solr+RabbitMQ (hotels **no** usa memcached).

## Pasos

### 1. Logging estructurado con `slog` (O2)

Una vez en cada `main.go`:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
  With("service", "hotels-api")
slog.SetDefault(logger)
```

Reemplazar los `log.Printf("warn: ...")` por `slog.Warn/Info/Error` con atributos; estandarizar mensajes en **inglés** (hoy hay mezcla, p.ej. `users_service.go:219`).

### 2. Gin release mode (O4)

`gin.SetMode(gin.ReleaseMode)` antes de `gin.Default()` en los 3, o `GIN_MODE=release` en las **5 entradas** del compose (3 réplicas de users + hotels + search).

### 3. Request-ID end-to-end (O1)

Middleware por servicio:

```go
func RequestID() gin.HandlerFunc {
  return func(c *gin.Context) {
    id := c.GetHeader("X-Request-ID")
    if id == "" { id = uuid.NewString() }   // github.com/google/uuid
    c.Set("request_id", id)
    c.Writer.Header().Set("X-Request-ID", id)
    // logger scoped: slog.With("request_id", id) disponible vía contexto
    c.Next()
  }
}
```

- Consumer de search-api: generar un `request_id` por mensaje y adjuntarlo al logger + reenviarlo como header en el fetch a hotels-api (`hotels_http.go:32`, junto con `http.NewRequestWithContext` — el client con timeout completo lo hace el plan 06/E4; acá alcanza con propagar el header).
- nginx: agregar `$request_id` al `log_format json_combined` y activar `access_log ... json_combined` (hoy el formato existe pero el request-id no está en él).

### 4. `/livez` + `/readyz` reales (O3)

- Agregar `Ping(ctx) error` / `IsConnected() bool` a los repos que no lo tengan (Mongo/MySQL ya tienen ping del plan 03; falta exponer conectividad de RabbitMQ, Memcached y Solr).
- `/livez`: 200 barato, sin tocar dependencias.
- `/readyz`: pinguea las deps de **su** servicio y devuelve **503** con mapa de estado si alguna cae:

```json
{"status":"degraded","checks":{"mongo":"ok","rabbitmq":"down"}}
```

- Mantener `/health` como alias de `/livez` (compat con lo ya documentado) o redirigir; decidir y ser consistente.

### 5. Healthchecks del compose (I5)

- Cada servicio Go: `healthcheck: test: ["CMD", "wget", "-q", "--spider", "http://localhost:PORT/readyz"]` (o el binario `curl`/`wget` disponible en la imagen — verificar; alpine trae wget de busybox).
- nginx: `depends_on: condition: service_healthy` para el tier de apps.
- `start_period: 90s` para search-api (Solr tarda en estar listo).

## Verificar

```bash
docker compose up -d --build
curl -s localhost:8081/readyz | jq .        # (o vía puertos internos) todos ok
curl -si localhost/hotels | grep -i x-request-id   # el header vuelve

# Correlación: pedir con ID propio y encontrarlo en los logs de nginx y del servicio
curl -s -H 'X-Request-ID: test-trace-123' localhost/hotels >/dev/null
docker compose logs nginx hotels-api | grep test-trace-123   # aparece en ambos, en JSON

# Readiness real: bajar mongo → hotels /readyz pasa a 503; nginx deja de rutear a esa instancia
docker compose stop mongo && sleep 5 && curl -si localhost:8081/readyz   # 503 con {"mongo":"down"}

# Debug mode fuera: los logs de arranque ya no muestran el warning de Gin ni el dump de rutas
docker compose logs hotels-api | grep -c "running in .debug. mode"   # → 0

# I5: docker compose ps → todos "healthy"; reinicio en frío no produce 502 tempranos
```

## Al terminar

Tildá el plan 05 en `plans/README.md`. Commit sugerido: `observability: slog JSON, request-id propagation, livez/readyz, gin release mode, container healthchecks (O1-O4,I5)`.
