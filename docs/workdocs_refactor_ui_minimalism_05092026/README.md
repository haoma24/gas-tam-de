# Refactor giao diện + luồng sử dụng theo hướng modern minimalism

- **Thư mục:** `docs/workdocs_refactor_ui_minimalism_05092026`
- **Ngày:** 05/09/2026
- **Loại:** refactor
- **Liên quan:** PRD §1 (CTA đặt giao gas), §2.1 (khách "không muốn form dài"), §3.1–3.2 (user flow khách / admin)

## Mục tiêu

Đưa toàn bộ app Flutter về **một** ngôn ngữ thị giác đơn sắc, tiết chế, dùng chung
cho cả khách hàng và admin; thay điều hướng push-based rời rạc bằng shell thường
trực; rút ngắn phễu đặt hàng; và bỏ phần "trang trí kiểu sàn thương mại điện tử"
không phục vụ nghiệp vụ đặt gas.

## Hiện trạng trước khi sửa

- **Hai ngôn ngữ thị giác song song.** Màn khách/auth dùng nền kem `#FAF7F4`,
  gradient tối `heroGradient`, `FlameAmbientPainter`, nút pill gradient cam,
  `AppShadow` blur 20–40px, chữ `w800`/`w900`. Màn admin dùng Material 3 mặc định
  với `BorderRadius.circular(12)` — khác hẳn.
- `AppTextStyles` chỉ được gọi **3 lần** trong khi có **62 `TextStyle` inline** và
  **24 mã hex cứng** ngoài file token. Không có thang spacing.
- **Không có thư viện widget dùng chung**: ~10 bản copy khối lỗi, ~8 empty state,
  15+ spinner, 3 product card, 3 qty stepper, 4 nút press-scale tự viết,
  2 `_MoneyRow`, 3 biến thể "card có tiêu đề".
- **Router phẳng 20 route, không ShellRoute**: bottom nav nằm bên trong
  `CustomerShopPage` nên biến mất ở mọi màn khác; admin không có điều hướng
  thường trực (dashboard là danh sách 9 tile).
- Guard đăng nhập là `Consumer + addPostFrameCallback + spinner` **copy 4 lần**;
  mọi page nhận điều hướng qua `VoidCallback` prop (dashboard: 10 prop).
- **~1.008 dòng code chết**: `customer_auth_flow_page`, `otp_page`, `phone_page`.
- Phễu đặt hàng **4 màn**, có **2 ô tìm kiếm sản phẩm** cho một cửa hàng vài SKU.

## Quyết định chính

| Hạng mục | Chốt | Lý do |
|---|---|---|
| Palette | Đơn sắc than/trắng; cam là accent rất tiết chế | Chốt với maintainer |
| Nút chính | Khối `ink` (than), **không** phải cam | Cam giữ cho CTA đặt gas, badge khẩn, tab đang chọn — tối đa 1 accent/viewport |
| Độ nổi | Viền hairline 1px, **không** đổ bóng | Bỏ `AppShadow` |
| Gradient | Bỏ hoàn toàn | Xoá `heroGradient`, `subtleHeroGradient`, `FlameAmbientPainter` |
| Dark mode | Có — light + dark theo system | Token semantic theo cặp |
| Admin | Responsive: bottom nav < 900px, NavigationRail + 2 cột ≥ 900px | Chủ cửa hàng dùng cả điện thoại và máy |
| Cân chữ | Tối đa `w600` | `w800`/`w900` không phải minimalism |
| Bo góc | 3 mức: 8 / 12 / 999 | Rút từ 5 mức (bỏ 16/24/32) |

### Token là `ThemeExtension`, không phải hằng số tĩnh

`AppPalette` là `ThemeExtension<AppPalette>` nên light/dark cùng resolve qua một
call site `context.palette.ink`. Đây là điều kiện để có dark mode mà không phải
viết nhánh `if (isDark)` ở từng màn.

### Token đặt tên theo VAI TRÒ, không theo màu

`ink` / `inkMuted` / `surface` / `border` / `accent` thay cho `fire` / `obsidian` /
`amber`. Tên theo màu là lý do bảng màu cũ không thể có dark mode.

### Gộp Báo cáo + Công nợ

`/admin` (dashboard) và `/admin/debts` là hai đích riêng cho cùng một câu hỏi
("cửa hàng đang lời/lỗ và ai còn nợ"). Gộp thành tab **Báo cáo**; Order Desk trở
thành màn đích khi admin đăng nhập, đúng nhu cầu đầu tiên theo PRD §2.1.

### Phễu đặt hàng 4 màn → 2

