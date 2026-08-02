# Schema products + product_prices (T2.1.2)

- **Thư mục:** `docs/workdocs_catalog_products_schema_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.2

## Mục tiêu

Chốt schema catalog SQLite: `products` + lịch sử giá (PRD `product_prices` ≡ `product_price_history`) — indexes, constraints, comments, migrate-on-start, tests assert schema. Phần lớn table đã có từ T2.1.1; task này siết gap.

## Phạm vi

- Trong scope:
  - Siết `schema.sql`: CHECK `sale_price >= 0`, `active IN (0,1)`, FK history → products
  - Index `idx_products_active`, `idx_price_history_product`
  - Tests columns/indexes + constraint/FK
  - Sync architecture §6.2; mark PRD `[DONE] T2.1.2`
- Ngoài scope:
  - T2.1.3 Event `catalog.product.updated`
  - T2.1.4 Flutter admin UI
  - T2.2.1 public list active products
  - Đổi CRUD API / đổi tên bảng sang `product_prices`

## Quyết định chính

- Giữ tên `product_price_history` (architecture §6.2 + code T2.1.1); PRD `product_prices` là equivalent naming.
- Giá hiện tại trên `products.sale_price`; history append-only khi tạo/đổi giá.
- `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` — migrate embed tại process start (pattern auth/catalog).

## Đã làm

- [x] Siết schema comments + CHECK + FK + indexes
- [x] Tests `TestMigrateCreatesCatalogSchema` + `TestProductsSchemaConstraints`
- [x] Sync `docs/architecture.md` §6.2
- [x] Mark `[DONE] T2.1.2` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/catalog-service/schema.sql` | modified | CHECK/FK/indexes/comments |
| `services/catalog-service/products_test.go` | modified | Schema + constraint tests |
| `docs/architecture.md` | modified | §6.2 sync indexes/constraints |
| `docs/prd.md` | modified | `[DONE] T2.1.2` |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_catalog_products_schema_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/catalog-service/ -count=1`
2. Confirm PRD: `- [DONE] T2.1.2 Schema products, product_prices`

## Ghi chú / blocker

- DB file đã tồn tại từ trước không nhận CHECK/FK mới qua `IF NOT EXISTS` (chỉ indexes mới được thêm); DB mới / test temp đủ contract.
- Next unfinished: **T2.1.3** Event `catalog.product.updated`.
