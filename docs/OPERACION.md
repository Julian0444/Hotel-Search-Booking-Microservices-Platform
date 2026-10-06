# Guía de demo y operación

Complemento del [README](../README.md). Ejecutá los comandos desde la raíz del
repositorio. Las garantías están en [ARCHITECTURE.md](ARCHITECTURE.md) y los
resultados de las pruebas en [CIERRE.md](CIERRE.md).

## Levantar la demo

Requisitos: Docker con Compose, puertos locales 80/443/5173 libres y memoria
suficiente para el stack (~6 GB asignados a Docker es una referencia práctica).
Node 22+ y Go 1.26.8+ sólo son necesarios para ejecutar herramientas fuera de Docker.

Desde la raíz, **para una instalación nueva**:

```bash
# No sobrescribe un .env existente.
test -f .env || cp .env.example .env

# Conserva el par de certificados local si ya existe.
if [ ! -f nginx/certs/key.pem ] || [ ! -f nginx/certs/cert.pem ]; then
  mkdir -p nginx/certs
  openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
    -keyout nginx/certs/key.pem -out nginx/certs/cert.pem \
    -subj '/CN=localhost' -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1'
fi

docker compose --profile frontend up -d --build
docker compose ps -a
```

Esperá que las APIs y bases estén healthy y que `migrate` y `mongo-init` terminen
con código 0. Abrí **[http://localhost:5173](http://localhost:5173)**. La SPA y
`/api/v1` comparten origen; el proxy interno llega al gateway TLS. El catálogo de
cinco hoteles argentinos se carga sólo en un volumen nuevo. Solr se reconcilia
al arrancar y puede tardar en mostrar el primer resultado.

| Cuenta | Usuario | Contraseña local |
|---|---|---|
| Cliente demo | `demo` | `DemoCliente123` |
| Administrador | `ADMIN_USERNAME` del `.env` | `ADMIN_PASSWORD` del `.env` |

`.env.example` contiene exclusivamente credenciales de demo. Para otro entorno,
usá valores propios y un JWT secret aleatorio; no publiques `.env`. El registro
público siempre crea clientes. Las claves de ejemplo no son secretos de producción.

`make down` detiene el proyecto **conservando sus volúmenes**. No hace `down -v`.
Los nombres de contenedor los asigna Compose, lo que permite proyectos de prueba
separados. Los puertos publicados siguen necesitando estar libres.

### Si ya tenés datos

No borres volúmenes para actualizar. Detené primero las escrituras:

```bash
docker compose stop hotels-api search-api
docker compose up -d mongo mongo-init
docker compose wait mongo-init
bash scripts/audit-mongo.sh
# Sólo tras un audit correcto: migra claves antiguas con respuesta 201 comprobada.
AUDIT_APPLY=1 bash scripts/audit-mongo.sh
docker compose up -d --build --wait hotels-api solr
bash search-api/scripts/apply-solr-schema.sh
docker compose --profile frontend up -d --build
```

`mongo-init` convierte el nodo a `rs0` sin borrar datos ni cambiar contraseñas.
La clave interna se genera una vez y permanece en el volumen. La auditoría
compara reservas confirmadas con inventario y detecta referencias inválidas,
contadores negativos/faltantes y resultados antiguos no demostrables. Si falla,
**detenete a investigar los registros que enumera**: no corrige contadores a
ciegas. El servicio también rechaza arrancar sobre inventario inconsistente.

La migración de idempotencia sólo deriva una huella cuando la respuesta histórica
201 apunta a una reserva del mismo usuario. Es idempotente y no elimina historial.
Volúmenes mucho más antiguos pueden requerir además la migración ya existente
`hotels-api/seed/rename-available-rooms.js`; revisá su contenido antes de aplicarla.

El script Solr detiene el único escritor, guarda una copia del schema en su
volumen, aplica/reload la configuración y ejecuta reconciliación completa antes
de devolver tráfico. Conserva el índice y elimina huérfanos. Si falla, deja
search-api detenido e informa el backup; corregí la causa y repetí. Cambiar el
XML del repositorio por sí solo no actualiza un core persistido.

## Recorrido de demostración

1. Buscá `Mendoza`, `Bariloche`, `Salta`, `Buenos Aires`, `Córdoba`, `cordoba`,
   `CORDOBA` y `Argentina`. El catálogo canónico conserva los textos con tildes.
2. Abrí un hotel, elegí fechas/habitaciones/huéspedes y continuá por login o
   registro: la selección vuelve con vos al mismo detalle.
3. Consultá disponibilidad, reservá y revisá el importe en centavos y la
   confirmación. En “My reservations”, cargá más páginas y cancelá. La
   disponibilidad se vuelve a consultar tras crear/cancelar.
4. Como admin, creá/editá un hotel, guardá, recargá y comprobá los campos.
   Precio/rating/capacidad cero y listas opcionales vacías se guardan. Capacidad
   cero cierra reservas; una reducción por debajo del inventario ocupado devuelve
   conflicto. Un hotel con historial no se puede borrar.
5. Observá el cambio en búsqueda con la misma consulta: la propagación por eventos
   es eventual. El índice también se reconcilia cada minuto y mediante `/reindex`.
6. Mostrá usuarios, paginación admin y protección de cambios sin guardar. El panel
   de servicios informa salud real y no simula operaciones de escalado.

## Pruebas y CI

```bash
make build
make test                 # cuatro módulos Go, con -race
make lint
make test-integration     # Mongo rs0 + Solr y RabbitMQ reales, aislados
cd frontend
npm ci
npm run check             # lint, cobertura con umbrales y build
npm audit --audit-level=high
npx playwright install chromium
cd ..
make e2e                  # SPA Docker y APIs reales
```

Para una verificación de fallos en un stack **descartable separado**, sin tocar
los volúmenes de la demo (primero liberá sus puertos):

```bash
COMPOSE_PROJECT_NAME=hotel-closure docker compose --profile frontend up -d --build
COMPOSE_PROJECT_NAME=hotel-closure python3 scripts/verify-broker-outage.py
```

Este script detiene/reinicia RabbitMQ y hotels-api, verifica Mongo directamente,
replay tras reinicio, reserva/cancelación sin broker y CRUD persistido aunque no
se publique. Comprueba recuperación por reconciliación sin reindex manual y
conserva la reserva cancelada como evidencia. No lo ejecutes en paralelo a E2E.
`test_load_balancer.sh` incluye una prueba explícita de rate limit, OPTIONS y
recuperación; ejecutala separada de otros logins.

CI ejecuta Go, frontend y escaneo/build de imágenes en push/PR. Integración y E2E
corren en **pull request o ejecución manual (`workflow_dispatch`)**, no en cada
push. Las imágenes sólo se publican en los eventos/configuración autorizados de
main/tags. Este cierre no hizo push, merge ni PR: la configuración remota nueva
queda preparada, sin una ejecución remota atribuida a estos cambios.

## Desarrollo y dominio público

Para Vite: `cd frontend && npm ci && npm run dev`, con el backend levantado. Usa
el mismo `/api/v1` relativo. Para llamar directo al gateway local, `curl -k
https://localhost/api/v1/hotels`; el certificado autofirmado es sólo local.

Una configuración opcional para un dominio público está preparada en
[docs/public-domain.nginx.conf](public-domain.nginx.conf), con
[docker-compose.public.yml](../docker-compose.public.yml). El nginx del host
termina TLS real y dirige todo a la SPA; `/api/v1` sigue en el mismo origen.
Reemplazá dominio y rutas de certificados; Compose 2.24.4+ acepta el overlay:

```bash
docker compose -f docker-compose.yml -f docker-compose.public.yml \
  --profile frontend config --quiet
```

El overlay deja el frontend sólo en loopback y retira puertos públicos de bases,
broker y gateway interno. No hace falta recompilar con un hostname en JavaScript.
Esta configuración se entrega para revisión; **no se publicó ningún dominio**.
[Kubernetes](../k8s/README.md) queda como material opcional histórico, no como el
entorno validado de este cierre.