`/order` gộp 4 khối trên một màn cuộn: Sản phẩm → Giao đến → Người nhận →
Thanh toán. `/order/address` vẫn giữ **nguyên vẹn** 657 dòng logic (search
debounce 550ms, GPS, sổ địa chỉ, `POST /v1/geo/check`) nhưng chuyển thành picker
push-rồi-pop, chỉ mở khi cần đổi địa chỉ. Cơ chế **re-quote ngay trước khi submit**
được giữ nguyên — đây là ràng buộc đúng để tổng tiền khớp fee engine phía server.

### Đặt lại đơn trước

`GET /v1/orders/me` đã trả về line items nhưng chưa được dùng để tăng tốc mua lại.
Thêm `lastOrderProvider` + card "Đặt lại" trên trang chủ: nạp sẵn giỏ + địa chỉ
rồi nhảy thẳng vào `/order`. Khách quay lại còn **2 tap**.

## Đã làm

- [x] Tầng token semantic + theme dùng chung (`core/ui/app_tokens.dart`, `app_theme_data.dart`)
- [x] Theme hoá **mọi** component đang render (bổ sung NavigationBar/Rail, ListTile, Chip, Dialog, SnackBar, SegmentedButton, Switch, Checkbox, Radio, BottomSheet, PopupMenu, TabBar, Tooltip)
- [x] Thư viện widget dùng chung `core/ui/` (10 file) thay ~40 bản copy
- [x] Router tập trung `core/router.dart` + **một** redirect guard thay 4 bản copy
- [x] `StatefulShellRoute` cho khách (3 tab) và admin (4 tab), admin responsive
- [x] Bỏ toàn bộ `onBack`/`onContinue`/`onOpenX` prop → `context.go`/`popOrGo`
- [x] Gộp phễu đặt hàng 4 màn → 2; thêm "Đặt lại đơn trước"
- [x] Gộp dashboard + công nợ thành tab Báo cáo; 9 tile → 4 tab + hub Cài đặt
- [x] Reskin toàn bộ màn khách và admin theo token mới
- [x] Sửa lỗi thật: `my_orders_page` in chuỗi thô `PENDING` → `orderStatusLabelVi()`
- [x] Xoá ~1.008 dòng code chết (OTP pages) + `core/app_theme.dart` + `_auth_widgets.dart`
- [x] Thêm test theme (light/dark) và test breakpoint shell
- [x] Chạy analyzer, test, build web, và kiểm tay trên stack thật

## File đụng tới

### Thêm mới

| Path | Ghi chú |
|------|---------|
| `apps/mobile/lib/core/format.dart` | `formatVnd` thuần, không phụ thuộc Flutter |
| `apps/mobile/lib/core/router.dart` | Router + redirect guard tập trung |
| `apps/mobile/lib/core/ui/app_tokens.dart` | `AppPalette` (ThemeExtension), `AppSpacing`, `AppRadius`, `AppDuration`, `VGap`/`HGap` |
| `apps/mobile/lib/core/ui/app_theme_data.dart` | `buildAppTheme(Brightness)` |
| `apps/mobile/lib/core/ui/app_breakpoints.dart` | `compact` / `medium` / `expanded` (900) |
| `apps/mobile/lib/core/ui/app_{states,button,section,money,badge,field,tile,scaffold}.dart` | Primitive dùng chung |
| `apps/mobile/lib/core/ui/auth_layout.dart` | `AuthScrollBody`, `AuthCard`, `AuthErrorText` (chuyển từ `_auth_widgets.dart`) |
| `apps/mobile/lib/core/ui/ui.dart` | Barrel export |
| `apps/mobile/lib/features/shell/customer_shell.dart` | Bottom nav thường trực 3 tab |
| `apps/mobile/lib/features/shell/admin_shell.dart` | 4 tab, bottom nav ↔ rail theo bề rộng |
| `apps/mobile/lib/features/home/welcome_page.dart` | Guest landing (thay `home_page.dart`) |
| `apps/mobile/lib/features/order/order_page.dart` | Màn đặt hàng gộp |
| `apps/mobile/lib/features/order/last_order.dart` | `lastOrderProvider` cho "Đặt lại" |
| `apps/mobile/lib/features/dashboard/admin_reports_page.dart` | Metrics + công nợ |
| `apps/mobile/lib/features/dashboard/admin_settings_page.dart` | Hub cấu hình |
| `apps/mobile/test/app_theme_test.dart` | Theme dựng được cả 2 brightness |
| `apps/mobile/test/shell_responsive_test.dart` | Bottom nav ↔ rail tại breakpoint |
| `apps/mobile/test/welcome_page_test.dart` | Guest chỉ có 1 nút (thay `home_page_test.dart`) |

### Xoá

