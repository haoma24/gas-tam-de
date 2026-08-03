# Customer shop home + hồ sơ + admin redirect theo role

- **Thư mục:** `docs/workdocs_customer_shop_profile_03082026`
- **Ngày:** 03/08/2026
- **Loại:** feature
- **Liên quan:** UX Home / US-1.1 OTP / US-1.2 Admin / US-2.2 xem sản phẩm

## Mục tiêu

Sau OTP, khách vào trang cửa hàng mang cảm giác brand (không nhảy thẳng form đặt hàng tối giản). Có màn hồ sơ cá nhân riêng. Bỏ CTA «Dành cho cửa hàng» trên Home; session `role=admin` tự điều hướng `/admin`.

## Phạm vi

- Trong scope:
  - Landing khách (guest) + shop shell sau OTP
  - Hồ sơ: SĐT ẩn, tên (`GET/PATCH /v1/me`), đơn của tôi, đăng xuất
  - Auto-redirect admin; ẩn nút admin trên Home (login admin vẫn qua `/admin/login`)
- Ngoài scope:
  - Đổi backend OTP để trả role admin theo SĐT
  - Redesign toàn bộ admin desk widgets
  - Persist theme / dark mode

## Quyết định chính

- OTP verify → `/` (shop) thay vì `/order`; đặt hàng vẫn từ CTA trên shop.
- Guest Home giữ CTA «Đặt giao gas» / đăng nhập OTP; không hiện cửa hàng.
- Admin vào bằng deep link `/#/admin/login`; nếu đã có session admin thì mọi lần mở `/` → `/admin`.
- Typography: `google_fonts` Be Vietnam Pro (hỗ trợ tiếng Việt).

## Đã làm

- [x] Workdocs + CHANGESLOG
- [x] Guest landing bỏ nút admin
- [x] Customer shop shell (hero brand + catalogue + CTA)
- [x] Profile page
- [x] Router: OTP → shop; admin redirect
- [x] Verify analyze / test

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/home/home_page.dart` | modified | Guest landing |
| `apps/mobile/lib/features/home/customer_shop_page.dart` | added | Shop sau OTP |
| `apps/mobile/lib/features/auth/customer_profile_page.dart` | added | Hồ sơ |
| `apps/mobile/lib/features/dashboard/admin_dashboard_page.dart` | modified | Ẩn back khi đã login |
| `apps/mobile/lib/main.dart` | modified | Routes + redirect + font |
| `apps/mobile/pubspec.yaml` | modified | google_fonts |
| `apps/mobile/README.md` | modified | DX / verify |
| `CHANGESLOG.md` | modified | Entry |

## Cách verify

1. Guest: mở `/` — thấy brand + CTA, **không** có «Dành cho cửa hàng».
2. OTP xong → shop (hero + sản phẩm), bottom nav Hồ sơ / Đơn.
3. Hồ sơ: sửa tên, lưu, đăng xuất.
4. `/#/admin/login` → đăng nhập admin → `/admin`; refresh `/` vẫn về admin.
5. `flutter analyze` / `flutter test` dưới `apps/mobile`.

## Ghi chú / blocker

- Admin không còn entry trên Home; nhân viên dùng bookmark `/#/admin/login`.
