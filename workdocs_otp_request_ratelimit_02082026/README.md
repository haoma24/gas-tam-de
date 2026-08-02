# OTP request API + rate limit (T1.1.1)

- **Thư mục:** `workdocs_otp_request_ratelimit_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.1 / architecture §4.4, §7.2

## Mục tiêu

Implement `POST /v1/auth/otp/request` trên auth-service kèm rate limit (cooldown + quota theo phone_hash và IP) để khách có thể yêu cầu OTP trước khi verify.

## Phạm vi

- Trong scope:
  - Validate/normalize SĐT VN → E.164
  - Rate limit OTP request (cooldown + max/hour theo phone & IP)
  - Sinh OTP 6 số, lưu `otp_challenges` (hash + expiry) để request có ý nghĩa
  - Response có `phone_masked`, `expires_in_sec`, `resend_after_sec`
  - Dev: `OTP_DEV_REVEAL` trả `dev_code` (không log plaintext)
- Ngoài scope:
  - `POST /auth/otp/verify` → JWT (T1.1.2)
  - SMS adapter interface/production (T1.1.3)
  - Flutter màn OTP (T1.1.4)
  - Gateway proxy/JWT/RBAC (T9.1.x)

## Quyết định chính

- Rate limit in-process (đủ cho 1 instance MVP); đếm theo `phone_hash` + IP.
- Hash SĐT bằng HMAC-SHA256 (pepper); hash OTP bằng SHA256(pepper:challenge_id:code) — không lưu raw OTP.
- Persist challenge vào SQLite ngay khi request (schema đã có); T1.1.5 có thể refine thêm nếu cần.
- SMS: chỉ `slog` “otp issued” (không raw code); gửi thật để T1.1.3.

## Đã làm

- [x] `POST /v1/auth/otp/request` handler
- [x] Phone normalize/mask/hash
- [x] Rate limiter + `429 RATE_LIMITED` + `Retry-After`
- [x] Migrate `schema.sql` lúc start
- [x] Unit tests
- [x] Mark T1.1.1 DONE trên PRD
- [x] Env keys trong `deploy/.env.example`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/main.go` | modified | wire OTP request + migrate |
| `services/auth-service/otp_request.go` | added | handler + challenge insert |
| `services/auth-service/phone.go` | added | VN phone helpers |
| `services/auth-service/ratelimit.go` | added | IP + phone limiter |
| `services/auth-service/otp_request_test.go` | added | unit/handler tests |
| `deploy/.env.example` | modified | OTP_* env |
| `docs/prd.md` | modified | T1.1.1 DONE |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_otp_request_ratelimit_02082026/` | added | this folder |

## Cách verify

1. `go test ./services/auth-service/...`
2. `go run ./services/auth-service` rồi:

```bash
curl -s -X POST http://127.0.0.1:8081/v1/auth/otp/request -H "Content-Type: application/json" -d "{\"phone\":\"0901234567\"}"
```

3. Gọi lại ngay → kỳ vọng HTTP 429 + `Retry-After`.

## Ghi chú / blocker

- Gateway vẫn stub proxy auth (`501`) — gọi thẳng auth-service `:8081` để test T1.1.1.
- Persist challenge đụng nhẹ T1.1.5; chưa mark DONE T1.1.5 (có thể chỉ cần xác nhận/refine sau).
