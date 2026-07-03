# Plan 07 — Contratos de API: envelope, versionado, idempotencia

> **Alcance:** A1, A2, A3, A4, A5, A6, A7, A8
> **Base:** playbook 7.5 de `plantofinish.md` (A1, A2, A3, A6) + resto de la familia A por afinidad (A4, A5, A7, A8)
> **Prerequisitos:** plan 04 (la idempotencia compone con el inventario), plan 06 (rutas estables antes de versionar)
> **Esfuerzo:** ~1 día (L)

## Contexto

Superficie HTTP "hecha a mano por handler": errores que filtran texto de Mongo/Solr al cliente (A1), sin `/v1` (A2 — el rename C11 del plan 11 necesita esto), sin idempotencia en POST (A3), paginación y envelopes inconsistentes (A4, A5), semántica HTTP floja (A6), `user_id` int64 en users-api vs string en hotels-api (A7), sin content negotiation (A8).

## Correcciones de la Sección 7.0 que aplican

- **A1**: son **~18 sitios** de fuga (no 4): hotels-api `hotels_controller.go:49,73,104,174` y más; search-api filtra `strconv` y texto de Solr (`search_controller.go:39,48,56`). Falta además `error_page` 5xx en nginx: con upstream caído devuelve HTML default, no el envelope.
- **A2**: versionar rompe el **frontend** (base URL + proxy Vite) → actualizarlo en el mismo cambio; en nginx hay que cambiar **location y `proxy_pass` target** en cada bloque, **incluidas las 2 regex** de reservas-por-usuario; `health`/`livez`/`readyz` quedan **sin versionar**.
- **A3**: la clave de idempotencia ≠ `X-Request-ID` (ese es de tracing); header `Idempotency-Key` propio.

## Pasos

### 1. Paquete de error estándar (A1)

`apperr` **copiado en cada módulo** (`internal/` no cruza módulos; cuando exista más contrato compartido puede migrar a `platform-contracts`):

```go
func Abort(c *gin.Context, status int, code, msg string, cause error) {
  if cause != nil { _ = c.Error(cause) }              // la causa va al log...
  c.AbortWithStatusJSON(status, gin.H{"error": gin.H{ // ...nunca al body
    "code": code, "message": msg, "trace_id": c.GetString("request_id"),
  }})
}
```

- Reemplazar los ~18 sitios `fmt.Sprintf("...: %s", err.Error())` por `apperr.Abort` con códigos estables (`hotel_not_found`, `invalid_body`, `no_availability`, `internal`...).
- nginx: alinear 404/5xx al mismo shape:

```nginx
error_page 500 502 503 504 = @api_error;
location @api_error {
  default_type application/json;
  return 502 '{"error":{"code":"upstream_unavailable","message":"service temporarily unavailable"}}';
}
```

### 2. Versionado `/api/v1` (A2)

- `v1 := router.Group("/api/v1")` en los 3 servicios; `health`/`livez`/`readyz` **fuera** del grupo.
- nginx: prefijar cada `location` **y** su `proxy_pass` target, incluidas las 2 regex de `/users/{id}/reservations`.
- Frontend en el **mismo commit**: base URL del service layer + proxy de Vite.
- Documentar la estrategia de versionado en una línea del README (lo retoma el plan 12).

### 3. Idempotencia en POST (A3)

- Middleware para `POST /reservations` (y opcional `/hotels`): header `Idempotency-Key`.
- Store en Mongo: clave compuesta **`(Idempotency-Key, userID)`** con índice **único** + índice TTL 24h; guardar status+body de la primera respuesta y devolverla en replays.
- Replay **concurrente** (dos requests con la misma key en vuelo): el segundo insert da `IsDuplicateKeyError` → 409 con `code:"request_in_flight"`.
- Composición con el plan 04: la idempotencia dedupea reintentos del mismo cliente/key; el inventario evita overbooking entre usuarios distintos. Son capas complementarias, no redundantes.

### 4. Paginación consistente (A4)

- Una sola convención: `?limit&offset` con defaults (`20/0`) y clamp (max 100) en **todos** los list endpoints.
- `/search` deja de exigirlos (hoy: 400 con fuga `strconv.Atoi: parsing ""` — `search_controller.go:35-50`); usa los defaults.

### 5. Envelope de respuesta único (A5)

- `{"data": ..., "meta": {"total": n, "limit": l, "offset": o}}` en listas; objeto pelado dentro de `data` en gets.
- Es breaking → entra junto con `/api/v1` (el paso 2 es el punto natural de corte) actualizando el service layer del frontend en el mismo cambio. Reemplaza el interino `X-Total-Count` del plan 03/DB5.

### 6. Semántica HTTP (A6)

- 201 + header `Location: /api/v1/hotels/{id}` en creates (`hotels_controller.go:79,180`).
- DELETE → **204** sin body; PUT → 200 con la representación actualizada (hoy `{message:id}` — `:110,131`).

### 7. Unificar tipo de `user_id` (A7)

- Canónico: **string** (es lo que viaja en el JWT y usa hotels-api en `reservations.go:9`). users-api sigue con PK int64 interna pero **serializa `id` como string** en sus DTOs de respuesta (`json:"id,string"` o mapping explícito en `users_domain.go:13,21`).
- Dejarlo fijado con un comentario en `platform-contracts` (el tipo compartido formal puede llegar con más contratos).
- Breaking para el frontend → mismo cambio versionado del paso 2/5.

### 8. Content negotiation (A8)

- Decisión documentada: **JSON-only**. Middleware simple: si hay `Accept` y no admite `application/json` → 406 con el envelope. `Content-Type: application/json` garantizado también en errores (apperr ya lo hace vía `AbortWithStatusJSON`).

## Verificar

```bash
# Envelope de error sin internals
curl -s localhost/api/v1/hotels/nonexistent | jq .   # {"error":{"code":"hotel_not_found",...,"trace_id":...}} sin texto de Mongo
docker compose stop hotels-api && curl -s localhost/api/v1/hotels | jq .error.code   # "upstream_unavailable" (JSON, no HTML)

# Versionado
curl -si localhost/api/v1/search?q=spa | head -1     # 200
curl -si localhost/hotels | head -1                  # 404 (o redirect, según decisión)
curl -si localhost/health | head -1                  # 200 sin versión

# Idempotencia: mismo key 2 veces → misma respuesta, una sola reserva
KEY=$(uuidgen)
curl -s -X POST localhost/api/v1/reservations -H "Idempotency-Key: $KEY" -H "Authorization: Bearer $TOKEN" -d '{...}'
curl -s -X POST localhost/api/v1/reservations -H "Idempotency-Key: $KEY" -H "Authorization: Bearer $TOKEN" -d '{...}'
# → mismos body/status; en Mongo hay 1 reserva

# Semántica
curl -si -X POST localhost/api/v1/hotels ... | grep -i '^location:'   # presente
curl -si -X DELETE localhost/api/v1/reservations/$RID ... | head -1   # 204

# Paginación con defaults
curl -s 'localhost/api/v1/search?q=hotel' | jq .meta                  # sin offset/limit explícitos → 200

# user_id como string en users-api
curl -s localhost/api/v1/users/1 -H "Authorization: Bearer $ATOKEN" | jq '.data.id | type'   # "string"

# Frontend sigue funcionando end-to-end (login → búsqueda → reserva)
cd frontend && npm run build
```

## Al terminar

Tildá el plan 07 en `plans/README.md`. Commit sugerido: `api: problem envelope, /api/v1, idempotency keys, consistent pagination/envelopes/semantics, user_id type (A1-A8)`.
