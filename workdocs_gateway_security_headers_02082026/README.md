# Gateway security headers + ẩn internal error (T9.1.3)

- **Thư mục:** `workdocs_gateway_security_headers_02082026`
- **Ngày:** 02/08/2026
- **Loại:** security
- **Liên quan:** US-9.1 / T9.1.3

## Mục tiêu

Cứng hóa API Gateway: gắn security headers trên mọi response và đảm bảo lỗi proxy/panic không lộ chi tiết nội bộ (host:port upstream, dial error, stack trace) ra client.

## Phạm vi

- Trong scope:
  - Security headers middleware trên gateway
  - Generic `502 BAD_GATEWAY` cho proxy / misconfigured upstream
  - Strip header fingerprint `Server` / `X-Powered-By` từ upstream
  - `httpx.SafeRecover` → JSON `500 INTERNAL_ERROR` (không leak panic/stack)
  - Unit tests + cập nhật architecture / PRD
- Ngoài scope:
  - Audit log admin actions (T9.1.4)
  - Full CSP cho static web (gateway chỉ API JSON)
  - HTTPS terminate (vẫn đứng sau Caddy/Nginx theo architecture)

## Quyết định chính

- Headers theo architecture §7.2: nosniff + frame deny; thêm Referrer-Policy / Permissions-Policy; CSP chỉ `frame-ancestors 'none'` (API, không serve HTML).
- Proxy error: message cố định `upstream unavailable` — log `err` (có thể chứa internal URL) chỉ server-side.
- Misconfigured upstream URL: cùng message generic (không còn `"invalid upstream URL"`).
- Safe recover đặt trong `pkg/httpx` (thay chi `Recoverer`) vì Recoverer đăng ký sớm trong `NewRouter` — gateway middleware bên ngoài không bắt được panic handler.

## Đã làm

- [x] `security.go` — `SecurityHeaders`
- [x] `proxy.go` — log + generic 502; strip Server / X-Powered-By
- [x] `pkg/httpx` — `SafeRecover` JSON INTERNAL_ERROR
- [x] Unit tests gateway + httpx
- [x] architecture §4.2 / §7.2; mark `- [DONE] T9.1.3`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/api-gateway/security.go` | added | Security headers middleware |
| `services/api-gateway/security_test.go` | added | Headers + no-leak + strip tests |
| `services/api-gateway/proxy.go` | modified | Generic errors + strip headers |
| `services/api-gateway/main.go` | modified | Wire `SecurityHeaders` |
| `pkg/httpx/httpx.go` | modified | `SafeRecover` thay chi Recoverer |
| `pkg/httpx/httpx_recover_test.go` | added | Panic → generic JSON |
| `docs/architecture.md` | modified | Gateway + §7.2 Headers |
| `docs/prd.md` | modified | `[DONE] T9.1.3` |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_gateway_security_headers_02082026/` | added | Workdoc này |

## Cách verify

1. Unit tests:

```bash
go test ./services/api-gateway/ ./pkg/httpx/ -count=1
```

2. Manual (gateway chạy):

```bash
# Security headers trên OK
curl -si http://127.0.0.1:8080/v1/hello | head -n 20

# Upstream down — body không chứa 127.0.0.1 / dial
curl -si -X POST http://127.0.0.1:8080/v1/auth/otp/request \
  -H "Content-Type: application/json" \
  -d '{"phone":"0901234567"}'
```

## Ghi chú / blocker

- Không implement T9.1.4 (audit log).
