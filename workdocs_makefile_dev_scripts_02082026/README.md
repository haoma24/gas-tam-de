# Makefile / scripts chạy dev (T9.2.3)

- **Thư mục:** `workdocs_makefile_dev_scripts_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.3

## Mục tiêu

Có một bộ shortcut DX thống nhất để chạy local: NATS, bootstrap streams, Go services, health check, Flutter — không phải nhớ lệnh compose/`go run` dài.

## Phạm vi

- Trong scope:
  - `Makefile` đầy đủ target Sprint 0 (help, nats*, services, test/build, flutter*)
  - `scripts/dev.ps1` mirror cùng lệnh cho Windows (không phụ thuộc GNU Make)
  - Cập nhật `README.md` hướng dẫn DX
- Ngoài scope:
  - Flutter CTA / platform folders (T9.2.4–T9.2.5)
  - CI GitHub Actions
  - Orchestrator chạy nhiều service song song (để sprint sau nếu cần)

## Quyết định chính

- Tên lệnh giống nhau giữa Make và PowerShell (`nats`, `gateway`, `health`, …).
- Windows: PowerShell là entry chính vì máy dev thường không có GNU Make trên PATH.
- `make nats` / `dev.ps1 nats` = up + wait healthz :8222 + `cmd/nats-init`.
- Không bắt buộc Docker Desktop trong lúc verify script syntax — smoke `help` + `build`/`test` không cần container.

## Đã làm

- [x] Viết lại `Makefile` với `help` mặc định + nats/compose/services/flutter
- [x] Thêm `scripts/dev.ps1` mirror targets
- [x] Cập nhật README DX
- [x] Mark `- [DONE] T9.2.3` trên PRD
- [x] Verify `.\scripts\dev.ps1 help` + `build` + `test`
- [ ] `.\scripts\dev.ps1 nats` khi Docker Desktop đang chạy (engine offline trong phiên này)

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `Makefile` | modified | DX targets đầy đủ |
| `scripts/dev.ps1` | added | Windows mirror |
| `README.md` | modified | Hướng dẫn Make + PS1 |
| `docs/prd.md` | modified | T9.2.3 DONE |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_makefile_dev_scripts_02082026/` | added | this folder |

## Cách verify

1. `.\scripts\dev.ps1 help` — in danh sách lệnh
2. `.\scripts\dev.ps1 build` — `go build ./services/...`
3. `.\scripts\dev.ps1 test` — `go test ./...`
4. (Docker up) `.\scripts\dev.ps1 nats` → `NATS JetStream OK`
5. Terminal 2: `.\scripts\dev.ps1 gateway` rồi `.\scripts\dev.ps1 health`
6. Nếu có GNU Make: `make help` / `make nats` tương đương

## Ghi chú / blocker

- Docker Desktop engine offline → chưa smoke `nats-up` thật trong phiên này.
- GNU Make không có trên PATH máy Windows hiện tại → PS1 là đường chạy chính.
