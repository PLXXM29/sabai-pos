#!/usr/bin/env bash
# Restore a gzipped pg_dump into the running compose DB. DESTRUCTIVE.
# Usage:  ./scripts/restore.sh backups/minimart-YYYYmmdd-HHMMSS.sql.gz
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
[ -f .env ] && set -a && . ./.env && set +a

FILE="${1:?usage: restore.sh <backup.sql.gz>}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
: "${POSTGRES_USER:?}" ; : "${POSTGRES_DB:?}"

echo "Restoring $FILE into $POSTGRES_DB — existing data will be overwritten."
read -r -p "Type 'yes' to continue: " ok
[ "$ok" = "yes" ] || { echo "aborted"; exit 1; }

gunzip -c "$FILE" | docker compose -f "$COMPOSE_FILE" exec -T db \
  psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"

echo "restore complete"
