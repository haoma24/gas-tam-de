#!/usr/bin/env bash
# VPS / staging: pull GHCR images and start without building on the host.
# Usage (from repo root):
#   ./scripts/vps-compose-up.sh
#   COMPOSE_PROJECT_NAME=ts-tamde-stag ./scripts/vps-compose-up.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="${ROOT}/deploy/docker-compose.yml"
PROJECT="${COMPOSE_PROJECT_NAME:-ts-tamde-stag}"
PROXY_NETWORK="${PROXY_NETWORK:-tensorship-net}"

cd "$ROOT/deploy"

if ! docker network inspect "$PROXY_NETWORK" >/dev/null 2>&1; then
  echo "==> creating docker network $PROXY_NETWORK (platform usually creates this first)"
  docker network create "$PROXY_NETWORK"
fi

echo "==> pull (project=$PROJECT)"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" pull

echo "==> up --no-build"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d --no-build --remove-orphans

echo "==> proxy network check"
COMPOSE_PROJECT="$PROJECT" PROXY_NETWORK="$PROXY_NETWORK" "$ROOT/scripts/vps-net-check.sh" --fix || true

docker compose -f "$COMPOSE_FILE" -p "$PROJECT" ps -a
