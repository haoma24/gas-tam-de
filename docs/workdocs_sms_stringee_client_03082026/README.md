# Client SMS OTP Stringee (thật)

- **Thư mục:** `docs/workdocs_sms_stringee_client_03082026`
- **Ngày:** 03/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.3 (nối tiếp `docs/workdocs_sms_adapter_mock_02082026`), architecture §2 + §9.7

## Mục tiêu

Thay seam `ProductionSMSSender` (luôn trả `ErrSMSNotConfigured`) bằng **client Stringee SMS REST thật**, để `POST /v1/auth/otp/request` gửi được OTP tới SĐT khách khi cấu hình credentials Stringee.

## Phạm vi

- Trong scope:
  - `StringeeSMSSender` — JWT `X-STRINGEE-AUTH` (HS256, `cty=stringee-api;v=1`) + `POST /v1/sms`
  - Chọn adapter: `SMS_PROVIDER=stringee` hoặc `SMS_PROVIDER=production` + `SMS_VENDOR=stringee`
  - Env mới: `SMS_API_SID`, `SMS_API_SECRET`, `SMS_TIMEOUT_SEC`, `SMS_JWT_TTL_SEC` (+ fallback `SMS_API_KEY="sid:secret"`)
  - Map lỗi vendor → `ErrSMSRejected` → handler trả `502 SMS_FAILED`
  - Pass-through env SMS trong `deploy/docker-compose.yml`
- Ngoài scope:
  - Stringee **Verify API** (`verify/start` / `verify/check`) — repo vẫn tự sinh + tự verify OTP trong `otp_challenges`
  - Voice OTP fallback, delivery-report webhook, theo dõi chi phí/quota
  - Client eSMS (giữ nguyên seam `ProductionSMSSender`)

## Quyết định chính

- **Dùng SMS REST API (`POST https://api.stringee.com/v1/sms`)**, không dùng Verify API: giữ nguyên OTP tự sinh + hash trong `otp_challenges` (rate limit, TTL, `max_attempts` đã có), tránh phụ thuộc state OTP ở vendor.
- **JWT ký mỗi request** (TTL mặc định 3600s) thay vì cache token: chi phí HMAC không đáng kể so với một HTTP call, đổi lại không phải xử lý refresh/expiry.
- **Fail-closed khi thiếu credential**: thiếu `SMS_API_SID` / `SMS_API_SECRET` / `SMS_SENDER` → `ErrSMSNotConfigured`, **không** gọi vendor.
- **Không retry**: Stringee có thể đã gửi tin trước khi trả lỗi/timeout; retry dễ gây SMS trùng + tốn phí. Khách bấm «gửi lại» sau `resend_after_sec`.
- Coi là thất bại khi `result[0].r != 0`, khi `smsSent < 1`, hoặc HTTP non-2xx (kể cả body có `r` ở top-level khi token sai).
- `to` gửi dạng `84xxxxxxxxx` (bỏ `+` của E.164) theo yêu cầu Stringee; log chỉ `phone_masked`, tuyệt đối không log OTP hay token.
- `SMS_API_KEY="sid:secret"` được parse thành cặp SID/secret cho môi trường chỉ có một secret slot; `SMS_API_SID`/`SMS_API_SECRET` set tường minh thì ưu tiên.

## Đã làm

- [x] `StringeeSMSSender` + JWT + parse response/error vendor
- [x] Factory `newSMSSenderFromEnv` nhận `stringee` (và `production` + `SMS_VENDOR=stringee`)
- [x] `stringeeConfigFromEnv` + fallback `SMS_API_KEY="sid:secret"`
- [x] Unit tests (httptest): success, vendor reject, HTTP 401, `smsSent=0`, thiếu credential, defaults, MSISDN, factory, config env
- [x] `deploy/.env.example` + `deploy/docker-compose.yml` (pass-through `SMS_*`, `OTP_DEV_REVEAL`)
- [x] `docs/architecture.md` §2 + checklist §9.7
- [x] Verify thủ công end-to-end với fake vendor (success + reject)

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/sms_stringee.go` | added | client Stringee SMS REST + JWT + map lỗi |
| `services/auth-service/sms_stringee_test.go` | added | tests httptest (success / reject / 401 / config) |
| `services/auth-service/sms.go` | modified | factory `stringee`, `stringeeConfigFromEnv`, `splitAPIKeyPair` |
| `services/auth-service/sms_test.go` | modified | production seam dùng vendor `esms`; thêm case stringee + config env |
| `services/auth-service/sms_production.go` | modified | comment: stringee đã có client riêng |
| `services/auth-service/main.go` | modified | `smsProviderName` biết `stringee` |
| `deploy/.env.example` | modified | block env Stringee |
| `deploy/docker-compose.yml` | modified | truyền `SMS_*` / `OTP_DEV_REVEAL` vào `auth-service` |
| `docs/architecture.md` | modified | §2 stack OTP + §9.7 checklist prod |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_sms_stringee_client_03082026/` | added | this folder |

## Cấu hình

```bash
SMS_PROVIDER=stringee
SMS_API_SID=SKxxxxxxxx        # API Key SID (dashboard Stringee)
SMS_API_SECRET=xxxxxxxx       # API Key Secret
SMS_SENDER=GASTAMDE           # brandname đã được nhà mạng duyệt
# optional
SMS_API_URL=https://api.stringee.com/v1/sms
SMS_TIMEOUT_SEC=10
SMS_JWT_TTL_SEC=3600
OTP_DEV_REVEAL=0              # bắt buộc 0 ngoài local
```

## Cách verify

1. `go test ./services/auth-service/...`
2. Fake vendor local (không tốn tin thật):

```bash
# server giả trả {"smsSent":1,"result":[{"r":0,"msg":"Success"}]} tại :9099/v1/sms
SMS_PROVIDER=stringee SMS_API_SID=SKlocal SMS_API_SECRET=localsecret \
SMS_SENDER=GASTAMDE SMS_API_URL=http://127.0.0.1:9099/v1/sms \
AUTH_DB=/tmp/auth-stringee.db AUTH_ADDR=:8091 go run ./services/auth-service

curl -s -X POST http://127.0.0.1:8091/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"0901234567"}'
```

Kỳ vọng: log `sms provider selected provider=stringee` → `sms sent provider=stringee phone_masked=090***4567 sms_sent=1`; vendor nhận `{"sms":[{"from":"GASTAMDE","to":"84901234567","text":"Ma OTP Gas Tam De: …"}]}`.

3. Vendor trả `{"smsSent":0,"result":[{"r":2,"msg":"Brandname not approved"}]}` → API trả `502 SMS_FAILED`, log `sms send failed err="sms vendor rejected the message: code=2 …"`.
4. Bỏ `SMS_API_SECRET` → `502 SMS_FAILED` + log `sms stringee not configured missing=SMS_API_SECRET`, không có request nào tới vendor.
5. Smoke thật (tốn phí): 1 SĐT nội bộ, `OTP_DEV_REVEAL=0`, kiểm tra tin về đúng brandname.

## Ghi chú / blocker

- Brandname phải được nhà mạng duyệt trước; chưa duyệt thì Stringee trả `r != 0` → OTP request `502`.
- Chưa có delivery report: hệ thống chỉ biết Stringee **nhận** tin, không biết đã tới máy khách. Nếu cần, bổ sung webhook sau.
- Chưa theo dõi số dư/quota Stringee; hết tiền sẽ biểu hiện thành `502 SMS_FAILED` (đọc log `msg=` của vendor).
