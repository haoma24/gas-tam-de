#!/usr/bin/env bash
# Report (and optionally repair) containers that are missing from the platform
# proxy network.
#
# The Cursor Cloud / Coolify style deploy starts the stack with docker compose
# and then runs `docker network connect <proxy net> <container>` so Traefik can
# route to it. That attach fails when the container is in `restarting` state, or
# when a container from the previous deploy still holds the endpoint name, and
# the deploy ends with:
#   [HEALTHCHECK FAILED] cause=NotOnNet labeledPort=8080
#   Warning: failed to connect container <id> to tensorship-net
#
# Usage (on the VPS, from the repo root):
#   ./scripts/vps-net-check.sh              # report only
#   ./scripts/vps-net-check.sh --fix        # attach whatever is missing
#   PROXY_NETWORK=other-net ./scripts/vps-net-check.sh --fix
#   COMPOSE_PROJECT=gas-tam-de ./scripts/vps-net-check.sh
#
# Exit code 0 = every running container is attached, 1 = something is missing
# (or was attached by --fix), 2 = the script could not inspect Docker at all.

set -uo pipefail

PROXY_NETWORK="${PROXY_NETWORK:-tensorship-net}"
FIX=0
[[ "${1:-}" == "--fix" ]] && FIX=1

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker not found in PATH" >&2
  exit 2
fi

if ! docker info >/dev/null 2>&1; then
  echo "error: cannot talk to the Docker daemon (not running, or missing permission)." >&2
  exit 2
fi

if ! docker network inspect "$PROXY_NETWORK" >/dev/null 2>&1; then
  echo "error: network '$PROXY_NETWORK' does not exist on this host." >&2
  echo "       Networks available:" >&2
  docker network ls --format '  {{.Name}} ({{.Driver}})' >&2
  echo "       Set PROXY_NETWORK=<name> if the proxy uses another network." >&2
  exit 2
fi

# Restrict to this stack when a compose project is known, otherwise look at all
# containers that carry a compose project label.
if [[ -n "${COMPOSE_PROJECT:-}" ]]; then
  filter=(--filter "label=com.docker.compose.project=$COMPOSE_PROJECT")
else
  filter=(--filter "label=com.docker.compose.project")
fi

mapfile -t containers < <(docker ps -a "${filter[@]}" --format '{{.ID}}')
if [[ ${#containers[@]} -eq 0 ]]; then
  echo "error: no compose-managed containers found (is the stack deployed?)" >&2
  exit 2
fi

problems=0
printf '%-28s %-12s %-10s %s\n' CONTAINER STATE "ON-NET" NOTE
for id in "${containers[@]}"; do
  name=$(docker inspect --format '{{.Name}}' "$id" | sed 's|^/||')
  state=$(docker inspect --format '{{.State.Status}}' "$id")
  attached=$(docker inspect --format "{{if index .NetworkSettings.Networks \"$PROXY_NETWORK\"}}yes{{else}}no{{end}}" "$id")
  note=""

  if [[ "$attached" == "no" ]]; then
    problems=$((problems + 1))
    case "$state" in
      restarting)
        # Docker refuses `network connect` while a container is restarting, so
        # the crash loop is the thing to fix — attaching cannot succeed first.
        note="crash loop — fix the container, then re-attach (docker logs $name)"
        ;;
      running|created|exited)
        note="not on $PROXY_NETWORK"
        if [[ $FIX -eq 1 ]]; then
          svc=$(docker inspect --format '{{index .Config.Labels "com.docker.compose.service"}}' "$id")
          if out=$(docker network connect --alias "$svc" "$PROXY_NETWORK" "$id" 2>&1); then
            note="attached to $PROXY_NETWORK (alias $svc)"
          else
            note="attach FAILED: $(printf '%s' "$out" | tail -1)"
          fi
        fi
        ;;
    esac
  fi

  printf '%-28s %-12s %-10s %s\n' "$name" "$state" "$attached" "$note"
done

echo
if [[ $problems -eq 0 ]]; then
  echo "OK: every container is on '$PROXY_NETWORK'."
  exit 0
fi

if [[ $FIX -eq 1 ]]; then
  echo "Re-run without --fix to confirm, then redeploy if Traefik still 404s."
else
  echo "Run './scripts/vps-net-check.sh --fix' to attach them, or redeploy."
fi
exit 1
