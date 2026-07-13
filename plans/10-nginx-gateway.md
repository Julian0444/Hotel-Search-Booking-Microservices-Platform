# Plan 10 — Gateway nginx: TLS y hardening

> **Alcance:** SD4, SD5, I3, I7
> **Base:** sin playbook — redactado desde los IDs de las Secciones 2 y 6
> **Prerequisitos:** ninguno duro; **ideal después del plan 07** (las locations ya quedaron versionadas y no se re-tocan)
> **Esfuerzo:** medio día

## Contexto

El gateway es el activo estrella para entrevistas, pero tiene 4 gotchas que un revisor con nginx ve rápido: rate-limit devuelve 503 en vez de 429 (I7, `nginx.conf:88-90`), `/search` declara un cache de 5 min que **no existe** — no hay `proxy_cache_path` ni zona (I3, `nginx.conf:418-419`), los security headers definidos a nivel `server` **nunca llegan** a las rutas de API por la herencia de `add_header` (SD5, `nginx.conf:115-118` vs los `add_header` CORS en cada location), y todo viaja en texto plano (SD4, `listen 80`).

## Pasos

### 1. 429 en rate-limit (I7)

Junto a las zonas existentes (`nginx.conf:88-90`):

```nginx
limit_req_status 429;
limit_conn_status 429;
```

Así "me limitaste" deja de confundirse con "upstream caído" (503).

### 2. Cache real en `/search` (I3)

Recomendado hacerla real (mejor tema de entrevista que borrar la directiva):

```nginx
# http {}
proxy_cache_path /var/cache/nginx/search levels=1:2 keys_zone=search_cache:10m
                 max_size=100m inactive=10m use_temp_path=off;

# location /api/v1/search
proxy_cache search_cache;
proxy_cache_key "$request_uri";           # incluye la query string
proxy_cache_valid 200 5m;
proxy_cache_use_stale error timeout updating;
add_header X-Cache-Status $upstream_cache_status;   # ver gotcha SD5: si agregás add_header acá, incluí el snippet de headers
```

Nota: cachear `/search` implica resultados hasta 5 min viejos — coherente con el modelo CQRS-lite (Solr ya es eventual). Documentarlo.

### 3. Security headers que realmente lleguen (SD5)

Gotcha de herencia de nginx: **cualquier** `add_header` en una `location` descarta TODOS los `add_header` del nivel `server`. Fix:

- Crear `nginx/snippets/security-headers.conf` con los headers actuales (`X-Frame-Options`, `X-Content-Type-Options`, etc.).
- `include /etc/nginx/snippets/security-headers.conf;` en **cada location** que tenga su propio `add_header` (las de CORS: `nginx.conf:164-269+`), y montar el snippet en el contenedor.
- Verificación obligatoria con `curl -I` contra una ruta de API real (no solo `/`).

### 4. TLS local + redirect + HSTS (SD4)

- Cert self-signed para local (`mkcert localhost` si está instalado, si no):

```bash
mkdir -p nginx/certs && openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout nginx/certs/key.pem -out nginx/certs/cert.pem -subj "/CN=localhost"
```

- Server block 443 + redirect:

```nginx
server { listen 80; return 301 https://$host$request_uri; }
server {
  listen 443 ssl;
  ssl_certificate     /etc/nginx/certs/cert.pem;
  ssl_certificate_key /etc/nginx/certs/key.pem;
  ssl_protocols TLSv1.2 TLSv1.3;
  add_header Strict-Transport-Security "max-age=31536000" always;  # solo en 443 (+ incluir el snippet del paso 3)
  # ... locations existentes ...
}
```

- Compose: publicar `443:443`, montar `certs/` (los `.pem` van al `.gitignore`; en el repo queda solo un `certs/README.md` con el comando de generación).
- Frontend/docs: base URL a `https://localhost` (el browser avisará por el self-signed — documentarlo); nota en el README de producción: cert real (Let's Encrypt) + `redirect` ya listo (lo consolida el plan 12/P9).
- El server de monitoreo `:8090` puede quedar HTTP-only interno.

## Verificar

```bash
docker compose up -d --build nginx

# TLS + redirect
curl -sI http://localhost | head -1                    # 301 → https
curl -skI https://localhost/api/v1/search?q=spa | head -1   # 200 (‑k por self-signed)

# SD5: headers presentes en una ruta de API (antes: solo en /)
curl -skI https://localhost/api/v1/hotels | grep -Ei 'x-frame-options|x-content-type|strict-transport'

# I3: segunda pegada es HIT
curl -sk https://localhost/api/v1/search?q=spa -o /dev/null
curl -skI https://localhost/api/v1/search?q=spa | grep -i x-cache-status   # HIT

# I7: burst sobre /login → 429 (no 503)
for i in $(seq 1 10); do curl -sk -o /dev/null -w "%{http_code}\n" -X POST https://localhost/api/v1/login -d '{}'; done | sort | uniq -c

docker compose exec nginx nginx -t   # config válida
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
