# Plan 01 — Seguridad: auth hardening + secretos

> **Alcance:** S1, S2, C4, C8, I1, SD2, SD3, T5
> **Base:** playbook 7.1 de `plantofinish.md` + ítems de seguridad sin playbook (C8, SD2, SD3, T5, I1)
> **Prerequisitos:** ninguno — es el primer plan (bloqueante de seguridad)
> **Esfuerzo:** ~1 día

## Contexto

- **S1**: cualquiera se registra como `administrador` — `POST /users` sin auth, el service acepta `tipo` del body verbatim (`users_service.go:94-101`, `users_controller.go:77-103`), el dropdown del Register ofrece "Administrator" (`frontend/src/pages/Register.jsx:254-255`) y ese JWT desbloquea todo `/admin/*` de hotels-api (`hotels-api/.../auth.go:92`).
- **S2**: `users-api` no tiene NINGÚN middleware de auth: `GET /users`, `GET /users/:id` y `DELETE /users/:id` están 100% abiertos (`users-api/cmd/main.go:61-65`, `users_controller.go:107-124`).
- S1 y S2 son el mismo problema: users-api emite JWT pero nunca implementó el lado *verify*.

## Correcciones de la Sección 7.0 que aplican

- **S1**: forzar `tipo="cliente"` **solo en el handler público** (`Controller.Create`), NO en `service.Create` — el seed de admin lo necesita intacto. Un endpoint admin bajo `/admin` sería inalcanzable (nginx rutea `/admin/*` a hotels-api) → el primer admin se crea vía **seed env-driven**. El test `TestController_Create` codifica la vulnerabilidad y se pondrá rojo: actualizarlo.
- **C4**: `getEnv` devuelve el default si la env var está vacía → el secreto **nunca** es `""`; el guard debe comparar contra el **placeholder literal**, y va en **ambos** servicios (`users-api/internal/config/config.go:25` y `hotels-api/internal/config/config.go:32` comparten el mismo default peligroso). **Ojo con los nombres**: el campo se llama `JWTKey` en users-api pero **`JWTSecret`** en hotels-api.

## Pasos

### 1. Middleware de auth en users-api (S2)

Crear `users-api/internal/middlewares/auth.go` (`package middleware` — mismo quirk de hotels-api: package singular en dir plural; la convención se arregla en CQ4/plan 11). Copiar de hotels-api **verbatim**: el struct `JWTMiddleware{SecretKey}` + su constructor `NewJWTMiddleware(secretKey)` + el método `Authenticate()` (`auth.go:14-22`), y la función libre `AdminOnly()` (`auth.go:84`). Agregar `OwnerOrAdmin()` (nuevo — hotels-api no lo tiene):

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

Nota: `golang-jwt/jwt/v5` ya es dependencia de users-api (no hace falta `go get`). Las réplicas de users-api no exponen puertos → verificar todo por el gateway `:80`.

### 2. Forzar rol en el registro público (S1) — con DTO propio

Hecho verificado: hoy `Controller.Create` bindea `usersDomain.LoginRequest` (`users_controller.go:78-79`), **el mismo struct que bindea `Login`** (`:128-148`). No agregarle tags de política a `LoginRequest` (rompería el Login — ver paso 8).

- Crear `RegisterRequest` en `users_domain.go` (los tags de política van acá, paso 8).
- `Controller.Create` bindea `RegisterRequest`, fuerza `Tipo = "cliente"` y lo mapea a `LoginRequest` para llamar a `service.Create`. **No** tocar `service.Create` (el seed lo necesita intacto).

### 3. Config para el seed de admin

Agregar `SeedAdminUsername` / `SeedAdminPassword` (envs `ADMIN_USERNAME` / `ADMIN_PASSWORD`) en `users-api/config.go`.

### 4. `cmd/main.go`: fail-fast, grupos de rutas y seed (S2, C4)

a) Fail-fast del secreto — en users-api:

```go
if config.JWTKey == "" || config.JWTKey == "your-secret-key-change-in-production" {
  log.Fatal("JWT_SECRET must be set to a non-default value")
}
```

Repetir el guard en `hotels-api/cmd/main.go`, pero ahí el campo es **`config.JWTSecret`** (`hotels-api/internal/config/config.go:32`) — con `JWTKey` no compila.

b) Wiring del middleware + grupos de rutas (hoy las rutas viven en `main.go:61-65`):

