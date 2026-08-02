# Flutter admin: màn sản phẩm (T2.1.4)

- **Thư mục:** `workdocs_flutter_admin_products_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.4 / architecture §4.4 + §8

## Mục tiêu

CCH (admin) quản lý sản phẩm trên Flutter (Web + Android + iOS): xem danh sách, thêm, sửa giá/thông tin, ẩn/hiện — gọi catalog admin CRUD đã có (T2.1.1).

## Phạm vi

- Trong scope:
  - `CatalogApi` → `GET/POST /v1/admin/products`, `GET/PATCH /v1/admin/products/{id}`
  - Màn list + form create/edit; toggle ẩn (`active=false`) / hiện
  - Wire từ desk `/admin` → `/admin/products`
  - Material 3 style khớp auth/home hiện có
- Ngoài scope:
  - T2.2.x customer product pick / public list active
  - Gateway reverse-proxy catalog admin (vẫn stub; local gọi thẳng `:8082`)
  - Persist token / auto-refresh
  - Upload ảnh (chỉ URL text)

## Quyết định chính

- Tái dùng Dio + `authSessionProvider` (Bearer nếu đã login); catalog-service không enforce JWT (authz ở gateway khi proxy).
- Local: `API_BASE_URL=http://127.0.0.1:8082` cho CRUD sản phẩm; login admin vẫn `:8081`.
- Không hard-delete — ẩn bằng `PATCH { "active": false }` (đúng story «thêm/sửa/ẩn»).
- Routes flat go_router: `/admin/products`, `/new`, `/:id`.

## Đã làm

- [x] `catalog_models` / `catalog_api` + `CatalogApiException`
- [x] `AdminProductsPage` (list, refresh, hide/show)
- [x] `AdminProductFormPage` (create + edit)
- [x] Admin desk tile **Sản phẩm** + routes
- [x] README mobile + `ApiConfig` notes
- [x] Mark T2.1.4 DONE trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/catalog/catalog_models.dart` | added | Product + formatVnd |
| `apps/mobile/lib/features/catalog/catalog_api.dart` | added | list/get/create/patch |
| `apps/mobile/lib/features/catalog/admin_products_page.dart` | added | list UI |
| `apps/mobile/lib/features/catalog/admin_product_form_page.dart` | added | create/edit form |
| `apps/mobile/lib/main.dart` | modified | routes + admin desk |
| `apps/mobile/lib/core/api_config.dart` | modified | catalog `:8082` note |
| `apps/mobile/README.md` | modified | verify products |
| `docs/prd.md` | modified | T2.1.4 DONE |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_flutter_admin_products_02082026/` | added | this folder |

## Cách verify

1. Catalog: `go run ./services/catalog-service` (port 8082).
2. Flutter (có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8082
```

3. Mở `/admin/products` (hoặc Home → cửa hàng → login nếu cần → **Sản phẩm**).
4. **Thêm** → SKU `GAS12`, tên `Gas 12kg`, giá `450000` → thấy trong list.
5. Tap item → sửa giá → **Lưu**; icon mắt → ẩn → chip «Đã ẩn».

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — UI theo style admin login / OTP.
- Gateway `/v1/admin/*` chưa split upstream catalog → E2E login+products qua một `API_BASE_URL` cần proxy sau.
- Next unfinished PRD: **T2.2.1** API list products `active` (public/authenticated).
