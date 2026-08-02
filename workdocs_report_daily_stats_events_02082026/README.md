# T8.1.1 — report-service subscribe events → daily_stats

- **Thư mục:** `workdocs_report_daily_stats_events_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-8.1 / T8.1.1 / Sprint 5

## Mục tiêu

report-service consume JetStream order events và upsert read-model `daily_stats` (doanh thu, COGS, phí ship, số đơn, profit theo ngày VN) để phục vụ dashboard.

## Phạm vi

- Trong scope:
  - Durable consumers `report-order-placed`, `report-order-completed`
  - Upsert `daily_stats` + idempotency `processed_events`
  - Wire NATS + migrate schema trong `main`
  - Unit / JetStream tests
  - Sync architecture §5.4 / §6.7; mark `[DONE] T8.1.1`
- Ngoài scope:
  - API `GET /v1/admin/dashboard/summary` (T8.1.2)
  - Flutter dashboard widgets (T8.1.3)
  - Aggregate `billing.debt.updated` → `dashboard_snapshot.debt_total` (cần read-model per-customer; để T8.1.2)

## Quyết định chính

- **Ngày:** `daily_stats.day` = `YYYY-MM-DD` theo `Asia/Ho_Chi_Minh` từ `created_at` / `completed_at` / `occurred_at`.
- **`order.placed`:** chỉ `orders_placed++`.
- **`order.completed`:** cộng revenue (Σ qty×unit_price), COGS (Σ qty×unit_cost optional), delivery_fee, `orders_completed++`, `profit = revenue − cogs`.
- **Fee:** ưu tiên `delivery_fee` trong payload; nếu thiếu derive `max(0, total − revenue)`.
- **COGS:** đọc `unit_cost` trên item nếu publisher gửi; thiếu → 0 (không cross-query inventory.db).
- **Billing events:** chưa ghi `daily_stats` (không có cột công nợ ngày); snapshot debt để T8.1.2.

## Đã làm

- [x] `daily_stats.go` — DayKeyVN, upsert delta, processed_events helpers
- [x] `order_stats.go` — consumers + handlers
- [x] `main.go` — migrate, NATS, start consumers
- [x] `order_stats_test.go` — place/complete/idempotent/JetStream
- [x] Architecture + PRD `[DONE] T8.1.1`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/report-service/daily_stats.go` | added | upsert + day VN |
| `services/report-service/order_stats.go` | added | JetStream consumers |
| `services/report-service/order_stats_test.go` | added | tests |
| `services/report-service/main.go` | modified | NATS + migrate |
| `docs/architecture.md` | modified | §5.4 / §6.7 T8.1.1 |
| `docs/prd.md` | modified | `[DONE] T8.1.1` |
| `CHANGESLOG.md` | modified | entry mới |

## Cách verify

```powershell
go test ./services/report-service/ -count=1
```

1. Publish `order.placed` → `orders_placed` ngày VN = 1
2. Publish `order.completed` (có `unit_price` + optional `unit_cost`) → revenue/cogs/fee/profit cộng dồn
3. Republish cùng `event_id` → không double-count

## Ghi chú / blocker

- Publisher hiện tại (`order-service`) chưa gửi `unit_cost` / `delivery_fee` trên `order.completed`; fee vẫn derive được từ `total − revenue`. COGS chính xác cần enrichment payload hoặc event inventory riêng (future).
- `billing.debt.updated` / `dashboard_snapshot` sẽ làm cùng T8.1.2 khi expose summary API.
