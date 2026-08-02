# API list orders admin FIFO (T5.1.1)

- **Thư mục:** `workdocs_order_admin_list_fifo_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.1 / Sprint 3 / T5.1.1

## Mục tiêu

Admin Order Desk cần API danh sách đơn theo FIFO (`created_at ASC`) — đơn cũ nhất trước — để CCH biết giao ai trước. Path dưới `/v1/admin/orders` (gateway RBAC `role=admin`).

## Phạm vi

- Trong scope:
  - `GET /v1/admin/orders` — list theo `ORDER BY created_at ASC`
  - Optional `?status=` (`PENDING` | `COMPLETED` | `CANCELLED`); mặc định `PENDING` (đơn chờ giao)
  - Response basic fields đã có trên `orders` + `order_items` (id, customer_name, phone_masked, address, km, totals, status, created_at, items)
- Ngoài scope:
  - Cột STT / làm giàu response desk (T5.1.2)
  - Flutter Order Desk UI (T5.1.3)
  - Polling / SSE / NATS bridge (T5.1.4)
  - Gateway reverse-proxy thật (RBAC `/v1/admin/*` đã có; upstream vẫn stub)

## Quyết định chính

- Default `status=PENDING` khi omit query — khớp story “đơn chờ giao”.
- Sort luôn `created_at ASC` (FIFO); index `idx_orders_admin_fifo` đã có từ schema.
- Reuse `orderView` / `customerOrderView` (phone masked); chưa expose full phone (desk cột SĐT đầy đủ → T5.1.2 nếu cần).
- Trust gateway RBAC giống admin delivery-fee (không parse JWT trong order-service).

## Đã làm

- [x] `handleListAdminOrders` + parse status filter
- [x] Wire `GET /v1/admin/orders` (thay stub `notImplemented`)
- [x] Unit tests FIFO, status filter, invalid status
- [x] Mark `[DONE] T5.1.1` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/list_orders.go` | modified | admin list + helpers |
| `services/order-service/list_admin_orders_test.go` | added | FIFO / filter tests |
| `services/order-service/main.go` | modified | wire handler |
| `services/order-service/create_order_test.go` | modified | mount admin route in test router |
| `docs/prd.md` | modified | `[DONE] T5.1.1` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/order-service/ -count=1`
2. Với order-service chạy + DB có ≥2 PENDING:

```bash
curl -s "http://127.0.0.1:8084/v1/admin/orders"
curl -s "http://127.0.0.1:8084/v1/admin/orders?status=COMPLETED"
```

3. Kỳ vọng: PENDING list sắp xếp `created_at` tăng dần; đơn cũ đứng trước.

## Ghi chú / blocker

- Next candidate: **T5.1.2** Cột STT, tên, SĐT, địa chỉ, km, thời gian (làm giàu payload / contract desk).
- Gateway proxy upstream vẫn 501 stub — gọi thẳng order-service `:8084` khi dev local.