```go
jwtMiddleware := middleware.NewJWTMiddleware(config.JWTKey)   // Authenticate es método del struct

router.POST("/users", controller.Create); router.POST("/login", controller.Login) // públicos
auth := router.Group("/", jwtMiddleware.Authenticate())
auth.GET("/users", middleware.AdminOnly(), controller.GetAll)         // AdminOnly es función libre
auth.GET("/users/:id", middleware.OwnerOrAdmin(), controller.GetByID)
auth.DELETE("/users/:id", middleware.OwnerOrAdmin(), controller.Delete)
```

c) Seed idempotente del admin: si `ADMIN_USERNAME`/`ADMIN_PASSWORD` están seteados, crear el admin al arranque vía `service.Create` tragando el error de duplicado (las 3 réplicas corren el seed; el índice único de `username` hace fallar a 2 — es esperado).

### 5. Arreglar tests (S1) + tests negativos nuevos (T5)

- `TestController_Create` (`users_controller_test.go:165-237`) tiene **4 subtests** — qué hacer con cada uno:
  - `"invalid body -> 400"`: queda como está.
  - `"validation error -> 400"` (manda `tipo:"hacker"` y mockea el service devolviendo `invalid tipo`): **se elimina** — con el handler forzando `tipo="cliente"`, el service nunca recibe un tipo inválido desde el endpoint público.
  - `"success -> 201"`: su body usa password `"pass123"` (**7 chars**) → con `min=8` ahora falla el binding: actualizar el body (password ≥8) y la expectativa del mock (`Create` llamado con `Tipo:"cliente"`).
  - `"create admin -> 201"`: renombrar a "public register cannot self-assign admin"; su body usa password `"secret"` (**6 chars**) → actualizarlo también; el mock espera `Create` con `Tipo:"cliente"`.
  - Los service tests (`TestService_Create`: "success with admin tipo", "invalid tipo") **no se tocan** — el service queda intacto para el seed.
- **T5 (nuevo):**
  - Test de `GenerateToken` en `tokenizers/`: round-trip de claims (userID, tipo, exp, iss/aud del paso 7).
  - Tests negativos del middleware `Authenticate` (en ambos servicios): token expirado → 401, firma con otra key → 401, `alg=none` → 401, header malformado (`Bearer` sin token, token basura) → 401.

### 6. Compose + frontend + docs mínimos

- `ADMIN_USERNAME`/`ADMIN_PASSWORD` en las 3 réplicas de users-api en `docker-compose.yml` (services `users-api-1/2/3`).
- **Register.jsx**: quitar el **selector de tipo completo** (no solo la opción admin — un selector con una sola opción no tiene sentido). El selector es un `<Controller name="tipo">` con MenuItems en `:254-255` que usan `USER_ROLES.CLIENT/ADMIN` (`frontend/src/constants/index.js:21-23`). Al quitarlo: ajustar `onSubmit` (`Register.jsx:57-58`) y `AuthContext.register` (`AuthContext.jsx:87,92`) para no pasar `tipo` — `authService.register` ya tiene default `tipo='cliente'` (`auth.service.js:37`).
- **users.md:129-133**: la sección "Registrar administrador" documenta el curl sin token que crea un admin — actualizarla (el admin ahora sale del seed). Hoy documenta la vulnerabilidad como feature.
- **Bruno**: `Post User.bru:17,23` muestra `tipo` elegible incluyendo administrador → actualizar la doc del request. Get All Users / Get User by ID / Delete User ya declaran `auth: bearer` — coinciden con la nueva postura, solo revisar las notas.
- Actualización mínima del README (tabla `README.md:202-204` con Auth `—`) para no contradecir la nueva postura (la pasada completa de docs es el plan 12).

### 7. Secretos a `.env` (I1)

Evidencia: `docker-compose.yml:37,65-66,85-86,139,165,193,235` (root/root en todas las DBs + `JWT_SECRET: ThisIsAnExampleJWTKey!` compartido, commiteados). **El alcance real es mayor**: los consumidores repiten las mismas creds en claro — users-api `MYSQL_USERNAME/PASSWORD` (`:135-136,:161-162,:189-190`), hotels-api `MONGO_*`/`RABBIT_*` (`:222-223,:232-233`), search-api `RABBIT_*` (`:261-262`). Hoy no hay `env_file` ni `${}` en ninguna parte del compose.

