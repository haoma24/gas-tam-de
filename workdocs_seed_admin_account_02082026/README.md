# Seed admin account (T1.2.1)

- **Thư mục:** `workdocs_seed_admin_account_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.1

## Mục tiêu

Seed tài khoản admin mặc định vào SQLite `admin_accounts` với **password hash** (không plaintext), cấu hình qua env cho local — nền cho T1.2.2 login API.

## Phạm vi

- Trong scope:
  - Bootstrap sau migrate khi username chưa tồn tại
  - Hash bcrypt (`golang.org/x/crypto/bcrypt`)
  - Env: `ADMIN_USERNAME` / `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_DISPLAY_NAME`, `ADMIN_SEED`
  - Tests + `.env.example` placeholders (không secret thật)
- Ngoài scope:
  - T1.2.2 API login admin + refresh
  - T1.2.3 Flutter admin login
  - T1.2.4 Gateway RBAC
  - Đổi password / disable admin API

## Quyết định chính

- Dùng **bcrypt** (DefaultCost) — đã có trong `x/crypto`, đủ MVP; architecture ghi nhận bcrypt cho seed/login.
- Username ưu tiên `ADMIN_USERNAME`; `ADMIN_EMAIL` là alias (login contract vẫn là `username`).
- Default local: `admin` / `admin-change-me`; log `default_password=true` khi dùng default — đổi trước deploy chung.
- Idempotent: **không** reset hash nếu username đã có (tránh mất mật khẩu khi restart).
- `ADMIN_SEED=0|false` tắt seed (prod có thể provision tay).

## Đã làm

- [x] `admin_seed.go` + gọi từ `main` sau migrate
- [x] Comment schema `admin_accounts`; cập nhật architecture password note
- [x] Env placeholders trong `deploy/.env.example`
- [x] Unit tests create/hash/idempotent/disable/config
- [x] Mark `[DONE] T1.2.1` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/admin_seed.go` | added | Seed + bcrypt helpers |
| `services/auth-service/admin_seed_test.go` | added | Tests |
| `services/auth-service/main.go` | modified | Gọi seed sau migrate |
| `services/auth-service/schema.sql` | modified | Comment T1.2.1 / bcrypt |
| `deploy/.env.example` | modified | ADMIN_* placeholders |
| `docs/architecture.md` | modified | bcrypt note |
| `docs/prd.md` | modified | `[DONE] T1.2.1` |
| `CHANGESLOG.md` | modified | Entry mới |
| `go.mod` / `go.sum` | modified | `golang.org/x/crypto` direct |
| `workdocs_seed_admin_account_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/auth-service/ -count=1`
2. Chạy auth-service với `.env` local → log `admin account seeded` lần đầu; lần sau `admin seed skipped` / `already exists`
3. Confirm PRD: `- [DONE] T1.2.1 Seed admin account (password hash)`

## Ghi chú / blocker

- Chưa có login API — chỉ seed row. Next unfinished: **T1.2.2** API login admin + refresh.
