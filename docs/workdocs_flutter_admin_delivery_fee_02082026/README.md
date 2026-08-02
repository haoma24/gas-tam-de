# Flutter admin: màn phí giao hàng (T4.1.4)

- **Thư mục:** `docs/workdocs_flutter_admin_delivery_fee_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.4 / architecture §4.4

## Mục tiêu

CCH (admin) bật/tắt phí ship và chỉnh bậc khoảng cách trên Flutter (Web + Android + iOS), gọi `GET/PUT /v1/admin/delivery-fee` (T4.1.2).

## Phạm vi

- Trong scope:
  - Models + API client admin delivery fee (Bearer JWT qua Dio session)
  - Màn toggle `enabled` + edit bands (`min_km` / `max_km` / `fee_vnd` / `active`)
  - Wire từ desk `/admin` → `/admin/delivery-fee`
  - Material 3 style khớp admin products / login
- Ngoài scope:
  - T4.2.x customer quote / hiển thị phí trên review
  - Gateway reverse-proxy order admin (local gọi thẳng `:8084`)

## Quyết định chính

- Tái dùng Dio + `authSessionProvider` (Authorization Bearer nếu đã login).
- Local: `API_BASE_URL=http://127.0.0.1:8084` cho order-service admin fee.
- Toggle `enabled` → PUT chỉ `enabled` ngay; **Lưu bậc** → PUT `enabled` + full `rules`.
- `max_km` trống = open-ended (`null`); half-open `[min, max)` khớp backend.
- Validate local overlap trước khi gọi API; lỗi `INVALID_RULES` hiện tiếng Việt.
- Route flat: `/admin/delivery-fee`.

## Đã làm

- [x] `delivery_fee_models` / `delivery_fee_api` + `DeliveryFeeApiException`
- [x] `AdminDeliveryFeePage` (toggle, edit bands, add/remove, save)
- [x] Admin desk tile **Phí giao hàng** + route
- [x] README mobile + `ApiConfig` notes
- [x] Mark T4.1.4 DONE trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/delivery_fee_models.dart` | added | Config + Rule |
| `apps/mobile/lib/features/order/delivery_fee_api.dart` | added | GET/PUT client |
| `apps/mobile/lib/features/order/admin_delivery_fee_page.dart` | added | admin UI |
| `apps/mobile/lib/main.dart` | modified | route + desk tile |
| `apps/mobile/lib/core/api_config.dart` | modified | order admin fee note |
| `apps/mobile/README.md` | modified | verify delivery fee |
| `docs/prd.md` | modified | T4.1.4 DONE |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_flutter_admin_delivery_fee_02082026/` | added | this folder |

## Cách verify

1. Order: `go run ./services/order-service` (port 8084, seed fee).
2. Flutter (có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

3. Mở `/admin/delivery-fee` (hoặc Home → cửa hàng → login nếu cần → **Phí giao hàng**).
4. Switch bật/tắt → snackbar; sửa bậc → **Lưu bậc**.

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — UI theo style admin products.
- Gateway `/v1/admin/*` chưa split upstream order → E2E login+fee qua một `API_BASE_URL` cần proxy sau; order-service upstream không tự check JWT.
- Next unfinished PRD: **T4.2.1** API quote: distance + fee + total.
