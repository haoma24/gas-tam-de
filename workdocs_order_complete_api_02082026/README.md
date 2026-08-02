# API hoàn tất đơn + payment payload (T6.1.1)

- **Thư mục:** `workdocs_order_complete_api_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-6.1 / Sprint / T6.1.1

## Mục tiêu

Admin hoàn tất giao hàng qua `POST /v1/admin/orders/{id}/complete` kèm `payment_type` (FULL / PARTIAL / UNPAID) và `amount_paid` (khi cần), chuyển đơn `PENDING` → `COMPLETED`, trả settlement (paid / debt) theo PRD M6.

## Phạm vi

- Trong scope:
  - API complete + validate payment payload
  - Persist `status`, `completed_at`, `payment_type`, `amount_paid` trên `orders`
  - Unit tests (PARTIAL AC debt, FULL, UNPAID, validation, 404/409)
  - Gateway RBAC: path đã nằm dưới `/v1/admin/*`; bổ sung assert trong `rbac_test`
- Ngoài scope:
  - Billing ghi `payments` / `debts` (T6.1.2)
  - Events `order.completed`, `billing.payment.recorded`, `billing.debt.updated` (T6.1.3)
  - Flutter dialog hoàn tất (T6.1.4)

## Quyết định chính

- Path canonical: `/v1/admin/orders/{id}/complete` (khớp architecture §4.4; PRD rút gọn `/orders/{id}/complete`).
- `payment_type`: `FULL` | `PARTIAL` | `UNPAID` (không dùng tên DEBT — unpaid = nợ full).
- Rules: FULL → paid=total, debt=0; PARTIAL → `0 < paid < total`, debt=total−paid; UNPAID → paid=0, debt=total.
- Snapshot payment trên order để T6.1.3 đọc khi publish event; billing.db vẫn do T6.1.2.
- Trust gateway RBAC `/v1/admin/*`; order-service không parse JWT.
- Already COMPLETED / CANCELLED → 409 (status machine).

## Đã làm

- [x] `handleCompleteOrder` + `settlePayment`
- [x] Schema `payment_type`, `amount_paid` trên `orders`
- [x] Wire route (thay stub 501)
- [x] Tests complete + gateway RBAC complete path
- [x] Mark `[DONE] T6.1.1` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/complete_order.go` | added | handler + settle |
| `services/order-service/complete_order_test.go` | added | T6.1.1 tests |
| `services/order-service/schema.sql` | modified | payment columns |
| `services/order-service/main.go` | modified | wire handler; drop stub |
| `services/order-service/create_order_test.go` | modified | mount complete route |
| `services/api-gateway/rbac_test.go` | modified | admin complete RBAC |
| `docs/prd.md` | modified | `[DONE] T6.1.1` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/order-service/ ./services/api-gateway/ -count=1`
2. Với order-service + đơn PENDING:

```bash
curl -s -X POST "http://127.0.0.1:8084/v1/admin/orders/<id>/complete" \
  -H "Content-Type: application/json" \
  -d '{"payment_type":"PARTIAL","amount_paid":100000}'
```

3. Kỳ vọng: `status=COMPLETED`, `debt = total - amount_paid`; lần 2 → 409.

## Ghi chú / blocker

- Next candidate: **T6.1.2** Billing ghi `payments` + cập nhật `debts`.
- Gateway reverse-proxy upstream vẫn stub — gọi thẳng order-service `:8084` khi dev local.
