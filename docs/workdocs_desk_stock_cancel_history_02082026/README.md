# Order desk wait badge / TTS interval / stock reserve / hủy đơn / lịch sử

- **Thư mục:** `docs/workdocs_desk_stock_cancel_history_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Order Desk / Inventory / Customer orders

## Mục tiêu

1. Badge thời gian chờ trên Order Desk (xanh/cam/đỏ) — ngưỡng chỉnh trong admin.
2. TTS tiếng Việt, interval báo «Bạn có N đơn chưa giao» — bật/tắt + interval trong admin.
3. Thêm SP → bắt buộc phiếu nhập; tồn 0 = «Tạm hết hàng»; đặt đơn trừ tồn; hủy hoàn tồn; complete giữ đã trừ.
4. Khách xem lịch sử đơn + hủy PENDING.

## Phạm vi

- Trong scope: desk-settings API, reserve/release inventory HTTP, cancel order, Flutter UX.
- Ngoài scope: SSE realtime, multi-warehouse.

## Quyết định chính

- Trừ tồn **lúc place** (HTTP reserve sync); **complete** không trừ thêm; **cancel** release.
- `order.completed` consumer: bỏ qua nếu đã có movement `ref_id=order_id` (idempotent / tránh double).
- Desk settings lưu order-service SQLite singleton.

## Đã làm

- [x] Backend desk-settings + cancel + reserve/release
- [x] Flutter desk badge/TTS/settings
- [x] Product inbound + stock UI + history/cancel
- [x] CHANGESLOG

## Cách verify

1. Desk: badge đổi màu theo phút chờ; TTS VI mỗi N phút khi bật.
2. Tạo SP → nhập tồn → khách thấy còn hàng; đặt → tồn giảm; hủy → tồn tăng.
3. `/orders/history` list + hủy PENDING.
