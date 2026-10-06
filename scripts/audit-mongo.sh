#!/usr/bin/env sh
# Read-only by default. AUDIT_APPLY=1 migrates only proven legacy keys.
# Stop hotels-api first; preserves credentials and all reservation/inventory data.
set -eu
cd "$(dirname "$0")/.."
docker compose exec -T -e AUDIT_APPLY="${AUDIT_APPLY:-0}" mongo sh -c \
  'exec mongosh --quiet -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --file /dev/stdin' \
  < hotels-api/seed/audit-reservations.js
