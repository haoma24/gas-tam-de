# Event catalog.product.updated (T2.1.3)

- **Thư mục:** `docs/workdocs_catalog_product_updated_event_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-2.1 / Sprint 2 / T2.1.3

## Mục tiêu

Publish event `catalog.product.updated` lên NATS JetStream khi admin tạo / sửa / ẩn sản phẩm, để inventory & report đồng bộ tên/SKU/giá/`active` (architecture §5.1).

## Phạm vi

- Trong scope:
  - Publish sau create + patch thành công (ẩn = `active=false`)
  - Envelope chuẩn (`event_id`, `subject`, `occurred_at`, `schema_version`, payload)
  - Payload: `product_id`, `sku`, `sale_price`, `active`
  - Helper `natsx.PublishEnvelope` + wire catalog-service → JetStream
  - Tests: mock recorder + embedded JetStream
- Ngoài scope:
  - T2.1.4 Flutter admin UI
  - Consumers (inventory / report)
  - Outbox / transactional publish

## Quyết định chính

- Subject dùng constant `events.CatalogProductUpdated` (`catalog.product.updated`).
- Publish **sau** commit DB; lỗi publish chỉ log (không 500) để tránh client retry tạo trùng SKU.
- `MsgId` = `event_id` để JetStream dedupe trong cửa sổ `Duplicates`.
- Startup catalog: `ConnectJS` + `EnsureStreams` (cần NATS local/compose); unit tests inject mock/`noop`.

## Đã làm

- [x] `pkg/natsx.PublishEnvelope`
- [x] `jsProductPublisher` + hook create/patch
- [x] Mock + embedded JetStream tests
- [x] Mark `[DONE] T2.1.3` trên `docs/prd.md`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `pkg/natsx/publish.go` | added | PublishEnvelope + MsgId |
| `pkg/natsx/publish_test.go` | added | embed + validation |
| `services/catalog-service/main.go` | modified | NATS connect / EnsureStreams |
| `services/catalog-service/product_events.go` | added | publisher interface + JS impl |
| `services/catalog-service/product_events_test.go` | added | mock + embed publish |
| `services/catalog-service/products.go` | modified | bus field + publish hooks |
| `services/catalog-service/products_test.go` | modified | noop bus in router helper |
| `docs/prd.md` | modified | `[DONE] T2.1.3` |
| `CHANGESLOG.md` | modified | entry T2.1.3 |
| `docs/workdocs_catalog_product_updated_event_02082026/README.md` | added | this file |

## Cách verify

1. `go test ./pkg/natsx/... ./services/catalog-service/... -count=1`
2. NATS up (`docker compose` hoặc local) + `go run ./cmd/nats-init` (optional — catalog tự EnsureStreams)
3. `go run ./services/catalog-service`
4. Create/patch product rồi subscribe:

```bash
# ví dụ với nats CLI nếu có
nats sub "catalog.product.updated"
curl -s -X POST http://127.0.0.1:8082/v1/admin/products \
  -H "Content-Type: application/json" \
  -d '{"sku":"GAS12","name":"Gas 12kg","sale_price":450000}'
```

## Ghi chú / blocker

- Catalog service giờ **phụ thuộc NATS** lúc start (compose đã inject `NATS_URL`).
- Next unfinished PRD: **T2.1.4** Flutter admin: màn sản phẩm.
