#!/usr/bin/env bash
# Fail if deploy/.env.example values would break Cursor Cloud's generated
# docker-compose.override.yml.
#
# That platform converts each KEY=VALUE into unquoted YAML:
#   KEY: VALUE
# so any value containing ": " (colon + space) is parsed as a nested mapping
# and Stage 5 ("Run Container") dies with:
#   did not find expected key / mapping values are not allowed in this context
#
# Usage (from repo root):
#   ./scripts/check-env-yaml-safe.sh
#   ./scripts/check-env-yaml-safe.sh deploy/.env.example

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${1:-$ROOT/deploy/.env.example}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "error: env file not found: $ENV_FILE" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

override="$tmp/docker-compose.override.yml"
{
  echo "services:"
  echo "  _yaml_safe_probe:"
  echo "    image: alpine:3.20"
  echo "    environment:"
  while IFS= read -r line || [[ -n "$line" ]]; do
    # skip blank / full-line comments
    [[ -z "${line//[[:space:]]/}" ]] && continue
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    # only KEY=VALUE assignments
    [[ "$line" == *=* ]] || continue
    key="${line%%=*}"
    val="${line#*=}"
    # trim key whitespace
    key="$(printf '%s' "$key" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    [[ -n "$key" ]] || continue

    if [[ "$val" == *": "* ]]; then
      echo "ERROR: $ENV_FILE — value for $key contains ': ' (colon+space)." >&2
      echo "       Cursor Cloud writes unquoted YAML into docker-compose.override.yml;" >&2
      echo "       ': ' breaks parsing (Stage 5 Run Container)." >&2
      echo "       Offending value: $val" >&2
      exit 1
    fi

    # Emit the same unquoted form the platform uses.
    printf '      %s: %s\n' "$key" "$val"
  done < "$ENV_FILE"
} >"$override"

# Minimal base so compose accepts the override merge.
cat >"$tmp/docker-compose.yml" <<'YAML'
services:
  _yaml_safe_probe:
    image: alpine:3.20
    command: ["true"]
YAML

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  echo "Simulated override written; validating with docker compose ..."
  if ! docker compose -f "$tmp/docker-compose.yml" -f "$override" config >/dev/null; then
    echo "ERROR: generated override failed docker compose config parse." >&2
    echo "---- override (first 40 lines) ----" >&2
    sed -n '1,40p' "$override" >&2
    exit 1
  fi
else
  echo "WARN: docker compose not available — skipped full YAML parse (colon-space scan passed)."
fi

echo "OK: $ENV_FILE is safe for unquoted YAML override generation."
