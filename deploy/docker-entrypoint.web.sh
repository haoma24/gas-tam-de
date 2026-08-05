#!/bin/sh
# Run embedded api-gateway (loopback) then nginx on :8080 for platforms that
# only deploy the public `web` container without a separate api-gateway task.
set -eu

GATEWAY_ADDR="${GATEWAY_ADDR:-127.0.0.1:8081}"
export GATEWAY_ADDR

if [ "${WEB_EMBED_GATEWAY:-1}" != "0" ]; then
  echo "starting embedded api-gateway on $GATEWAY_ADDR"
  /usr/local/bin/gas-api-gateway &
  gw_pid=$!
  i=0
  while [ "$i" -lt 45 ]; do
    if wget -qO- "http://127.0.0.1:8081/healthz" >/dev/null 2>&1; then
      echo "embedded api-gateway healthy"
      break
    fi
    if ! kill -0 "$gw_pid" 2>/dev/null; then
      echo "embedded api-gateway exited early" >&2
      exit 1
    fi
    i=$((i + 1))
    sleep 1
  done
  if [ "$i" -ge 45 ]; then
    echo "embedded api-gateway did not become healthy in time" >&2
    exit 1
  fi
fi

exec nginx -g 'daemon off;'
