# Flutter Order Desk UI (T5.1.3)

- **Thư mục:** `docs/workdocs_flutter_order_desk_ui_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.1 / Sprint 3 / T5.1.3

## Mục tiêu

CCH xem danh sách đơn chờ giao trên Flutter admin desk: cột STT | Tên | SĐT (masked) | Địa chỉ | km | Thời gian đặt, sort FIFO (cũ nhất trước), gọi API `GET /v1/admin/orders` đã có (T5.1.1/T5.1.2).

## Phạm vi

- Trong scope:
  - Flutter client list admin orders + model `AdminOrder` (`stt`, …)
  - Màn Order Desk list + pull-to-refresh
  - Link từ `/admin` → `/admin/orders`
  - Chi tiết đọc-only đơn giản từ payload list (không gọi `GET /admin/orders/{id}` — vẫn stub)
- Ngoài scope:
  - Polling / SSE / NATS bridge (T5.1.4)
  - Deep-link Google Maps / nút Dẫn đường (T5.2.x)
  - Hoàn tất giao / thanh toán
  - Gateway reverse-proxy thật (local trỏ `:8084`)

## Quyết định chính

- Style khớp `AdminProductsPage` (Material 3, surface tiles, refresh).
- Default status API = `PENDING` (omit query) — khớp desk «đơn chờ giao».
- Chi tiết dùng `extra: AdminOrder` trên go_router — không phụ thuộc detail API.
- SĐT = `phone_masked` từ API (không plaintext).

## Đã làm

- [x] `AdminOrder` + `OrderApi.listAdminOrders`
- [x] `AdminOrdersPage` (list FIFO columns + empty/error/refresh)
- [x] `AdminOrderDetailPage` (read-only từ list payload)
- [x] Routes `/admin/orders`, `/admin/orders/detail` + tile admin home
- [x] README verify + `ApiConfig` note
- [x] Mark `[DONE] T5.1.3` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/order_models.dart` | modified | `AdminOrder` |
| `apps/mobile/lib/features/order/order_api.dart` | modified | `listAdminOrders` |
| `apps/mobile/lib/features/order/admin_orders_page.dart` | added | desk list + detail |
| `apps/mobile/lib/main.dart` | modified | routes + nav tile |
| `apps/mobile/lib/core/api_config.dart` | modified | Order Desk note |
| `apps/mobile/README.md` | modified | verify T5.1.3 |
| `docs/prd.md` | modified | `[DONE] T5.1.3` |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_flutter_order_desk_ui_02082026/` | added | this folder |

## Cách verify

1. `go run ./services/order-service` (có ≥1 đơn PENDING nếu muốn thấy list).
2. Flutter (có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

3. Home → cửa hàng → login (nếu cần) → **Order Desk** (hoặc `/admin/orders`).
4. Kỳ vọng: STT tăng theo FIFO; cột tên / SĐT / địa chỉ / km / thời gian; tap → chi tiết.

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — UI theo style admin products / delivery fee.
- Next candidate: **T5.1.4** (Should) Polling hoặc SSE/NATS bridge báo đơn mới.
