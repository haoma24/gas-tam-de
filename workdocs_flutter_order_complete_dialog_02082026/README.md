# Flutter dialog hoàn tất đơn (T6.1.4)

- **Thư mục:** `workdocs_flutter_order_complete_dialog_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-6.1 / Sprint / T6.1.4

## Mục tiêu

Admin (CCH) trên Order Desk chi tiết chọn đã thu đủ / một phần / nợ khi giao xong, gọi API complete (T6.1.1), rồi list PENDING tự cập nhật.

## Phạm vi

- Trong scope:
  - Models + `OrderApi.completeOrder` → `POST /v1/admin/orders/{id}/complete`
  - Dialog FULL / PARTIAL / UNPAID (+ `amount_paid` khi PARTIAL)
  - Nút «Hoàn tất» trên `AdminOrderDetailPage`
  - Sau success: SnackBar + quay về Order Desk (list reload, đơn biến mất khỏi PENDING)
- Ngoài scope:
  - UI list công nợ (T6.2.x)
  - Thay đổi backend complete / billing / events

## Quyết định chính

- Dialog `AlertDialog` Material 3 khớp style admin products confirm.
- Radio: Đã thu đủ / Thu một phần / Chưa thu (nợ); preview «Công nợ dự kiến».
- PARTIAL validate local: `0 < paid < total` trước khi gọi API (khớp backend).
- `AdminOrderDetailPage` → `ConsumerWidget`; `onCompleted` navigate `/admin/orders` (page mới → `_load`).
- Không làm debt list UI.

## Đã làm

- [x] `CompleteOrderRequest` / `CompletedOrder` / `OrderPaymentType`
- [x] `OrderApi.completeOrder` + map lỗi `ORDER_ALREADY_COMPLETED` / …
- [x] Dialog + nút «Hoàn tất» trên chi tiết đơn
- [x] Wire `onCompleted` trong `main.dart`
- [x] Mark `[DONE] T6.1.4` trên PRD + CHANGESLOG + README verify

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/order_models.dart` | modified | complete request/response |
| `apps/mobile/lib/features/order/order_api.dart` | modified | `completeOrder` + error messages |
| `apps/mobile/lib/features/order/admin_orders_page.dart` | modified | dialog + nút Hoàn tất |
| `apps/mobile/lib/main.dart` | modified | `onCompleted` → desk list |
| `apps/mobile/README.md` | modified | verify T6.1.4 |
| `docs/prd.md` | modified | `[DONE] T6.1.4` |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_flutter_order_complete_dialog_02082026/` | added | this folder |

## Cách verify

1. Admin login → Order Desk → mở chi tiết đơn PENDING.
2. Bấm **Hoàn tất** → chọn **Thu một phần**, nhập `amount_paid` (vd. 100000 khi total 450000) → **Xác nhận**.
3. Kỳ vọng: SnackBar thu/nợ; quay về list; đơn không còn trong PENDING.
4. FULL / UNPAID tương tự; PARTIAL sai (0 / ≥ total) → lỗi local, không gọi API.

## Ghi chú / blocker

- Flutter có thể không có trên PATH — verify thủ công khi có SDK.
- Local: `API_BASE_URL` trỏ order-service (`:8084`) + Bearer admin JWT.
- Next candidate: **T6.2.1** API list/aggregate debts.
