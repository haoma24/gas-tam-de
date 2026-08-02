# CRUD APIs catalog (T2.1.1)

- **Thư mục:** `docs/workdocs_catalog_crud_apis_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-2.1 / Sprint 2 / T2.1.1

## Mục tiêu

Admin quản lý sản phẩm và giá bán qua HTTP APIs trên `catalog-service` (thêm / sửa / ẩn), dùng schema SQLite hiện có.

## Phạm vi

- Trong scope:
  - `GET/POST /v1/admin/products`
  - `GET/PATCH /v1/admin/products/{id}`
  - Migrate `schema.sql` lúc start
  - Ghi `product_price_history` khi tạo / đổi `sale_price`
  - Unit tests
- Ngoài scope:
  - T2.1.2 schema `product_prices` (đặt tên / refine riêng)
  - T2.1.3 event `catalog.product.updated`
  - T2.1.4 Flutter admin UI
  - T2.2.1 public list `active` (`GET /v1/products` vẫn stub rỗng)
  - Reverse-proxy thật trên gateway (vẫn stub 501)

## Quyết định chính

- Paths theo architecture §4.4 trên **catalog-service** (`:8082`). Gateway `/v1/admin/*` vẫn stub → gọi trực tiếp catalog cho tới khi proxy split.
- Không hard-delete: ẩn bằng `PATCH { "active": false }` (story «thêm/sửa/ẩn»).
- Giá bán nằm trên `products.sale_price`; lịch sử giá dùng `product_price_history` sẵn có (không tạo bảng `product_prices` — để T2.1.2).
- `changed_by` lấy từ header `X-User-Id` (gateway sẽ forward khi proxy).

## Đã làm

- [x] Migrate embed `schema.sql` trong `main.go`
- [x] Admin list / create / get / patch products
- [x] Price history on create + price change
- [x] Unit tests (happy path, validation, SKU conflict, 404)
- [x] Mark `[DONE] T2.1.1` trên `docs/prd.md`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/catalog-service/main.go` | modified | migrate + mount handlers |
| `services/catalog-service/products.go` | added | CRUD handlers |
| `services/catalog-service/products_test.go` | added | unit tests |
| `services/catalog-service/schema.sql` | modified | comment migrate / T2.1.2 note |
| `docs/prd.md` | modified | `[DONE] T2.1.1` |
| `CHANGESLOG.md` | modified | entry T2.1.1 |
| `docs/workdocs_catalog_crud_apis_02082026/README.md` | added | this file |

## Cách verify

1. `go test ./services/catalog-service/ -count=1`
2. Chạy service: `go run ./services/catalog-service` (DB `data/catalog.db`)
3. Create:

```bash
curl -s -X POST http://127.0.0.1:8082/v1/admin/products \
  -H "Content-Type: application/json" \
  -H "X-User-Id: admin-dev" \
  -d '{"sku":"GAS12","name":"Gas 12kg","sale_price":450000}'
```

4. List / patch:

```bash
curl -s http://127.0.0.1:8082/v1/admin/products
curl -s -X PATCH http://127.0.0.1:8082/v1/admin/products/<id> \
  -H "Content-Type: application/json" \
  -d '{"sale_price":460000,"active":false}'
```

## Ghi chú / blocker

- Gateway chưa proxy catalog admin; RBAC chỉ áp dụng khi đi qua gateway sau khi split upstream.
- Next unfinished PRD: **T2.1.2** Schema `products`, `product_prices`.
