## REGLAS DEL PROYECTO (hotels-api)

Estas reglas son **no negociables** para IA y desarrolladores. Mantener alineado con:
- `cmd/main.go` (rutas + middlewares)
- `internal/services/*` (orquestación, cache, eventos)
- `README_HOTELS.md` (documentación de alto nivel)

---

## 1) Arquitectura por capas (obligatoria)

Regla base: **nada de lógica de negocio en controllers**.

- `internal/controllers/`:
  - parseo de request (params, JSON), status codes, respuesta JSON
  - NO DB, NO cache, NO lógica de fechas/negocio
- `internal/services/`:
  - lógica de negocio/orquestación
  - cache-aside
  - conversiones DAO ↔ Domain
  - eventos (cuando aplique)
- `internal/repositories/`:
  - acceso a datos (Mongo o Cache)
  - sin decisiones de negocio
- `internal/domain/`:
  - modelos de dominio (contrato JSON que expone la API)
- `internal/dao/`:
  - modelos/persistencia (Mongo)

---

## 2) Contrato HTTP (respetar routing real)

**Fuente de verdad:** `cmd/main.go`.

Reglas:
- No “inventar” endpoints ni cambiar paths sin actualizar `cmd/main.go` + tests + README.
- Rutas admin viven bajo `/admin/*`.
- El JSON de errores usa `{"error": "..."}`
- Respuestas “ok” típicas usan `200/201` y a veces `{"message": id}` o `{"id": id}` según controller.

---

## 3) JWT / AuthN / AuthZ (obligatorio)

### Claims esperados
El middleware JWT (`internal/middlewares/auth.go`) espera:
- `tipo`: string (`cliente` o `administrador`)
- `user_id`: puede venir como número o string (el middleware lo normaliza a string)

### Reglas de autorización
- **Admin** (`tipo=administrador`):
  - CRUD hoteles (`/admin/hotels...`)
  - endpoints de microservices (`/admin/microservices...`)
- **Usuario autenticado** (cualquier `tipo` válido):
  - crear/cancelar reservas
  - consultar reservas por usuario en rutas protegidas
- **Reglas de ownership**:
  - un usuario solo puede **crear** reservas para su propio `user_id`
  - un usuario solo puede **cancelar** sus propias reservas

### Qué NO hacer
- No bypass de auth en tests con `ctx.Set("userID")` manual (salvo unit tests de controller muy específicos).

---

## 4) Cache-aside y consistencia (obligatorio)

### Principios
- Lecturas: **cache-aside** (cache → fallback main → poblar cache).
- Escrituras: primero persistir en **main repository**, luego reflejar en **cache**.

### Alcance actual (no inventar)
- `GetHotelByID`: cache-aside, pobla cache en miss.
- `GetReservationByID`: cache-aside, pobla cache en miss (si falla “set cache”, no debería romper la lectura).
- Listas de reservas (`GetReservationsByHotelID`, `GetReservationsByUserID`, `GetReservationsByUserAndHotelID`):
  - cache → fallback main → se guardan reservas en cache (por reserva) para poblar listas agregadas.
- `GetAvailability`:
  - intenta cache; si falla usa main (no cachea el resultado del main actualmente).

### TTL
- TTL y tamaño se configuran en `internal/config/config.go` (default actual: 30s).

---

## 5) Eventos (RabbitMQ) - alcance actual

Reglas:
- Solo se publican eventos para **CRUD de hoteles** (`HotelNew` con `Operation=CREATE|UPDATE|DELETE`).
- No documentar ni implementar eventos de reservas salvo requerimiento explícito.
- Si RabbitMQ no está disponible, el publish puede fallar y el CRUD de hoteles puede devolver error (diseño actual).

---

## 6) “Microservices controller” (nota importante)

El controller `internal/controllers/microservices` es **simulado/mock**:
- No interactúa realmente con Docker/Kubernetes.
- Se usa como endpoint administrativo de demostración.
- Mantenerlo bajo `/admin/*` y con `AdminOnly()`.

---

## 7) Naming y compatibilidad (no romper)

Reglas:
- No renombrar campos públicos sin migración/compatibilidad.
- Ojo con el naming existente: `AvaiableRooms` (se mantiene por contrato actual).
- No cambiar firmas de interfaces (`Service`/`Repository`) sin actualizar implementaciones y tests.

---

## 8) Testing (mínimo obligatorio)

### Services
- Unit tests con repos mocks (main + cache) y queue mock.
- Validar fechas: el día de checkout es excluido en ocupación (según lógica actual).

### Controllers
- Tests con Gin + `httptest`, usando rutas reales (incl. `/admin`).
- JWT real en `Authorization: Bearer ...` para probar 401/403 y happy paths.
- Mantener un set pequeño, de alto valor (routing + auth + 400 por bind).

---

## 9) Prohibiciones explícitas

- NO lógica de negocio en controllers.
- NO DB/calls a repos desde controllers.
- NO mezclar DAO y Domain fuera de `services`.
- NO inventar campos/claims.
- NO introducir env vars “de mentira” (si se cambia config a env, hacerlo end-to-end).
