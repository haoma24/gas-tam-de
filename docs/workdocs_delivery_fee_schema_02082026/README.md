# Schema delivery_fee_settings + delivery_fee_rules (T4.1.1)

- **Thư mục:** `docs/workdocs_delivery_fee_schema_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.1

## Mục tiêu

Chốt schema SQLite order.db cho phí giao: toggle singleton `delivery_fee_settings` + bậc khoảng cách `delivery_fee_rules`; migrate-on-start; seed local theo ví dụ architecture; tests assert schema/constraints/seed. Không làm admin API hay fee engine.

## Phạm vi

- Trong scope:
  - Siết `schema.sql`: CHECK enabled/active/fee/min_km/max_km, index rules
  - Seed idempotent (settings + 3 bậc 0–5 / 5–10 / 10–∞)
  - Wire seed trong `order-service` main
  - Unit tests schema + seed
  - Sync architecture §6.4; mark PRD `[DONE] T4.1.1`
- Ngoài scope:
  - T4.1.2 Admin APIs cấu hình phí
  - T4.1.3 Engine tính phí khi preview/place order
  - T4.1.4 Flutter admin UI
  - T4.2.x Quote preview API / Flutter

## Quyết định chính

- Delivery fee sống trong **order-service** / `order.db` (architecture §6.4), không billing-service.
- Singleton settings id = `default`; `enabled` mặc định seed = 0 (fee off — khớp stub place-order hiện tại).
- Bậc seed theo bảng ví dụ architecture: 10k / 20k / 30k VND.
- Seed không ghi đè khi settings đã tồn tại (`DELIVERY_FEE_SEED=0` tắt seed).

## Đã làm

- [x] Siết schema comments + CHECK + index
- [x] `delivery_fee.go` seed + env `DELIVERY_FEE_*`
- [x] Wire main migrate → seed
- [x] Tests migrate / constraints / seed / idempotent
- [x] Sync `docs/architecture.md` §6.4
- [x] Mark `[DONE] T4.1.1` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/schema.sql` | modified | CHECK/index/comments fee tables |
| `services/order-service/delivery_fee.go` | added | seed settings + rules |
| `services/order-service/delivery_fee_test.go` | added | schema + seed tests |
| `services/order-service/main.go` | modified | seed on start |
| `deploy/.env.example` | modified | `DELIVERY_FEE_*` |
| `docs/architecture.md` | modified | §6.4 sync constraints |
| `docs/prd.md` | modified | `[DONE] T4.1.1` |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_delivery_fee_schema_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/order-service/ -count=1 -run DeliveryFee`
2. Confirm PRD: `- [DONE] T4.1.1 Schema delivery_fee_settings, delivery_fee_rules`
3. Chạy order-service lần đầu → log `delivery fee seeded`; restart → `already exists`

## Ghi chú / blocker

- DB file cũ đã tạo trước khi thêm CHECK: `CREATE TABLE IF NOT EXISTS` không áp CHECK mới — DB mới / test temp đủ contract (giống catalog T2.1.2).
- Next unfinished: **T4.1.2** Admin APIs cấu hình phí.
