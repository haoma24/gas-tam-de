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

check_one() {
  local ENV_FILE="$1"
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "error: env file not found: $ENV_FILE" >&2
    return 1
  fi

  local tmp override
  tmp="$(mktemp -d)"
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
      local key="${line%%=*}"
      local val="${line#*=}"
      # trim key whitespace
      key="$(printf '%s' "$key" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
      [[ -n "$key" ]] || continue

      # Strip surrounding quotes (dotenv style) — platform often does the same
      # before emitting unquoted YAML.
      if [[ ${#val} -ge 2 ]]; then
        if [[ "${val:0:1}" == '"' && "${val: -1}" == '"' ]] || [[ "${val:0:1}" == "'" && "${val: -1}" == "'" ]]; then
          val="${val:1:${#val}-2}"
        fi
      fi

      if [[ "$val" == *": "* ]]; then
        echo "ERROR: $ENV_FILE — value for $key contains ': ' (colon+space)." >&2
        echo "       Cursor Cloud writes unquoted YAML into docker-compose.override.yml;" >&2
        echo "       ': ' breaks parsing (Stage 5 Run Container)." >&2
        echo "       Offending value: $val" >&2
        rm -rf "$tmp"
        return 1
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
    echo "Simulated override for $ENV_FILE; validating with docker compose ..."
    if ! docker compose -f "$tmp/docker-compose.yml" -f "$override" config >/dev/null; then
      echo "ERROR: generated override failed docker compose config parse." >&2
      echo "---- override (first 40 lines) ----" >&2
      sed -n '1,40p' "$override" >&2
      rm -rf "$tmp"
      return 1
    fi
  else
    echo "WARN: docker compose not available — skipped full YAML parse for $ENV_FILE (colon-space scan passed)."
  fi

  rm -rf "$tmp"
  echo "OK: $ENV_FILE is safe for unquoted YAML override generation."
}

if [[ $# -gt 0 ]]; then
  for f in "$@"; do
    check_one "$f" || exit 1
  done
else
  check_one "$ROOT/deploy/.env.example" || exit 1
  check_one "$ROOT/deploy/.env.vps.example" || exit 1
fi
