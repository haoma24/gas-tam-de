# Scaffold monorepo boilerplate

- **Thư mục:** `workdocs_scaffold_monorepo_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 0 / architecture §2.1, §4, §6, §8, §9

## Mục tiêu

Scaffold cấu trúc monorepo theo `docs/architecture.md`: Go services + shared pkg, Flutter app stub (brand + CTA), NATS/docker-compose, schema SQL tham chiếu — đủ để Sprint 0 health/hello.

## Phạm vi

- Trong scope:
  - `apps/mobile/` Flutter source (home CTA)
  - `services/*` Go stubs + `schema.sql`
  - `pkg/` shared helpers
  - `deploy/docker-compose.yml`, Dockerfile, `.env.example`
  - Root `README.md`, `Makefile`, `.gitignore`, `go.mod`
- Ngoài scope:
  - Implement OTP / place-order / JWT thật
  - Reverse-proxy đầy đủ trong gateway
  - JetStream consumers / publishers
  - CI GitHub Actions
  - Generate đầy đủ `android/` `ios/` `web/` (cần Flutter SDK)

## Quyết định chính

- HTTP framework: **Chi** (chọn một theo architecture “Chi hoặc Fiber”).
- State Flutter: **Riverpod** + **go_router** + **dio** (architecture §8.3).
- Một Go module root `gas-tam-de` (không tách go.mod per service ở MVP).
- Shared code nằm ở `pkg/` (không có trong sơ đồ §2.1 nhưng cần cho boilerplate — ghi nhận lệch nhẹ so với tree tối thiểu).
- Schema SQL copy từ architecture §6, chưa auto-migrate khi start.

## Đã làm

- [x] Tạo cây monorepo theo architecture
- [x] Stub 8 services + gateway `/healthz` + `/v1/hello`
- [x] Shared `pkg/{config,httpx,sqlite,events,natsx}`
- [x] `deploy/docker-compose.yml` (NATS + services)
- [x] Flutter home brand + CTA placeholder
- [x] CHANGESLOG + workdocs
- [x] Verify `go build` + gateway `/healthz` + `/v1/hello` (local binary smoke OK)
- [x] **T9.2.1 accepted** — layout khớp architecture §2.1; `go build ./services/...` OK (02/08/2026); đánh dấu DONE trên `docs/prd.md` (không scaffold lại)
- [ ] Verify `flutter create` / `flutter run` (Flutter chưa có trên PATH máy này) — thuộc T9.2.4/T9.2.5
- [ ] `docker compose up --build` full stack (chưa chạy trong phiên này) — thuộc T9.2.2/T9.2.3

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `go.mod` / `go.sum` | added | Module root |
| `pkg/**` | added | Shared Go |
| `services/**` | added | 8 services stubs + schema |
| `apps/mobile/**` | added | Flutter source |
| `deploy/**` | added | Compose + Dockerfile + env example |
| `README.md`, `Makefile`, `.gitignore` | added | DX |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_scaffold_monorepo_02082026/` | added | Workdocs |

## Cách verify

1. `go build ./services/...` (hoặc từng service)
2. `go run ./services/api-gateway` → `GET http://127.0.0.1:8080/v1/hello`
3. `docker compose -f deploy/docker-compose.yml up nats -d`
4. Cài Flutter → `cd apps/mobile && flutter create . --platforms=web,android,ios && flutter run -d chrome`

## Ghi chú / blocker / assumptions

- **Flutter SDK không có trên PATH** → chưa generate platform folders; hướng dẫn trong `apps/mobile/README.md`.
- Gateway route stubs trả `501 NOT_IMPLEMENTED` (chưa proxy HTTP thật).
- NATS client helper có sẵn nhưng services chưa connect JetStream.
- Dockerfile build context dùng Go 1.22 image; local Go là 1.26 — `go.mod` theo `go mod tidy`.
- Module path tạm: `gas-tam-de` (chưa gắn GitHub org).
