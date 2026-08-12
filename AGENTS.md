# AGENTS.md

## Cursor Cloud specific instructions

**Git / PR:** Mọi thay đổi merge vào nhánh **`stag`** (deploy VPS / CI push `:stag`).
Không mở PR target `master` trừ khi maintainer yêu cầu rõ.

**VPS deploy:** Platform chỉ load `deploy/docker-compose.yml` (pull GHCR `:stag`, không
build trên host). Local `make compose-up` merge `docker-compose.local.yml` (có `build:`).

Gas Tam Đệ monorepo: Flutter (`apps/mobile`) + Go microservices (`services/*`) + SQLite files under `data/` + NATS JetStream. Standard commands live in root `README.md` and `Makefile` (`make help`).

### Services (local DX)

| Need | How |
|------|-----|
| NATS JetStream | `make nats` (Docker Compose `deploy/docker-compose.yml` + `go run ./cmd/nats-init`). Catalog/order/inventory/billing/report still **serve HTTP** when NATS is down; they connect in the background and report it on `/readyz`. |
| Go APIs | One process each: `make gateway` `:8080`, `auth` `:8081`, `catalog` `:8082`, `geo` `:8083`, `order` `:8084`, `inventory` `:8085`, `billing` `:8086`, `report` `:8087`. |
| Flutter Web (dev) | `make flutter-web` or `cd apps/mobile && flutter run -d chrome` (API defaults to `http://127.0.0.1:8080`). |
| Website in Docker | `make web-up` → release Flutter Web build behind nginx on `:8090`, proxying `/v1` + `/healthz` to `api-gateway`. |
| Whole stack | `make compose-up` (builds everything, waits for healthy). `make nats` starts **only** NATS. |
| Smoke | `make health` → gateway `/healthz` + `/v1/hello`; `make stack-health` covers containers + gateway + web. |
| Logs | `make compose-logs` (all services), `make web-logs` (nginx), `make nats-logs` (NATS only). |
| Debug unhealthy | `make doctor` → for each non-healthy container: last health probe output + 40 log lines. |
| Go tests | `make test` (`go test ./...`). |

Every compose service has a `healthcheck` + `restart: unless-stopped`, so a
crashed container shows up as `unhealthy` in `make compose-ps` instead of failing
silently. Successful `/healthz` requests are not access-logged (`pkg/httpx`), which
keeps `docker compose logs` readable.

`/healthz` is **liveness** (process is serving; never touches NATS) and is what
compose healthchecks and `depends_on` use. `/readyz` is **readiness**: 200 when
dependencies pass, 503 plus the failing dependency name otherwise. Do not put
`/readyz` in a container healthcheck — that reintroduces the failure where a slow
broker fails the whole `docker compose up`.

`natsx.NewBackground(url).Start(onReady)` connects JetStream off the critical
path, retrying forever (backoff 1s → 30s), so NATS-dependent services no longer
exit at boot. Publishers take a `natsx.JSProvider` (use `natsx.Static(js)` in
tests); consumers attach from the `onReady` callback. `NATS_STARTUP_TIMEOUT_SEC`
(default 60s) now bounds a single connect round, not the process lifetime.

`httpx.ListenAndServe` normalizes `:port` to `0.0.0.0:port` and listens on
`tcp4`, because an IPv6-only bind makes `wget http://127.0.0.1:<port>/healthz`
probes fail on hosts with `net.ipv6.bindv6only=1`.

Dev defaults (no `.env` required): see `deploy/.env.example`. OTP returns
`dev_code` because `docker-compose.local.yml` sets `OTP_DEV_REVEAL=1`;
`docker-compose.yml` defaults it to **0**, since a revealed code lets anyone log
in as any phone. Note the GCP stag deploy merges the local overlay too, so
staging still reveals codes until `deploy/.env` on that VM sets `0`. Seeded admin: `admin` / `admin-change-me` (`ADMIN_SEED=1`), plus the phone allow-list `ADMIN_PHONES` (default `0909777020`) whose numbers get `role=admin` straight from the customer OTP flow.

`JWT_SECRET` must be identical for **auth-service** (signs the access token) and
both api-gateways — the standalone container and the one embedded in `web`.
A mismatch 401s every authenticated route immediately after a successful login,
which no client-side token refresh can recover from. Each process logs
`jwt_secret_fp=<8 hex>` at startup; compare those before debugging anything else.

### Cloud VM gotchas

- **Docker**: required for NATS. If the daemon is not running, start `dockerd` (this environment uses `fuse-overlayfs` + `iptables-legacy`). Prefer `chmod 666 /var/run/docker.sock` or `sudo docker` when the user is not yet in the `docker` group for the current shell.
- **Flutter on Chrome in Cloud VMs**: pass `--web-browser-flag=--no-sandbox --web-browser-flag=--disable-dev-shm-usage` or Chrome may hang on “Waiting for connection from debug service”.
- **go_router URLs**: Flutter Web uses **hash** routes (`http://localhost:<port>/#/admin/login`, `/#/admin`, …). Path-only `/admin/login` after a full reload can look wrong; prefer `#/...`.
- **Flutter `web-server` device**: serves for curl/HTML checks, but CanvasKit debugging is weaker than `-d chrome`; prefer Chrome for UI verification.
- **Outbound geo**: address autocomplete uses Photon by default; GPS + Haversine still work if geocode is unreachable.

### Lint / test notes

- Go: `make test` should be green.
- Flutter: `flutter analyze` / `flutter test` under `apps/mobile`. There are known pre-existing analyzer infos (deprecated APIs) and some test SDK constraint issues around digit separators in `test/dashboard_models_test.dart` (pubspec SDK floor vs Dart 3.6+ syntax)—do not treat those as environment setup failures unless you are changing that package.
