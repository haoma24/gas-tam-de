# API list products active (T2.2.1)

- **Thư mục:** `docs/workdocs_catalog_list_active_products_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-2.2 / Sprint 2 / T2.2.1

## Mục tiêu

Khách (public hoặc đã đăng nhập) có thể lấy danh sách sản phẩm đang bán (`active=1`) qua `GET /v1/products` để dùng khi đặt giao gas.

## Phạm vi

- Trong scope:
  - `GET /v1/products` trên catalog-service — chỉ `active = 1`
  - Response `{ "items": [...] }` cùng shape product với admin
  - Unit tests (empty / filter inactive / admin vẫn thấy ẩn)
  - Mark `[DONE] T2.2.1` trên PRD
- Ngoài scope:
  - T2.2.2 Flutter bước chọn SP trong flow đặt hàng
  - Reverse-proxy thật trên gateway (vẫn stub 501; path public đã đúng)
  - Admin CRUD / events (đã DONE T2.1.x)

## Quyết định chính

- Filter tại SQL `WHERE active = 1 ORDER BY created_at DESC` (index `idx_products_active` sẵn có).
- Không yêu cầu JWT trên catalog; gateway đã mount `/v1/products` ngoài nhóm auth → public + authenticated đều dùng được khi proxy live.
- Tái dùng `collectProducts` / `scanProduct` với admin list để tránh drift shape JSON.

## Đã làm

- [x] Thay stub `GET /v1/products` bằng `handleListActiveProducts`
- [x] Refactor list admin dùng `collectProducts`
- [x] Unit test public list filters inactive
- [x] Mark `[DONE] T2.2.1` trên `docs/prd.md`
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/catalog-service/main.go` | modified | mount handler active list |
| `services/catalog-service/products.go` | modified | `handleListActiveProducts` + `collectProducts` |
| `services/catalog-service/products_test.go` | modified | `TestListActiveProductsPublic` |
| `docs/prd.md` | modified | `[DONE] T2.2.1` |
| `CHANGESLOG.md` | modified | entry T2.2.1 |
| `docs/workdocs_catalog_list_active_products_02082026/README.md` | added | this file |

## Cách verify

1. `go test ./services/catalog-service/ -count=1`
2. Chạy service: `go run ./services/catalog-service`
3. Seed active + inactive qua admin, rồi:

```bash
curl -s http://127.0.0.1:8082/v1/products
# chỉ items có "active": true
```

## Ghi chú / blocker

- Gateway vẫn stub catalog → gọi trực tiếp `:8082` cho tới khi proxy wiring.
- Next unfinished PRD: **T2.2.2** Flutter bước chọn SP trong flow đặt hàng.
