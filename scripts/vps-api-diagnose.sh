#!/usr/bin/env bash
# Diagnose «API gateway không sẵn sàng» (nginx api_unavailable) on VPS.
# Usage:
#   COMPOSE_PROJECT_NAME=ts-tamde-stag ./scripts/vps-api-diagnose.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="${ROOT}/deploy/docker-compose.yml"
PROJECT="${COMPOSE_PROJECT_NAME:-ts-tamde-stag}"
PROXY_NETWORK="${PROXY_NETWORK:-tensorship-net}"

cd "$ROOT/deploy"

echo "=== compose ps (project=$PROJECT) ==="
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" ps -a || true

web_id=$(docker compose -f "$COMPOSE_FILE" -p "$PROJECT" ps -q web 2>/dev/null | head -1)
gw_id=$(docker compose -f "$COMPOSE_FILE" -p "$PROJECT" ps -q api-gateway 2>/dev/null | head -1)

if [[ -z "$web_id" ]]; then
  echo "ERROR: no web container — deploy full stack, not only Traefik target." >&2
  exit 1
fi
if [[ -z "$gw_id" ]]; then
  echo "ERROR: no api-gateway container — OTP needs api-gateway + auth + nats." >&2
  exit 1
fi

echo
echo "=== proxy network ($PROXY_NETWORK) ==="
COMPOSE_PROJECT="$PROJECT" PROXY_NETWORK="$PROXY_NETWORK" "$ROOT/scripts/vps-net-check.sh" || true

echo
echo "=== from web: DNS + wget api-gateway:8080/healthz ==="
docker exec "$web_id" sh -c 'getent hosts api-gateway || true; wget -qSO- http://api-gateway:8080/healthz 2>&1 | head -20' || true

echo
echo "=== from web: nginx /gateway-healthz (same path as OTP) ==="
docker exec "$web_id" wget -qSO- http://127.0.0.1:8080/gateway-healthz 2>&1 | head -20 || true

echo
echo "=== api-gateway logs (last 30 lines) ==="
docker logs --tail=30 "$gw_id" 2>&1 || true
