# Thông báo đơn mới (TTS) + nhớ tên/địa chỉ theo SĐT

- **Thư mục:** `docs/workdocs_order_alert_customer_profile_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Order Desk poll / US-3.x place order / auth users.full_name

## Mục tiêu

1. Admin Order Desk: khi có đơn mới → phát âm to «Bạn có đơn hàng mới» (+ SnackBar).
2. Khách lần đầu nhập tên → lưu DB; lần sau theo SĐT (JWT user) lấy lại tên + địa chỉ giao trước.

## Phạm vi

- Trong scope:
  - Flutter TTS / SpeechSynthesis trên desk poll
  - Auth `GET/PATCH /v1/me` (`full_name`)
  - Order `GET /v1/orders/me/defaults` (đơn gần nhất)
  - Prefill review name + nút dùng địa chỉ lần trước
- Ngoài scope: push notification native, SSE realtime

## Quyết định chính

- Tên nguồn chính: `auth.users.full_name` (PATCH khi đặt đơn).
- Địa chỉ lần trước: từ đơn gần nhất (`orders`), không dual-write tọa độ sang auth.
- TTS volume max; Web dùng `flutter_tts` (speechSynthesis).

## Đã làm

- [x] Backend me + defaults + gateway
- [x] Flutter TTS desk
- [x] Flutter prefill name/address
- [x] CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/` | added/modified | `/v1/me` |
| `services/order-service/` | added/modified | `/v1/orders/me/defaults` |
| `services/api-gateway/` | modified | route `/me` |
| `apps/mobile/` | modified | TTS + profile UX |

## Cách verify

1. Desk mở → tạo đơn từ tab khác → nghe «Bạn có đơn hàng mới».
2. Khách mới: nhập tên lần đầu → đặt đơn → logout/login OTP cùng SĐT → tên đã điền; địa chỉ có «Dùng địa chỉ lần trước».

## Ghi chú / blocker

- TTS cần user gesture lần đầu trên một số trình duyệt (desk đã tương tác thì ổn).
