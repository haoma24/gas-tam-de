# Khôi phục phiên đăng nhập và điều hướng hai chiều

- **Thư mục:** `docs/workdocs_fix_phien_va_dieu_huong_07082026`
- **Ngày:** 07/08/2026
- **Loại:** fix
- **Liên quan:** Báo lỗi màn Hồ sơ và Đơn hàng của tôi

## Mục tiêu

Tự làm mới access token hết hạn trước khi gọi API hoặc sau một phản hồi 401, để khách đang có refresh token hợp lệ không bị báo hết phiên khi mở hồ sơ hay lịch sử đơn. Giữ navigation stack để màn mở trượt từ phải sang trái và thao tác quay lại trượt theo chiều ngược lại.

## Phạm vi

- Trong scope: Flutter auth session, Dio API client và route khách/admin.
- Ngoài scope: thay đổi thời hạn JWT hoặc API refresh phía backend.

## Quyết định chính

- Gom các lệnh refresh đồng thời vào một request vì refresh token được xoay vòng.
- Retry đúng một lần sau 401; request refresh không tự retry để tránh vòng lặp.
- Dùng `push()` cho điều hướng tiến và `pop()` (có fallback cho deep link) cho thao tác quay lại.

## Đã làm

- [x] Làm mới token chủ động khi biết access token sắp hết hạn.
- [x] Làm mới rồi gửi lại một lần khi API trả 401, hỗ trợ phiên cũ chưa lưu thời điểm hết hạn.
- [x] Chuyển các luồng màn hình sang navigation stack hai chiều.
- [x] Thêm test cho refresh đồng thời và retry sau 401.

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/auth/auth_session.dart` | modified | Refresh token đồng bộ và trả trạng thái thành công |
| `apps/mobile/lib/features/auth/auth_api.dart` | modified | Không áp dụng retry cho refresh endpoint |
| `apps/mobile/lib/core/api_client.dart` | modified | Refresh/retry access token tập trung |
| `apps/mobile/lib/main.dart` | modified | Push khi mở, pop khi quay lại |
| `apps/mobile/test/auth_session_refresh_test.dart` | added | Test refresh và retry |

## Cách verify

1. Chạy `flutter test test/auth_session_refresh_test.dart`.
2. Đăng nhập bằng tài khoản khách, mở Hồ sơ rồi Đơn hàng của tôi; API 401 với refresh token còn hạn phải tự khôi phục và tải dữ liệu.
3. Mở Hồ sơ từ Cửa hàng: màn đi từ phải sang trái; nhấn quay lại: màn đóng từ trái sang phải.
4. Mở trực tiếp một URL con rồi nhấn quay lại: ứng dụng đi tới route cha/fallback thay vì lỗi stack rỗng.

## Ghi chú / blocker

- `flutter analyze` chỉ còn các info/warning có sẵn ở những file không thuộc thay đổi này.
