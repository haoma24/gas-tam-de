# T8.1.2 — API dashboard summary

- **Thư mục:** `workdocs_report_dashboard_summary_api_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-8.1 / T8.1.2 / Sprint 5 / architecture §4.4 / §6.7

## Mục tiêu

Admin có HTTP API đọc tổng hợp dashboard từ `daily_stats` (doanh thu, COGS, phí ship, profit, số đơn) theo hôm nay / một ngày / khoảng ngày, kèm `debt_total` từ debt read-model.

## Phạm vi

- Trong scope:
  - `GET /v1/admin/dashboard/summary` trên report-service
  - Query `day` **hoặc** `from`+`to` (VN `YYYY-MM-DD`); mặc định = hôm nay
  - Durable consumer `report-billing-debt-updated` → `customer_debt_balances` + `dashboard_snapshot.debt_total`
  - Gateway RBAC assert path `/v1/admin/dashboard/summary`
  - Sync architecture §4.4 / §5.4 / §6.7; mark `[DONE] T8.1.2`
- Ngoài scope:
  - Flutter dashboard widgets (T8.1.3)
  - Tồn kho trong summary (dùng `GET /v1/admin/inventory`)
  - Gateway reverse-proxy thật tới report-service

## Quyết định chính

- **Nguồn kỳ:** SUM `daily_stats` trong khoảng; `profit = revenue − cogs` trên aggregate (không SUM cột profit để tránh lệch nếu schema đổi).
- **Timezone:** `Asia/Ho_Chi_Minh` (khớp `daily_stats.day`).
- **Công nợ:** event `billing.debt.updated` gửi balance tuyệt đối per `customer_key` → cần bảng `customer_debt_balances`; `debt_total = SUM(balance > 0)`.
- **Authz:** trust gateway `/v1/admin/*`; upstream không parse JWT.

## Đã làm

- [x] `dashboard_summary.go` — handler + parse range + SUM daily_stats
- [x] `debt_stats.go` — consumer + snapshot debt
- [x] Schema `customer_debt_balances` + comment snapshot
- [x] Wire route + consumer trong `main` / `startReportConsumers`
- [x] Unit tests summary + debt idempotent; gateway RBAC path
- [x] Architecture + PRD `[DONE] T8.1.2`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/report-service/dashboard_summary.go` | added | HTTP API |
| `services/report-service/dashboard_summary_test.go` | added | tests |
| `services/report-service/debt_stats.go` | added | billing debt consumer |
| `services/report-service/schema.sql` | modified | customer_debt_balances |
| `services/report-service/order_stats.go` | modified | start debt consumer |
| `services/report-service/main.go` | modified | wire handler |
| `services/api-gateway/rbac_test.go` | modified | dashboard path |
| `docs/architecture.md` | modified | §4.4 / §5.4 / §6.7 |
| `docs/prd.md` | modified | `[DONE] T8.1.2` |
| `CHANGESLOG.md` | modified | entry mới |

## Cách verify

```powershell
go test ./services/report-service/ ./services/api-gateway/ -count=1
```

1. Seed `daily_stats` hoặc publish order events →  
   `curl "http://127.0.0.1:8087/v1/admin/dashboard/summary?day=2026-08-02"`
2. Range:  
   `curl "http://127.0.0.1:8087/v1/admin/dashboard/summary?from=2026-08-01&to=2026-08-02"`
3. Publish `billing.debt.updated` → `debt_total` trong summary tăng tương ứng
4. Gateway (stub proxy): customer JWT → 403 trên `/v1/admin/dashboard/summary`

## Ghi chú / blocker

- Gateway vẫn `proxyStub` cho `/v1/admin/*` → E2E qua `:8080` trả 501 cho đến khi split upstream report.
- Cột `revenue_today` / `revenue_month` / `profit_month` trên `dashboard_snapshot` chưa refresh theo lịch (API tính live từ `daily_stats`); snapshot chủ yếu giữ `debt_total`.
