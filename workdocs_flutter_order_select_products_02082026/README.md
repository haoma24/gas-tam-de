# Flutter: bước chọn SP trong flow đặt hàng (T2.2.2)

- **Thư mục:** `workdocs_flutter_order_select_products_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 2 / US-2.2 / T2.2.2 / architecture §8

## Mục tiêu

Khách (sau OTP) chọn sản phẩm đang bán trong flow đặt giao gas trên Flutter (Web + Android + iOS), lưu giỏ local cho bước địa chỉ / place-order sau.

## Phạm vi

- Trong scope:
  - `CatalogApi.listActiveProducts` → `GET /v1/products`
  - Màn chọn SP + stepper số lượng
  - Giỏ hàng in-memory (Riverpod) — qty / tổng tiền
  - Wire `/order` (thay placeholder); **Tiếp tục** → placeholder địa chỉ
- Ngoài scope:
  - Geo / map / radius (E3)
  - Place-order API / quote
  - Persist giỏ / secure storage
  - Gateway reverse-proxy catalog (local vẫn `:8082`)

## Quyết định chính

- Tái dùng `Product` + `formatVnd` + Dio/`catalogApiProvider` từ admin catalog.
- Cart `StateNotifier` giữ snapshot `Product` theo `id` — đủ cho E3/E4 sau.
- Style Material 3 khớp admin products / OTP (amber seed, tile `surfaceContainerLowest`).
- Không bắt JWT trên catalog list (public/authenticated theo T2.2.1); local override `API_BASE_URL=:8082`.

## Đã làm

- [x] `listActiveProducts` + refactor list helper trong `CatalogApi`
- [x] `OrderCart` / `orderCartProvider`
- [x] `SelectProductsPage` (load, +/- qty, footer tổng, Tiếp tục)
- [x] Routes `/order`, `/order/address` placeholder
- [x] README mobile + `ApiConfig` note
- [x] Mark T2.2.2 DONE trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/order_cart.dart` | added | Local cart state |
| `apps/mobile/lib/features/order/select_products_page.dart` | added | Chọn SP UI |
| `apps/mobile/lib/features/catalog/catalog_api.dart` | modified | `listActiveProducts` |
| `apps/mobile/lib/main.dart` | modified | Wire `/order` |
| `apps/mobile/lib/core/api_config.dart` | modified | Customer products note |
| `apps/mobile/README.md` | modified | Flow + verify |
| `docs/prd.md` | modified | T2.2.2 DONE |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_flutter_order_select_products_02082026/` | added | this folder |

## Cách verify

1. Catalog: `go run ./services/catalog-service` (port 8082); có ≥1 SP active.
2. Flutter:

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8082
```

3. Mở `/order` (hoặc Home → OTP → `/order`).
4. Thêm qty → footer hiện số lượng + tổng VND → **Tiếp tục** → placeholder địa chỉ.
5. Back về `/order` — giỏ vẫn giữ.

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH.
- E2E OTP+products qua một `API_BASE_URL` cần gateway proxy auth+catalog.
- Next unfinished PRD: **T3.1.1** Xin quyền location (Web / Android / iOS).
