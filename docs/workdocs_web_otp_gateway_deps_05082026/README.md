# Fix OTP «API gateway chưa sẵn sàng» — web stack thiếu auth/gateway

- **Thư mục:** `docs/workdocs_web_otp_gateway_deps_05082026`
- **Ngày:** 05/08/2026
- **Loại:** fix
- **Liên quan:** đăng nhập OTP qua website Docker / Traefik

## Mục tiêu

Người dùng mở app web, nhập SĐT → «Gửi mã OTP» báo **API gateway chưa sẵn sàng**
(JSON `api_unavailable` từ nginx).

## Nguyên nhân

1. **nginx** (`deploy/nginx.web.conf`) trả `503` + `code: api_unavailable` khi
   không dial được `http://api-gateway:8080` (container gateway tắt hoặc không
   cùng network).
2. **`make web-up` cũ** chỉ `compose up web` → Docker khởi động **web +
   api-gateway**, không bật **nats/auth-service**. Gateway chạy nhưng OTP vẫn
   lỗi `BAD_GATEWAY` nếu auth down; trên VPS nếu **chỉ web** lên mà gateway
   không chạy thì đúng lỗi `api_unavailable`.

## Quyết định

- Chuỗi `depends_on`: `api-gateway` ← `auth-service` (healthy); `web` ←
  `api-gateway` + `auth-service` (healthy).
- `make web-up` / `dev.ps1 web-up`: `up --wait nats auth-service api-gateway web`.
- Flutter: map `api_unavailable` / `BAD_GATEWAY` sang tiếng Việt rõ hơn.
- `make web-health`: thêm `/gateway-healthz`.

## Đã làm

- [x] `deploy/docker-compose.yml` — depends_on OTP path
- [x] `Makefile`, `scripts/dev.ps1` — web-up services
- [x] `apps/mobile/lib/features/auth/auth_models.dart` — displayMessage
- [x] `README.md` — troubleshooting OTP
- [x] Verify: `POST /v1/auth/otp/request` qua `:8090` → `ok` + `dev_code`

## File đụng tới

| Path | Ghi chú |
|------|---------|
| `deploy/docker-compose.yml` | gateway→auth, web→gateway+auth |
| `Makefile` | web-up, web-health, help |
| `scripts/dev.ps1` | web-up |
| `apps/mobile/lib/features/auth/auth_models.dart` | lỗi UX |
| `README.md` | hướng dẫn VPS/local |

## Verify

```bash
make web-up
curl -sf -X POST http://127.0.0.1:8090/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"0901234567"}'
make web-health
```
