# Plan 01 — Seguridad: auth hardening + secretos

> **Alcance:** S1, S2, C4, C8, I1, SD2, SD3, T5
> **Base:** playbook 7.1 de `plantofinish.md` + ítems de seguridad sin playbook (C8, SD2, SD3, T5, I1)
> **Prerequisitos:** ninguno — es el primer plan (bloqueante de seguridad)
> **Esfuerzo:** ~1 día

## Contexto

- **S1**: cualquiera se registra como `administrador` — `POST /users` sin auth, el service acepta `tipo` del body verbatim (`users_service.go:95-101`, `users_controller.go:86`), el dropdown del Register ofrece "Administrator" (`frontend/src/pages/Register.jsx:255`) y ese JWT desbloquea todo `/admin/*` de hotels-api (`hotels-api/.../auth.go:92`).
- **S2**: `users-api` no tiene NINGÚN middleware de auth: `GET /users`, `GET /users/:id` y `DELETE /users/:id` están 100% abiertos (`users-api/cmd/main.go:58-64`, `users_controller.go:107-124`).
- S1 y S2 son el mismo problema: users-api emite JWT pero nunca implementó el lado *verify*.

## Correcciones de la Sección 7.0 que aplican

- **S1**: forzar `tipo="cliente"` **solo en el handler público** (`Controller.Create`), NO en `service.Create` — el seed de admin lo necesita intacto. Un endpoint admin bajo `/admin` sería inalcanzable (nginx rutea `/admin/*` a hotels-api) → el primer admin se crea vía **seed env-driven**. El test `TestController_Create` codifica la vulnerabilidad y se pondrá rojo: actualizarlo.
- **C4**: `getEnv` devuelve el default si la env var está vacía → `JWTKey` **nunca** es `""`; el guard debe comparar contra el **placeholder literal**, y va en **ambos** servicios (`users-api/config.go:25` y `hotels-api/config.go:32` comparten el mismo default peligroso).

## Pasos

### 1. Middleware de auth en users-api (S2)

Crear `users-api/internal/middlewares/auth.go` (`package middleware`): copiar `Authenticate` + `AdminOnly` de hotels-api **verbatim** y agregar `OwnerOrAdmin()` (nuevo — hotels-api no lo tiene):

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

### 2. Forzar rol en el registro público (S1)

En `Controller.Create`, justo tras `ShouldBindJSON`: `request.Tipo = "cliente"`. **No** tocar `service.Create`.

### 3. Config para el seed de admin

Agregar `SeedAdminUsername` / `SeedAdminPassword` (envs `ADMIN_USERNAME` / `ADMIN_PASSWORD`) en `users-api/config.go`.

### 4. `cmd/main.go`: fail-fast, grupos de rutas y seed (S2, C4)

a) Fail-fast del secreto (repetir el mismo guard en `hotels-api/cmd/main.go`):

```go
if config.JWTKey == "" || config.JWTKey == "your-secret-key-change-in-production" {
  log.Fatal("JWT_SECRET must be set to a non-default value")
}
```

b) Grupos de rutas:

```go
router.POST("/users", controller.Create); router.POST("/login", controller.Login) // públicos
auth := router.Group("/", jwtMiddleware.Authenticate())
auth.GET("/users", middleware.AdminOnly(), controller.GetAll)
auth.GET("/users/:id", middleware.OwnerOrAdmin(), controller.GetByID)
auth.DELETE("/users/:id", middleware.OwnerOrAdmin(), controller.Delete)
```

c) Seed idempotente del admin: si `ADMIN_USERNAME`/`ADMIN_PASSWORD` están seteados, crear el admin al arranque vía `service.Create` tragando el error de duplicado (las 3 réplicas corren el seed; el índice único de `username` hace fallar a 2 — es esperado).

### 5. Arreglar tests (S1) + tests negativos nuevos (T5)

- `users_controller_test.go`: los 3 subtests de `Create` esperan ahora `Tipo:"cliente"`; renombrar "create admin→201" a "public register cannot self-assign admin". El subtest **"invalid tipo → 400" se elimina** (no alcanza con cambiarle el `Tipo` esperado): con el handler forzando `tipo="cliente"`, ese caso queda inalcanzable.
- **T5 (nuevo):**
  - Test de `GenerateToken` en `tokenizers/`: round-trip de claims (userID, tipo, exp, iss/aud del paso 7).
  - Tests negativos del middleware `Authenticate` (en ambos servicios): token expirado → 401, firma con otra key → 401, `alg=none` → 401, header malformado (`Bearer` sin token, token basura) → 401.

### 6. Compose + frontend + docs mínimos

- `ADMIN_USERNAME`/`ADMIN_PASSWORD` en las 3 réplicas de users-api en `docker-compose.yml`.
- Quitar la opción "Administrator" del dropdown de `Register.jsx` (línea ~255).
- Actualización mínima de README/Bruno para que no contradigan la nueva postura (la pasada completa de docs es el plan 12).

### 7. Secretos a `.env` (I1)

Evidencia: `docker-compose.yml:37,65-66,85-86,139,165,193,235` (root/root en todas las DBs + `JWT_SECRET` compartido, commiteados).

- Reemplazar cada literal por `${MYSQL_ROOT_PASSWORD}`, `${MONGO_PASSWORD}`, `${JWT_SECRET}`, `${ADMIN_PASSWORD}`, etc.
- Crear `.env` (gitignored) con valores reales de demo y `.env.example` commiteado con placeholders.
- Crear `.gitignore` raíz con al menos `.env` (el plan 02/I9 lo completa con el resto).
- Nota en README: son credenciales de demo local.

### 8. Validación de entrada + política de password (C8, SD2)

Evidencia: `users_service.go:86-92` (sin longitud/complejidad), `reservations.go:5-12` (sin tags de binding).

- Tags `binding` de gin en el DTO de registro: `username` `required,min=3,max=50`; `password` `required,min=8,max=72`; `nombre` `required,max=100`.
- Guard explícito de los 72 bytes de bcrypt (trunca en silencio): rechazar passwords > 72 bytes con 400 claro (`users_service.go:247-258`).
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
  -d '{"nombre":"Mallory","username":"mallory","password":"Password123","tipo":"administrador"}'
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
  -d '{"nombre":"x","username":"weak","password":"1","tipo":"cliente"}'  # → 400

# 5. Fail-fast del secreto: levantar users-api sin JWT_SECRET → el contenedor muere con log claro

# 6. Tests
cd users-api && go test ./... && cd ../hotels-api && go test ./...

# 7. Secretos fuera de git
git grep -n "root:root\|your-secret-key" -- docker-compose.yml   # → sin hits
```

## Al terminar

Tildá el plan 01 en `plans/README.md`. Commit sugerido: `security: JWT auth in users-api, no self-service admin, secrets to .env (S1,S2,C4,C8,I1,SD2,SD3,T5)`.
