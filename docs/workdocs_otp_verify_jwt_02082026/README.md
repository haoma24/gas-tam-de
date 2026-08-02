# OTP verify API → JWT (T1.1.2)

- **Thư mục:** `docs/workdocs_otp_verify_jwt_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.2 / architecture §4.4, §7.2

## Mục tiêu

Implement `POST /v1/auth/otp/verify` trên auth-service: kiểm tra OTP đã request, tạo/ cập nhật user khách, phát JWT access + refresh token (session SQLite).

## Phạm vi

- Trong scope:
  - `POST /v1/auth/otp/verify` `{ phone, code }` → access/refresh tokens
  - Max attempts / expire / consume challenge
  - Upsert `users` (phone encrypt at rest) + `sessions` (refresh_hash)
  - JWT HS256 claims: `sub`, `role=customer`, `phone_masked`, `sid`
- Ngoài scope:
  - SMS adapter (T1.1.3)
  - Flutter OTP UI (T1.1.4)
  - `POST /auth/refresh` rotation endpoint (T1.2.2)
  - Gateway JWT validation (T9.1.1)

## Quyết định chính

- Lookup challenge mới nhất theo `phone_hash` (API không cần `challenge_id`).
- So khớp OTP bằng `subtle.ConstantTimeCompare` trên hash peppered (cùng công thức T1.1.1).
- Sai mã → tăng `attempts`; hết `OTP_MAX_ATTEMPTS` → `429 OTP_LOCKED`.
- Phone at rest: AES-GCM với key = SHA256(`PHONE_ENC_KEY`).
- Access TTL mặc định 15 phút; refresh opaque 32 bytes, hash SHA256 lưu `sessions`.

## Đã làm

- [x] Handler verify + transaction (consume + invalidate open challenges)
- [x] Mint JWT access + refresh session
- [x] Encrypt phone khi tạo user
- [x] Unit tests (success, invalid, lockout, replay, no challenge)
- [x] Env keys JWT_* / OTP_MAX_ATTEMPTS trong `deploy/.env.example`
- [x] Mark T1.1.2 DONE trên PRD

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/otp_verify.go` | added | verify handler + user/session |
| `services/auth-service/tokens.go` | added | JWT + refresh helpers |
| `services/auth-service/phone_crypto.go` | added | AES-GCM phone at rest |
| `services/auth-service/otp_verify_test.go` | added | verify tests |
| `services/auth-service/main.go` | modified | wire verify + JWT config |
| `services/auth-service/otp_request.go` | modified | mở rộng `otpService` fields |
| `services/auth-service/otp_request_test.go` | modified | init fields mới |
| `deploy/.env.example` | modified | JWT TTL + OTP_MAX_ATTEMPTS |
| `go.mod` / `go.sum` | modified | `golang-jwt/jwt/v5` |
| `docs/prd.md` | modified | T1.1.2 DONE |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_otp_verify_jwt_02082026/` | added | this folder |

## Cách verify

1. `go test ./services/auth-service/...`
2. `go run ./services/auth-service` rồi:

```bash
curl -s -X POST http://127.0.0.1:8081/v1/auth/otp/request -H "Content-Type: application/json" -d "{\"phone\":\"0901234567\"}"
curl -s -X POST http://127.0.0.1:8081/v1/auth/otp/verify -H "Content-Type: application/json" -d "{\"phone\":\"0901234567\",\"code\":\"<dev_code>\"}"
```

3. Kỳ vọng `access_token`, `refresh_token`, `user.phone_masked`.

## Ghi chú / blocker

- Refresh endpoint vẫn stub `501` — client lưu refresh để dùng khi có T1.2.2.
- Gateway chưa validate JWT; test thẳng auth-service `:8081`.
