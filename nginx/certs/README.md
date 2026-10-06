# Certificados TLS locales (SD4 — plan 10)

Los `.pem` **no se commitean** (están en el `.gitignore`); acá solo vive esta
receta. El compose monta este directorio en `/etc/nginx/certs` y `nginx.conf`
referencia exactamente `cert.pem` / `key.pem`, así que **hay que generarlos
antes del primer `docker compose up`** (sin ellos, nginx no arranca).

Desde la **raíz del repo**:

```bash
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout nginx/certs/key.pem -out nginx/certs/cert.pem \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

- El SAN (`localhost` + `127.0.0.1`) es lo que validan los clientes modernos;
  el CN solo ya no alcanza.
- Al ser self-signed, el browser va a mostrar la advertencia de siempre y
  `curl` necesita `-k`. Es el tradeoff aceptado para el demo local.

## Alternativa: mkcert (cert confiable local, sin advertencia)

```bash
mkcert -key-file nginx/certs/key.pem -cert-file nginx/certs/cert.pem localhost 127.0.0.1
```

**Caveat HSTS**: con un cert confiable el browser SÍ registra el
`Strict-Transport-Security` del gateway, y HSTS aplica a todo el host
`localhost` **sin distinguir puertos** — `http://localhost:5173` (Vite dev)
pasaría a forzarse a https y dejaría de andar en ese browser hasta limpiar la
política. Con el self-signed de openssl los browsers ignoran HSTS (RFC 6797) y
no hay efecto. Si usás mkcert, sabé dónde se limpia (`chrome://net-internals/#hsts`).

## Producción

Cert real (p. ej. Let's Encrypt). El server block 443, el redirect 80→443 y
HSTS ya quedan listos en `nginx.conf`; solo cambian los archivos de cert.
