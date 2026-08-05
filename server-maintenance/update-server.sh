#!/usr/bin/env bash
set -euo pipefail
APP_DIR=${EITS_APP_DIR:-/srv/storage/eits-monitor}; BRANCH=${EITS_UPDATE_BRANCH:-main}; HEALTH_URL=${EITS_HEALTH_URL:-http://127.0.0.1:8088/api/health}; BACKUP_DIR=${EITS_BACKUP_DIR:-/srv/storage/backups/eits-monitor}; MODE=${EITS_SERVER_UPDATE_MODE:-automatic}
[[ "$MODE" == disabled ]] && exit 0
cd "$APP_DIR"; git fetch --tags origin; old=$(git rev-parse HEAD); remote=$(git rev-parse "origin/$BRANCH")
[[ "$old" == "$remote" ]] && exit 0
[[ "$MODE" == notify ]] && { logger -t eits-server-update "Update available $old -> $remote"; exit 10; }
ts=$(date +%Y%m%d-%H%M%S); mkdir -p "$BACKUP_DIR/$ts"; cp -a .env "$BACKUP_DIR/$ts/.env" 2>/dev/null || true; echo "$old" > "$BACKUP_DIR/$ts/commit"
if docker compose exec -T db pg_dump -U "${POSTGRES_USER:-eits}" "${POSTGRES_DB:-eits}" > "$BACKUP_DIR/$ts/database.sql"; then :; else logger -t eits-server-update 'Database backup failed'; exit 1; fi
git reset --hard "$remote"; docker compose build --pull; docker compose up -d
for _ in {1..30}; do curl -fsS "$HEALTH_URL" >/dev/null && { logger -t eits-server-update "Updated to $remote"; exit 0; }; sleep 5; done
logger -t eits-server-update 'Health check failed; rolling back'; git reset --hard "$old"; docker compose build; docker compose up -d; exit 1
