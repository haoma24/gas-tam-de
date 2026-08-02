# Gateway rate limit OTP / login / place-order (T9.1.2)

- **Thư mục:** `workdocs_gateway_rate_limit_02082026`
- **Ngày:** 02/08/2026
- **Loại:** security
- **Liên quan:** US-9.1 / T9.1.2

## Mục tiêu

Bảo vệ cạnh API Gateway: rate limit `POST /v1/auth/otp/request`, `POST /v1/auth/admin/login`, và `POST /v1/orders` theo IP (và user cho place-order), trả `429 RATE_LIMITED` + header `Retry-After`. Auth-service vẫn giữ OTP limiter theo phone_hash + IP (defense in depth).

## Phạm vi

- Trong scope:
  - Gateway sliding-window limiters (in-process)
  - Env cấu hình quota / phút
  - CORS expose `Retry-After`
  - Unit tests
- Ngoài scope:
  - Security headers / ẩn internal error (T9.1.3)
  - Audit log admin (T9.1.4)
  - Distributed / Redis rate limit
  - Rate limit OTP verify (vẫn do auth-service lockout attempts)

## Quyết định chính

- OTP + admin login: key theo IP (chưa có JWT; phone_hash vẫn do auth-service).
- Place-order: chỉ `POST /v1/orders` exact — không áp cho `/orders/quote` hay `/orders/me`.
- Place-order: kiểm tra IP rồi user (`JWT.sub`); middleware sau `RequireJWT`.
- Quota mặc định / phút: OTP IP 10, login IP 10, order IP 30, order user 10.

## Đã làm

- [x] `ratelimit.go` — sliding window + middleware OTP/login + place-order
- [x] Wire vào `main.go`; env `RATE_LIMIT_*`
- [x] CORS expose `Retry-After`
- [x] Unit tests
- [x] `.env.example` + compose
- [x] Mark `- [DONE] T9.1.2` trên PRD; cập nhật architecture §4.2 / §7.2
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/api-gateway/ratelimit.go` | added | Limiters + middleware |
| `services/api-gateway/ratelimit_test.go` | added | 429 + Retry-After tests |
| `services/api-gateway/main.go` | modified | Wire env + middleware |
| `services/api-gateway/cors.go` | modified | Expose Retry-After |
| `services/api-gateway/proxy_test.go` | modified | `testRouterWithLimits` |
| `deploy/.env.example` | modified | `RATE_LIMIT_*` |
| `deploy/docker-compose.yml` | modified | Pass rate limit env |
| `docs/architecture.md` | modified | Gateway + §7.2 |
| `docs/prd.md` | modified | `[DONE] T9.1.2` |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_gateway_rate_limit_02082026/` | added | Workdoc này |

## Cách verify

1. Unit tests:

```bash
go test ./services/api-gateway/ -count=1
```

2. Manual (gateway chạy, giảm quota tạm thời):

```bash
# Spam OTP — kỳ vọng 429 + Retry-After sau N request
for i in 1 2 3 4 5 6 7 8 9 10 11; do
  curl -si -X POST http://127.0.0.1:8080/v1/auth/otp/request \
    -H "Content-Type: application/json" \
    -d '{"phone":"0901234567"}' | head -n 5
  echo "---"
done
```

## Ghi chú / blocker

- In-process only — mỗi replica gateway có counter riêng (MVP OK).
