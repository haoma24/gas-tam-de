# Gateway audit log admin actions (T9.1.4)

- **Thư mục:** `workdocs_gateway_admin_audit_02082026`
- **Ngày:** 02/08/2026
- **Loại:** security
- **Liên quan:** US-9.1 / T9.1.4

## Mục tiêu

Ghi nhận hành động admin quan trọng tại API Gateway: **ai** (user id), **làm gì** (method + path), **khi nào**, **kết quả** (HTTP status) — phục vụ kiểm toán MVP.

## Phạm vi

- Trong scope:
  - Middleware audit trên nhóm route `/v1/admin/**` (sau JWT + role=admin)
  - Chỉ mutating: `POST` / `PUT` / `PATCH` / `DELETE`
  - Persist SQLite `gateway.db` bảng `admin_audit_logs` + structured slog `admin_audit`
  - Env / compose `GATEWAY_DB`; docs architecture + PRD DONE
- Ngoài scope:
  - T9.2.4 / T9.2.5 (Flutter CTA shell / platform checklist)
  - API đọc/query audit cho UI admin
  - Domain-level `auth.db.audit_logs` (giữ cho sau; gateway không cross-write)

## Quyết định chính

- Audit ở **gateway edge** (một chỗ cho mọi upstream) thay vì từng service.
- SQLite riêng `gateway.db` + slog — durable + ops-friendly; lỗi ghi DB chỉ log, không fail request.
- GET admin (list/dashboard) không audit để giảm nhiễu; mutating mới là “hành động”.

## Đã làm

- [x] `audit.go` — recorder interface, middleware, slog + SQLite + memory (test)
- [x] `schema.sql` — `admin_audit_logs`
- [x] Wire `main.go`; cập nhật test helpers
- [x] Unit tests mutating / skip GET / status / unauthorized / SQLite persist
- [x] `GATEWAY_DB` trong `.env.example` + docker-compose volume
- [x] architecture §4.2 / §6.8 / §7.2; mark `- [DONE] T9.1.4`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/api-gateway/audit.go` | added | Middleware + sinks |
| `services/api-gateway/audit_test.go` | added | Unit tests |
| `services/api-gateway/schema.sql` | added | `admin_audit_logs` |
| `services/api-gateway/main.go` | modified | Open DB, wire audit |
| `services/api-gateway/proxy_test.go` | modified | `testRouterWithAudit` |
| `deploy/.env.example` | modified | `GATEWAY_DB` |
| `deploy/docker-compose.yml` | modified | gateway volume + env |
| `docs/architecture.md` | modified | gateway.db + Audit row |
| `docs/prd.md` | modified | `[DONE] T9.1.4` |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_gateway_admin_audit_02082026/` | added | Workdoc này |

## Cách verify

1. Unit tests:

```bash
go test ./services/api-gateway/ -count=1
```

2. Manual (gateway chạy với `GATEWAY_DB=data/gateway.db`):

```bash
# Admin mutating → row trong admin_audit_logs + log admin_audit
# (dùng admin JWT hợp lệ)
curl -si -X POST http://127.0.0.1:8080/v1/admin/orders/<id>/complete \
  -H "Authorization: Bearer <admin_access_token>"
```

## Ghi chú / blocker

- Không implement T9.2.4 / T9.2.5.
