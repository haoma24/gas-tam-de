# Adapter SMS mock + production seam (T1.1.3)

- **Thư mục:** `docs/workdocs_sms_adapter_mock_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.3 / architecture §2 (OTP interface + mock)

## Mục tiêu

Tách gửi SMS OTP khỏi auth handler qua interface + mock local, giữ seam rõ ràng cho adapter production (eSMS / Stringee / tương đương VN) mà chưa bắt buộc tích hợp vendor thật.

## Phạm vi

- Trong scope:
  - `SMSSender` interface (`SendOTP`)
  - `MockSMSSender` (default) — in-memory + slog masked phone
  - `ProductionSMSSender` seam — config env, trả `ErrSMSNotConfigured` đến khi wire client
  - Wire vào `POST /v1/auth/otp/request` sau khi persist challenge
  - Env keys trong `deploy/.env.example`
- Ngoài scope:
  - HTTP client eSMS/Stringee thật
  - Flutter màn OTP (T1.1.4)
  - Refine schema `otp_challenges` (T1.1.5)

## Quyết định chính

- Chọn adapter bằng `SMS_PROVIDER` (`mock` default; `production` cho seam).
- Persist challenge trước, rồi `SendOTP`; lỗi SMS → `502 SMS_FAILED` (không lộ raw OTP).
- Production seam cố ý fail-closed: có API key vẫn chưa gửi cho đến khi implement vendor client — tránh silent no-op trên prod.
- `OTP_DEV_REVEAL` vẫn độc lập với SMS mock (tiện test verify không cần đọc SMS).

## Đã làm

- [x] Interface + mock + production seam
- [x] Wire OTP request + `502 SMS_FAILED`
- [x] Unit tests (adapter, handler send, SMS failure)
- [x] Env example `SMS_*`
- [x] Mark T1.1.3 DONE trên PRD

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/sms.go` | added | interface + factory từ env |
| `services/auth-service/sms_mock.go` | added | mock sender |
| `services/auth-service/sms_production.go` | added | production seam |
| `services/auth-service/sms_test.go` | added | adapter unit tests |
| `services/auth-service/otp_request.go` | modified | gọi `sms.SendOTP` |
| `services/auth-service/main.go` | modified | wire SMS từ env |
| `services/auth-service/otp_request_test.go` | modified | mock SMS + failure case |
| `services/auth-service/otp_verify_test.go` | modified | inject mock SMS |
| `deploy/.env.example` | modified | `SMS_PROVIDER` + vendor keys |
| `docs/prd.md` | modified | T1.1.3 DONE |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_sms_adapter_mock_02082026/` | added | this folder |

## Cách verify

1. `go test ./services/auth-service/...`
2. `go run ./services/auth-service` (mặc định mock) rồi:

```bash
curl -s -X POST http://127.0.0.1:8081/v1/auth/otp/request -H "Content-Type: application/json" -d "{\"phone\":\"0901234567\"}"
```

3. Log kỳ vọng `sms mock sent` + `phone_masked` (không có raw OTP). Response vẫn có `dev_code` nếu `OTP_DEV_REVEAL=1`.
4. `SMS_PROVIDER=production` không set `SMS_API_KEY` → request trả `502 SMS_FAILED`.

## Ghi chú / blocker

- Vendor HTTP client để empty có chủ đích; khi chọn nhà cung cấp VN chỉ cần điền body trong `ProductionSMSSender.SendOTP`.
- Next PRD task unfinished: **T1.1.4** Flutter màn nhập SĐT + OTP.
