# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repo

Monorepo for the **Gas Tam Đệ** gas-delivery shop: Flutter app (`apps/mobile`, Web + Android + iOS from one codebase) + 8 Go microservices (`services/*`) + one SQLite file per service (`data/*.db`) + NATS JetStream. Docs (`docs/prd.md`, `docs/architecture.md`), `README.md` and `CHANGESLOG.md` are written in Vietnamese — keep new docs/changelog entries in Vietnamese too.

## Mandatory workflow: CHANGESLOG + workdocs

`.cursor/rules/change-workdocs.mdc` is `alwaysApply` and binds any agent working here. For every meaningful change (feature, fix, refactor, large docs/config change):

1. Create/update `docs/workdocs_<mo-ta-khong-dau>_<ddmmyyyy>/README.md` (feature work: required; small fixes: optional).
2. Prepend a new entry to `CHANGESLOG.md` (never append at the end, never edit old entries) using the `## [YYYY-MM-DD] <title>` / Loại / Phạm vi / Tóm tắt / Chi tiết / Workdocs / Liên quan format already used there. Small change with no workdocs ⇒ `Workdocs: n/a`.

Full rules and templates: `.cursor/skills/change-workdocs/SKILL.md`.

## Branch / deploy convention

Everything merges into **`stag`** (not `main`/`master`) unless the maintainer says otherwise. Pushing `stag` triggers `backend-ci.yml` (Go tests → build+push all service images to GHCR `:stag`) and `web-image.yml` (Flutter Web + nginx image). The VPS deploys with `docker compose up --no-build`, so a service is only deployable once its image is on GHCR.

## Commands

Windows (no GNU Make) and POSIX have the same target names:

```powershell
.\scripts\dev.ps1 help          # make help
```

| Task | Command |
|------|---------|
| Go tests | `make test` (`go test ./...`) |
| One package | `go test ./services/order-service` |
| One test | `go test ./services/order-service -run TestCreateOrder -v` |
| Build all services | `make build` |
| NATS + bootstrap streams | `make nats` (compose `nats` + `go run ./cmd/nats-init`) — starts **only** NATS |
| One service on host | `make gateway` \| `auth` \| `catalog` \| `geo` \| `order` \| `inventory` \| `billing` \| `report` |
| Full stack in Docker | `make compose-up` (build + `--wait`), `make compose-down`, `make compose-ps`, `make compose-logs` |
| Website only (nginx + gateway + auth + nats) | `make web-up`, `make web-logs` |
| Smoke | `make health` (gateway), `make stack-health`, `make web-health` |
| Why is a container unhealthy | `make doctor` (prints last health probe + 40 log lines per non-healthy container) |
| Flutter | `make flutter-web` / `flutter-android` / `flutter-ios`; in `apps/mobile`: `flutter analyze`, `flutter test`, `flutter test test/otp_page_test.dart` |

Ports: gateway `8080`, auth `8081`, catalog `8082`, geo `8083`, order `8084`, inventory `8085`, billing `8086`, report `8087`, website `8090` (host) → nginx `:8080`, NATS monitoring `8222`.

Known-noisy checks: `flutter analyze` has pre-existing deprecation infos, and `test/dashboard_models_test.dart` has an SDK-constraint issue with digit separators. Don't treat those as new breakage.

## Architecture

**Gateway is the only entry point.** `services/api-gateway/main.go` builds one chi router that reverse-proxies `/v1/*` to upstreams by path, and that router *is* the authorization model — routes are grouped as public (`/v1/auth/*`, `/v1/products*`, `GET /v1/geo/store|search`, `GET /v1/stock/levels`), customer (`RequireJWT` + `RequireRole(roleCustomer)` + place-order rate limit), and admin (`RequireJWT` + `RequireRole(roleAdmin)` + `AuditAdminMutations`). Upstream services do **not** re-check roles, so adding an endpoint means adding it to the right group in `newGatewayRouter`. Client-supplied identity headers (`X-User-Id`, `X-User-Role`, …) are stripped in `stripInboundIdentityHeaders` before JWT middleware re-attaches trusted values. The gateway binds `:8080` with a health-only handler first and swaps in the real router once SQLite is ready (`atomicHandler`), so Traefik never waits on DB init.

**Service shape.** Every `services/*/main.go` follows the same pattern: `config.ListenAddr` → `sqlite.Open` → `migrate()` from an `//go:embed schema.sql` → optional `natsx.NewBackground(url).Start(...)` → `httpx.NewRouter` + `MountHealth` + `MountReady` → `httpx.ListenAndServe`. Each service is `package main` with tests in-package alongside the handlers.

