#!/usr/bin/env bash
# Sólo crea/elimina contenedores y cores propios; no usa volúmenes de la demo.
set -euo pipefail
cd "$(dirname "$0")/../.."
root=$PWD
suffix="$(date +%s)-$$"
solr_name="hotel-test-solr-$suffix"
rabbit_name="hotel-test-rabbit-$suffix"
cleanup() {
  docker rm -fv "$solr_name" "$rabbit_name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run -d --name "$solr_name" -p 127.0.0.1::8983 \
  -v "$root/search-api/internal/solr-config:/opt/test-config:ro" \
  solr:9 solr-precreate hotels /opt/test-config >/dev/null
docker run -d --name "$rabbit_name" -p 127.0.0.1::5672 \
  -e RABBITMQ_DEFAULT_USER=verify -e RABBITMQ_DEFAULT_PASS=verify-local \
  rabbitmq:3-management >/dev/null
solr_port=$(docker inspect -f '{{(index (index .NetworkSettings.Ports "8983/tcp") 0).HostPort}}' "$solr_name")
rabbit_port=$(docker inspect -f '{{(index (index .NetworkSettings.Ports "5672/tcp") 0).HostPort}}' "$rabbit_name")
export TEST_SOLR_URL="http://127.0.0.1:$solr_port"
export TEST_RABBIT_URL="amqp://verify:verify-local@127.0.0.1:$rabbit_port/"
export TEST_SOLR_CONFIGSET=hotels
ready=false
# El CLI también crea .erlang.cookie: usar el mismo usuario que el broker evita
# que un diagnóstico temprano la deje propiedad de root y rompa su arranque.
for ((attempt=0; attempt<90; attempt++)); do
  if curl --fail --silent "$TEST_SOLR_URL/solr/admin/info/system" >/dev/null && \
      docker exec --user rabbitmq "$rabbit_name" rabbitmq-diagnostics -q check_port_connectivity >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  docker logs "$solr_name"
  docker logs "$rabbit_name"
  printf 'Solr/RabbitMQ de pruebas no llegaron a ready en el plazo.\n' >&2
  exit 1
fi
docker exec "$solr_name" mkdir -p /var/solr/data/configsets
docker cp search-api/internal/solr-config "$solr_name:/var/solr/data/configsets/hotels"
go test -tags=integration -race -count=1 ./search-api/...

# Publisher real: confirma aceptación y recuperación de conexión en el mismo broker aislado.
go test -tags=integration -race -count=1 ./hotels-api/internal/clients/queues
