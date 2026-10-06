# Plan 12 — Documentación y presentación de portfolio

> **Alcance:** P1, P2, P3, P4, P5, P6, P8, P9 + Sección 4 completa (§4.1 README, §4.2 activos, §4.3 traducción/consolidación, §4.4 prep de entrevista)
> **Base:** Sección 4 de `plantofinish.md` (P7/seed ya quedó en el plan 03)
> **Prerequisitos:** planes 01–11 + núcleo del plan 13 (este plan retrata el estado final: frontend pulido, capturas, tabla de endpoints con auth real y quickstart con seed). Se puede adelantar parcialmente, pero las capturas y la tabla de endpoints se hacen al final.
> **Esfuerzo:** 1–1.5 días

## Contexto

Objetivo: que un hiring manager en un **skim de 5 minutos** entienda qué construiste, vea que funciona y quiera hablar con vos. Orden de lectura: *hook visual → arquitectura → cómo correrlo → profundidad técnica*. Las capturas y el GIF salen de la UI final verificada en el plan 13, no de una versión intermedia. Hoy: la única imagen del repo es `vite.svg` (P3), el mejor documento está solo en español (P5), hay archivos "para la IA" commiteados (P4), errores fácticos verificables (P6) y docs que contradicen al código (P1, P2).

## Pasos

### 1. Borrar el scaffolding de IA (P4)

- Borrar del repo `hotels-api/PLAN.md` y `RULES.md` (guardarlos fuera del repo si querés conservarlos). Leen como repo auto-generado y revelan que el panel era mock.
- Plegar lo útil en `ProyectoBackend.md` con audiencia humana antes de traducir.

### 2. Corregir errores fácticos (P6)

- `README.md:176`: "10 containers" → **11** (o el número real post-planes; contar con `docker compose config --services | wc -l`).
- `LOAD_BALANCER.md:68,80`: quitar las rutas fantasma `GET /hotels` y `/health/all`… ojo: tras el plan 06, `GET /hotels` **sí** existe — verificar contra el código real y documentar lo que hay.
- Diagrama con Users API y Search API ambos en "Port 8082": el error real está en **`README.md:25-26`** (la cita `ProyectoBackend.md:49` de plantofinish.md es incorrecta — re-verificado por revisión posterior). Corregir los puertos por servicio ahí, y revisar `ProyectoBackend.md` de paso por si repite el error.

### 3. Consistencia código ↔ docs (P1, P2)

- Tabla de endpoints del README con la **auth real por endpoint** (post plan 01: `GET /users` = Admin; `GET/DELETE /users/:id` = Owner/Admin; post plan 11: reservas de hotel = Admin). El "DELETE /users/:id | — |" de `README.md:202-204` desaparece.
- Una sola verdad: README + `users.md` + colección Bruno + OpenAPI (paso 5) coinciden con el código. Actualizar los `.bru` (auth bearer donde corresponda, rutas `/api/v1`).

### 4. Traducir y consolidar (P5, §4.3)

- Traducir `ProyectoBackend.md` a inglés → `docs/ARCHITECTURE.md` (es tu mejor pieza; opcional conservar `.es.md`).
- `LOAD_BALANCER.md` y `users.md`: traducir o fusionar su contenido vigente en ARCHITECTURE/README y retirarlos.
- Idioma estándar del repo: inglés (código, commits nuevos, docs).

### 5. Activos visuales + OpenAPI (P3, §4.2)

Crear `docs/` con:

- **Diagrama de arquitectura renderizado** (Mermaid en el README o export SVG/PNG de excalidraw), incluyendo el flujo del evento CREATE hotel → RabbitMQ → search-api → Solr. (Mermaid tiene la ventaja de diffear en git.)
- **3–4 capturas**: Home/Search, HotelDetail, MyReservations, Admin Dashboard (con el seed del plan 03 la app se ve poblada).
- **1 GIF** del flujo end-to-end (login → buscar → reservar), 15–30s (grabar con Kap/LICEcap/QuickTime+gifski).
- **OpenAPI por servicio**: anotaciones `swaggo` + `/swagger` servido, o `docs/openapi/{users,hotels,search}.yaml` a mano (más control, sin dependencia). Validar con `redocly lint` o editor.swagger.io. Reflejar `/api/v1`, envelopes y códigos del plan 07.
- (Opcional) diagrama de secuencia del flujo event-driven.

