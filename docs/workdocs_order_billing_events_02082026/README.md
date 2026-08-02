# Events order.completed + billing.* (T6.1.3)

- **Thư mục:** `docs/workdocs_order_billing_events_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-6.1 / T6.1.3 / PRD E6

## Mục tiêu

Publish JetStream events sau hoàn tất đơn / ghi thanh toán để inventory, report (và consumer billing sau này) cập nhật bất đồng bộ theo architecture §5.1.

## Phạm vi

- Trong scope:
  - `order.completed` từ order-service sau complete thành công
  - `billing.payment.recorded` + `billing.debt.updated` từ billing-service sau ghi payment (không republish khi idempotent)
  - Wire billing-service → NATS (`ConnectJS` + `EnsureStreams`)
  - Unit/integration tests (recording bus + embedded JetStream)
- Ngoài scope:
  - T6.1.4 Flutter dialog hoàn tất
  - Consumer inventory trừ tồn (T7.1.3)
  - Report consumer

## Quyết định chính

- Subject constants đã có trong `pkg/events` (`OrderCompleted`, `BillingPaymentRecorded`, `BillingDebtUpdated`).
- Stream subjects `order.>` / `billing.>` đã có trong `natsx.DomainStreams` — không cần đổi `nats-init`.
- Payload theo architecture §5.1:
  - `order.completed`: `order_id`, `items[]` (`product_id`/`qty`/`unit_price`/`sku`), `total`, `payment_type`, `amount_paid`
  - `billing.payment.recorded`: `order_id`, `amount_paid`, `payment_type`
  - `billing.debt.updated`: `customer_key`, `balance` (kể cả FULL balance=0)
- Publish **sau** commit DB; lỗi publish chỉ log (không 500).
- Mở rộng `orderPublisher` (thay `orderPlacedPublisher`) thêm `PublishOrderCompleted`.
- Idempotent payment replay: không republish events.

## Đã làm

- [x] `jsOrderPublisher.PublishOrderCompleted` + hook `handleCompleteOrder`
- [x] `jsBillingPublisher` + hook `recordPayment` (payment + debt)
- [x] Billing main: ConnectJS / EnsureStreams
- [x] Tests recording + embedded JetStream
- [x] Mark `- [DONE] T6.1.3` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/order_events.go` | modified | `orderPublisher` + `order.completed` |
| `services/order-service/order_events_test.go` | modified | complete publish + JS test |
| `services/order-service/complete_order.go` | modified | publish sau complete |
| `services/order-service/create_order.go` | modified | bus type `orderPublisher` |
| `services/order-service/create_order_test.go` | modified | bus type |
| `services/billing-service/billing_events.go` | added | publisher interface + JS |
| `services/billing-service/billing_events_test.go` | added | record + JS envelope tests |
| `services/billing-service/record_payment.go` | modified | publish sau commit |
| `services/billing-service/record_payment_test.go` | modified | noop / with-bus helpers |
| `services/billing-service/main.go` | modified | NATS wire |
| `docs/prd.md` | modified | `[DONE] T6.1.3` |
| `CHANGESLOG.md` | modified | entry mới |

## Cách verify

1. `go test ./services/order-service/ ./services/billing-service/ -count=1`
2. NATS up + services order/billing; admin complete một đơn PENDING
3. Subscribe:
   ```text
   nats sub "order.completed"
   nats sub "billing.payment.recorded"
   nats sub "billing.debt.updated"
   ```

## Ghi chú / blocker

- Next unfinished: **T6.1.4** Flutter dialog hoàn tất.
