# Fix F2 — users-api: login honesto, paginación determinística y clamp de bcrypt

> **Alcance:** RV6 (infra → 401), RV7 (timing oracle), RV8 (`ORDER BY`), RV9 (clamp `BCRYPT_COST`), RV10 (documentar tradeoff de caché stale en login)
> **Prerequisitos:** ninguno. Independiente de F1/F3/F4.
> **Esfuerzo:** ~1-2 h (S)
> Snippets validados contra el código del 2026-07-11.

## Contexto

- **RV6**: `Login` (`users-api/internal/services/users/users_service.go:155-163`) solo propaga `context.DeadlineExceeded`/`Canceled`; **cualquier otro error de infraestructura** (connection refused con MySQL caído — que falla en milisegundos sin agotar el timeout —, error de red, fila corrupta) colapsa a `ErrInvalidCredentials` → **401 para todos los usuarios** y sin rastro en logs. El sentinel correcto ya existe: `ErrUserNotFound` (`internal/repositories/users/users_mysql.go:34`), y `getByUsernameFromCaches` lo devuelve sin wrappear (`users_service.go:221-224`), así que `errors.Is` funciona directo.
- **RV7**: cuando el username no existe se responde 401 **sin ejecutar bcrypt** (~50-100 ms de diferencia medible con cost 10) → enumeración de usernames por timing.
- **RV8**: `GetAll` pagina sin `ORDER BY` (`users_mysql.go:95`) — `LIMIT/OFFSET` sobre orden indefinido puede repetir o saltear filas entre páginas.
- **RV9**: `hashPassword` (`users_service.go:268-271`) solo defiende `cost == 0`; `BCRYPT_COST=32` hace fallar `GenerateFromPassword` → **todos los registros dan 500**; 25–31 es un DoS práctico. La config está en `internal/config/config.go:37` (`BcryptCost = getIntEnv("BCRYPT_COST", 10)`).

## Pasos

### 1. Login: solo not-found es "credenciales inválidas" + costo constante (RV6 + RV7)

En `users_service.go`, reemplazar el bloque de `:155-167`:

```go
	user, err := s.getByUsernameFromCaches(ctx, username)
	if err != nil {
		if errors.Is(err, usersRepo.ErrUserNotFound) {
			// Igualar el costo del camino "no existe" al de un login real para
			// no filtrar existencia de usuarios por timing (RV7): el resultado
			// se descarta, solo importa quemar el mismo bcrypt.
			_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(password))
			return usersDomain.LoginResponse{}, ErrInvalidCredentials
		}
		// Infraestructura caída no es "credenciales inválidas" (RV6): se
		// propaga para que el controller responda 5xx en vez de 401.
		return usersDomain.LoginResponse{}, fmt.Errorf("error getting user for login: %w", err)
	}
```

Con, a nivel de paquete:

```go
// dummyBcryptHash es un hash válido de un valor descartable, usado solo para
// igualar el timing del camino "usuario inexistente" (RV7). Cost 10, como el
// default de producción.
var dummyBcryptHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
```

Notas de validación:
- Import nuevo: `usersRepo "github.com/.../users-api/internal/repositories/users"` — **verificar que no cree ciclo de imports** (el service no debería importar ya a repositories; si la interfaz del repo vive en el service y el sentinel en el paquete repo, el import es solo por el sentinel y es unidireccional: repo NO importa al service). Si hubiera ciclo, mover el sentinel a un paquete neutro o redeclararlo en el service y que el repo lo wrappee.
- El branch `DeadlineExceeded`/`Canceled` explícito de hoy queda cubierto por el fallthrough del wrap (el controller ya mapea deadline → 503 en su helper, `users_controller.go:45-46`; el resto cae al 500 "error during login" de `:201`). Comprobar que ningún test dependa del texto del error.

### 2. Paginación determinística (RV8)

`users_mysql.go:95`:

```go
	if err := repository.db.WithContext(ctx).Order("id ASC").Limit(limit).Offset(offset).Find(&usersList).Error; err != nil {
```

### 3. Clamp de `BCRYPT_COST` (RV9)

En `hashPassword` (`users_service.go:268-271`), reemplazar el `if cost == 0`:

```go
	cost := s.bcryptCost
	// Clamp defensivo (RV9): >31 hace fallar GenerateFromPassword (500 en todos
	// los registros) y 25-31 tarda minutos por hash; <MinCost lo salva la lib
	// pero mejor explícito. Rango sano para un servicio online: [10, 15].
	if cost < 10 || cost > 15 {
		slog.Warn("BCRYPT_COST out of sane range, using default", "configured", cost, "used", bcrypt.DefaultCost)
		cost = bcrypt.DefaultCost
	}
```

(Cubre también el `cost == 0` actual. `bcrypt.DefaultCost` = 10.)

### 4. Documentar el tradeoff de caché stale en login (RV10 — sin code-fix)

Con 3 réplicas y L1 por proceso, `invalidateCaches` (`users_service.go:240-248`) solo limpia la réplica que atendió el DELETE: durante ≤`CACHE_DURATION` (30 s) un usuario borrado puede loguearse contra otra réplica (Login lee a través de la caché) y obtener un JWT nuevo de 24 h; con JWT stateless sin revocación, el borrado nunca corta tokens ya emitidos. Es **inherente al diseño** (L1-por-réplica + JWT stateless) y aceptable para el alcance del proyecto — pero tiene que estar escrito, no descubierto:

- Comentario sobre `invalidateCaches` explicando la ventana multi-réplica y por qué se acepta.
- Punto en la sección de tradeoffs/roadmap del README (o donde el plan 12 la ponga): mitigaciones estándar si esto fuera producción (invalidación por pub/sub o bus, denylist de tokens, TTL corto + refresh tokens).

### 5. Tests

En `users_service_test.go`:

- Login con el mock del repo principal devolviendo un **error genérico** (no `ErrUserNotFound`): el error retornado NO es `ErrInvalidCredentials` (hoy este test fallaría — es la regresión de RV6).
- Login con `ErrUserNotFound`: sigue siendo `ErrInvalidCredentials` (401 en el controller).
- `hashPassword` con `bcryptCost` 32 y 0: no falla y produce un hash verificable con cost 10.

## Verificar

```bash
cd users-api && go test -race ./... && cd ..
make lint

# En vivo: parar MySQL y probar login → debe ser 5xx, NUNCA 401
docker compose stop mysql
curl -si -X POST localhost/login -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"DemoCliente123"}' | head -1   # esperado: 500/503
docker compose start mysql
# (esperar readyz) login correcto → 200; password mala → 401; username inexistente → 401

# Paginación estable:
curl -s "localhost/users?limit=2&offset=0" -H "Authorization: Bearer $ADMIN_JWT"
curl -s "localhost/users?limit=2&offset=2" -H "Authorization: Bearer $ADMIN_JWT"  # sin repetidos/salteados
```

Al terminar: tick en `plans/fixes/README.md` y sección nueva en `plans/HANDOFF.md`.