`core/app_theme.dart`, `features/auth/_auth_widgets.dart`,
`features/auth/customer_auth_flow_page.dart`, `features/auth/otp_page.dart`,
`features/auth/phone_page.dart`, `features/home/home_page.dart`,
`features/dashboard/admin_dashboard_page.dart`, `features/billing/admin_debts_page.dart`,
`features/order/select_products_page.dart`, `features/order/order_review_page.dart`,
`test/otp_page_test.dart`, `test/home_page_test.dart`.

### Sửa

`main.dart` (469 → 45 dòng), toàn bộ page khách và admin, `product_image.dart`,
`wait_time_badge.dart`, `catalog_models.dart` (re-export `formatVnd`),
`order_models.dart` (+ `OrderStatus`, `orderStatusLabelVi`), 4 file test.

**Tổng:** `lib/` từ 16.822 dòng / 68 file → 14.930 dòng / 80 file.

## Cách verify

```powershell
cd apps/mobile
flutter analyze     # còn 7 info, đều tồn tại từ trước (Radio deprecated x6 + dangling doc x1)
flutter test        # 54 test
flutter build web
```

Chạy thật (Docker hoặc Go trên host):

```powershell
.\scripts\dev.ps1 web-up
.\scripts\dev.ps1 web-health
```

Kiểm tay:

1. **Guest** `/#/welcome` — đúng một nút «Đăng nhập», không CTA đặt hàng, không lối vào admin.
2. **Admin** `/#/admin/login` (admin / admin-change-me khi `ADMIN_SEED=1`) — vào thẳng
   tab **Đơn**; ≥900px thấy NavigationRail + 2 cột (danh sách | chi tiết);
   <900px thấy bottom nav 4 tab và chi tiết mở full-screen.
3. **Tab Báo cáo** — SegmentedButton kỳ, 6 metric, danh sách công nợ ngay dưới.
4. **Tab Cài đặt** — 6 mục cấu hình; bấm vào push full-screen (không có rail), back về đúng chỗ.
5. **Dark mode** — bật dark ở OS, cả hai vai đọc được, card vẫn thấy viền hairline.
6. **Khách** — «Đặt lại đơn trước» nạp sẵn giỏ + địa chỉ; `/order` một màn; đặt đơn → success.

## Kết quả xác minh

- `flutter analyze`: **7 info**, giảm từ 16 baseline; không có error/warning.
  7 info còn lại đều tồn tại từ trước refactor.
- `flutter test`: **54/54 pass**.
- `flutter build web`: thành công.
- Kiểm tay trên stack thật (8 Go service chạy host + web build):
  đăng nhập admin, Order Desk 2 cột, chọn đơn → pane chi tiết, tab Báo cáo,
  tab Cài đặt, push `/admin/products` và back — tất cả đúng, ở **cả light và dark**.
- Kiểm tra hồi quy phong cách trong `lib/` (ngoài `core/ui/`): **0** `LinearGradient`,
  **0** `BoxShadow`, **0** `FlameAmbient`, **0** `FontWeight.w8/w9`,
  **0** `BorderRadius.circular(...)`. Chỉ còn 1 hex cứng: xanh thương hiệu Google
  trong `google_sign_in_button_stub.dart` (cố ý — màu của bên thứ ba).

## Ghi chú / blocker

- **Chưa kiểm được luồng khách end-to-end trên máy này**: đăng nhập khách dùng
  Google Sign-In, cần client ID cấu hình cho origin đang chạy. Phần khách đã được
  verify qua widget test (shell, welcome) + build; cần kiểm tay trên staging.
- **Docker Desktop không chạy** lúc verify nên dùng `go run` trực tiếp 8 service
  trên host thay cho `make web-up`. NATS không có — không ảnh hưởng vì `/healthz`
  không chạm NATS và các service kết nối nền.
- **Hai điểm ngoài phạm vi, nêu để ghi nhận:**
  1. Guest vẫn phải đăng nhập mới thấy sản phẩm/giá. PRD §1 và MoSCoW must-have
     nói CTA đặt gas phải thấy ngay khi mở app, nhưng PRD §3.1 (bản sửa sau) lại
     yêu cầu guest chỉ có một nút Đăng nhập. Giữ nguyên hành vi hiện tại; nếu muốn
     đổi thì đó là quyết định sản phẩm, nên tách PR riêng.
  2. Luồng OTP đã bị bỏ route nên SĐT giao hàng là text tự do **chưa xác thực**
     (`me_api.dart` `patchPhone`). Với nghiệp vụ gọi-điện-để-giao thì đây là
     trường quan trọng nhất.
- `test/product_image_test.dart` được sửa một dòng: icon fallback đổi từ
  `propane_tank_rounded` sang `propane_tank_outlined` cho khớp ngôn ngữ icon
  outlined dùng toàn app. Các test logic khác pass **không sửa** — đó là lưới an
  toàn chứng minh refactor không đụng nghiệp vụ.
