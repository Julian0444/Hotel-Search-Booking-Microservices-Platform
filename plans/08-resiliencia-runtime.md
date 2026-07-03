# Plan 08 — Resiliencia de runtime (prerequisito de k8s)

> **Alcance:** R2, R4, R5, C12, C14
> **Base:** sin playbook — redactado desde los IDs de las Secciones 2 y 6
> **Prerequisitos:** plan 04 (el fan-out de disponibilidad ya reescrito), plan 06 (cliente HTTP con timeout ya existe). **Va antes del plan 09 (k8s):** el drain limpio de pods en rolling deploys depende del manejo de SIGTERM (C12).
> **Esfuerzo:** ~1 día

## Contexto

Hay scaffolding real (LB, reconexión del productor RabbitMQ) pero la capa de aplicación casi no tiene tolerancia a fallos por request: sin graceful shutdown ni drenado ante SIGTERM (C12, `hotels-api/cmd/main.go:107-110`), sin deadlines en Mongo/Solr y el consumer usa `context.Background()` (R2, `search_service.go:81,111,119,129`), `GetAvailability` lanza una goroutine por hotel sin límite y aborta todo el batch si uno falla (R4, `hotels_mongo.go:342-351,357-359`), sin circuit breaker en llamadas inter-servicio (R5), y el productor RabbitMQ no reconecta entre reintentos de `Publish` (C14, `queue_rabbit.go:203-247`).

## Solapamientos con planes anteriores (verificar antes de codear)

- El **ping al arranque** de C12 probablemente ya lo hizo el plan 03 (DB3) → verificar y saltear.
- El **`$unionWith: {coll: nil}`** y el **cursor por día** de C14 probablemente desaparecieron con la reescritura de `IsHotelAvailable` del plan 04 → verificar; si quedó algún cursor en loop, cerrar por iteración.
- El **timeout del cliente HTTP** de search (E4) ya lo hizo el plan 06 — R2 acá es deadlines en Mongo/Solr/consumer.

## Pasos

### 1. Graceful shutdown en los 3 servicios (C12)

Reemplazar `router.Run(...)` por:

```go
srv := &http.Server{Addr: ":" + config.Port, Handler: router,
  ReadHeaderTimeout: 5 * time.Second}
go func() {
  if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
    slog.Error("server error", "err", err); os.Exit(1)
  }
}()
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
<-ctx.Done()
slog.Info("shutting down")
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = srv.Shutdown(shutdownCtx)   // drena requests en vuelo
queue.Close()                    // cierra RabbitMQ limpio
_ = mongoClient.Disconnect(shutdownCtx)
```

- En search-api además: al recibir la señal, **dejar de consumir** (cancelar el consumer) y terminar el mensaje en vuelo antes de salir (con manual ack del plan 06, un mensaje a medias se re-entrega — pero salir limpio evita el retrabajo).
- Nota k8s (plan 09): esto es lo que hace que un rolling restart no tire 502 — el pod termina de servir lo que tiene tras el SIGTERM del kubelet.

### 2. Deadlines en toda llamada saliente (R2)

- hotels-api: `context.WithTimeout(ctx, 3*time.Second)` en cada operación Mongo del repo (el ctx viene del request desde el plan 03/04).
- search-api: deadline en cada llamada a Solr; el **consumer** deja de usar `context.Background()`: un `context.WithTimeout(5s)` **por mensaje** (`search_service.go:81,111,119,129`).
- Driver Mongo: `SetSocketTimeout` además del `SetServerSelectionTimeout` que puso el plan 03.

### 3. Bulkhead en el fan-out de disponibilidad (R4)

`hotels_mongo.go:342-359` (o su forma post-plan-04):

```go
g, gctx := errgroup.WithContext(ctx)   // golang.org/x/sync/errgroup
g.SetLimit(8)                          // límite de concurrencia: bulkhead
```

- Devolver **disponibilidad parcial** en vez de abortar el batch: un hotel que falla se marca `available:false` (o se omite con log), los demás responden. Cambiar el `errgroup` por recolección de errores por-item si `errgroup` cancela demasiado agresivo.

### 4. Circuit breaker + retry con jitter (R5)

- `github.com/sony/gobreaker` alrededor del repo HTTP search→hotels (el único hop HTTP inter-servicio):

```go
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
  Name: "hotels-api", MaxRequests: 3, Timeout: 30 * time.Second,
  ReadyToTrip: func(c gobreaker.Counts) bool { return c.ConsecutiveFailures >= 5 },
})
```

- Retry: 2 reintentos con backoff + jitter solo para errores transitorios (conexión, 5xx) — nunca en 4xx.
- Opcional: exponer el estado del breaker en `/readyz` (plan 05).

### 5. Robustez fina del productor RabbitMQ (C14)

`hotels-api/.../queue_rabbit.go:203-247`: entre reintentos de `Publish`, forzar `ensureConnection()` — hoy los 3 intentos pegan al mismo canal muerto. Verificar el estado de los otros dos sub-ítems de C14 (ver "Solapamientos" arriba).

## Verificar

```bash
# C12: SIGTERM drena y sale limpio (exit 0, rápido)
docker compose kill -s SIGTERM hotels-api && docker compose logs --tail 5 hotels-api  # "shutting down", sin panics
docker inspect --format '{{.State.ExitCode}}' $(docker compose ps -q hotels-api)      # 0

# Bajo carga: mandar requests continuos y reiniciar el servicio → sin errores de conexión cortada a mitad
( while true; do curl -so /dev/null -w "%{http_code}\n" localhost/api/v1/hotels; sleep 0.1; done ) &
docker compose restart hotels-api   # los in-flight terminan; el LB cubre el hueco
kill %1

# R2: bajar Solr → el consumer no se cuelga; cada mensaje falla en ≤5s y va a retry/DLQ
# R4: disponibilidad de un rango con muchos hoteles → responde parcial aunque un hotel falle
# R5: bajar hotels-api → tras 5 fallos consecutivos el breaker abre (logs "circuit breaker open");
#      levantar hotels-api → half-open → cerrado, sin martillar durante la caída

go test -race ./...   # via go.work
```

## Al terminar

Tildá el plan 08 en `plans/README.md`. Commit sugerido: `resilience: graceful shutdown, outbound deadlines, bounded fan-out, circuit breaker, producer retry fix (C12,C14,R2,R4,R5)`.
