# AGENTS.md

## Cursor Cloud specific instructions

Gas Tam Đệ monorepo: Flutter (`apps/mobile`) + Go microservices (`services/*`) + SQLite files under `data/` + NATS JetStream. Standard commands live in root `README.md` and `Makefile` (`make help`).

### Services (local DX)

| Need | How |
|------|-----|
| NATS JetStream | `make nats` (Docker Compose `deploy/docker-compose.yml` + `go run ./cmd/nats-init`). Catalog/order/inventory/billing/report **exit if NATS is down**. |
| Go APIs | One process each: `make gateway` `:8080`, `auth` `:8081`, `catalog` `:8082`, `geo` `:8083`, `order` `:8084`, `inventory` `:8085`, `billing` `:8086`, `report` `:8087`. |
| Flutter Web (dev) | `make flutter-web` or `cd apps/mobile && flutter run -d chrome` (API defaults to `http://127.0.0.1:8080`). |
| Website in Docker | `make web-up` → release Flutter Web build behind nginx on `:8090`, proxying `/v1` + `/healthz` to `api-gateway`. |
| Whole stack | `make compose-up` (builds everything, waits for healthy). `make nats` starts **only** NATS. |
| Smoke | `make health` → gateway `/healthz` + `/v1/hello`; `make stack-health` covers containers + gateway + web. |
| Logs | `make compose-logs` (all services), `make web-logs` (nginx), `make nats-logs` (NATS only). |
| Go tests | `make test` (`go test ./...`). |

Every compose service has a `healthcheck` + `restart: unless-stopped`, so a
crashed container shows up as `unhealthy` in `make compose-ps` instead of failing
silently. Successful `/healthz` requests are not access-logged (`pkg/httpx`), which
keeps `docker compose logs` readable.

Dev defaults (no `.env` required): see `deploy/.env.example`. Local OTP returns `dev_code` when `OTP_DEV_REVEAL=1`. Seeded admin: `admin` / `admin-change-me` (`ADMIN_SEED=1`).

### Cloud VM gotchas

- **Docker**: required for NATS. If the daemon is not running, start `dockerd` (this environment uses `fuse-overlayfs` + `iptables-legacy`). Prefer `chmod 666 /var/run/docker.sock` or `sudo docker` when the user is not yet in the `docker` group for the current shell.
- **Flutter on Chrome in Cloud VMs**: pass `--web-browser-flag=--no-sandbox --web-browser-flag=--disable-dev-shm-usage` or Chrome may hang on “Waiting for connection from debug service”.
- **go_router URLs**: Flutter Web uses **hash** routes (`http://localhost:<port>/#/admin/login`, `/#/admin`, …). Path-only `/admin/login` after a full reload can look wrong; prefer `#/...`.
- **Flutter `web-server` device**: serves for curl/HTML checks, but CanvasKit debugging is weaker than `-d chrome`; prefer Chrome for UI verification.
- **Outbound geo**: address autocomplete uses Photon by default; GPS + Haversine still work if geocode is unreachable.

### Lint / test notes

- Go: `make test` should be green.
- Flutter: `flutter analyze` / `flutter test` under `apps/mobile`. There are known pre-existing analyzer infos (deprecated APIs) and some test SDK constraint issues around digit separators in `test/dashboard_models_test.dart` (pubspec SDK floor vs Dart 3.6+ syntax)—do not treat those as environment setup failures unless you are changing that package.