### 6. README nuevo (§4.1)

Reestructurar en este orden exacto:

1. Título + una línea + **badges** (CI del plan 02, versión Go, License — P9).
2. **GIF/captura** del app funcionando.
3. **Qué demuestra este proyecto** (3–5 bullets: microservicios, API gateway + LB, event-driven/CQRS-lite, caché multi-nivel, JWT/RBAC distribuido, no-overbooking atómico, k8s+HPA — robado de `ProyectoBackend.md` + lo ganado en los planes 04/09).
4. **Diagrama de arquitectura** renderizado.
5. **Stack** (tabla existente).
6. **Quickstart que funciona de verdad**: `cp .env.example .env` → `docker compose up -d --build` → `make seed` (si no es automático) → URL del frontend → **credenciales demo** (admin + cliente del plan 03).
7. **Tabla de endpoints** (auth real) + link a la **spec OpenAPI**.
8. **Testing** (comandos, `-race`, integración, cobertura).
9. **Decisiones de arquitectura / trade-offs** → link a `docs/ARCHITECTURE.md`.
10. **"Cómo lo llevaría a producción"** (P9): consolidar lo disperso en `LOAD_BALANCER.md`/`README_HOTELS.md` — secrets manager, observabilidad (métricas/trazas), managed DBs + replica set, outbox para RabbitMQ, CD, TLS real. Es lo que un senior lee para medir seniority.

### 7. Placeholders y lib deprecada (P8)

- `README_HOTELS.md:600,622-627`: rellenar `[Add your repo URL]`, quitar "Last Updated: January 2026" (manual), o directamente fusionar lo vigente en el README/ARCHITECTURE y retirar el archivo.
- Migrar `streadway/amqp` (archivada en 2021) → `rabbitmq/amqp091-go`: es drop-in (mismo API) — cambiar el import path en hotels-api y search-api, `go mod tidy`, correr los tests. Actualizar la mención en docs.

### 8. LICENSE, badges y cierre (P9)

- `LICENSE` MIT ya existe (plan 02/I9) → verificar; agregar badges de CI (`![ci](https://github.com/Julian0444/<repo>/actions/workflows/ci.yml/badge.svg)`), Go version y license al README.
- (Opcional) `CONTRIBUTING.md` breve.

### 9. Prep de entrevista (§4.4) — fuera del repo

Guardar **fuera del repo** (notas personales) las respuestas listas:

- *"¿Cómo evitás doble reserva?"* → contador de inventario por hotel-noche con `findOneAndUpdate` atómico + índice único + compensación; por qué una transacción sola no alcanza (TOCTOU, no hay conflicto de escritura entre dos inserts) y por qué no replica set (plan 04).
- *"¿Qué pasa si Solr/RabbitMQ se caen?"* → manual ack + retry + DLQ + backfill/reindex (plan 06); breaker (plan 08).
- *"¿Por qué caché de 2 niveles?"* → L1 local por réplica vs L2 compartida entre las 3 réplicas.
- *"¿Cómo asegurás la autorización entre servicios?"* → users-api emite (`iss/aud`), hotels-api valida HMAC-pinned; blast radius de un fallo acá (por eso el plan 01 fue primero).
- *"¿Consistencia fuerte o eventual?"* → Mongo fuente de verdad (fuerte), Solr proyección (eventual); dónde está el límite y por qué está bien para búsqueda.

## Verificar

```bash
# Quickstart en frío, siguiendo SOLO el README, en un clone limpio:
cd $(mktemp -d) && git clone <repo> . && cp .env.example .env && docker compose up -d --build
# → login con credenciales demo del README → buscar → reservar → panel admin. Sin pasos no documentados.

# Consistencia: cada endpoint de la tabla del README probado con/sin token da lo que la tabla dice
# OpenAPI válida
npx @redocly/cli lint docs/openapi/*.yaml

# Sin restos
git ls-files | grep -iE "PLAN.md|RULES.md"        # 0 (P4)
grep -rn "Add your repo URL\|Last Updated" *.md    # 0 (P8)
grep -rn "10 containers\|streadway" *.md docs/     # 0 (P6, P8)
# Imágenes y GIF renderizan en el remote de GitHub (revisar en la web)
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
