# Flutter nút «Dẫn đường» chi tiết đơn (T5.2.3)

- **Thư mục:** `docs/workdocs_flutter_order_nav_button_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.2 / Sprint 3 / T5.2.3

## Mục tiêu

CCH mở chỉ đường từ vị trí hiện tại tới điểm giao ngay trên màn chi tiết đơn Order Desk — nút «Dẫn đường» gọi helper T5.2.2.

## Phạm vi

- Trong scope:
  - Nút «Dẫn đường» trên `AdminOrderDetailPage`
  - Gọi `openNavigationTo(order.lat, order.lng)`
  - SnackBar khi thiếu toạ độ hoặc launch Maps thất bại
- Ngoài scope:
  - E6 hoàn tất giao / thanh toán / công nợ
  - Thay đổi deep-link helper (đã xong T5.2.2)
  - API mới

## Quyết định chính

- Đặt nút sau khối địa chỉ / thời gian (trước danh sách SP) — gần context giao hàng.
- Coi `lat == 0 && lng == 0` là thiếu toạ độ (API null → `_asDouble` default `0`).
- Lỗi launch dùng `NavigationLaunchResult.errorMessage` từ helper (SnackBar floating, khớp Order Desk).

## Đã làm

- [x] `AdminOrderDetailPage`: `FilledButton.icon` «Dẫn đường» + guard coords
- [x] Cập nhật comment helper / README verify
- [x] Mark `[DONE] T5.2.3` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/admin_orders_page.dart` | modified | nút + handler |
| `apps/mobile/lib/features/order/navigation_link.dart` | modified | comment T5.2.3 wired |
| `apps/mobile/README.md` | modified | verify T5.2.3 |
| `docs/prd.md` | modified | `[DONE] T5.2.3` |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_flutter_order_nav_button_02082026/` | added | this folder |

## Cách verify

1. Admin → Order Desk → mở chi tiết đơn có lat/lng.
2. Bấm **Dẫn đường** → Maps directions tới điểm giao.
3. Đơn `0,0` / thiếu coords → SnackBar «Đơn không có toạ độ điểm giao.»

## Ghi chú / blocker

- Next candidate: **T6.1.1** API `POST /orders/{id}/complete` + payment payload (E6 — không làm trong task này).
