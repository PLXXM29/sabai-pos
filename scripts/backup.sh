#!/usr/bin/env bash
# Dump the Postgres database from the running compose stack to ./backups/.
# Usage:  ./scripts/backup.sh            (uses docker-compose.prod.yml)
#         COMPOSE_FILE=docker-compose.yml ./scripts/backup.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
[ -f .env ] && set -a && . ./.env && set +a

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
: "${POSTGRES_USER:?set POSTGRES_USER (or provide .env)}"
: "${POSTGRES_DB:?set POSTGRES_DB (or provide .env)}"

mkdir -p backups
TS="$(date +%Y%m%d-%H%M%S)"
FILE="backups/minimart-${TS}.sql.gz"

docker compose -f "$COMPOSE_FILE" exec -T db \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" | gzip >"$FILE"

echo "backup written: $FILE ($(du -h "$FILE" | cut -f1))"

# Retention: keep the newest 14 dumps.
ls -1t backups/minimart-*.sql.gz 2>/dev/null | tail -n +15 | xargs -r rm -f
