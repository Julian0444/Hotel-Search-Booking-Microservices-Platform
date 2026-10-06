# Cierre del portfolio

Verificación local: 5 de octubre de 2026, hora de Los Ángeles / 6 de octubre UTC.
Base: `feat/plan-01-seguridad-auth`, `2738e8eb62463f8cf1a7a34f6bbbbee59363879f`.
La evidencia de este documento se obtuvo localmente, antes del commit y push a
esta misma rama autorizados posteriormente por el usuario. No se atribuye a la CI
histórica. Las ejecuciones remotas posteriores pueden consultarse en
[GitHub Actions](https://github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/actions?query=branch%3Afeat%2Fplan-01-seguridad-auth).
No se realizó PR, merge a main ni despliegue externo.

Se preservaron los cambios previos en `frontend/README.md`, la eliminación de
`hotels-api/seed-hotels.js` y `EXPLICACION-DEL-PROYECTO.md`. No se editaron los
documentos externos, los trece planes completados ni `plantofinish.md`.

## Resultado y decisiones

- [x] Reserva, inventario e idempotencia en una transacción Mongo snapshot/majority.
- [x] Cancelación atómica, coordinación con capacidad/borrado y recuperación de commit ambiguo.
- [x] Disponibilidad desde inventario persistente; PUT completo validado, ceros y arrays vacíos.
- [x] Destinos y acentos con Solr real; reconciliación de faltantes, vigentes y huérfanos.
- [x] Eventos transitorios recuperables, confirms y HTTP coherente cuando falla publicar.
- [x] Retiradas cachés de hotels-api, caché gateway de búsqueda, breaker y reservations-news.
- [x] SPA/API bajo el mismo origen, readiness degradada y recuperación DNS tras recrear APIs.
- [x] Corrida final completa de navegador: 32/32 aprobadas, y registro de evidencia cerrado.

Se eligió Mongo `rs0` de un nodo para evitar compensaciones manuales entre varios
documentos. Reservar, cancelar, cambiar capacidad y borrar escriben el mismo hotel
dentro de su transacción. Las referencias ObjectID se normalizan antes de operar;
el cliente no elige el ID de un hotel nuevo. La auditoría de arranque y la migración
reportan inconsistencias heredadas sin borrar historial ni adivinar contadores.

Search tiene un único escritor: eventos y reconciliación comparten exclusión.
La reconciliación enumera por cursor y vuelve a consultar el estado actual de
cada ID; incluye los IDs existentes sólo en Solr. Los eventos válidos se conservan
ante fallos transitorios con espera cancelable; los inválidos van a DLQ. Se mantiene
la ventana Mongo→RabbitMQ y se recupera mediante reconciliación periódica.

El CRUD devuelve el éxito persistido aunque publicar falle. La publicación tiene
un presupuesto de 2s que incluye espera concurrente, I/O y confirmación; un único
reconector trabaja fuera de los requests. Reservar y cancelar no lo invocan.

El frontend conserva la selección al autenticarse, pagina el historial más allá
de 100 reservas, corrige páginas admin vacías tras borrar y protege formularios
sucios incluso al cerrar sesión. Se corrigió también el ajuste de página que
usaba resultados anteriores mientras cambiaba una búsqueda.

## Verificación y evidencia

Entorno: macOS arm64, Docker Engine 29.1.5, Compose 5.0.1, Go 1.26.8,
Node 22.23.2, npm 10.9.8. APIs construidas con Go 1.26.8/Alpine 3.24.2.
El proyecto Compose `hotel-closure` usa volúmenes propios. Se conservaron los
volúmenes originales del proyecto. Las pruebas de integración crean y retiran
únicamente sus contenedores/volúmenes descartables.

Los logs se conservan en [evidence/2026-10-06](evidence/2026-10-06/).
El [manifiesto SHA-256](evidence/2026-10-06/source-sha256.txt) identifica fuentes,
configuración y tests usados en esa verificación local; las correcciones posteriores
detectadas por CI quedan versionadas en Git. Los [IDs de imágenes](evidence/2026-10-06/images.json)
identifican los builds locales. No son hashes de un commit nuevo.
Los resultados siguientes son de comandos ejecutados localmente:

| Comando o comprobación | Resultado | Evidencia |
|---|---|---|
| `make build`, `make test` | Cuatro módulos, build y tests con race aprobados; el target normal permite caché de Go | [build](evidence/2026-10-06/go-build.txt), [test](evidence/2026-10-06/go-test.txt) |
| `make lint`, `go vet`, gofmt | Cero issues en cuatro módulos, vet aprobado y formato limpio | [lint](evidence/2026-10-06/go-lint.txt) |
| `make test-integration` | Ejecución nueva con race/count=1: Mongo rs0, Solr, Rabbit y publisher aprobados | [integración](evidence/2026-10-06/integration.txt) |
| `npm run check` | 23 archivos, 140 tests; lint, cobertura y build aprobados | [frontend](evidence/2026-10-06/frontend-check.txt) |
| `npm audit --audit-level=high` | Cero vulnerabilidades tras instalación con npm ci | [audit](evidence/2026-10-06/frontend-audit.txt) |
| `COMPOSE_PROJECT_NAME=hotel-closure npm run test:e2e` desde frontend | 32/32 en 4,7 minutos; desktop/móvil, cero retries y omisiones | [E2E final](evidence/2026-10-06/e2e.txt) |
| govulncheck, cuatro módulos | Cero vulnerabilidades alcanzables; se conservan los avisos de módulos requeridos no invocados | Archivos `govulncheck-*.txt` |
| Trivy HIGH/CRITICAL, ignore-unfixed | Cero hallazgos en imágenes finales de las tres APIs; mismo gate existente | Archivos `trivy-*.txt` |
| `verify-broker-outage.py` | Broker detenido, reinicio API, replay, inventario y CRUD aprobados; recuperación sin reindex manual | [resultado](evidence/2026-10-06/broker-outage.json) |
| `test_load_balancer.sh` | TLS, rutas, balanceo, no-store, 429 JSON, OPTIONS y recuperación aprobados | [gateway](evidence/2026-10-06/gateway.txt) |
| Recreate APIs con gateway vivo | IP de hotels-api cambió; misma URL de SPA recuperó 200 sin reiniciar gateway | [DNS](evidence/2026-10-06/dns-recreate.txt) |
| Mongo init repetido / auditoría real | rs0 primary, audit OK; wrapper aborta con exit 1 ante throw controlado | [init](evidence/2026-10-06/mongo-init.txt), [audit](evidence/2026-10-06/mongo-audit.txt) |
| Aplicación Solr sobre core persistido | Conservó los cinco IDs, eliminó huérfano y verificó analyzers activos | [constancia retrospectiva](evidence/2026-10-06/schema-observations.txt) |
| Redocly 2.58.1, tres OpenAPI | Cero errores; tres avisos por servidores localhost, sin suprimir reglas | [OpenAPI](evidence/2026-10-06/openapi.txt) |
| Overlay público Compose | `config --quiet` aprobado; no desplegado | `docker-compose.public.yml` |
| Vite en puerto temporal 5178 | HTML, catálogo y búsqueda por `/api/v1` respondieron 200; servidor de prueba detenido | [Vite](evidence/2026-10-06/vite.txt) |

Cobertura frontend: statements 90,08%, branches 85,08%, functions 85,11%, lines
90,95%. No se redujeron umbrales. Los escaneos motivaron actualizar Go y dependencias
frontend; no se desactivaron controles para obtener el resultado.

### Qué demuestran las pruebas de fallos

En [hotels_mongo_integration_test.go](../hotels-api/internal/repositories/hotels/hotels_mongo_integration_test.go):
20 intentos concurrentes por el último cupo, 20 con la misma clave, cancelaciones
concurrentes, reserva de varias noches, conflictos de capacidad/borrado, IDs
canónicos, rollback de cancelación y migración repetida sin cambios de contadores.

La inserción no ejecutada se inyecta con `failCommand` antes de escribir. Los casos
de respuesta perdida interceptan una respuesta real `commitTransaction` con
`ok:1` **después de aplicar el commit**, la descartan y comprueban recuperación del
mismo resultado. Incluyen creación, cancelación, deadline vencido y replay tras
reconstruir el repositorio. Son escenarios controlados reproducidos en este cierre,
no incidentes históricos de producción.

En [las pruebas HTTP/Mongo](../hotels-api/internal/repositories/hotels/hotels_http_integration_test.go)
se comprueba disponibilidad tras abrir el detalle, persistencia de ceros/listas
vacías, replay, validación antes de escritura y respuesta de CRUD al fallar publicar.

Search verifica los cinco destinos canónicos, tildes, orden/paginación, DELETE
perdido, cambios durante paginación y eventos intercalados con una lectura antigua.
La prueba Rabbit conserva eventos durante respuestas 503 controladas de hoteles/
Solr y los procesa al recuperarse, sin reindex manual; verifica también DLQ y
reconexión TCP. La prueba de caída completa complementa esa inyección con un
**contenedor RabbitMQ realmente detenido**.

Durante esa caída completa, reserva **8,0 ms** y cancelación **8,1 ms** en una
medición local por operación; no son un benchmark ni una promesa de latencia.
Mongo terminó con una reserva cancelada, una clave y dos noches con `booked=0`.
Create/PUT/Delete mantuvieron 201/200/204 y sus estados persistidos. La misma URI
de búsqueda convergió por el proceso periódico mientras el broker seguía detenido.

El navegador repitió CREATE/PUT/DELETE con el contenedor RabbitMQ detenido:
éxito visible, recarga con valores persistidos y búsqueda convergente en la misma
URI, sin reindex manual. La corrida completa también cubrió login/registro con
retorno, destinos/tildes y los cinco hoteles de Argentina, último cupo/cancelación,
ediciones vacías/cero, paginación admin, formulario sucio y recuperación de APIs.
Las seis rutas evaluadas por axe no tuvieron violaciones serious/critical en
desktop ni móvil. El historial con la reserva 101 se cubrió en componentes.

Los E2E admiten datos de ejecuciones anteriores: recorren páginas reales,
identifican sus fixtures y comparan el orden con el catálogo persistido. Esperan
la convergencia repitiendo la misma consulta, sin parámetros para eludir cachés.
Las comprobaciones de consola no usan una exclusión global de errores CORS/red.

### Alcance honesto de la evidencia

Los intentos iniciales de integración encontraron restricciones de sandbox;
se repitieron en el entorno autorizado. Hubo fallos reales o de harness durante
el desarrollo: no se presentan esas corridas como aprobadas. El registro final
conserva los resultados posteriores a sus correcciones.

La constancia del cambio de schema es retrospectiva, identificada como tal, y
no sustituye un log original. Bruno se alineó y se validaron sus cuerpos JSON;
no se afirma una ejecución HTTP de todas las colecciones. No se ejecutó CI remota,
Kubernetes ni un despliegue público. Las capturas/Lighthouse históricos no son
mediciones nuevas de este cierre.

## Demostración y límites

[README](../README.md) conserva la presentación del proyecto y el arranque rápido.
La [guía de operación](OPERACION.md) contiene el recorrido de demo y la migración
no destructiva de volúmenes existentes. La SPA se abre en `http://localhost:5173`. El entorno dejado
para revisión usa `COMPOSE_PROJECT_NAME=hotel-closure`; usá ese mismo prefijo para
inspeccionarlo o detenerlo. No arranques otro proyecto sobre los mismos puertos.
Conserva hoteles y reservas canceladas de auditoría, además de los cinco hoteles
canónicos. Al finalizar, todas las APIs/bases estaban healthy, los checks de
RabbitMQ/Memcached en `ok` y la auditoría de inventario aprobada.
El [estado final de Compose](evidence/2026-10-06/runtime.json) incluye ambos
procesos one-shot terminados con código 0.

Límites: Mongo de un nodo sin alta disponibilidad; search de una instancia;
búsqueda eventual sin transacción Mongo/Rabbit; claves de idempotencia durante
24h y ninguna deduplicación sin header; DELETE de hoteles con historial rechazado;
capacidad respeta incluso reservas confirmadas históricas; JWT sin revocación;
cachés de usuarios con invalidación local y TTL sin garantía universal entre
réplicas. Los clientes del proxy frontend comparten el rate limit de login del
gateway. El proyecto calcula importes, no cobra. No se lo declara production ready.
