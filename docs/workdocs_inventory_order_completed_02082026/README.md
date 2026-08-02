# Consumer order.completed trừ tồn (T7.1.3)

- **Thư mục:** `docs/workdocs_inventory_order_completed_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-7.1 / T7.1.3 / PRD M7 / architecture §5.1 §5.4 §6.5 §10

## Mục tiêu

Inventory-service subscribe JetStream `order.completed` và trừ tồn (OUT + snapshot COGS) khi admin hoàn tất đơn — không trừ khi `order.placed` (tránh giữ tồn ảo nếu hủy).

## Phạm vi

- Trong scope:
  - Durable consumer `inventory-order-completed` trên subject `order.completed`
  - Apply OUT per item (`ref_type=ORDER`, `ref_id=order_id`), snapshot `unit_cost = cost_price`
  - Idempotency qua `processed_events(event_id)`
  - Wire NATS trong `inventory-service` main
  - Unit + embedded JetStream tests
  - Sync architecture; mark PRD `[DONE] T7.1.3`
- Ngoài scope:
  - T7.1.4 Flutter màn tồn kho
  - T7.2.x profit formula / report
  - Publish `inventory.low_stock` / `inventory.stock.adjusted`
  - Consumer `order.placed` (cố ý không subscribe)

## Quyết định chính

- **Trừ khi complete** (PRD default + architecture §3/§10) — không trừ trên `order.placed`.
- Durable name khớp architecture §5.4: `inventory-order-completed`.
- Một transaction: tất cả OUT lines + insert `processed_events`; redelivery cùng `event_id` → no-op.
- SP chưa có trong `stock_items`: tạo placeholder (sku từ event, `cost_price=0`) rồi OUT → `on_hand` có thể âm (MVP).
- Lỗi xử lý → `Nak` để JetStream redeliver; thành công → `Ack`.

## Đã làm

- [x] `order_completed.go` — parse envelope, apply OUT, processed_events, start consumer
- [x] Wire `ConnectJS` + `EnsureStreams` + consumer trong `main.go`
- [x] Tests: deduct, idempotent, multi-line, missing→negative, JetStream e2e
- [x] Sync architecture §6.5; mark `[DONE] T7.1.3`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/inventory-service/order_completed.go` | added | consumer + apply |
| `services/inventory-service/order_completed_test.go` | added | unit + JS tests |
| `services/inventory-service/main.go` | modified | NATS + start consumer |
| `services/inventory-service/schema.sql` | modified | comment processed_events |
| `docs/architecture.md` | modified | §6.5 consumer detail |
| `docs/prd.md` | modified | `[DONE] T7.1.3` |
| `CHANGESLOG.md` | modified | entry mới |
| `docs/workdocs_inventory_order_completed_02082026/README.md` | added | workdoc này |

## Cách verify

1. Unit:

```bash
go test ./services/inventory-service/ -count=1
```

2. Manual (cần NATS + inventory + order):

```bash
# Nhập kho
curl -s -X POST http://127.0.0.1:8085/v1/admin/inventory \
  -H "Content-Type: application/json" \
  -d '{"movement_type":"IN","product_id":"p1","sku":"GAS12","name":"Gas 12kg","qty":10,"unit_cost":150000}'

# Complete một đơn (admin) → publish order.completed
# Kiểm tra tồn giảm:
curl -s http://127.0.0.1:8085/v1/admin/inventory
```

## Ghi chú / blocker

- Next unfinished: **T7.1.4** Flutter màn tồn kho.
- Catalog sync consumer (tên/SKU từ `catalog.product.updated`) vẫn chưa có — placeholder name = sku khi thiếu stock.
