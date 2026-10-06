#!/usr/bin/env bash
# Actualiza un core existente sin borrar índices ni volúmenes. Ejecutar en
# mantenimiento: search-api queda detenido durante reload/reconciliación.
set -euo pipefail
cd "$(dirname "$0")/../.."

# El único escritor debe ser el servicio search-api de este proyecto Compose.
# COMPOSE_PROJECT_NAME / COMPOSE_FILE conservan la selección estándar de Docker.
docker compose build search-api
was_running=$(docker compose ps --status running --services search-api)
docker compose stop search-api
backup="schema.xml.before-$(date -u +%Y%m%dT%H%M%SZ)-$$"
printf 'Respaldando schema dentro del volumen Solr: %s\n' "$backup"
docker compose exec -T solr cp /var/solr/data/hotels/conf/schema.xml "/var/solr/data/hotels/conf/$backup"
docker compose cp search-api/internal/solr-config/conf/schema.xml solr:/var/solr/data/hotels/conf/schema.xml

on_error() {
  printf '\nLa actualización no terminó. search-api permanece detenido; no se borraron volúmenes.\n' >&2
  printf 'Corregí el fallo y repetí el script. Backup: /var/solr/data/hotels/conf/%s\n' "$backup" >&2
}
trap on_error ERR

docker compose exec -T solr curl --fail --silent --show-error --get \
  --data-urlencode action=RELOAD --data-urlencode core=hotels \
  http://localhost:8983/solr/admin/cores
# Mantiene el catálogo disponible en hotels-api; los eventos se acumulan en la
# cola durable. El proceso de mantenimiento es el único escritor de Solr.
docker compose run --rm --no-deps search-api ./search-api --reindex-once
# Hace visibles también las últimas escrituras antes de devolver tráfico.
docker compose exec -T solr curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  --data '{"commit":{"softCommit":true,"waitSearcher":true}}' \
  http://localhost:8983/solr/hotels/update
if [[ -n "$was_running" ]]; then
  docker compose up -d --no-deps search-api
fi
trap - ERR
printf '\nSchema aplicado y catálogo reconciliado. Backup conservado: %s\n' "$backup"
