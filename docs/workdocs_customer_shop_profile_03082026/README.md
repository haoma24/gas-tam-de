# Customer shop home + hồ sơ + admin redirect theo role

- **Thư mục:** `docs/workdocs_customer_shop_profile_03082026`
- **Ngày:** 03/08/2026
- **Loại:** feature
- **Liên quan:** UX Home / US-1.1 OTP / US-1.2 Admin / US-2.2 xem sản phẩm

## Mục tiêu

Sau OTP, khách vào trang cửa hàng mang cảm giác brand. Có màn hồ sơ cá nhân riêng (gồm quản lý đơn). Guest Home chỉ còn **Đăng nhập**. Session `role=admin` tự điều hướng `/admin`.

## Phạm vi

- Trong scope:
  - Landing khách: một nút **Đăng nhập**
  - Shop shell sau OTP + hồ sơ (đơn hàng trong hồ sơ)
  - Auto-redirect admin; login admin qua `/admin/login`
- Ngoài scope:
  - Đổi backend OTP để trả role admin theo SĐT
  - Redesign toàn bộ admin desk widgets

## Quyết định chính

- Guest: chỉ «Đăng nhập» → OTP; không CTA đặt gas / đơn / admin trên Home.
- OTP verify → `/` (shop); đặt hàng từ CTA trên shop.
- «Đơn hàng của tôi» chỉ từ hồ sơ; shop bottom nav: Cửa hàng | Hồ sơ.
- Admin: deep link `/#/admin/login`; session admin mở `/` → `/admin`.
- Typography: `google_fonts` Be Vietnam Pro.

## Đã làm

- [x] Workdocs + CHANGESLOG
- [x] Guest landing chỉ **Đăng nhập**
- [x] Customer shop shell (hero brand + catalogue + CTA)
- [x] Profile page — gồm Đơn hàng của tôi
- [x] Shop nav bỏ tab Đơn
- [x] Router: OTP → shop; admin redirect
- [x] Verify analyze / test

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/home/home_page.dart` | modified | Guest chỉ Đăng nhập |
| `apps/mobile/lib/features/home/customer_shop_page.dart` | added/modified | Shop; nav 2 tab |
| `apps/mobile/lib/features/auth/customer_profile_page.dart` | added | Hồ sơ + đơn |
| `apps/mobile/lib/main.dart` | modified | Routes |
| `CHANGESLOG.md` | modified | Entry |

## Cách verify

1. Guest `/` — chỉ **Đăng nhập**.
2. OTP → shop; Hồ sơ → Đơn hàng của tôi.
3. `/#/admin/login` → `/admin`.
4. `flutter analyze` / `flutter test`.

## Ghi chú / blocker

- Admin không còn entry trên Home; dùng bookmark `/#/admin/login`.
