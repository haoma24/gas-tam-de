# T7.2.2 — Công thức profit cho report-service

- **Thư mục:** `workdocs_report_profit_formula_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-7.2 / T7.2.2 / Sprint 4

## Mục tiêu

Định nghĩa và implement công thức lợi nhuận MVP trong report-service để tái sử dụng khi aggregate `daily_stats` (T8.1.x): doanh thu hàng − COGS snapshot; phí ship tách riêng.

## Phạm vi

- Trong scope:
  - Helper thuần: `SumSaleRevenue`, `SumCOGS`, `ComputeProfit`, `BuildDailyStatsAmounts`, `DailyStatsFromTotals`, `ApplyProfit`
  - Unit tests (fee không trừ vào profit; multi-line; profit âm)
  - Comment schema `daily_stats.profit_vnd` + sync architecture §6.7
  - Mark `[DONE] T7.2.2` trong PRD
- Ngoài scope:
  - Subscribe events → `daily_stats` (T8.1.1)
  - API dashboard summary (T8.1.2)
  - Flutter dashboard (T8.1.3)

## Quyết định chính

- **Công thức:** `profit_vnd = revenue_vnd - cogs_vnd`.
- **Revenue:** Σ(qty × unit_price) — subtotal sản phẩm, không gồm delivery fee.
- **COGS:** Σ(qty × unit_cost) từ snapshot OUT/`ORDER` (T7.2.1), không dùng `cost_price` hiện tại.
- **Delivery fee:** ghi `delivery_fee_vnd` riêng; không trừ vào profit (architecture §6.7).

## Đã làm

- [x] `profit.go` — công thức + types map `daily_stats` money columns
- [x] `profit_test.go` — cover revenue/COGS/profit/fee separation
- [x] Comment `schema.sql` + architecture §6.7
- [x] Mark `[DONE] T7.2.2` trong `docs/prd.md`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/report-service/profit.go` | added | công thức profit tái sử dụng |
| `services/report-service/profit_test.go` | added | unit tests |
| `services/report-service/schema.sql` | modified | comment T7.2.2 trên cột money |
| `docs/architecture.md` | modified | §6.7 reference helper + T7.2.2 |
| `docs/prd.md` | modified | `[DONE] T7.2.2` |

## Cách verify

```powershell
go test ./services/report-service/ -count=1
```

1. Sale 300k, COGS 200k, fee 25k → `profit_vnd=100000` (fee không trừ)
2. Aggregate `ApplyProfit` sau khi cộng `revenue`/`cogs` → `profit = revenue - cogs`

## Ghi chú / blocker

- T8.1.1 sẽ gọi `BuildDailyStatsAmounts` / `ApplyProfit` khi consume `order.completed` (+ nguồn COGS từ inventory movements).
