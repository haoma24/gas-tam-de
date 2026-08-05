#!/usr/bin/env bash
# VPS / staging: pull GHCR images and start without building on the host.
# Usage (from repo root):
#   ./scripts/vps-compose-up.sh
#   COMPOSE_PROJECT_NAME=ts-tamde-stag ./scripts/vps-compose-up.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="${ROOT}/deploy/docker-compose.yml"
PROJECT="${COMPOSE_PROJECT_NAME:-ts-tamde-stag}"

cd "$ROOT/deploy"

echo "==> pull (project=$PROJECT)"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" pull

echo "==> up --no-build"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d --no-build --remove-orphans

docker compose -f "$COMPOSE_FILE" -p "$PROJECT" ps -a
