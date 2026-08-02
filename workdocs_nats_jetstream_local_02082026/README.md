# NATS JetStream local (T9.2.2)

- **Thư mục:** `workdocs_nats_jetstream_local_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.2 / architecture §5.1, §5.4

## Mục tiêu

Chạy NATS với JetStream trên máy local (compose), bootstrap stream theo bounded context, và có cách verify (CLI + test) — nền cho event publish/consume các sprint sau.

## Phạm vi

- Trong scope:
  - `deploy/nats.conf` + compose healthcheck / volume JetStream
  - `pkg/natsx` ConnectJS + EnsureStreams + PingJS
  - `cmd/nats-init` bootstrap/verify streams
  - Unit + embedded JetStream test
- Ngoài scope:
  - Durable consumers / publishers trong từng service (sprint sau)
  - Makefile / full dev scripts (T9.2.3)
  - Flutter CTA / platform checklist (T9.2.4–T9.2.5)

## Quyết định chính

- Stream theo bounded context: `AUTH`, `CATALOG`, `GEO`, `ORDERS`, `INVENTORY`, `BILLING` với subject `auth.>`, `catalog.>`, …
- File storage JetStream trong Docker volume `nats-data`; retention Limits + MaxAge 7 ngày (dev).
- Verify không phụ thuộc Docker Desktop: embedded `nats-server` trong `natsx_embed_test.go`.

## Đã làm

- [x] `deploy/nats.conf` + compose mount/healthcheck/volume
- [x] `depends_on` NATS `service_healthy` cho services
- [x] `pkg/natsx` JetStream helpers
- [x] `cmd/nats-init`
- [x] `go test ./pkg/natsx/...` OK (embedded JetStream)
- [x] Mark T9.2.2 DONE trên PRD
- [ ] `docker compose ... up nats` + `go run ./cmd/nats-init` trên máy có Docker Desktop đang chạy (chưa verify trong phiên này vì engine offline)

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/nats.conf` | added | JetStream config |
| `deploy/docker-compose.yml` | modified | healthcheck, volume, healthy depends |
| `pkg/natsx/natsx.go` | modified | ConnectJS / EnsureStreams |
| `pkg/natsx/natsx_test.go` | added | stream defs unit test |
| `pkg/natsx/natsx_embed_test.go` | added | embedded JetStream integration |
| `cmd/nats-init/main.go` | added | CLI bootstrap |
| `go.mod` / `go.sum` | modified | nats-server test dep |
| `README.md` | modified | NATS JetStream verify steps |
| `docs/prd.md` | modified | T9.2.2 DONE |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_nats_jetstream_local_02082026/` | added | this folder |

## Cách verify

1. `go test ./pkg/natsx/...`
2. (Docker Desktop running) `docker compose -f deploy/docker-compose.yml up nats -d`
3. `go run ./cmd/nats-init` → in stream list + `NATS JetStream OK`
4. Optional: mở `http://127.0.0.1:8222/jsz`

## Ghi chú / blocker

- Docker Desktop engine không chạy trong phiên implement → không smoke compose thật; embedded test thay thế.
- T9.2.3 sẽ gom Makefile shortcuts (`nats-up`, `nats-init`, …) nếu cần.
