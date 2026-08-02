# Gateway routing, CORS, JWT validation (T9.1.1)

- **Thư mục:** `docs/workdocs_gateway_routing_cors_jwt_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature / security
- **Liên quan:** Sprint / US-9.1 / T9.1.1

## Mục tiêu

Hoàn thiện API Gateway: reverse-proxy thật tới mọi upstream (không còn stub 501), CORS cho Flutter Web, và JWT validation vững (cùng secret/claims với auth-service + RBAC đã có từ T1.2.4).

## Phạm vi

- Trong scope:
  - `httputil.ReverseProxy` giữ path `/v1/...`
  - Split `/v1/admin/**` theo service (catalog / geo / order / inventory / billing / report)
  - CORS middleware + `CORS_ORIGINS` (default localhost / 127.0.0.1 wildcard port)
  - Strip client-spoofable identity / internal headers; JWT gắn lại `X-User-*`
  - Require `exp` trên access token; upstream down → `502 BAD_GATEWAY`
- Ngoài scope:
  - Rate limit (T9.1.2)
  - Security headers / ẩn internal error chi tiết (T9.1.3)
  - Audit log admin (T9.1.4)

## Quyết định chính

- Proxy giữ nguyên path — services đã mount `/v1/...` giống gateway.
- Admin không còn một stub `orderURL` chung; map rõ:
  - products → catalog; geo → geo; orders + delivery-fee → order
  - inventory → inventory; debts → billing; dashboard → report
- CORS preflight `OPTIONS` trả 204 trước JWT/RBAC.
- Không expose `/v1/internal/**` qua gateway.

## Đã làm

- [x] `proxy.go` — reverse proxy + strip inbound identity headers
- [x] `cors.go` — CORS + origin matching
- [x] `main.go` — wire all upstreams + admin split
- [x] JWT harden: reject missing `exp`
- [x] Unit tests proxy / CORS / RBAC
- [x] `CORS_ORIGINS` trong `.env.example` + compose; `JWT_SECRET` trên gateway compose
- [x] Sync `api_config.dart` comment + architecture §4.2
- [x] Mark `- [DONE] T9.1.1` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/api-gateway/main.go` | modified | Real proxy + admin split |
| `services/api-gateway/proxy.go` | added | Reverse proxy |
| `services/api-gateway/cors.go` | added | CORS |
| `services/api-gateway/jwt.go` | modified | Require `exp` |
| `services/api-gateway/*_test.go` | added/modified | Proxy/CORS/RBAC |
| `deploy/.env.example` | modified | `CORS_ORIGINS` |
| `deploy/docker-compose.yml` | modified | CORS + JWT trên gateway |
| `apps/mobile/lib/core/api_config.dart` | modified | Prefer gateway `:8080` |
| `docs/architecture.md` | modified | Gateway responsibilities |
| `docs/prd.md` | modified | `[DONE] T9.1.1` |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_gateway_routing_cors_jwt_02082026/` | added | Workdoc này |

## Cách verify

1. Unit tests:

```bash
go test ./services/api-gateway/ -count=1
```

2. Manual (gateway + auth chạy):

```bash
# CORS preflight
curl -si -X OPTIONS http://127.0.0.1:8080/v1/auth/otp/request \
  -H "Origin: http://localhost:54321" \
  -H "Access-Control-Request-Method: POST"

# Proxy auth
curl -s -X POST http://127.0.0.1:8080/v1/auth/otp/request \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"0901234567\"}"

# Orders không token — 401
curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:8080/v1/orders
```

## Ghi chú / blocker

- Next unfinished PRD trong US-9.1: **T9.1.2** Rate limit OTP / login / place-order.
- Next unfinished toàn PRD (sau E9 tasks còn lại / US-9.2): T9.1.2 cũng là candidate tiếp theo trong cùng story.
