# API list/aggregate debts (T6.2.1)

- **Thư mục:** `workdocs_billing_admin_debts_api_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-6.2 / T6.2.1 / PRD M6 / architecture §4.4 `GET /v1/admin/debts`

## Mục tiêu

Admin xem danh sách khách còn công nợ và tổng nợ còn lại (aggregate), phục vụ tab Công nợ / dashboard (UI = T6.2.2).

## Phạm vi

- Trong scope:
  - `GET /v1/admin/debts` trên billing-service
  - Response: `items[]` + `total_balance` + `count`
  - Chỉ `balance > 0`; sort `balance DESC`
  - Gateway admin RBAC coverage (path đã nằm dưới `/v1/admin/*`)
- Ngoài scope:
  - Flutter UI công nợ (T6.2.2)
  - Reverse-proxy thật gateway → billing (vẫn stub 501 như các admin route khác)
  - Ledger chi tiết / filter / pagination

## Quyết định chính

- Aggregate trên billing `debts` table (không qua report-service `dashboard_snapshot` — đó là E8).
- Ẩn `balance = 0` khỏi list (FULL không tạo nợ; khách đã trả hết không hiện).
- `phone_masked` trả về cho admin desk; không lộ `phone_e164`.

## Đã làm

- [x] `list_debts.go` — handler + query
- [x] Wire `GET /v1/admin/debts` (bỏ stub `notImplemented`)
- [x] Unit tests empty / multi-customer aggregate / omit zero
- [x] Gateway RBAC test path `/v1/admin/debts`
- [x] Mark `[DONE] T6.2.1` trên PRD; CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/billing-service/list_debts.go` | added | list + aggregate |
| `services/billing-service/list_debts_test.go` | added | unit/API tests |
| `services/billing-service/main.go` | modified | wire route; remove stub |
| `services/api-gateway/rbac_test.go` | modified | debts RBAC coverage |
| `docs/prd.md` | modified | `[DONE] T6.2.1` |
| `CHANGESLOG.md` | modified | entry mới |
| `workdocs_billing_admin_debts_api_02082026/` | added | workdoc này |

## Cách verify

1. Unit:

```bash
go test ./services/billing-service/ ./services/api-gateway/ -count=1
```

2. Manual (billing :8086):

```bash
# Sau khi có payment PARTIAL/UNPAID ghi debts:
curl -s http://127.0.0.1:8086/v1/admin/debts
# Kỳ vọng: {"items":[...],"total_balance":...,"count":...}
```

3. Gateway RBAC (proxy vẫn 501):

```bash
# Customer token → 403; admin token → 501 stub
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/v1/admin/debts \
  -H "Authorization: Bearer <token>"
```

## Ghi chú / blocker

- Next PRD unfinished: **T6.2.2** Flutter UI công nợ.
- Gateway reverse-proxy tới billing chưa wire (cùng pattern stub các `/v1/admin/*`).
