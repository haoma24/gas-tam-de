# Middleware RBAC trên gateway (T1.2.4)

- **Thư mục:** `workdocs_gateway_rbac_middleware_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature / security
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.4

## Mục tiêu

Gateway validate JWT access token (cùng `JWT_SECRET` / claims với auth-service) và enforce RBAC: `role=admin` cho `/v1/admin/**`, `role=customer` cho orders + `POST /geo/check`.

## Phạm vi

- Trong scope:
  - Parse/verify HS256 access JWT (`sub`, `role`, `sid`, issuer `gas-tam-de-auth`)
  - Middleware `RequireJWT` + `RequireRole`
  - Wire route groups trên `api-gateway`
  - Unit tests authn/authz
- Ngoài scope:
  - Reverse-proxy thật tới upstream (vẫn stub 501)
  - Full E9 (CORS, rate limit, audit) — T9.1.x
  - Refactor claims sang shared `pkg/` (auth-service giữ issuer riêng)

## Quyết định chính

- Validate JWT tại gateway (local secret), không RPC auth-service mỗi request.
- Public: `/healthz`, `/v1/hello`, `/v1/auth/*`, catalog browse, `GET /geo/store|search`.
- Customer-only: `/v1/orders`, `/v1/orders/*`, `POST /v1/geo/check`.
- Admin-only: `/v1/admin/*`.
- Forward `X-User-Id` / `X-User-Role` / `X-Session-Id` (+ optional `X-Phone-Masked`) cho proxy sau này.

## Đã làm

- [x] `jwt.go` — parse Bearer + claims
- [x] `rbac.go` — RequireJWT / RequireRole + context
- [x] Wire groups trong `main.go`
- [x] Unit tests (401/403/public/admin/customer)
- [x] Mark `- [DONE] T1.2.4` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/api-gateway/jwt.go` | added | Parse access JWT |
| `services/api-gateway/rbac.go` | added | Middleware RBAC |
| `services/api-gateway/rbac_test.go` | added | Unit tests |
| `services/api-gateway/main.go` | modified | Route groups + JWT secret |
| `docs/prd.md` | modified | `[DONE] T1.2.4` |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_gateway_rbac_middleware_02082026/` | added | Workdoc này |

## Cách verify

1. Unit tests:

```bash
go test ./services/api-gateway/ -count=1
```

2. Manual (gateway chạy, token từ auth-service):

```bash
# Public — OK
curl -s http://127.0.0.1:8080/v1/hello

# Orders không token — 401 UNAUTHORIZED
curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:8080/v1/orders

# Admin token trên /v1/admin/* — 501 stub (auth OK)
# Customer token trên /v1/admin/* — 403 FORBIDDEN
```

## Ghi chú / blocker

- Upstream vẫn `proxyStub` 501; RBAC chặn trước stub.
- Next unfinished PRD: **T2.1.1** CRUD APIs catalog.
