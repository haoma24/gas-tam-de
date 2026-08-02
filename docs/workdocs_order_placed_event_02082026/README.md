# Persist order + publish order.placed (T3.3.2)

- **Thư mục:** `docs/workdocs_order_placed_event_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.3 / Sprint 2 / T3.3.2

## Mục tiêu

Sau `POST /v1/orders` thành công: bảo đảm order + items được persist chắc chắn (schema/index), và publish event `order.placed` lên NATS JetStream cho report (architecture §5.1).

## Phạm vi

- Trong scope:
  - Polish schema: CHECK non-negative money / qty≥1; index `created_at`, `order_items.product_id`
  - Publish `order.placed` sau commit (envelope chuẩn)
  - Payload: `order_id`, `total`, `distance_km`, `created_at`
  - Wire order-service → JetStream (`ConnectJS` + `EnsureStreams`)
  - Tests: mock recorder + embedded JetStream
- Ngoài scope:
  - T3.3.3 Flutter review/success UI
  - T3.3.4 PII masking nâng cao
  - Consumers (report)
  - Outbox / transactional publish

## Quyết định chính

- Subject dùng `events.OrderPlaced` (`order.placed`).
- Publish **sau** commit DB; lỗi publish chỉ log (không 500) — đơn đã tạo, client không nên retry tạo trùng.
- Pattern publisher giống catalog `catalog.product.updated` (`orderPlacedPublisher` + `jsOrderPublisher` + `noop`).
- `MsgId` = `event_id` qua `natsx.PublishEnvelope`.

## Đã làm

- [x] Schema CHECK + indexes bổ sung
- [x] `jsOrderPublisher` + hook sau create order
- [x] Wire NATS trong `main.go`
- [x] Mock + embedded JetStream tests
- [x] Mark `[DONE] T3.3.2` trên `docs/prd.md`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/order_events.go` | added | publisher interface + JS impl |
| `services/order-service/order_events_test.go` | added | mock + embed publish |
| `services/order-service/create_order.go` | modified | bus field + publish after commit |
| `services/order-service/create_order_test.go` | modified | noop bus; assert persisted columns |
| `services/order-service/main.go` | modified | ConnectJS / EnsureStreams |
| `services/order-service/schema.sql` | modified | CHECK + indexes |
| `deploy/.env.example` | modified | comment NATS / T3.3.2 |
| `docs/prd.md` | modified | `[DONE] T3.3.2` |
| `CHANGESLOG.md` | modified | entry T3.3.2 |
| `docs/workdocs_order_placed_event_02082026/README.md` | added | this file |

## Cách verify

1. `go test ./services/order-service/... -count=1`
2. NATS up + `go run ./cmd/nats-init` (optional — order tự EnsureStreams)
3. `go run ./services/order-service`
4. Place order rồi subscribe:

```bash
nats sub "order.placed"
# POST /v1/orders qua gateway (customer JWT) hoặc gọi thẳng order-service với X-User-* headers
```

## Ghi chú / blocker

- Order-service giờ **phụ thuộc NATS** lúc start (compose đã inject `NATS_URL`).
- Next unfinished PRD: **T3.3.3** Flutter: review + success screen.
