# Kubernetes: material opcional de referencia

El camino reproducible de este cierre es **Docker Compose**, documentado en el
[README](../README.md). Estos manifests proceden del ejercicio anterior de kind;
no constituyen un despliegue validado de la implementación actual. Las imágenes
`sha-1408d76` conservadas en los ejemplos son históricas.

Se retiró el quickstart `kubectl apply -f k8s/`: el Mongo standalone incluido en
`13-mongo.yaml` no admite las transacciones que ahora exige hotels-api. No debe
usarse ese manifest con el backend actual. Un despliegue adaptado requiere:

- Mongo replica set (nombre configurado mediante `MONGO_REPLICA_SET`, `rs0` en
  Compose), autenticación interna y almacenamiento persistente. Aplicar primero
  la migración/auditoría de reservas descrita en la documentación principal.
- Una sola instancia de search-api, con `Recreate` (ya reflejado en el template).
  El bloqueo que coordina eventos y reconciliación es local al proceso.
- Regenerar el ConfigMap de Solr desde su configuración canónica actual y ejecutar
  el procedimiento de actualización/reconciliación; el XML copiado aquí es histórico.
- Imágenes de los archivos modificados, en lugar de las referencias antiguas.
- Frontend y API bajo el mismo origen y certificados apropiados para ese entorno.

Las probes se mantienen como referencia: Mongo es obligatorio para hotels-api;
RabbitMQ es secundario y su fallo se informa sin retirar reservas del tráfico.
MySQL es obligatorio para users-api; Memcached es opcional. Solr es obligatorio
para search-api; RabbitMQ es secundario para atender búsquedas ya indexadas.

Para construir imágenes locales, los **tres** Dockerfiles Go necesitan contexto
raíz porque importan `platform-contracts`:

```bash
docker build -f users-api/Dockerfile -t users-api:local .
docker build -f hotels-api/Dockerfile -t hotels-api:local .
docker build -f search-api/Dockerfile -t search-api:local .
```

El material de Deployments, Services, HPA y probes sirve para estudiar Kubernetes.
No se ha ejecutado kind, un rollout ni un HPA como parte de este cierre. No se
requiere ampliar esa infraestructura para demostrar el portfolio.
