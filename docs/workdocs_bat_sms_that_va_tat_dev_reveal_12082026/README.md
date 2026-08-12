# Sẵn sàng gửi OTP thật: tắt dev reveal mặc định + cảnh báo SMS_PROVIDER sai

- **Thư mục:** `docs/workdocs_bat_sms_that_va_tat_dev_reveal_12082026`
- **Ngày:** 12/08/2026
- **Loại:** security + fix
- **Liên quan:** chuẩn bị test SMS thật qua Stringee

## Mục tiêu

Client Stringee (`sms_stringee.go`) đã hoàn chỉnh và compose đã truyền đủ biến
`SMS_*`, nên bật SMS thật chỉ là việc cấu hình. Nhưng hai thứ khiến việc đó dễ
hỏng âm thầm:

1. **`OTP_DEV_REVEAL` mặc định `1`** ở cả `docker-compose.yml` lẫn
   `.env.vps.example`. API trả thẳng `dev_code` trong response, nên **ai gọi được
   `/v1/auth/otp/request` cũng đăng nhập được bằng bất kỳ số nào**. Bật SMS thật
   mà quên tắt cái này thì lỗ hổng vẫn nguyên vẹn — gửi SMS thật không đóng nó.
2. **`SMS_PROVIDER` gõ sai âm thầm về mock.** Nhánh `default` trả
   `NewMockSMSSender()` không một dòng log. Đặt nhầm `stringee-sms` chẳng hạn:
   request trả 200, không tin nào tới máy, không dấu hiệu nào để lần ra.

## Phạm vi

- Trong scope: mặc định `OTP_DEV_REVEAL`, cảnh báo provider lạ, test chặn hồi quy
- Ngoài scope: client cho vendor khác Stringee (`sms_production.go` vẫn là seam),
  đặt credential thật (chủ repo làm trên dashboard + env của project)

## Quyết định chính

- **Tách mặc định theo file compose thay vì đổi một giá trị.** `docker-compose.yml`
  (file deploy dùng chung) về `0`; `docker-compose.local.yml` đặt `1` để dev
  không cần vendor SMS.
- ⚠️ **`docker-compose.local.yml` KHÔNG chỉ dùng cho local.** Job `deploy-gcp`
  trong `.github/workflows/web-image.yml` (dòng 160, 168) merge cả file này khi
  deploy lên VM stag. Nên **tamde-stag vẫn lộ mã OTP** sau thay đổi này. Đó là
  điều chủ repo đang muốn (tạm dùng dev mode vì chưa có brandname), nhưng phải
  đóng trước khi có khách thật:

  ```bash
  # trên VM, thêm vào /opt/gas-tam-de/deploy/.env
  OTP_DEV_REVEAL=0
  ```

  `--env-file deploy/.env` thắng giá trị mặc định `:-1` của overlay. Nền tảng
  nào chỉ nạp `docker-compose.yml` (Coolify/Cursor Cloud) thì đã tự động là `0`.
- **Vẫn fallback về mock khi provider lạ, nhưng log `ERROR`** kèm giá trị sai và
  danh sách giá trị hợp lệ. Không cho boot fail: một biến gõ sai không đáng làm
  chết auth-service, nhưng phải nhìn thấy được.
- **Chốt cả hai mặc định bằng test compose** (`deploy/compose_env_test.go`) —
  cùng kiểu với các guard sẵn có cho `JWT_SECRET` và `INVENTORY_SERVICE_URL`.
- `deploy/.env.example` **giữ `OTP_DEV_REVEAL=1`** vì đó là file local, nhưng
  comment nói rõ hậu quả và cảnh báo đừng để dòng đó lọt vào env của VPS.

## Đã làm

- [x] `docker-compose.yml`: `OTP_DEV_REVEAL` mặc định `0` + comment giải thích
- [x] `docker-compose.local.yml`: override `1` cho dev local
- [x] `.env.vps.example`: `OTP_DEV_REVEAL=0`
- [x] `.env.example`: giữ `1`, comment mạnh hơn
- [x] `sms.go`: log `ERROR` khi `SMS_PROVIDER` không nhận dạng được
- [x] 3 test: 2 guard compose + 1 test bắt đúng nội dung log cảnh báo
- [x] Cập nhật `AGENTS.md`, `CLAUDE.md` (đang mô tả hành vi cũ)

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/docker-compose.yml` | modified | mặc định `0` |
| `deploy/docker-compose.local.yml` | modified | override `1` cho local |
| `deploy/.env.vps.example` | modified | `0` + cảnh báo |
| `deploy/.env.example` | modified | comment hậu quả |
| `deploy/compose_env_test.go` | modified | 2 guard |
| `services/auth-service/sms.go` | modified | log provider lạ |
| `services/auth-service/sms_test.go` | modified | test nội dung cảnh báo |
| `AGENTS.md`, `CLAUDE.md` | modified | mô tả lại mặc định |

## Cách bật SMS thật

Env của project (staging):

```bash
SMS_PROVIDER=stringee
SMS_API_SID=SK...          # dashboard Stringee
SMS_API_SECRET=...
SMS_SENDER=<brandname đã duyệt>
OTP_DEV_REVEAL=0           # nay đã là mặc định, đặt lại cho chắc
```

Redeploy rồi kiểm tra:

```bash
docker logs gas-tamde-stag-auth-service-1 | grep "sms provider selected"
# provider=stringee   ← nếu thấy provider=mock thì SMS_PROVIDER sai/không tới container

curl -sS -X POST https://<domain>/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"09xxxxxxxx"}'
# response KHÔNG được có "dev_code"
docker logs gas-tamde-stag-auth-service-1 | grep "sms sent"
# provider=stringee sms_sent=1
```

Thiếu SID/SECRET/SENDER ⇒ log `sms stringee not configured missing=...`, API trả
**502 `SMS_FAILED`**, không gọi vendor nên không tốn tiền.

## Ghi chú / blocker

- **Mock sender không log mã OTP** (chỉ log số đã che, `sms_mock.go:41`). Nghĩa là
  `OTP_DEV_REVEAL=0` + `SMS_PROVIDER=mock` = **không ai lấy được mã**, kể cả admin
  đọc log. Tắt reveal chỉ có nghĩa khi đã có vendor thật.
- **Challenge được ghi DB trước khi gửi SMS** (`otp_request.go:85` → `:92`). SMS
  lỗi thì rate limit đã bị tính: `OTP_COOLDOWN_SEC=60`,
  `OTP_MAX_PER_PHONE_HOUR=5`. Lúc dò credential nên nới tạm
  `OTP_MAX_PER_PHONE_HOUR` để khỏi tự khoá.
- Stringee hay từ chối vì **brandname chưa duyệt** hoặc **hết credit**; mã lỗi
  vendor được in nguyên trong log (`code=... msg=...`).
- Sau khi tắt dev reveal, mọi bài test thủ công cần **số điện thoại thật nhận
  được SMS**. Kiểm thử tự động không bị ảnh hưởng (dùng mock sender).
