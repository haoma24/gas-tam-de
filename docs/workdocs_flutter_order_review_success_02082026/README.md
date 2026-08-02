# Flutter: review + success place order (T3.3.3)

- **Thư mục:** `docs/workdocs_flutter_order_review_success_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.3

## Mục tiêu

Khách xác nhận đơn trên Flutter (giỏ + địa chỉ + tổng), gọi `POST /v1/orders` với JWT, rồi thấy màn thành công — hoàn tất luồng đặt giao gas phía client (PII mask sâu = T3.3.4).

## Phạm vi

- Trong scope:
  - `OrderApi.createOrder` → `POST /v1/orders`
  - Màn review: cart lines, địa chỉ + geo hint, tên người nhận, tạm tính / phí stub 0 / tổng
  - Place order → clear cart → màn success (mã đơn, tổng từ API)
  - Wire `/order/address` → `/order/review` → `/order/success`
- Ngoài scope:
  - Mask PII nâng cao (T3.3.4) — chỉ hiện `phone_masked` API trả về
  - Quote / fee engine (E4 / T4.2.x) — phí hiển thị 0 khớp stub server
  - Gateway reverse-proxy thật (local: `API_BASE_URL` → order `:8084` + `X-User-*` từ session)

## Quyết định chính

- Style Material 3 khớp address / select-products (amber seed, `surfaceContainerLowest`).
- Local order-service cần `X-User-Id` / `X-User-Role` / `X-Phone-Masked` (gateway stub chưa proxy) → OrderApi gắn từ `authSessionProvider` kèm Bearer JWT.
- Phí giao preview = 0 (cùng `computeDeliveryFeeStub`); success dùng `delivery_fee` / `total` từ response.
- Success nhận `PlacedOrder` qua `go_router` `extra`; deep-link không có extra → về `/`.

## Đã làm

- [x] `order_models.dart` + `order_api.dart`
- [x] `OrderReviewPage` + `OrderSuccessPage`
- [x] Routes `/order/review`, `/order/success`; address continue → review
- [x] README + `ApiConfig` note `:8084`
- [x] Mark `[DONE] T3.3.3` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/order_models.dart` | added | Request/response models |
| `apps/mobile/lib/features/order/order_api.dart` | added | POST /v1/orders + errors VN |
| `apps/mobile/lib/features/order/order_review_page.dart` | added | Review + place |
| `apps/mobile/lib/features/order/order_success_page.dart` | added | Success UI |
| `apps/mobile/lib/main.dart` | modified | Wire routes |
| `apps/mobile/lib/core/api_config.dart` | modified | order `:8084` |
| `apps/mobile/README.md` | modified | flow + verify T3.3.3 |
| `docs/prd.md` | modified | T3.3.3 DONE |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_flutter_order_review_success_02082026/` | added | this folder |

## Cách verify

1. Chạy catalog `:8082`, geo `:8083` (đã seed store + SP active), order `:8084` (+ NATS).
2. Flutter (cần SDK):

```powershell
cd apps/mobile
flutter pub get
# Full E2E cần gateway proxy; hoặc OTP với :8081 rồi đổi base / mock session rồi:
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

3. Có cart + địa chỉ in-range → `/order/review` → nhập tên → **Đặt đơn** → success.
4. Out-of-range từ API → banner lỗi trên review.

## Ghi chú / blocker

- Máy agent có thể chưa có Flutter trên PATH — chưa `flutter analyze` / run tại đây.
- Next unfinished PRD: **T3.3.4** Mask PII trong response.
