# 🚀 Load Balancer & API Gateway

## Arquitectura

```
                                    ┌─────────────────────┐
                                    │   Frontend (React)  │
                                    └──────────┬──────────┘
                                               │
                                    ┌──────────▼──────────┐
                                    │    Nginx Gateway    │
                                    │    (Port 80/8090)   │
                                    └──────────┬──────────┘
                                               │
              ┌────────────────────────────────┼────────────────────────────────┐
              │                                │                                │
    ┌─────────▼─────────┐           ┌─────────▼─────────┐           ┌──────────▼─────────┐
    │    Users API      │           │    Hotels API     │           │    Search API      │
    │  (Load Balanced)  │           │                   │           │                    │
    └─────────┬─────────┘           └─────────┬─────────┘           └──────────┬─────────┘
              │                               │                                │
    ┌─────────┼─────────┐                     │                                │
    │         │         │                     │                                │
┌───▼───┐ ┌───▼───┐ ┌───▼───┐          ┌─────▼─────┐                    ┌──────▼──────┐
│ API-1 │ │ API-2 │ │ API-3 │          │  MongoDB  │                    │    Solr     │
└───────┘ └───────┘ └───────┘          └───────────┘                    └─────────────┘
    │         │         │                     │                                │
    └─────────┴─────────┴─────────────────────┴────────────────────────────────┘
                                    │
                        ┌───────────┼───────────┐
                        │           │           │
                   ┌────▼────┐ ┌────▼────┐ ┌────▼────┐
                   │  MySQL  │ │Memcached│ │RabbitMQ │
                   └─────────┘ └─────────┘ └─────────┘
```

## Características

### ⚖️ Load Balancing
- **Algoritmo**: `least_conn` (menor número de conexiones activas)
- **Health Checks**: Automáticos con `max_fails=3` y `fail_timeout=30s`
- **Keepalive Connections**: Optimizado para reducir latencia

### 🛡️ Seguridad
- **Rate Limiting**: 
  - API general: 10 req/seg (burst: 20)
  - Login: 5 req/min (burst: 3)
- **Connection Limits**: 20 conexiones por IP
- **Security Headers**: X-Frame-Options, X-Content-Type-Options, X-XSS-Protection

### ⚡ Performance
- **Gzip Compression**: Habilitado para JSON, XML, JavaScript, CSS
- **Proxy Buffering**: Optimizado para respuestas rápidas
- **TCP Optimizations**: `tcp_nopush`, `tcp_nodelay`, `sendfile`

## Endpoints

### API Gateway (Puerto 80)

| Método | Endpoint | Servicio | Descripción |
|--------|----------|----------|-------------|
| GET | `/health` | Gateway | Health check del gateway |
| GET | `/users` | Users API | Listar usuarios |
| GET | `/users/:id` | Users API | Obtener usuario por ID |
| POST | `/users` | Users API | Crear usuario |
| DELETE | `/users/:id` | Users API | Eliminar usuario |
| POST | `/login` | Users API | Autenticación |
| GET | `/hotels` | Hotels API | Listar hoteles |
| GET | `/hotels/:id` | Hotels API | Obtener hotel por ID |
| POST | `/reservations` | Hotels API | Crear reserva |
| GET | `/search` | Search API | Buscar hoteles |
| GET/POST | `/admin/*` | Hotels API | Endpoints de administración |

### Monitoring (Puerto 8090)

| Endpoint | Descripción |
|----------|-------------|
| `/nginx_status` | Estadísticas de Nginx (conexiones activas, requests) |
| `/status` | JSON con configuración del load balancer |
| `/health/all` | Health check general |

## Uso

### Iniciar todos los servicios

```bash
docker-compose up -d --build
```

### Verificar el estado

```bash
# Health check del gateway
curl http://localhost/health

# Estado de Nginx
curl http://localhost:8090/nginx_status

# Configuración del load balancer
curl http://localhost:8090/status | jq
```

### Ejecutar tests del load balancer

```bash
chmod +x test_load_balancer.sh
./test_load_balancer.sh
```

### Ver logs de Nginx

```bash
docker logs -f api-gateway
```

## Escalamiento

### Agregar más instancias de Users API

1. Duplicar la configuración en `docker-compose.yml`:
```yaml
users-api-4:
  build:
    context: ./users-api
    dockerfile: Dockerfile
  image: users-api:latest
  container_name: users-api-4
  # ... misma configuración
```

2. Actualizar `nginx.conf`:
```nginx
upstream users_api {
    least_conn;
    server users-api-1:8082 weight=1 max_fails=3 fail_timeout=30s;
    server users-api-2:8082 weight=1 max_fails=3 fail_timeout=30s;
    server users-api-3:8082 weight=1 max_fails=3 fail_timeout=30s;
    server users-api-4:8082 weight=1 max_fails=3 fail_timeout=30s;  # Nueva
    keepalive 32;
}
```

3. Reiniciar:
```bash
docker-compose up -d --build
```

## Algoritmos de Balanceo Disponibles

| Algoritmo | Descripción | Uso recomendado |
|-----------|-------------|-----------------|
| `round_robin` | Default. Distribuye en orden | Servidores homogéneos |
| `least_conn` | Menor cantidad de conexiones | Requests de duración variable |
| `ip_hash` | Sticky sessions por IP | Cuando necesitas persistencia |
| `weighted` | Por peso asignado | Servidores heterogéneos |

## Monitoreo en Producción

Para un entorno de producción, considera agregar:

- **Prometheus + Grafana**: Para métricas detalladas
- **ELK Stack**: Para análisis de logs
- **Jaeger/Zipkin**: Para distributed tracing
- **Healthchecks.io**: Para alertas de uptime

## Troubleshooting

### El gateway retorna 502 Bad Gateway

```bash
# Verificar que los servicios backend están corriendo
docker ps

# Ver logs del servicio que falla
docker logs users-api-1
```

### Rate limiting muy agresivo

Ajustar en `nginx.conf`:
```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=50r/s;
```

### Ver qué servidor maneja cada request

El header `X-Upstream-Server` muestra el servidor que manejó la request:
```bash
curl -I http://localhost/users
# Buscar: X-Upstream-Server: users-api-1:8082
```
