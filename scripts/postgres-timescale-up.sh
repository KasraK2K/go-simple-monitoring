#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.timescale.yml"

# Load .env if present to get POSTGRES_* variables (and/or POSTGRES_DSN for the app)
if [ -f "$REPO_ROOT/.env" ]; then
  # shellcheck disable=SC1090
  set -a
  . "$REPO_ROOT/.env"
  set +a
fi

# Provide sensible defaults if not set
: "${POSTGRES_USER:=monitoring}"
: "${POSTGRES_PASSWORD:=monitoring}"
: "${POSTGRES_DB:=monitoring}"
export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
export COMPOSE_PROJECT_NAME=monitoring

echo "Starting TimescaleDB container (project: $COMPOSE_PROJECT_NAME)..."
docker compose -f "$COMPOSE_FILE" up -d

echo "Waiting for database to become healthy..."
ATTEMPTS=0
until docker compose -f "$COMPOSE_FILE" exec -T timescaledb pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; do
  ATTEMPTS=$((ATTEMPTS+1))
  if [ "$ATTEMPTS" -gt 60 ]; then
    echo "Database did not become ready in time." >&2
    exit 1
  fi
  sleep 1
done

echo "TimescaleDB is up. Container credentials: user=$POSTGRES_USER db=$POSTGRES_DB"
echo "The app will build DSN from POSTGRES_* automatically."