**Events.** Subjects are constants in `pkg/events` (`<context>.<entity>.<verb_past>`) wrapped in `events.Envelope` (`event_id`, `occurred_at`, `schema_version`). Publishers take a `natsx.JSProvider` (`natsx.Static(js)` in tests); consumers are durable JetStream subscriptions attached from the `Start(onReady)` callback, use `ManualAck` + `Nak` on failure, and stay idempotent via a `processed_events(event_id PRIMARY KEY)` table in each service DB. Stock/debt/report updates happen on `order.completed`, never on `order.placed`.

**Persistence.** One SQLite file per service, WAL, `SetMaxOpenConns(1)` (single writer per process) — see `pkg/sqlite`. Schema lives in each service's `schema.sql` and is applied at startup; there is no migration tool, so schema changes must be idempotent (`CREATE TABLE IF NOT EXISTS` / additive).

**Flutter.** Riverpod + go_router + dio. `lib/core/api_client.dart` owns the single `Dio` provider whose interceptor refreshes the access token pre-emptively when expired and once more on a 401, clearing the session if the retry still 401s. `lib/core/api_config.dart` reads `--dart-define=API_BASE_URL` (default `http://127.0.0.1:8080`; empty in the Docker web image = same-origin through nginx). Feature folders under `lib/features/<context>/` mirror the backend bounded contexts and pair `*_api.dart` + `*_models.dart` + pages. Customer vs admin is one app switched by `session.isAdmin` in the router.

## Things that bite

- **`/healthz` is liveness, `/readyz` is readiness.** `/healthz` never touches NATS and is what compose healthchecks and `depends_on` use; putting `/readyz` in a healthcheck reintroduces "whole stack fails because the broker is slow". NATS-dependent services serve HTTP immediately and connect in the background forever (`pkg/natsx.Background`, backoff 1s→30s); `NATS_STARTUP_TIMEOUT_SEC` bounds one connect round, not the process.
- **`JWT_SECRET` must match** across auth-service (signer) and *both* gateways (standalone container and the one behind `web`). A mismatch 401s every authenticated route right after a successful login. Each process logs `jwt_secret_fp=<8 hex>` (`config.SecretFingerprint`) — compare those first.
- **Listen addresses**: never `127.0.0.1`. `httpx.ListenAndServe` normalizes `:port`/loopback to `0.0.0.0:port` on `tcp4` because IPv6-only binds make container probes fail.
- **Two compose files.** `deploy/docker-compose.yml` is the VPS/Coolify file: no `build:`, no published host ports, images pulled from GHCR, all services joined to `tensorship-net` for Traefik. Host-port mappings and build contexts live in `deploy/docker-compose.local.yml`, merged automatically by `make`. Adding `build:` or `ports:` to the main file breaks deploy.
- **`.env` values must be YAML-safe**: the deploy platform copies `KEY=VALUE` into an unquoted `docker-compose.override.yml`, so no `": "` inside a value. `make check-env-yaml` enforces this and runs in CI.
- **VPS triage**: `make vps-net-check[ --fix]` for `cause=NotOnNet`, `make vps-api-diagnose` for web→gateway, `make vps-up` for a manual pull+up. Details in `README.md` and `docs/workdocs_vps_*`.
- **Flutter Web uses hash routes** (`/#/admin/login`); a path-only URL after a full reload looks broken but isn't.
- Dev defaults need no `.env` (`deploy/.env.example` documents everything). OTP returns `dev_code` because `docker-compose.local.yml` sets `OTP_DEV_REVEAL=1`; `docker-compose.yml` defaults it to `0`, since a revealed code lets anyone log in as any phone. **The GCP stag deploy merges the local overlay** (`web-image.yml`), so staging reveals codes until `deploy/.env` on that VM sets `0` — the overlay is not local-only despite its name. Seeded admin `admin` / `admin-change-me` with `ADMIN_SEED=1`; phones in `ADMIN_PHONES` (default `0909777020`) get `role=admin` straight from the customer OTP flow.
- **Real SMS**: `SMS_PROVIDER=stringee` + `SMS_API_SID`/`SMS_API_SECRET`/`SMS_SENDER` (approved brandname). `production` with any vendor other than Stringee is an unimplemented seam that always fails. An unrecognized `SMS_PROVIDER` falls back to mock and logs an error — check for `sms provider selected` at startup.
