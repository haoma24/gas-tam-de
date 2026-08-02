# T7.2.1 — Lưu cost tại thời điểm xuất/bán (snapshot)

- **Thư mục:** `docs/workdocs_inventory_cogs_snapshot_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-7.2 / T7.2.1 / Sprint 4

## Mục tiêu

Đảm bảo mọi phiếu **OUT** (xuất tay admin + bán qua `order.completed`) lưu **giá vốn (COGS) tại thời điểm xuất** trên `stock_movements.unit_cost`, không phụ thuộc `stock_items.cost_price` sau này — nền tảng cho T7.2.2 profit report.

## Phạm vi

- Trong scope:
  - Snapshot `unit_cost` trên mọi OUT (admin API + order.completed consumer)
  - Bỏ qua `unit_cost` client gửi kèm OUT
  - Guard: insert OUT bắt buộc có `unit_cost`
  - Tests freeze sau khi IN đổi giá nhập
  - Docs architecture + mark PRD DONE
- Ngoài scope:
  - Công thức profit / report-service (T7.2.2)
  - FIFO/LIFO layer costing (MVP: giá nhập hiện tại tại lúc xuất)

## Quyết định chính

- **Nguồn snapshot:** `stock_items.cost_price` tại thời điểm ghi OUT (= giá nhập hiện tại sau IN gần nhất).
- **Hai đường xuất:** `POST /v1/admin/inventory` `OUT` và consumer `inventory-order-completed` đều dùng `snapshotOUTCost`.
- **Immutability:** movement append-only; IN/ADJUST sau chỉ đổi `cost_price` hiện tại.
- **ADJUST** không bắt buộc snapshot COGS (không phải xuất/bán).

## Đã làm

- [x] Audit: OUT admin + order.completed đã snapshot từ T7.1.2 / T7.1.3
- [x] Helper `snapshotOUTCost` + ignore client `unit_cost` trên OUT
- [x] `insertMovementTx` reject OUT thiếu `unit_cost`
- [x] Tests: ignore client cost, freeze sau later IN (API + order.completed)
- [x] Sync `schema.sql` comment + `docs/architecture.md` COGS contract
- [x] Mark `[DONE] T7.2.1` trong `docs/prd.md`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/inventory-service/stock_api.go` | modified | snapshot helper, OUT ignore client cost, insert guard |
| `services/inventory-service/order_completed.go` | modified | dùng `snapshotOUTCost` |
| `services/inventory-service/stock_api_test.go` | modified | ignore client + freeze tests |
| `services/inventory-service/order_completed_test.go` | modified | freeze ORDER OUT after later IN |
| `services/inventory-service/schema.sql` | modified | comment T7.2.1 |
| `docs/architecture.md` | modified | COGS snapshot contract |
| `docs/prd.md` | modified | `[DONE] T7.2.1` |

## Cách verify

```powershell
go test ./services/inventory-service/ -count=1
```

1. IN `unit_cost=100000` → OUT → `movement.unit_cost=100000`
2. IN lại `unit_cost=200000` → OUT cũ vẫn `100000`; OUT mới = `200000`
3. Complete đơn → `stock_movements` `ref_type=ORDER` có `unit_cost` = cost lúc bán

## Ghi chú / blocker

- Phần lớn logic đã có từ T7.1.2/T7.1.3; T7.2.1 chốt contract + harden + tests immutability cho report.
