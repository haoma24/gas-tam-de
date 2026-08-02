# Schema stock + movements + cost (T7.1.1)

- **Thư mục:** `docs/workdocs_inventory_stock_schema_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint / US-7.1 / T7.1.1

## Mục tiêu

Chốt schema SQLite `inventory.db`: `stock_items` (tồn + giá vốn), `stock_movements` (nhập/xuất/điều chỉnh), `processed_events` (idempotency consumer sau này); migrate-on-start; seed empty defaults; tests assert schema/constraints.

## Phạm vi

- Trong scope:
  - Siết `schema.sql`: CHECK cost/reorder/qty/movement_type/unit_cost; comments; index movements
  - Migrate embed tại process start
  - Seed empty (không hardcode SP — chờ catalog sync / IN API)
  - Unit tests schema + constraints + seed
  - Sync architecture §6.5; mark PRD `[DONE] T7.1.1`
- Ngoài scope:
  - T7.1.2 APIs nhập/xuất/điều chỉnh
  - T7.1.3 Consumer `order.completed` trừ tồn
  - T7.1.4 Flutter màn tồn kho
  - T7.2.x snapshot COGS / profit formula

## Quyết định chính

- Schema khớp architecture §6.5; bổ sung CHECK (`cost_price >= 0`, `reorder_level >= 0`, `qty > 0`, `unit_cost` null-or-`>= 0`).
- **MVP cho phép `on_hand` âm** (architecture: low_stock + admin xử lý tay) — không CHECK `on_hand >= 0`.
- Seed **empty**: không insert product giả; `INVENTORY_SEED=0` chỉ tắt log readiness.
- `processed_events` tạo sẵn trong schema (dùng ở T7.1.3), không implement consumer ở task này.

## Đã làm

- [x] Siết `schema.sql` comments + CHECK + index
- [x] `stock.go` seed empty + env `INVENTORY_SEED`
- [x] Wire `main` migrate → seed
- [x] Tests migrate / constraints / seed / idempotent
- [x] Sync `docs/architecture.md` §6.5
- [x] Mark `[DONE] T7.1.1` trên PRD
- [x] `deploy/.env.example` `INVENTORY_SEED`
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/inventory-service/schema.sql` | modified | CHECK/comments |
| `services/inventory-service/main.go` | modified | embed migrate + seed |
| `services/inventory-service/stock.go` | added | empty seed |
| `services/inventory-service/stock_test.go` | added | schema + seed tests |
| `deploy/.env.example` | modified | `INVENTORY_SEED` |
| `docs/architecture.md` | modified | §6.5 sync constraints |
| `docs/prd.md` | modified | `[DONE] T7.1.1` |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_inventory_stock_schema_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/inventory-service/ -count=1`
2. Confirm PRD: `- [DONE] T7.1.1 Schema stock + movements + cost`
3. Chạy inventory-service lần đầu → log `inventory ready` với counts 0

## Ghi chú / blocker

- DB file cũ tạo trước khi thêm CHECK: `CREATE TABLE IF NOT EXISTS` không áp CHECK mới — DB mới / test temp đủ contract (giống catalog T2.1.2).
- Next unfinished: **T7.1.2** APIs nhập/xuất/điều chỉnh.