- Reemplazar cada literal por `${MYSQL_ROOT_PASSWORD}`, `${MONGO_PASSWORD}`, `${RABBIT_PASSWORD}`, `${JWT_SECRET}`, `${ADMIN_PASSWORD}`, etc. — en los servicios de infra **y** en todos los consumidores listados arriba.
- Nota: el valor actual del secreto no es el placeholder (el guard de C4 protege el default de código); el `.env` lleva el valor de demo.
- Crear `.env` (gitignored) con valores reales de demo y `.env.example` con placeholders (queda en el repo; el versionado lo hace el usuario).
- Crear `.gitignore` raíz con al menos `.env` (el plan 02/I9 lo completa con el resto).
- Nota en README: son credenciales de demo local.

### 8. Validación de entrada + política de password (C8, SD2)

Evidencia: `users_service.go:86-92` (sin longitud/complejidad), `reservations.go:5-12` (sin tags de binding).

- Los tags de política van en el **`RegisterRequest` nuevo** del paso 2 — NO en `LoginRequest`, que también bindea `Login` (ponerle `min=8` ahí rompería el login de usuarios existentes con passwords cortas, devolviendo 400 en vez de 401):

```go
type RegisterRequest struct {
  Username string `json:"username" binding:"required,min=3,max=50"`
  Password string `json:"password" binding:"required,min=8,max=72"`
  Tipo     string `json:"tipo,omitempty"` // ignorado: el handler siempre fuerza "cliente"
}
```

- Guard explícito de los 72 bytes de bcrypt (trunca en silencio): rechazarlo en `hashPassword` (`users_service.go:247-258`) — así cubre también el path del seed, no solo el binding del endpoint público.
- (No hay campo `nombre` en users-api — el display name llega recién con DM7/plan 13.)
- Tags `binding:"required"` en el DTO de reservas (la validación semántica de fechas se profundiza en el plan 04 — acá solo `required`).
- Mensajes 400 claros (campo + regla), sin filtrar internals.

### 9. Claims `iss`/`aud`/`nbf` en el JWT (SD3)

Evidencia: `tokenizers_jwt.go:38-44` (emisión sin iss/aud), `auth.go:37-48` (validación sin scoping), secreto único compartido en compose.

- Emitir en users-api: `iss="users-api"`, `aud=["hotels-api"]`, `nbf=now`.
- Validar en el middleware de hotels-api (y en el nuevo de users-api): `jwt.WithIssuer("users-api")`, `jwt.WithAudience(...)`, `jwt.WithLeeway(30*time.Second)`.
- Ojo: los tokens viejos dejan de validar — aceptable (demo local, TTL 24h).

## Verificar

```bash
# 1. Intento de auto-registro como admin → el JWT resultante dice cliente
curl -s -X POST http://localhost/users -H 'Content-Type: application/json' \
  -d '{"username":"mallory","password":"Password123","tipo":"administrador"}'
TOKEN=$(curl -s -X POST http://localhost/login -H 'Content-Type: application/json' \
  -d '{"username":"mallory","password":"Password123"}' | jq -r .token)
echo $TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq .   # → "tipo":"cliente", iss/aud presentes

# 2. Endpoints protegidos
curl -i http://localhost/users                              # → 401
curl -i -X DELETE http://localhost/users/1                  # → 401
curl -i -H "Authorization: Bearer $TOKEN" http://localhost/users        # → 403 (no admin)
curl -i -H "Authorization: Bearer $TOKEN" http://localhost/users/1      # → 403 si el id 1 no es mallory

# 3. Admin del seed puede todo
ATOKEN=$(curl -s -X POST http://localhost/login -H 'Content-Type: application/json' \
  -d "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\"}" | jq -r .token)
curl -i -H "Authorization: Bearer $ATOKEN" http://localhost/users       # → 200

# 4. Password débil rechazada
curl -i -X POST http://localhost/users -H 'Content-Type: application/json' \
  -d '{"username":"weak","password":"1","tipo":"cliente"}'  # → 400

# 5. Fail-fast del secreto: levantar users-api sin JWT_SECRET → el contenedor muere con log claro

# 6. Tests
cd users-api && go test ./... && cd ../hotels-api && go test ./...

# 7. Secretos fuera de git
git grep -n "root:root\|your-secret-key" -- docker-compose.yml   # → sin hits
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
