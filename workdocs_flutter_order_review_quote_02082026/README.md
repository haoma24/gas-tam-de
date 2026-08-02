# Flutter review: hiển thị quote phí giao (T4.2.2)

- **Thư mục:** `workdocs_flutter_order_review_quote_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-4.2 / Sprint 2 / T4.2.2

## Mục tiêu

Thay stub phí giao = 0 trên màn xác nhận đơn bằng báo giá live từ `POST /v1/orders/quote`: khoảng cách, phí, tạm tính, tổng — trước khi khách đặt.

## Phạm vi

- Trong scope:
  - Flutter models + `OrderApi.quoteOrder`
  - `OrderReviewPage`: load quote khi mở, refresh, hiển thị distance/fee/totals
  - Re-quote trước place; chặn đặt khi `in_range=false` hoặc quote lỗi
- Ngoài scope:
  - E5 admin order desk
  - Đổi backend quote / place order

## Quyết định chính

- Dùng response quote (`subtotal` / `delivery_fee` / `total` / `distance_km`) thay vì cộng fee stub ở client.
- Ngoài bán kính: vẫn hiện preview phí (API 200 + `in_range=false`); disable nút Đặt đơn + message rõ.
- Re-quote ngay trước place để khớp fee engine server (admin có thể sửa bậc giữa lúc review).

## Đã làm

- [x] `QuoteOrderRequest` + `OrderQuote` trong `order_models.dart`
- [x] `OrderApi.quoteOrder` + reuse customer identity headers
- [x] `OrderReviewPage` load/display quote; bỏ hint stub E4
- [x] Mark `[DONE] T4.2.2` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/order_models.dart` | modified | quote request/response |
| `apps/mobile/lib/features/order/order_api.dart` | modified | `quoteOrder` |
| `apps/mobile/lib/features/order/order_review_page.dart` | modified | live quote UI |
| `apps/mobile/lib/core/api_config.dart` | modified | comment quote route |
| `docs/prd.md` | modified | `[DONE] T4.2.2` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. Chạy geo + catalog + order-service; Flutter với `--dart-define=API_BASE_URL=http://127.0.0.1:8084` (sau OTP/auth tùy flow).
2. Flow đặt hàng → chọn SP + địa chỉ trong phạm vi → màn Xác nhận đơn.
3. Kỳ vọng: Thanh toán hiện khoảng cách (km), tạm tính, phí giao (theo bậc / 0 nếu tắt), tổng = tạm + phí.
4. Refresh icon tải lại quote; ngoài bán kính → cảnh báo + nút Đặt đơn disabled.
5. Đặt đơn thành công → success screen totals khớp server.

## Ghi chú / blocker

- Flutter có thể không có trên PATH — không chạy `flutter analyze` trong session này.
- Next unfinished PRD: **T5.1.1** API list orders (admin), sort `created_at ASC` (E5).
