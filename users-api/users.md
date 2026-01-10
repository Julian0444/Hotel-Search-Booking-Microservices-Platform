# Users API

API de gestión de usuarios y autenticación JWT para el sistema de reservas de hoteles.

## Endpoints

| Método | Ruta          | Descripción                              |
|--------|---------------|------------------------------------------|
| GET    | `/health`     | Health check del servicio                |
| GET    | `/users`      | Lista todos los usuarios                 |
| GET    | `/users/:id`  | Obtiene un usuario por ID                |
| POST   | `/users`      | Crea un nuevo usuario (registro)         |
| DELETE | `/users/:id`  | Elimina un usuario                       |
| POST   | `/login`      | Autentica y retorna JWT                  |

## Modelos

### LoginRequest
Se usa tanto para **login** como para **registro** de usuarios.

```json
{
  "username": "string (required)",
  "password": "string (required)",
  "tipo": "string (optional: 'cliente' | 'administrador', default: 'cliente')"
}
```

### LoginResponse
Respuesta del endpoint `/login`.

```json
{
  "user_id": 1,
  "username": "user1",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "tipo": "cliente"
}
```

### User
Respuesta de endpoints que retornan usuarios (sin password).

```json
{
  "id": 1,
  "username": "user1",
  "tipo": "cliente"
}
```

## Configuración (Variables de Entorno)

| Variable             | Default                              | Descripción                    |
|----------------------|--------------------------------------|--------------------------------|
| `MYSQL_HOST`         | `localhost`                          | Host de MySQL                  |
| `MYSQL_PORT`         | `3306`                               | Puerto de MySQL                |
| `MYSQL_DATABASE`     | `users_db`                           | Nombre de la base de datos     |
| `MYSQL_USERNAME`     | `root`                               | Usuario de MySQL               |
| `MYSQL_PASSWORD`     | `root`                               | Contraseña de MySQL            |
| `MEMCACHED_HOST`     | `localhost`                          | Host de Memcached              |
| `MEMCACHED_PORT`     | `11211`                              | Puerto de Memcached            |
| `JWT_SECRET`         | `your-secret-key-change-in-production` | Clave secreta para JWT       |
| `JWT_DURATION`       | `24h`                                | Duración del token             |
| `BCRYPT_COST`        | `10`                                 | Costo de hashing bcrypt        |
| `PORT`               | `8082`                               | Puerto del servidor            |
| `CACHE_DURATION`     | `30s`                                | TTL del cache L1               |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:5173` | Origins CORS permitidos |

## JWT Claims

El token JWT generado es compatible con `hotels-api`:

```json
{
  "username": "user1",
  "user_id": 1,
  "tipo": "cliente",
  "iat": 1704825600,
  "exp": 1704912000
}
```

**Importante**: `JWT_SECRET` debe coincidir entre `users-api` y `hotels-api`.

## Arquitectura

```
┌─────────────────┐
│   Controller    │  ← HTTP handlers (Gin)
├─────────────────┤
│    Service      │  ← Lógica de negocio (bcrypt, JWT)
├─────────────────┤
│  Repositories   │  ← Cache L1 → Cache L2 → MySQL
└─────────────────┘
```

### Repositorios

- **MySQL** (`users_mysql.go`): Source of truth
- **Cache L1** (`users_cache.go`): In-process cache (ccache)
- **Cache L2** (`users_memcached.go`): Distributed cache (Memcached)

Patrón de lectura: **L1 → L2 → DB** (con backfill en miss)

## Desarrollo Local

```bash
# Compilar
cd users-api
go build ./...

# Ejecutar tests
go test ./... -v

# Ejecutar servidor (requiere MySQL y Memcached)
go run ./cmd/main.go
```

## Ejemplos de Uso

### Registrar usuario cliente
```bash
curl -X POST http://localhost:8082/users \
  -H "Content-Type: application/json" \
  -d '{"username":"user1","password":"secret123"}'
```

### Registrar administrador
```bash
curl -X POST http://localhost:8082/users \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","tipo":"administrador"}'
```

### Login
```bash
curl -X POST http://localhost:8082/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user1","password":"secret123"}'
```

### Usar token en hotels-api
```bash
curl http://localhost:8081/admin/hotels \
  -H "Authorization: Bearer <token>"
```

## Integración con hotels-api

1. `users-api` genera tokens JWT con claims `user_id` y `tipo`
2. `hotels-api` valida estos tokens en su middleware
3. Rutas `/admin/*` requieren `tipo = "administrador"`
4. Rutas de usuario usan `user_id` para validar ownership
