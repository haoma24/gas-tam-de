# Flutter Order Desk polling báo đơn mới (T5.1.4)

- **Thư mục:** `workdocs_flutter_order_desk_polling_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.1 / Sprint 3 / T5.1.4

## Mục tiêu

CCH đang mở Order Desk được báo / thấy đơn mới mà không cần bấm tải lại thủ công (PRD Should: Polling hoặc SSE/NATS bridge).

## Phạm vi

- Trong scope:
  - Poll định kỳ `GET /v1/admin/orders` trên `AdminOrdersPage`
  - Giữ pull-to-refresh + nút AppBar refresh
  - SnackBar khi xuất hiện order id mới (sau lần load đầu)
  - Pause poll khi app vào background; resume + fetch khi foreground
- Ngoài scope:
  - SSE / WebSocket / NATS→SSE bridge (ghi nhận là hướng tương lai)
  - Deep-link maps / nút Dẫn đường (T5.2.x)
  - Push notification native
  - Sound / badge ngoài SnackBar

## Quyết định chính

- **Polling** (10s) thay SSE/NATS bridge cho MVP: cùng hành vi trên Flutter Web + Android/iOS, không cần gateway streaming hay consumer JetStream phía client.
- Silent poll: không bật full-page spinner; lỗi transient không xóa list đã có.
- So sánh set `order.id` để đếm đơn mới (ổn định hơn so count).
- Future: có thể thay poll bằng NATS bridge (subscribe `order.placed` → SSE) mà vẫn giữ cùng list API.

## Đã làm

- [x] `Timer.periodic` + lifecycle pause/resume trên Order Desk
- [x] Silent refresh + SnackBar «Có N đơn mới»
- [x] Empty state kéo-refresh + copy nhắc chu kỳ
- [x] README verify T5.1.4
- [x] Mark `[DONE] T5.1.4` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/admin_orders_page.dart` | modified | poll 10s + SnackBar |
| `apps/mobile/README.md` | modified | verify bước 5 |
| `docs/prd.md` | modified | `[DONE] T5.1.4` |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_flutter_order_desk_polling_02082026/` | added | this folder |

## Cách verify

1. Chạy order-service + Flutter (xem `apps/mobile/README.md` — Order Desk).
2. Mở `/admin/orders`, để màn hình mở.
3. Tạo thêm đơn PENDING (client khác / API).
4. Trong ~10s: list cập nhật + SnackBar đơn mới; kéo refresh vẫn hoạt động.

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — logic poll thuần Dart/`Timer`.
- Next candidate: **T5.2.1** Lấy lat/lng đơn (US-5.2 Dẫn đường).
