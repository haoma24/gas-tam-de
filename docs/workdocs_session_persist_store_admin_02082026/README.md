# Persist session + admin vị trí cửa hàng

- **Thư mục:** `docs/workdocs_session_persist_store_admin_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Auth session UX / T3.2.1 store settings / Admin desk

## Mục tiêu

1. Giữ phiên đăng nhập (khách OTP + admin) qua reload / mở lại app — không phải login lại mỗi lần.
2. Thêm màn admin cấu hình vị trí cửa hàng (lat/lng, bán kính, địa chỉ) gọi API geo đã có.

## Phạm vi

- Trong scope:
  - Persist tokens + user qua `shared_preferences` (Web + Android + iOS)
  - Bootstrap lúc mở app: load → refresh token nếu cần
  - Home CTA nhảy thẳng vào flow nếu đã có session đúng role
  - Đăng xuất admin (xóa session)
  - Flutter admin: GET `/v1/geo/store` + PUT `/v1/admin/geo/store`
- Ngoài scope:
  - `flutter_secure_storage` (có thể nâng cấp sau)
  - Auto-refresh interceptor mọi 401 (chỉ refresh lúc bootstrap)

## Quyết định chính

- `shared_preferences` đủ cho MVP local/dev; không chặn Web.
- Lưu `access_expires_at` tuyệt đối để quyết định refresh lúc bootstrap.
- Màn cửa hàng dùng public GET store (đủ field) + PUT admin; map + search geo tái dùng pattern địa chỉ đơn.

## Đã làm

- [x] Persist `AuthSession` + refresh lúc bootstrap
- [x] Wire Home / login / OTP / logout
- [x] `AdminStorePage` + desk tile + route
- [x] CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/pubspec.yaml` | modified | `shared_preferences` |
| `apps/mobile/lib/features/auth/*` | modified/added | session store + notifier + refresh |
| `apps/mobile/lib/main.dart` | modified | bootstrap, routes, CTA |
| `apps/mobile/lib/features/dashboard/admin_dashboard_page.dart` | modified | store tile + logout |
| `apps/mobile/lib/features/order/geo_*` + `admin_store_page.dart` | modified/added | store API + UI |

## Cách verify

1. Login admin → F5 / restart app → vẫn vào được `/admin` (CTA cửa hàng).
2. OTP khách → restart → **Đặt giao gas** vào thẳng chọn SP.
3. Desk → **Vị trí cửa hàng** → sửa lat/lng/bán kính → Lưu → `GET /v1/geo/store` thấy giá trị mới.
4. Đăng xuất → session hết → phải login lại.

## Ghi chú / blocker

- Access JWT ngắn (~15 phút); refresh ~30 ngày — bootstrap gọi `/v1/auth/refresh`.
