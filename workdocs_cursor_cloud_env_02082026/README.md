# Cursor Cloud environment setup + OrderCart fix

- **Thư mục:** `workdocs_cursor_cloud_env_02082026`
- **Ngày:** 02/08/2026
- **Loại:** chore / fix / docs
- **Liên quan:** DevEx / Cursor Cloud agents

## Mục tiêu

Chuẩn bị môi trường phát triển Cloud Agent chạy được full stack Gas Tam Đệ (NATS + Go services + Flutter Web) và ghi chú vận hành cho agent sau.

## Phạm vi

- Trong scope: AGENTS.md cloud instructions; fix compile Flutter `OrderCart.isNotEmpty`; CHANGESLOG.
- Ngoài scope: sửa toàn bộ flutter analyze warnings / digit-separator tests; Android/iOS emulator.

## Quyết định chính

- Update script chỉ refresh deps: `go mod download` + `flutter pub get` (không start services).
- NATS qua Docker Compose; Go services chạy host `go run`.
- Flutter Web trên Chrome cần `--no-sandbox` trong Cloud VM.

## Đã làm

- [x] Cài Docker + Flutter SDK trên VM; `go mod download`; `flutter pub get`
- [x] `make nats` + 8 Go services healthy; `make test` green
- [x] Flutter Web `flutter run -d chrome` — admin login → dashboard (doanh thu từ order hello-world)
- [x] API hello-world: admin login → tạo SP → OTP → place order → complete → dashboard
- [x] Fix `OrderCart.isNotEmpty` (thiếu getter chặn compile)
- [x] `AGENTS.md` section Cursor Cloud

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `AGENTS.md` | added | Cloud-specific run notes |
| `apps/mobile/lib/features/order/order_cart.dart` | modified | `isNotEmpty` getter |
| `CHANGESLOG.md` | modified | entry mới |
| `workdocs_cursor_cloud_env_02082026/` | added | workdocs |

## Cách verify

1. `make nats` rồi start gateway + auth + catalog + geo + order + inventory + billing + report
2. `make health` và `make test`
3. `cd apps/mobile && flutter run -d chrome --web-browser-flag=--no-sandbox`
4. Admin login `admin` / `admin-change-me` → dashboard

## Ghi chú / blocker

- Không có pre-commit / husky trong repo.
- Flutter unit tests: một số file dùng digit separators cần SDK ≥3.6 — pre-existing.
