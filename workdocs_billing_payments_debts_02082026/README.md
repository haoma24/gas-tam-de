# Billing ghi payments + cập nhật debts (T6.1.2)

- **Thư mục:** `workdocs_billing_payments_debts_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-6.1 / T6.1.2 / PRD M6 / architecture §6.6

## Mục tiêu

Khi admin hoàn tất đơn, billing-service ghi `payments` và cập nhật `debts` / `debt_ledger` theo FULL / PARTIAL / UNPAID (debt delta = amount_due − amount_paid).

## Phạm vi

- Trong scope:
  - Migrate `billing.db` schema (`payments`, `debts`, `debt_ledger`, `processed_events`)
  - Core `recordPayment` + `POST /v1/internal/payments`
  - Order complete gọi billing sync sau khi order đã COMPLETED
  - Idempotent theo `order_id` (UNIQUE)
- Ngoài scope:
  - Events `order.completed` / `billing.payment.recorded` / `billing.debt.updated` (T6.1.3)
  - Flutter dialog hoàn tất (T6.1.4)
  - `GET /v1/admin/debts` aggregate UI (T6.2.x)

## Quyết định chính

- Sync HTTP từ order-service → billing (không chờ NATS consumer T6.1.3) để payments/debts có ngay sau complete.
- `customer_key` = `orders.phone_hash` (fallback `uid:<user_id>`).
- Lỗi billing chỉ log; response complete vẫn 200 (order đã commit; retry/eventual qua T6.1.3).
- FULL: debt delta 0 → không tạo `debts` / `debt_ledger` nếu chưa có nợ.

## Đã làm

- [x] Embed + migrate `schema.sql` trên billing-service
- [x] `recordPayment` + handler internal
- [x] HTTP client order-service + gọi sau complete
- [x] Tests PARTIAL AC (100k/450k → debt 350k), FULL, UNPAID, accumulate, idempotent
- [x] Mark `[DONE] T6.1.2` trên PRD; CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/billing-service/main.go` | modified | migrate + route internal |
| `services/billing-service/record_payment.go` | added | payments + debts |
| `services/billing-service/record_payment_test.go` | added | unit/API tests |
| `services/order-service/clients.go` | modified | billing HTTP client |
| `services/order-service/complete_order.go` | modified | call billing after complete |
| `services/order-service/create_order.go` | modified | `billing` field |
| `services/order-service/list_orders.go` | modified | load `phone_hash` |
| `services/order-service/main.go` | modified | `BILLING_SERVICE_URL` |
| `services/order-service/*_test.go` | modified | noop/stub billing |
| `deploy/docker-compose.yml` | modified | order→billing URL + depends |
| `deploy/.env.example` | modified | `BILLING_SERVICE_URL` |
| `docs/prd.md` | modified | `[DONE] T6.1.2` |

## Cách verify

1. Chạy billing + order (local hoặc compose).
2. Tạo đơn PENDING rồi:
   ```bash
   curl -s -X POST http://127.0.0.1:8084/v1/admin/orders/<id>/complete \
     -H "Content-Type: application/json" \
     -H "X-User-Id: admin-dev" \
     -d '{"payment_type":"PARTIAL","amount_paid":100000}'
   ```
3. Kiểm tra billing DB / internal replay:
   ```bash
   curl -s -X POST http://127.0.0.1:8086/v1/internal/payments \
     -H "Content-Type: application/json" \
     -d '{"order_id":"<id>","customer_key":"uid:u1","phone_masked":"090***1111","payment_type":"PARTIAL","amount_due":450000,"amount_paid":100000,"recorded_by":"admin-dev"}'
   ```
   Kỳ vọng `debt_delta=350000`, `idempotent=true` lần 2.
4. Unit: `go test ./services/billing-service/ ./services/order-service/ -count=1`

## Ghi chú / blocker

- `GET /v1/admin/debts` vẫn stub (T6.2.1).
- JetStream consumer `billing-order-completed` để dành T6.1.3.
