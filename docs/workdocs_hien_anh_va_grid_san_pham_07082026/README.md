# Hiển thị ảnh và lưới sản phẩm

- **Thư mục:** `docs/workdocs_hien_anh_va_grid_san_pham_07082026`
- **Ngày:** 07/08/2026
- **Loại:** fix + feature
- **Liên quan:** phản hồi giao diện sản phẩm staging

## Mục tiêu

Hiển thị URL ảnh đã lưu cho sản phẩm và đổi các danh sách sản phẩm sang dạng
grid responsive, dễ xem trên cả điện thoại và màn hình rộng.

## Phạm vi

- Trong scope: cửa hàng khách, bước chọn sản phẩm đặt hàng, danh sách quản trị,
  fallback khi ảnh lỗi và kiểm tra định dạng URL ở form quản trị.
- Ngoài scope: upload/lưu trữ file ảnh, resize ảnh phía server và thay đổi API.

## Quyết định chính

- Dùng chung `ProductImage` để tải ảnh HTTP/HTTPS; ảnh trống, URL sai hoặc tải
  lỗi đều trở về biểu tượng bình gas thay vì để ô ảnh vỡ.
- Grid dùng `SliverGridDelegateWithMaxCrossAxisExtent` để tự đổi số cột theo
  chiều rộng màn hình mà không cần breakpoint cố định.
- Chỉ kiểm tra cấu trúc URL ở form. Server ảnh bên ngoài vẫn phải cho phép trình
  duyệt tải ảnh và URL phải trỏ trực tiếp tới nội dung ảnh.

## Đã làm

- [x] Render `image_url` trên toàn bộ card sản phẩm.
- [x] Đổi cửa hàng, màn chọn hàng và danh sách quản trị từ list sang grid.
- [x] Thêm trạng thái loading và fallback khi ảnh không tải được.
- [x] Kiểm tra URL HTTP/HTTPS trước khi lưu.
- [x] Thêm widget test cho URL hợp lệ và fallback URL lỗi.

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/catalog/product_image.dart` | added | Widget ảnh dùng chung |
| `apps/mobile/lib/features/home/customer_shop_page.dart` | modified | Grid cửa hàng |
| `apps/mobile/lib/features/order/select_products_page.dart` | modified | Grid chọn sản phẩm |
| `apps/mobile/lib/features/catalog/admin_products_page.dart` | modified | Grid quản trị |
| `apps/mobile/lib/features/catalog/admin_product_form_page.dart` | modified | Validate URL |
| `apps/mobile/test/product_image_test.dart` | added | Test render/fallback |

## Cách verify

1. Chạy `flutter test test/product_image_test.dart`.
2. Chạy `flutter analyze`.
3. Tạo hoặc sửa sản phẩm với URL ảnh HTTPS trực tiếp.
4. Mở cửa hàng, chọn sản phẩm và quản trị sản phẩm; kiểm tra ảnh cùng số cột
   thay đổi theo chiều rộng.
5. Thử URL hỏng; card phải hiện biểu tượng bình gas và không vỡ layout.

## Ghi chú / blocker

- Ứng dụng không proxy ảnh bên thứ ba. Host ảnh cần còn hoạt động và cho phép
  trình duyệt tải tài nguyên.
