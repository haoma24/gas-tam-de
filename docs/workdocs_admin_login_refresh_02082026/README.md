# API login admin + refresh (T1.2.2)

- **Thư mục:** `docs/workdocs_admin_login_refresh_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.2 / architecture §4.4, §7.2

## Mục tiêu

Implement `POST /v1/auth/admin/login` và `POST /v1/auth/refresh` trên auth-service: xác thực admin bằng bcrypt, phát JWT `role=admin`, và xoay vòng refresh cho cả admin lẫn customer (token OTP đã phát từ T1.1.2).

## Phạm vi

- Trong scope:
  - `POST /v1/auth/admin/login` `{ username, password }` → access + refresh
  - `POST /v1/auth/refresh` `{ refresh_token }` → token mới (rotation)
  - Session SQLite `sessions` với `role=admin|customer`
  - Unit tests login / wrong creds / disabled / refresh rotate / customer refresh
- Ngoài scope:
  - T1.2.3 Flutter admin login screen
  - T1.2.4 Middleware RBAC trên gateway
  - Rate limit login (T9.1.2)
  - Đổi password / disable admin API

## Quyết định chính

- Tái dùng `issueAccessToken` / `generateRefreshToken` / `insertSession` từ OTP verify.
- JWT admin: `role=admin`, `sub=admin_accounts.id`, không có `phone_masked`.
- Refresh **xoay vòng**: revoke session cũ + insert session mới; reuse refresh cũ → `401 INVALID_TOKEN`.
- Sai username/password/disabled → cùng `401 INVALID_CREDENTIALS` (tránh enumerate).
- Refresh phục vụ cả customer (OTP) và admin — một endpoint theo architecture §4.4.

## Đã làm

- [x] `tokenService` + `handleAdminLogin` + `handleRefresh`
- [x] Wire routes trong `main.go` (bỏ stub 501)
- [x] Unit tests admin login + refresh rotation (admin + customer)
- [x] Mark `[DONE] T1.2.2` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/token_service.go` | added | Shared JWT TTL + parse helper |
| `services/auth-service/admin_login.go` | added | Admin password login |
| `services/auth-service/refresh.go` | added | Refresh rotation |
| `services/auth-service/admin_login_test.go` | added | Login tests |
| `services/auth-service/refresh_test.go` | added | Refresh tests |
| `services/auth-service/main.go` | modified | Wire login/refresh; share JWT config |
| `docs/prd.md` | modified | `[DONE] T1.2.2` |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_admin_login_refresh_02082026/` | added | Workdoc này |

## Cách verify

1. `go test ./services/auth-service/ -count=1`
2. Chạy auth-service (seed admin mặc định `admin` / `admin-change-me`):

```bash
curl -s -X POST http://127.0.0.1:8081/v1/auth/admin/login -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"admin-change-me\"}"
curl -s -X POST http://127.0.0.1:8081/v1/auth/refresh -H "Content-Type: application/json" -d "{\"refresh_token\":\"<refresh>\"}"
```

3. Kỳ vọng `access_token`, `refresh_token` mới, `user.role=admin`.

## Ghi chú / blocker

- Gateway chưa validate JWT / RBAC — test thẳng auth-service `:8081`.
- Next unfinished: **T1.2.3** Flutter admin login screen.
