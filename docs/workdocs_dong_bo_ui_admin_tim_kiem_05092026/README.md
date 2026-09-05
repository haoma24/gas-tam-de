# Đồng bộ UI admin và sửa ô tìm kiếm trang chủ

- **Thư mục:** `docs/workdocs_dong_bo_ui_admin_tim_kiem_05092026`
- **Ngày:** 05/09/2026
- **Loại:** fix + refactor
- **Liên quan:** phản hồi giao diện admin và trang chủ khách hàng

## Mục tiêu

Đưa khu vực quản trị về cùng ngôn ngữ thị giác với phần còn lại của Gas Tam Đệ,
đồng thời bảo đảm ô tìm kiếm sản phẩm trên trang chủ khách hàng luôn hiển thị và
nhận tương tác đầy đủ, không bị hero banner phía trên che khuất.

## Phạm vi

- Trong scope: theme Material dùng chung, dashboard quản trị và vị trí ô tìm kiếm
  trong `CustomerShopPage`.
- Ngoài scope: thay đổi luồng nghiệp vụ, API, schema dữ liệu, nội dung các form
  quản trị và thiết kế lại từng màn admin riêng biệt.

## Quyết định chính

- Dùng các token đã có trong `core/app_theme.dart` làm nguồn nhận diện duy nhất:
  nền kem, surface trắng/kem nhạt, màu cam `fire`, màu than `obsidian` và radius
  dùng chung.
- Cấu hình component theme ở `MaterialApp` để toàn bộ màn admin hiện tại và màn
  thêm sau này tự kế thừa style nhất quán, thay vì lặp màu và shape ở từng page.
- Dashboard admin được nhấn mạnh bằng header tối và welcome card dùng cùng hero
  gradient với cửa hàng khách; các metric/menu vẫn giữ nguyên hành vi.
- Không dùng `Transform.translate` để tạo overlap giữa hai sliver. Ô tìm kiếm nằm
  sau hero trong normal layout với khoảng cách trên 16 px, tránh clipping và sai
  vùng hit-test trên các kích thước màn hình/trình duyệt khác nhau.

## Đã làm

- [x] Ánh xạ palette, surface, radius và style component vào theme toàn app.
- [x] Đồng bộ app bar, input, button, card, FAB, progress và divider.
- [x] Làm mới phần nhận diện và card điều hướng của dashboard admin.
- [x] Đưa ô tìm kiếm xuống dưới banner, bỏ offset âm gây che/cắt.
- [x] Format và chạy analyzer cho các file thay đổi.
- [x] Chạy widget test trang chủ và build Flutter Web release.

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/main.dart` | modified | Theme Material dùng chung từ design token |
| `apps/mobile/lib/features/dashboard/admin_dashboard_page.dart` | modified | Header, welcome card, metric và menu admin |
| `apps/mobile/lib/features/home/customer_shop_page.dart` | modified | Bỏ search overlap, thêm khoảng cách an toàn |
| `CHANGESLOG.md` | modified | Ghi nhận thay đổi mới nhất |
| `docs/workdocs_dong_bo_ui_admin_tim_kiem_05092026/README.md` | added | Lịch sử và hướng dẫn verify |

## Cách verify

1. Chạy analyzer:
   `dart analyze lib/main.dart lib/features/home/customer_shop_page.dart lib/features/dashboard/admin_dashboard_page.dart`.
2. Chạy `flutter test test/home_page_test.dart`.
3. Chạy `flutter build web`.
4. Đăng nhập khách, kiểm tra ô «Tìm bình gas, phụ kiện…» nằm hoàn toàn dưới
   hero banner trên mobile và desktop; nhập từ khóa để kiểm tra suggestion.
5. Đăng nhập admin, kiểm tra dashboard và các màn con dùng nền kem, màu cam,
   app bar, input/button/card thống nhất; xác nhận các menu vẫn điều hướng đúng.

## Kết quả xác minh

- Dart analyzer: `No issues found`.
- Widget test `test/home_page_test.dart`: pass.
- Flutter Web release build: thành công (`build/web`).

## Ghi chú / blocker

- Working tree có sẵn các thay đổi trong `services/auth-service` và dữ liệu local
  dưới `services/api-gateway/data/`; thay đổi UI/docs này không chỉnh các file đó.
- Theme dùng chung có chủ đích tác động tới mọi màn kế thừa `ThemeData`; các màn
  đã đặt màu cục bộ (ví dụ hero hoặc app bar tối của luồng đặt hàng) vẫn giữ style
  riêng.
