# CHANGESLOG — Gas Tam Đệ

Nhật ký thay đổi của repo. Entry mới nhất ở **trên cùng**.  
Quy trình: skill `.cursor/skills/change-workdocs`.

---

## [2026-09-05] Lợi nhuận tính theo giá nhập, lịch sử đơn có bộ lọc, thống kê theo khách, admin thấy SĐT khách

- **Loại:** fix + feat
- **Phạm vi:** `services/order-service`, `services/inventory-service`, `services/auth-service`, `services/report-service` (test), `apps/mobile` (tab Đơn + tab Báo cáo), `deploy/docker-compose.yml`, `docs/codemap.md`
- **Tóm tắt:** Bốn việc chủ shop báo, cùng một lần. (1) Lợi nhuận trên tab «Báo cáo» đang **bằng doanh thu**: công thức `profit = revenue − cogs` ở `report-service/profit.go` vốn đúng, nhưng `cogs` luôn bằng 0 vì `order_items` không có cột giá vốn và payload `order.completed` không mang `unit_cost` — report đọc `m["unit_cost"]` và luôn nhận `nil`. (2) Đơn hoàn tất **biến mất khỏi màn hình**: backend đã nhận `?status=` từ lâu nhưng Flutter không bao giờ truyền, nên tab «Đơn» luôn chỉ thấy `PENDING`. (3) Không có endpoint nào thống kê theo khách. (4) Admin chỉ thấy `090***7020` — số thật **chưa từng được lưu** trong `order.db`, chỉ auth-service có ở dạng mã hoá AES-GCM; không sửa được ở tầng UI.
- **Chi tiết:**
  - **Giá vốn (COGS).** Tồn kho bị trừ ngay lúc **đặt đơn** (`inventory-service/reserve.go`), nên bản chụp giá vốn đã sẵn có trong `stock_movements.unit_cost` từ thời điểm đó — chỉ là order-service không nhận lại. Nay `POST /v1/internal/stock/reserve` trả thêm `items: [{product_id, unit_cost}]`; order-service lưu vào cột mới `order_items.unit_cost` và đưa vào payload `order.completed`. Không thêm stream/consumer NATS nào: giá vốn đi nhờ lời gọi đồng bộ vốn đã tồn tại. Bên report **không phải sửa** — nó vốn đã đọc `unit_cost`
  - Cột mới của DB đã deploy được vá bằng `ensureColumn` trong `migrate()` (copy khuôn của auth-service): `CREATE TABLE IF NOT EXISTS` không tiến hoá được bảng đã tồn tại
  - **Lịch sử đơn.** `GET /v1/admin/orders?status=` nhận thêm `ALL`; sắp xếp theo ngữ cảnh — `PENDING` giữ FIFO (cũ nhất trước, có `stt`), mọi trạng thái khác trả mới nhất trước và không đánh số. Response bổ sung `completed_at`, `cancelled_at`, `payment_type`, `amount_paid` để mở lại đơn là thấy đã thu bao nhiêu, còn nợ bao nhiêu
  - Tab «Đơn» có hàng chip **Chưa giao · Đã giao · Bị hủy · Tất cả**. Chỉ «Chưa giao» mới tự làm mới 10s và đọc/thông báo đơn mới — lịch sử không đổi theo thời gian, poll nó chỉ là nhiễu. Tile lịch sử hiện badge trạng thái thay cho badge thời gian chờ; nút «Hoàn tất» tự ẩn với đơn không còn `PENDING`
  - **Thống kê theo khách.** `GET /v1/admin/orders/customers?from=&to=&limit=` (mặc định 30 ngày gần nhất) gộp thẳng từ bảng `orders`: số đơn theo từng trạng thái, tổng chi, đã trả, còn nợ, lần đặt đầu/cuối. Tiền chỉ cộng đơn `COMPLETED`, đúng luật «chỉ tính khi hoàn tất» của kiến trúc; lọc theo **ngày VN (UTC+7)** để khớp `daily_stats.day` của report-service. Không dựng read-model mới: đây là một câu `GROUP BY` trên một bảng
  - Tab «Báo cáo» thêm mục «Khách hàng» (top 10 + «Xem tất cả»), SĐT bấm gọi được. Thêm cảnh báo khi `cogs = 0` mà `revenue > 0`: nói thẳng «lợi nhuận đang bằng doanh thu, vào tab Kho nhập giá nhập» thay vì im lặng hiện con số đẹp
  - **SĐT khách.** auth-service mở `POST /v1/internal/users/phones` (theo lô, **không** qua gateway, cùng lớp với `/v1/internal/stock/*`), giải mã `contact_phone_e164_enc` → `phone_e164_enc` và trả dạng `0…`. order-service snapshot vào cột mới `orders.customer_phone` lúc đặt đơn; đơn cũ được vá ngược **lười theo lô** khi admin liệt kê, rồi ghi lại nên lần sau không gọi nữa. Không nhét số vào JWT: làm vậy là để số thật nằm trong localStorage của mọi phiên trình duyệt
  - Số thật **chỉ** ra ở view admin (`pii.go:adminOrderView`); `GET /v1/orders/me` của khách vẫn masked như cũ, có test chặn rò rỉ. Trang chi tiết đơn: chạm số để gọi (`tel:`), giữ để chép; chưa có số thì không hiện nút gọi mà nói rõ khách chưa thêm SĐT liên hệ
  - Lời gọi auth là **best-effort** ở mọi chỗ: auth chết thì khách vẫn đặt được đơn và Order Desk vẫn liệt kê được, chỉ là chưa hiện số. Vì vậy **không** thêm auth vào `/readyz` hay `depends_on` của order-service — đó đúng là cái bẫy «cả stack hỏng vì một service chậm» mà `CLAUDE.md` cảnh báo
  - `deploy/docker-compose.yml`: order-service nhận `AUTH_SERVICE_URL`
- **Rủi ro đã biết:** đơn đặt **trước** bản này có `unit_cost = 0` nên báo cáo kỳ cũ vẫn hiện lợi nhuận = doanh thu (giá nhập tại thời điểm đó không còn khôi phục được) — số liệu đúng dần từ ngày deploy. Nếu admin chưa nhập giá nhập ở tab «Kho» thì COGS vẫn 0, đã có cảnh báo trên UI. Khách đăng nhập Google chưa thêm SĐT liên hệ thì admin vẫn thấy `—`.
- **Workdocs:** `docs/workdocs_bao-cao-loi-nhuan-lich-su-don-sdt-khach_05092026/`
- **Liên quan:** `docs/codemap.md` §1, §3, §5.1, §5.2, §7, §8


## [2026-09-05] Thêm docs/codemap.md — bản đồ chức năng ⇄ file

- **Loại:** docs
- **Phạm vi:** `docs/codemap.md` (mới), `CLAUDE.md`
- **Tóm tắt:** `docs/` đã có 95 thư mục workdocs ghi *vì sao* từng thay đổi được làm, nhưng không có chỗ nào trả lời *hiện tại chức năng X nằm ở file nào*. Thêm một file tra cứu duy nhất, dựng theo lát cắt dọc: màn hình → route → trang Flutter → file gọi API → endpoint → nhóm quyền ở gateway → service → bảng SQLite → sự kiện.
- **Chi tiết:**
  - §1 bảng tra nhanh 9 chức năng khách + 13 chức năng admin, mỗi dòng là một lát cắt dọc đầy đủ nên sửa một chức năng là thấy hết chỗ phải đụng
  - §2 tầng chung Flutter (`core/`, `core/ui/` 13 file design system) — vùng sửa vào là ảnh hưởng toàn app
  - §3 bảng 8 service kèm cổng, endpoint, bảng SQLite, sự kiện; và `pkg/` dùng chung
  - §4 ba nhóm quyền ở gateway, kèm cảnh báo service phía sau không kiểm role lần nữa nên thêm endpoint là phải thêm vào đúng nhóm
  - §5 tách rõ **hai** đường giao tiếp hay bị nhầm: gọi HTTP nội bộ đồng bộ (`order-service` → inventory `/v1/internal/stock/{reserve,release}`, → billing `/v1/internal/payments`, không đi qua gateway) và sự kiện NATS bất đồng bộ
  - Rà thực tế phát hiện 4 subject **đã khai báo trong `pkg/events/events.go` nhưng chưa ai phát và chưa ai nghe** (`auth.otp.verified`, `geo.store_config.updated`, `inventory.stock.adjusted`, `inventory.low_stock`) — ghi rõ để không tưởng nhầm là đang chạy
  - §6 hạ tầng/deploy/CI, §7 sửa gì chạy test gì kèm 13 test Flutter và thứ mỗi test bảo vệ, §8 bẫy hay dính, §9 quy ước cập nhật theo từng loại thay đổi
  - `CLAUDE.md` trỏ vào file này ngay đầu mục Repo, kèm yêu cầu cập nhật cùng commit
- **Workdocs:** n/a (bản thân file là tài liệu)
- **Liên quan:** `docs/architecture.md`, `docs/prd.md`, `.cursor/rules/change-workdocs.mdc`


## [2026-09-05] Thêm nút chọn giao diện Sáng/Tối và sửa mất icon tab admin

- **Loại:** feat + fix
- **Phạm vi:** `apps/mobile` (theme mode, trang Cài đặt admin, trang Hồ sơ khách), `deploy/nginx.web.conf`
- **Tóm tắt:** Bản refactor minimalism thêm dark theme nhưng `main.dart` để cứng `themeMode: ThemeMode.system` và không có chỗ ghi đè, nên máy đang để dark thì app luôn tối không có đường ra. Nay có khối «Giao diện» (Hệ thống / Sáng / Tối) lưu vào máy, xuất hiện ở cả Cài đặt admin lẫn Hồ sơ khách. Hai tab «Báo cáo» và «Cài đặt» mất icon **không phải do code** mà do trình duyệt dùng lại file font tree-shaken của bản deploy trước — bản đó chưa dùng `bar_chart`/`settings` nên font thiếu đúng hai glyph này; nginx trước đây không gửi `Cache-Control` cho các URL không hash của Flutter.
- **Chi tiết:**
  - `core/ui/app_theme_mode.dart` (mới): `ThemeModeController extends StateNotifier<ThemeMode>` đọc/ghi `SharedPreferences` khoá `gas_tam_de.theme_mode.v1`, dùng lại `sharedPreferencesProvider` sẵn có nên khôi phục ngay ở build đầu, không nhấp nháy theme
  - `AppThemeModeSection` — `AppSection` «Giao diện» chứa `SegmentedButton<ThemeMode>` ba lựa chọn, dùng chung cho admin và khách để hai vai không lệch nhau
  - `main.dart`: `themeMode: ThemeMode.system` → `themeMode: ref.watch(themeModeProvider)`
  - Gắn vào `admin_settings_page.dart` (dưới thẻ tài khoản) và `customer_profile_page.dart` (trên «Đơn hàng của tôi»)
  - Đã loại giả thuyết `const_finder` bỏ sót `IconData` trong const record: parse bảng `cmap` của `MaterialIcons-Regular.otf` vừa build cho thấy đủ cả 8 codepoint, build lại và mở Chrome thì bốn icon hiện đủ
  - Nguyên nhân thật: `git grep` trên `8d8294c` xác nhận bản trước không dùng `bar_chart`/`settings` ⇒ font subset của bản đó thiếu glyph; Flutter phục vụ `assets/**` và `main.dart.js` ở URL cố định không hash; service worker của SDK này đã deprecated và tự huỷ đăng ký; nginx không gửi `Cache-Control` ⇒ trình duyệt áp heuristic freshness, tải JS mới nhưng dùng lại font cũ
  - `deploy/nginx.web.conf`: thêm `Cache-Control "no-cache"` cho `main.dart.js|flutter.js|flutter_bootstrap.js`, `/assets/` và `/canvaskit/`. `no-cache` không tắt cache, chỉ ép revalidate bằng ETag (`etag on` đã bật sẵn) — file không đổi trả 304
  - **Máy đang bị lỗi cần hard reload một lần** (`Ctrl+Shift+R`) để bỏ font cũ; từ sau đó mọi deploy tự lấy đúng asset
  - Thêm `test/theme_mode_test.dart` (2 test: khôi phục sau restart, và đổi theme qua UI); `flutter analyze` vẫn 7 info cũ, 56/56 test pass, build web thành công
  - Chưa kiểm được: `nginx -t` (Docker daemon không chạy) và rà mắt admin nav ở light mode (extension Chrome mất kết nối)
- **Workdocs:** `docs/workdocs_theme_toggle_va_icon_admin_05092026/`
- **Liên quan:** `docs/workdocs_refactor_ui_minimalism_05092026/`


## [2026-09-05] Refactor giao diện + luồng sử dụng theo hướng modern minimalism

- **Loại:** refactor
- **Phạm vi:** `apps/mobile` (design system, router, shell điều hướng, toàn bộ màn khách và admin)
- **Tóm tắt:** App đang tồn tại hai ngôn ngữ thị giác song song — màn khách dùng gradient tối, hiệu ứng lửa và nút pill cam; màn admin dùng Material mặc định — cùng với một router phẳng không shell khiến bottom nav biến mất ngoài trang chủ và admin phải đi qua danh sách 9 tile. Thay bằng một hệ token đơn sắc than–trắng có dark mode, một thư viện widget dùng chung, shell điều hướng thường trực cho cả hai vai, và phễu đặt hàng rút từ 4 màn xuống 2.
- **Chi tiết:**
  - Token semantic đặt theo vai trò (`ink`, `surface`, `border`, `accent`) dưới dạng `ThemeExtension`, nên light và dark dùng chung một call site `context.palette`; bỏ toàn bộ gradient, `FlameAmbientPainter` và `AppShadow` — độ nổi thể hiện bằng viền hairline 1px
  - Cam chỉ còn là accent tiết chế (CTA đặt gas, badge khẩn, tab đang chọn); nút chính là khối `ink`
  - Theme hoá mọi component đang render, bổ sung NavigationBar/Rail, ListTile, Chip, Dialog, SnackBar, SegmentedButton, Switch, Checkbox, Radio, BottomSheet, PopupMenu, TabBar, Tooltip — đây là lý do gốc khiến 24 mã hex cứng tồn tại rải rác
  - Thư viện `core/ui/` (AppScaffold, AppSection, AppButton, AppStates, AppBadge, MoneyRow, QtyStepper, AppNavTile…) thay ~40 bản copy widget riêng lẻ: 10 khối lỗi, 8 empty state, 15 spinner, 3 product card, 3 qty stepper, 4 nút press-scale tự viết
  - Router chuyển sang `core/router.dart` với **một** redirect guard thay 4 bản copy `Consumer + addPostFrameCallback`; bỏ toàn bộ prop `onBack`/`onContinue`/`onOpenX` (riêng dashboard cũ 10 prop) → dùng `context.go`/`popOrGo`
  - `StatefulShellRoute` cho khách (Cửa hàng | Đơn hàng | Hồ sơ) — bottom nav nay thường trực thay vì nằm trong `CustomerShopPage`; admin gom 9 tile thành 4 tab (Đơn | Kho | Báo cáo | Cài đặt) và Order Desk trở thành màn đích khi đăng nhập
  - Admin responsive: bottom nav dưới 900px, NavigationRail + bố cục 2 cột (danh sách | chi tiết đơn) từ 900px trở lên
  - Phễu đặt hàng gộp còn `/order` (sản phẩm + địa chỉ + người nhận + thanh toán trên một màn) và `/order/success`; `/order/address` giữ nguyên logic search/GPS/sổ địa chỉ nhưng thành picker chỉ mở khi cần đổi. Cơ chế re-quote trước khi submit được bảo toàn
  - Thêm «Đặt lại đơn trước» trên trang chủ dùng `GET /v1/orders/me` sẵn có: nạp sẵn giỏ và địa chỉ, khách quay lại chỉ còn 2 tap
  - Gộp dashboard và `/admin/debts` thành tab Báo cáo (metrics kỳ + danh sách công nợ)
  - Sửa lỗi thật: `my_orders_page` in chuỗi thô `PENDING` cho khách; tách `orderStatusLabelVi()` dùng chung thay 4 chỗ so sánh string literal
  - Xoá ~1.008 dòng code chết của luồng OTP cũ (`customer_auth_flow_page`, `otp_page`, `phone_page`) cùng `core/app_theme.dart` và `_auth_widgets.dart`; `main.dart` từ 469 xuống 45 dòng; `lib/` từ 16.822 còn 14.930 dòng
  - Thêm test theme (dựng được cả light và dark) và test breakpoint shell (bottom nav ↔ rail); `flutter analyze` còn 7 info đều tồn tại từ trước (giảm từ 16), 54/54 test pass, build web thành công, đã kiểm tay trên stack thật ở cả light và dark
- **Workdocs:** `docs/workdocs_refactor_ui_minimalism_05092026/`
- **Liên quan:** PRD §1 (CTA đặt giao gas), §2.1 (khách không muốn form dài), §3.1–3.2 (user flow khách / admin)


## [2026-09-05] Đồng bộ giao diện admin và sửa ô tìm kiếm trang chủ

- **Loại:** fix + refactor
- **Phạm vi:** `apps/mobile` (theme, admin dashboard, cửa hàng khách)
- **Tóm tắt:** Giao diện quản trị trước đây dùng phần lớn style Material mặc định nên lệch khỏi nhận diện than–cam–kem của khu vực khách. Ô tìm kiếm tại cửa hàng khách còn được kéo chồng lên hero bằng transform nên có thể bị banner/sliver che phần đầu; nay theme được chuẩn hóa toàn app và ô tìm kiếm nằm trọn bên dưới banner.
- **Chi tiết:**
  - Ánh xạ `AppColors`/`AppRadius` vào `ThemeData` dùng chung cho nền, app bar, card, input, button, FAB, progress và divider
  - Làm mới dashboard admin bằng app bar tối, welcome card gradient và card điều hướng theo palette Gas Tam Đệ
  - Bỏ offset âm của ô tìm kiếm, thay bằng khoảng cách 16 px sau hero để vùng hiển thị và tương tác không bị cắt
  - Dart analyzer sạch, widget test trang chủ pass và Flutter Web release build thành công
- **Workdocs:** `docs/workdocs_dong_bo_ui_admin_tim_kiem_05092026/`
- **Liên quan:** phản hồi giao diện ngày 05/09/2026

## [2026-08-12] Tắt lộ mã OTP mặc định + cảnh báo SMS_PROVIDER sai

- **Loại:** security + fix
- **Phạm vi:** `deploy`, `auth-service`
- **Tóm tắt:** Chuẩn bị gửi OTP thật qua Stringee. `OTP_DEV_REVEAL` đang mặc định `1` khiến API trả thẳng mã OTP trong response — ai gọi được endpoint cũng đăng nhập được bằng bất kỳ số nào, gửi SMS thật không đóng được lỗ này. `SMS_PROVIDER` gõ sai thì âm thầm về mock, request vẫn 200 mà không tin nào tới máy.
- **Chi tiết:**
  - `docker-compose.yml` mặc định `OTP_DEV_REVEAL=0`; `docker-compose.local.yml` override `1` để `make compose-up` vẫn không cần vendor SMS
  - `.env.vps.example` đổi sang `0`; `.env.example` giữ `1` (file local) nhưng cảnh báo rõ hậu quả
  - `SMS_PROVIDER` không nhận dạng được nay log `ERROR` kèm giá trị sai và các giá trị hợp lệ, vẫn fallback mock để không chết boot
  - 3 test: 2 guard mặc định compose + 1 test chốt nội dung cảnh báo
  - Client Stringee vốn đã hoàn chỉnh — bật SMS thật chỉ cần `SMS_PROVIDER=stringee` + SID/SECRET/SENDER
- **Workdocs:** `docs/workdocs_bat_sms_that_va_tat_dev_reveal_12082026/`
- **Liên quan:** chuẩn bị test SMS thật

## [2026-08-12] Validate số điện thoại theo đầu số di động Việt Nam

- **Loại:** fix
- **Phạm vi:** `auth-service`, `apps/mobile` (đăng nhập)
- **Tóm tắt:** Luật cũ chỉ kiểm tra độ dài (`^0\d{9}$`) nên `0123456789`, `0000000000`, `0212345678`, `+84012345678` đều lọt và gửi SMS thật qua Stringee trước khi thất bại. Nay chỉ nhận đúng bộ đầu số di động sau đợt chuyển đổi 2018.
- **Chi tiết:**
  - `vnMobilePrefix` = `3[2-9]|5[25689]|7[06-9]|8[1-9]|9[0-9]`, áp cho cả dạng local và E.164
  - Giữ nguyên chuẩn hoá `0…` / `+84…` / `84…` / `0084…` và khoảng trắng, gạch, chấm
  - `phone_utils.dart` soi gương lại luật của server; thông báo lỗi kèm ví dụ số hợp lệ
  - 11 test mới (5 Go + 6 Dart) gồm toàn bộ đầu số nhà mạng đang phát hành và các ca từng lọt
- **Workdocs:** `docs/workdocs_validate_dau_so_di_dong_vn_12082026/`
- **Liên quan:** rà soát luồng đăng nhập OTP 12/08/2026

## [2026-08-12] Order Desk đọc thông báo bằng giọng tiếng Việt

- **Loại:** fix
- **Phạm vi:** `apps/mobile` (Order Desk)
- **Tóm tắt:** Thông báo «Bạn có N đơn chưa giao» phát bằng giọng tiếng Anh. Trình duyệt nạp danh sách giọng bất đồng bộ nên lần tìm đầu tiên gặp list rỗng, mà code lại đánh dấu đã sẵn sàng và không bao giờ thử lại — kẹt vĩnh viễn ở giọng mặc định.
- **Chi tiết:**
  - Không cache kết quả tìm giọng thất bại; mỗi lần thông báo thử lại một lần
  - `prewarm()` khi mở Order Desk, poll ~2s cho danh sách giọng của trình duyệt
  - Xếp hạng giọng: locale `vi-VN` > `vi*` > tên chứa "vietnam"/"việt" (mỗi engine ghi nhãn một kiểu)
  - Cấu hình Order Desk có nút «Nghe thử giọng đọc», hiện giọng đang dùng hoặc hướng dẫn cài giọng Việt khi máy chưa có
- **Workdocs:** `docs/workdocs_giong_doc_tieng_viet_order_desk_12082026/`
- **Liên quan:** phản hồi cửa hàng 12/08/2026

## [2026-08-12] Chẩn đoán được lỗi «Không trừ được tồn kho» khi đặt hàng

- **Loại:** fix + feature
- **Phạm vi:** `order-service`, `inventory-service`, `catalog-service`, `apps/mobile` (tồn kho), `scripts`, `deploy`
- **Tóm tắt:** Checkout gọi inventory đồng bộ, nhưng khi gọi hỏng thì mọi container vẫn `healthy` và không có chỗ nào cho biết order-service đang quay số vào URL nào — sự cố chỉ lộ ra khi khách đặt đơn thất bại. Nay `/readyz` của order-service báo cáo từng upstream kèm URL thật, và màn Nhập kho chọn sản phẩm từ danh mục thay vì gõ tay `product_id`.
- **Chi tiết:**
  - `/readyz` probe `GET <base>/healthz` của geo, catalog, billing, inventory; 503 kèm tên dependency hỏng
  - Lỗi và log của inventory client kèm base URL — `127.0.0.1:8085` trong container chính là chữ ký của việc thiếu `INVENTORY_SERVICE_URL`
  - Phủ test lần đầu cho nhánh reserve của `handleCreateOrder`: happy path, 409 → `INSUFFICIENT_STOCK`, không gọi được → `INVENTORY_UNAVAILABLE`
  - `vps-api-diagnose.sh` in env `*_SERVICE_URL` của order-service, gọi `/readyz` và lọc log `inventory reserve`
  - `/healthz` giữ nguyên là liveness thuần — không đưa upstream vào compose healthcheck
  - Nhập kho: dropdown sản phẩm từ `GET /v1/admin/products`, tự điền `product_id`/`sku`/`name`; ẩn sản phẩm đã có dòng tồn; catalog lỗi thì quay về nhập tay kèm cảnh báo
  - inventory-service consume `catalog.product.updated` (durable `inventory-catalog-product-updated`, stream `CATALOG`): sản phẩm mới tự có dòng tồn `on_hand=0`, đổi tên/SKU đồng bộ sang kho mà không đụng số lượng hay giá vốn
  - Payload `catalog.product.updated` thêm `name` — §5.1 giao inventory "đồng bộ tên/sku" nhưng payload cũ thiếu name; thêm field tương thích ngược nên `schema_version` giữ 1
  - Xung đột SKU chỉ ghi log ERROR + đánh dấu đã xử lý, không Nak vô hạn thành poison message
- **Workdocs:** `docs/workdocs_chan_doan_khong_tru_ton_kho_12082026/`
- **Liên quan:** sự cố staging `tamde-stag` 12/08/2026; nối tiếp #38

## [2026-08-12] Thêm CLAUDE.md hướng dẫn Claude Code

- **Loại:** docs
- **Phạm vi:** root repo
- **Tóm tắt:** Tạo `CLAUDE.md` tóm tắt lệnh phát triển, kiến trúc gateway/service/event và các cạm bẫy vận hành để agent nắm ngữ cảnh nhanh.
- **Chi tiết:**
  - Nhắc lại quy trình bắt buộc CHANGESLOG + workdocs và quy ước merge vào nhánh `stag`
  - Ghi rõ `/healthz` vs `/readyz`, đồng bộ `JWT_SECRET`, tách `docker-compose.yml` (VPS) và `docker-compose.local.yml`
- **Workdocs:** `n/a` (chỉ thêm file hướng dẫn)
- **Liên quan:** `.cursor/rules/change-workdocs.mdc`

## [2026-08-09] Fix đặt hàng không kết nối được dịch vụ tồn kho

- **Loại:** fix
- **Phạm vi:** `deploy`, `order-service` → `inventory-service`
- **Tóm tắt:** `order-service` trong container thiếu `INVENTORY_SERVICE_URL` nên gọi fallback `127.0.0.1:8085` và mọi đơn hàng báo không thể trừ tồn kho. Cấu hình nay dùng đúng DNS nội bộ `inventory-service:8085`.
- **Chi tiết:**
  - Order chờ healthcheck inventory khi stack khởi động
  - Thêm test hồi quy khóa URL inventory trong Compose
- **Workdocs:** `docs/workdocs_fix_ket_noi_tru_ton_kho_09082026/`
- **Liên quan:** lỗi đặt hàng trên staging ngày 09/08/2026

## [2026-08-07] Hiển thị ảnh và grid sản phẩm responsive

- **Loại:** fix + feature
- **Phạm vi:** `apps/mobile` (catalog, cửa hàng, đặt hàng)
- **Tóm tắt:** `image_url` trước đây đã được lưu và trả về từ API nhưng giao diện chỉ vẽ biểu tượng bình gas, nên thêm URL vẫn không thấy ảnh. Các màn sản phẩm nay tải ảnh thật và hiển thị dạng grid responsive.
- **Chi tiết:**
  - Widget ảnh dùng chung có loading và fallback an toàn khi URL trống/sai hoặc host ảnh lỗi
  - Cửa hàng khách, bước chọn sản phẩm và danh sách quản trị tự đổi số cột theo chiều rộng
  - Form quản trị kiểm tra URL HTTP/HTTPS và hướng dẫn dùng link ảnh trực tiếp
  - Widget test bao phủ URL hợp lệ và fallback
- **Workdocs:** `docs/workdocs_hien_anh_va_grid_san_pham_07082026/`
- **Liên quan:** phản hồi staging ngày 07/08/2026

## [2026-08-07] Fix 401 mọi route có JWT (auth-service thiếu JWT_SECRET) + đăng nhập admin bằng SĐT

- **Loại:** fix + feature
- **Phạm vi:** `deploy` (compose env), `services/auth-service`, `services/api-gateway`, `apps/mobile`
- **Tóm tắt:** `auth-service` ký access token nhưng compose không truyền `JWT_SECRET` cho nó, nên trên deploy có secret thật gateway từ chối **mọi** token vừa phát hành — Hồ sơ báo `invalid or expired access token`, Đơn hàng báo "Phiên đăng nhập hết hạn", và refresh phía client không cứu được. Đồng thời thêm allow-list số điện thoại admin: `0909777020` đăng nhập OTP là vào thẳng trang quản trị và tự thêm/bớt được số khác.
- **Chi tiết:**
  - `auth-service` nhận `JWT_SECRET`, TTL token, giới hạn OTP và biến seed admin; cả bên ký lẫn bên xác minh log `jwt_secret_fp` (8 hex của SHA-256) để so lệch secret mà không lộ secret
  - `deploy/compose_env_test.go` chạy trong `make test`, chặn tái diễn; `PHONE_ENC_KEY`/`PHONE_HASH_PEPPER` cố tình vẫn để trống vì đổi pepper sẽ mồ côi user cũ trong `auth.db`
  - Bảng `admin_phones` khoá bằng cùng peppered hash với `users.phone_hash`, seed từ `ADMIN_PHONES` (mặc định `0909777020`); OTP verify cấp `role=admin` cho số trong danh sách
  - `refresh` đọc lại allow-list mỗi lần xoay token → thêm/bớt số có hiệu lực ngay ở lần refresh kế tiếp; không xoá được entry cuối cùng
  - `GET/POST/DELETE /v1/admin/admin-phones` (auth-service, gateway route admin) + màn **Quản trị → Số điện thoại admin**
  - Client xoá session khi 401 vẫn còn sau refresh + retry, để router đưa về đăng nhập thay vì hiện lỗi tiếng Anh
- **Workdocs:** `docs/workdocs_fix_jwt_secret_va_admin_theo_sdt_07082026/`
- **Liên quan:** phản hồi staging ngày 07/08/2026, tiếp nối PR #33

## [2026-08-07] Khôi phục phiên khách khi mở hồ sơ/đơn hàng và điều hướng đảo chiều

- **Loại:** fix
- **Phạm vi:** `apps/mobile` (auth session, API client, router)
- **Tóm tắt:** Access token hết hạn được làm mới trước request hoặc sau 401 và request được thử lại một lần, nên phiên còn refresh token hợp lệ không còn báo hết hạn khi vào Hồ sơ hay Đơn hàng của tôi. Điều hướng tiến/lùi giữ stack để animation chạy đúng chiều.
- **Chi tiết:**
  - Gom refresh đồng thời để không dùng hai lần refresh token xoay vòng; refresh endpoint không tự retry.
  - Route mở màn dùng `push()`, route quay lại dùng `pop()` với fallback cho deep link.
- **Workdocs:** `docs/workdocs_fix_phien_va_dieu_huong_07082026/`
- **Liên quan:** phản hồi staging ngày 07/08/2026

## [2026-08-07] OTP UI: một hàng 6 ô (bỏ ô nhập thứ hai)

- **Loại:** fix
- **Phạm vi:** `apps/mobile` (`OtpEntryBlock`)
- **Tóm tắt:** Màn OTP hiển thị trùng (6 ô + ô «Nhập 6 số OTP»). Giữ hàng 6 ô, field thật phủ lên (text alpha 0.01, full chiều cao) — bàn phím vẫn mở nhờ luồng `CustomerAuthFlowPage` + `Listener`.
- **Workdocs:** n/a (chỉnh UI nhỏ)

## [2026-08-07] Fix bàn phím OTP (mobile web): luồng một màn + ô nhập hiển thị

- **Loại:** fix
- **Phạm vi:** `apps/mobile` (`CustomerAuthFlowPage`, `OtpEntryBlock`, router)
- **Tóm tắt:** Sau PR #31 bàn phím vẫn không mở trên thiết bị thật vì chuyển màn OTP **sau** `await requestOtp()` làm mất user-gesture; input trong suốt trên Safari cũng hay bị chặn. Gộp bước SĐT + OTP trong `IndexedStack`, focus OTP đồng bộ khi bấm «Gửi mã OTP», API gửi OTP chạy sau; thêm ô «Nhập 6 số OTP» hiển thị rõ.
- **Chi tiết:**
  - `CustomerAuthFlowPage`: hai bước luôn mount, đổi bước không qua `go_router` → giữ chuỗi gesture mobile web
  - `OtpEntryBlock`: hàng ô số + `TextField` thật (không trong suốt), `Listener.onPointerDown` focus khi chạm ô
  - Router `/auth/phone` và `/auth/otp` trỏ flow mới; `OtpNavArgs.requestOtpOnMount` cho deep link
  - Dòng test CI/CD đã bỏ từ trước trong source — staging cần deploy image mới
- **Workdocs:** `docs/workdocs_fix_otp_ban_phim_va_load_cham_06082026/` (bổ sung)
- **Liên quan:** PR #31, feedback sau deploy

## [2026-08-06] Fix bàn phím màn OTP + load trang chậm ~30s

- **Loại:** fix
- **Phạm vi:** `apps/mobile` (auth screens, startup), `apps/mobile/web/index.html`, `deploy/Dockerfile.web`, `deploy/nginx.web.conf`
- **Tóm tắt:** Màn OTP không mở được bàn phím trên mobile web vì field thật nằm trong `SizedBox(height: 0)`; trang đăng nhập chờ ~30s vì startup `await` refresh token với Dio timeout 15s + 15s và first load kéo CanvasKit từ gstatic sau một `index.html` trống.
- **Chi tiết:**
  - 6 ô số thành decoration dưới một `TextField` trong suốt phủ kín → tap focus trực tiếp, bàn phím mở; `autofocus` giữ bàn phím từ bước 1; đủ 6 số tự xác nhận
  - `AuthScrollBody` cho màn OTP / SĐT / admin login: bàn phím không còn làm overflow layout hay che ô nhập
  - `bootstrap()` publish session đã lưu trước, refresh chỉ chặn UI tối đa 4s và tiếp tục chạy nền
  - Build `--no-web-resources-cdn` (CanvasKit từ origin của mình) + nén sẵn `brotli -q 11`/`gzip -9` lúc build, serve bằng `brotli_static`/`gzip_static`; runtime image đổi sang nginx của Alpine vì image `nginx` chính thức không có module brotli
  - First load: 3.26 MB → **2.48 MB**, first frame @4 Mbps 7.3s → **5.8s** (đo bằng CDP trên image production)
  - `index.html`: splash logo + spinner xoá khi `flutter-first-frame`, preload `main.dart.js`, preconnect `fonts.gstatic.com`
  - `/healthz` dùng `default_type` (trước đó trả hai header `Content-Type`)
  - Bỏ dòng test “🚀 Test CI/CD tự động — GCP stag v2” trên màn đăng nhập
  - Thêm `apps/mobile/test/otp_page_test.dart` (field có kích thước, tap focus, auto-verify, layout khi bàn phím mở)
- **Workdocs:** `docs/workdocs_fix_otp_ban_phim_va_load_cham_06082026/`
- **Liên quan:** báo cáo staging `tamde-stag.tinhgon.xyz`

## [2026-08-05] Fix GCP SSH deploy: publickey auth rejected

- **Loại:** fix
- **Phạm vi:** `.github/actions/ssh-deploy-gcp`, `scripts/ci-normalize-ssh-key.sh`, `deploy-gcp-stag.yml`, `web-image.yml`
- **Tóm tắt:** Job CD fail `ssh: unable to authenticate, attempted methods [none publickey]`. Secrets đã có nhưng key format/khớp VM sai (trước đó từng thiếu secret). CI normalize literal `\n`, validate private key, dùng `key_path`; docs hướng dẫn ed25519 + `authorized_keys`.
- **Chi tiết:**
  - Composite `.github/actions/ssh-deploy-gcp` dùng chung cho web + backend deploy
  - Optional secret `GCP_VM_SSH_PASSPHRASE`
  - Cảnh báo RSA PEM (drone-ssh hay reject `ssh-rsa`)
- **Workdocs:** `docs/workdocs_fix_gcp_ssh_auth_05082026/`
- **Liên quan:** GHA runs `31019115236`, `31025224189`

## [2026-08-05] CD: deploy-gcp-stag.yml — SSH deploy sau web-image build

- **Loại:** ci/cd
- **Phạm vi:** `.github/workflows/deploy-gcp-stag.yml`, `web-image.yml`
- **Tóm tắt:** Thêm workflow CD SSH vào GCP VM sau khi `web-image` push `:stag`. Secrets: `GCP_VM_HOST`, `GCP_VM_USER`, `GCP_VM_SSH_KEY`.

## [2026-08-05] Web image: nhúng api-gateway (sidecar) cho VPS một container

- **Loại:** fix deploy
- **Phạm vi:** `deploy/Dockerfile.web`, `deploy/nginx.web.conf`, `deploy/docker-entrypoint.web.sh`, compose `web`
- **Tóm tắt:** Staging vẫn `api_unavailable` vì platform không chạy container `api-gateway` riêng. nginx proxy `127.0.0.1:8081`; gateway chạy trong container `web`. OTP vẫn cần `auth-service` + `nats`.
- **Workdocs:** cập nhật `docs/workdocs_vps_deploy_khong_ssh_05082026/`

## [2026-08-05] Staging không SSH: Traefik route `/v1` → api-gateway

- **Loại:** fix deploy
- **Phạm vi:** `deploy/docker-compose.yml`, README, workdocs VPS không SSH
- **Tóm tắt:** Site staging chỉ serve static Flutter (nginx 1.29); `/v1` trả index.html / POST 405. Thêm label Traefik `PathPrefix(/v1)` + `/gateway-healthz` tới api-gateway; hướng dẫn redeploy qua UI không SSH.
- **Workdocs:** `docs/workdocs_vps_deploy_khong_ssh_05082026/`

## [2026-08-05] Fix OTP api_unavailable: mọi service trên tensorship-net

- **Loại:** fix deploy
- **Phạm vi:** `deploy/docker-compose.yml`, Makefile, scripts VPS diagnose
- **Tóm tắt:** nginx trong `web` không tới được `api-gateway` khi hai container không cùng proxy network Traefik. Join stack vào `tensorship-net` (external); healthcheck web qua `/gateway-healthz`; `make vps-api-diagnose`.
- **Workdocs:** `docs/workdocs_web_gateway_proxy_net_05082026/`

## [2026-08-05] VPS: bỏ `build:` khỏi compose chính — tránh containerd CreateDiff

- **Loại:** fix deploy
- **Phạm vi:** `deploy/docker-compose.yml`, `deploy/docker-compose.local.yml`, `scripts/vps-compose-up.sh`, Makefile, README
- **Tóm tắt:** Stage 5 `compose build` trên VPS vỡ containerd khi build Flutter/web. Compose VPS chỉ còn `image: ...:stag`; build context chuyển sang `.local.yml` cho dev. Stage build → `No services to build`; runtime pull GHCR.
- **Workdocs:** `docs/workdocs_vps_no_compose_build_05082026/`

## [2026-08-05] Fix OTP: web stack phụ thuộc auth + gateway

- **Loại:** fix
- **Phạm vi:** `deploy/docker-compose.yml`, Makefile, Flutter auth UX, README
- **Tóm tắt:** Lỗi «API gateway chưa sẵn sàng» khi gửi OTP = nginx không tới được `api-gateway`. `make web-up` giờ bật `nats`, `auth-service`, `api-gateway`, `web`; compose thêm `depends_on` healthy cho OTP path.
- **Workdocs:** `docs/workdocs_web_otp_gateway_deps_05082026/`

## [2026-08-04] Fix Stage 8 Unreachable: Traefik target = `web:8080`

- **Loại:** fix
- **Phạm vi:** `deploy/nginx.web.conf`, `deploy/docker-compose.yml`, `services/api-gateway`, Dockerfile.web
- **Tóm tắt:** Sau khi bỏ `ports` publish trên gateway, Stage 8 vẫn `Unreachable labeledPort=8080` (08:33). Chuyển Traefik sang service **web** (nginx lắng nghe `8080` ngay, `/healthz` không phụ thuộc gateway); gateway bind trước khi mở SQLite; không publish host port trên compose chính.
- **Chi tiết:**
  - nginx `listen 8080`; `/healthz` + `/web-healthz` trả JSON local; `/gateway-healthz` proxy API
  - Traefik labels trên `web` (`server.port=8080`, `tensorship-net`); bỏ labels ở api-gateway
  - Main compose: không `ports` cho nats/gateway/web (`expose` thôi); local map ở `.local.yml` (`8090:8080`)
  - Gateway: `atomicHandler` — ListenAndServe ngay với `/healthz`, swap full router sau SQLite
- **Coolify UI:** Ports Exposes vẫn **8080** (giờ = nginx/web)
- **Workdocs:** `docs/workdocs_stage8_traefik_web_front_04082026/`
- **Liên quan:** Stage 8 Unreachable; nối tiếp traefik-no-publish / NotOnNet

## [2026-08-04] Fix Stage 8 Unreachable: không publish host port gateway

- **Loại:** fix
- **Phạm vi:** `deploy/docker-compose.yml`, `deploy/docker-compose.local.yml`, `services/api-gateway`, Makefile
- **Tóm tắt:** Stage 8 vẫn `Unreachable labeledPort=8080` sau khi bỏ custom network. Nguyên nhân còn lại: `ports: "8080:8080"` trên api-gateway tạo thêm endpoint publish/ingress; Traefik hay chọn IP đó thay vì IP trên `tensorship-net`. Bỏ publish host trên compose chính (`expose: "8080"` thôi), hardcode `traefik.docker.network=tensorship-net`, bỏ `depends_on` gateway, local ports chuyển sang `docker-compose.local.yml`.
- **Chi tiết:**
  - Main compose: không `ports` cho api-gateway; `expose: ["8080"]`
  - Label network literal `tensorship-net` (không `${VAR}` — platform dễ bỏ qua expansion)
  - Gateway không `depends_on` → listen ngay khi container start
  - Gateway: DB lỗi vẫn listen `/healthz` (Traefik TCP :8080 vẫn xanh)
  - `deploy/docker-compose.local.yml` + Make/dev.ps1 merge file này cho DX local
- **VPS:** chỉ dùng `docker-compose.yml` (không mount file `.local.yml`); redeploy sau CI push `api-gateway:stag`
- **Workdocs:** `docs/workdocs_stage8_gateway_listen_04082026/traefik-no-publish.md`
- **Liên quan:** Stage 8 Unreachable; nối tiếp NotOnNet / network fix

## [2026-08-04] Fix Stage 8 `NotOnNet`: nginx crash loop + push image `web`

- **Loại:** fix
- **Phạm vi:** `deploy/nginx.web.conf`, `deploy/docker-compose.yml`, `.github/workflows/web-image.yml`, `scripts/vps-net-check.sh`, `Makefile`, `README.md`
- **Tóm tắt:** Deploy fail `HEALTHCHECK FAILED cause=NotOnNet labeledPort=8080` kèm `failed to connect container ... to tensorship-net`. nginx dùng hostname literal trong `proxy_pass` nên resolve `api-gateway` **lúc parse config** và thoát (`host not found in upstream`) khi gateway chưa lên; container crash loop ở trạng thái `restarting` thì `docker network connect` fail → NotOnNet. Ngoài ra image `gas-tam-de/web:stag` chưa từng được CI push nên `up --no-build` không pull được website.
- **Chi tiết:**
  - nginx: `resolver 127.0.0.11 ipv6=off valid=10s` + `proxy_pass $api_gateway$request_uri`; gateway chết → `503` JSON (`@api_unavailable`) thay vì nginx chết
  - `web.depends_on.api-gateway`: `service_healthy` → `service_started`
  - Label `traefik.docker.network=${PROXY_NETWORK:-tensorship-net}` cho api-gateway (container nằm trên network default + proxy net)
  - Workflow mới `web-image.yml`: build/push `ghcr.io/<owner>/gas-tam-de/web:stag`, có `workflow_dispatch`
  - `scripts/vps-net-check.sh` (+ `make vps-net-check` / `vps-net-fix`): liệt kê container thiếu proxy network, `--fix` attach lại, phân biệt crash loop
- **VPS:** merge để CI push `web:stag`, redeploy; nếu còn NotOnNet chạy `./scripts/vps-net-check.sh --fix`
- **Workdocs:** `docs/workdocs_stage8_notonnet_web_image_04082026/`
- **Liên quan:** Stage 8 Health Check; nối tiếp fix “Traefik Unreachable”

## [2026-08-04] Fix Stage 8 Traefik Unreachable: bỏ custom network Coolify

- **Loại:** fix
- **Phạm vi:** `deploy/docker-compose.yml`, `.github/workflows/backend-ci.yml`
- **Tóm tắt:** Stage 8 vẫn fail `HEALTHCHECK FAILED cause=Unreachable labeledPort=8080` dù gateway đã bind `0.0.0.0:8080`. Coolify/Traefik chỉ gắn vào network do platform tạo; compose định nghĩa `networks.gastamde` khiến container nằm 2 network và Traefik resolve IP sai. Xóa custom network + label `loadbalancer.server.port=8080`. Đồng thời sửa CI: `secrets.*` trong `if:` làm workflow không chạy (0s failure) nên image `:stag` không được push.
- **Chi tiết:**
  - Xóa mọi `networks: [gastamde]` và block `networks.gastamde`
  - `api-gateway` labels: `traefik.enable=true`, `traefik.http.services.api-gateway.loadbalancer.server.port=8080`
  - CI: không dùng `secrets` trong `if:`; check `GHCR_WRITE_TOKEN` trong script
  - Trigger CI khi đổi `deploy/docker-compose.yml`
- **Coolify UI:** Ports Exposes / public service = **8080** (api-gateway)
- **Workdocs:** `docs/workdocs_stage8_gateway_listen_04082026/coolify-traefik.md`
- **Liên quan:** Stage 8; Coolify “Do Not Define Custom Networks”

## [2026-08-04] Fix Stage 8 Health Check: gateway listen `0.0.0.0:8080` sớm

- **Loại:** fix
- **Phạm vi:** `pkg/httpx`, `deploy/docker-compose.yml`, `deploy/Dockerfile.service`, `.env*.example`
- **Tóm tắt:** Cursor Cloud Stage 8 (Health Check) fail — “không lắng nghe 8080 / bind 127.0.0.1”. Stage 5 chạy `up -d` không `--wait` trong khi gateway `depends_on: service_healthy` 8 backend nên chưa listen khi platform probe `host:8080`. Fix: start gateway ngay khi container backend tồn tại, pin `PORT`/`0.0.0.0:8080`, rewrite loopback trong httpx.
- **Chi tiết:**
  - `api-gateway` depends_on: `service_healthy` → `service_started`; healthcheck `start_period` 90s→20s
  - Compose: `PORT=8080`, `GATEWAY_ADDR=0.0.0.0:8080`
  - Dockerfile.service: `ENV PORT=${EXPOSE_PORT}` + `EXPOSE`
  - `NormalizeListenAddr`: `127.0.0.1`/`localhost` → `0.0.0.0`
  - `.env.vps.example` / `.env.example`: thêm `PORT=8080`
- **VPS:** thêm `PORT=8080` vào Environment; không set listen `127.0.0.1`. Redeploy sau khi CI push image mới.
- **Workdocs:** `docs/workdocs_stage8_gateway_listen_04082026/`
- **Liên quan:** Stage 8 Health Check; nối tiếp Stage 5 override YAML

## [2026-08-04] VPS env: template tối thiểu + checklist sửa Stage 5

- **Loại:** fix / docs
- **Phạm vi:** `deploy/.env.vps.example`, `deploy/.env.example`, `scripts/check-env-yaml-safe.sh`
- **Tóm tắt:** `.env` trên VPS vẫn còn `GEOCODE_USER_AGENT=... contact: local`, `API_BASE_URL` rác, `NATS_URL`/`*_SERVICE_URL=127.0.0.1`, `IMAGE_PREFIX`, `SERVICE=api-gateway` — đủ để Stage 5 vỡ YAML hoặc container mất DNS Docker. Thêm template tối thiểu + checklist thay env trên VPS.
- **Chi tiết:**
  - `deploy/.env.vps.example`: chỉ secret/override cần thiết; không set `NATS_URL` / `*_SERVICE_URL` / `*_DB` / `IMAGE_*`
  - `GEOCODE_USER_AGENT=... contact=local` (không `: `)
  - Script check strip quote dotenv rồi quét cả `.env.example` + `.env.vps.example`
  - Workdocs checklist: `docs/workdocs_compose_override_yaml_04082026/vps-env.md`
- **Việc cần làm trên VPS:** thay project Environment bằng nội dung `.env.vps.example` (điền secret), xóa các key độc hại ở trên, redeploy
- **Workdocs:** `docs/workdocs_compose_override_yaml_04082026/`
- **Liên quan:** Stage 5 Run Container

## [2026-08-04] Fix Stage 5: override YAML vỡ vì `GEOCODE_USER_AGENT` chứa `: `

- **Loại:** fix
- **Phạm vi:** `deploy/.env.example`, `services/geo-service`, `scripts/check-env-yaml-safe.sh`, CI
- **Tóm tắt:** Cursor Cloud Stage 5 (Run Container) fail parse `docker-compose.override.yml` với `did not find expected key` (khoảng dòng 19/23). Platform copy `KEY=VALUE` từ `.env.example` thành YAML **không quote**; `GEOCODE_USER_AGENT=... contact: local` chứa `: ` nên bị hiểu là nested mapping. Đổi thành `contact=local` và thêm guardrail CI/Make.
- **Chi tiết:**
  - `GEOCODE_USER_AGENT` / `defaultUserAgent`: `contact: local` → `contact=local` (không còn colon+space)
  - Ghi chú constraint Stage 5 trong `deploy/.env.example`
  - `scripts/check-env-yaml-safe.sh` + `make check-env-yaml` (+ `dev.ps1`): fail sớm nếu value có `: `
  - Backend CI: chạy check trong job test; trigger thêm khi đổi `.env.example` / script
  - Lưu ý: nếu project settings Cursor Cloud vẫn inject env cũ có `: `, xóa/sửa biến đó trên UI
- **Workdocs:** `docs/workdocs_compose_override_yaml_04082026/`
- **Liên quan:** Stage 5 Run Container; nối tiếp hardcode GHCR image names / CI push 2026-08-04

## [2026-08-04] CI backend: build + push images to GHCR; VPS dùng pre-built image

- **Loại:** fix / ci
- **Phạm vi:** `.github/workflows/backend-ci.yml`, `deploy/docker-compose.yml`, `deploy/.env.example`
- **Tóm tắt:** VPS deploy dùng `--no-build` nên cần image đã được build sẵn. Không có CI nào build Go service images → VPS reuse image cũ từ trước khi có fix background NATS → `catalog-service` và `billing-service` vẫn unhealthy dù code đã sửa. Fix: thêm GitHub Actions CI build mọi service image và push lên GHCR sau mỗi push vào `stag`; VPS chỉ cần set `IMAGE_PREFIX`/`IMAGE_TAG` để pull đúng image mới.
- **Chi tiết:**
  - `.github/workflows/backend-ci.yml`: trigger on push to `stag`/`master` hoặc PR chạm `services/**`, `pkg/**`, `go.mod`, Dockerfile. Job 1: `go test ./...`. Job 2 (parallel per service × 8): build image + push `ghcr.io/haoma24/gas-tam-de/<svc>:stag` + `:sha-<commit>` + `:latest` (chỉ trên stag). Dùng GHA layer cache (`type=gha`) nên build nhanh sau lần đầu.
  - `deploy/docker-compose.yml`: comment cập nhật hướng dẫn `IMAGE_PREFIX`/`IMAGE_TAG`
  - `deploy/.env.example`: thêm section VPS rõ ràng với giá trị mặc định `IMAGE_PREFIX=ghcr.io/haoma24/gas-tam-de` và `IMAGE_TAG=stag`
- **Biến môi trường cần thêm trên VPS** (trong `deploy/.env` hoặc host env):
  ```
  IMAGE_PREFIX=ghcr.io/haoma24/gas-tam-de
  IMAGE_TAG=stag
  ```
- **Verify:** CI chạy → GHCR có image mới → VPS `docker compose -p ts-tamde-stag up -d --no-build` pull image mới → containers healthy
- **Liên quan:** Nối tiếp toàn bộ chuỗi fix healthcheck stag 2026-08-04

## [2026-08-04] Fix deploy stag chạy image cũ: pin `image:` cho mọi service

- **Loại:** fix
- **Phạm vi:** `deploy/docker-compose.yml`
- **Tóm tắt:** Deploy stag vẫn fail `container ts-tamde-stag-catalog-service-1 is unhealthy` **dù source đã có fix**. Nguyên nhân: service chỉ khai báo `build:` mà không có `image:`, nên compose tự đặt tên image theo project name (`-p foo` ⇒ `foo-catalog-service`). Stage build và stage `up --no-build` truyền `-p` khác nhau ⇒ stage run lấy đúng image **cũ** của lần deploy trước, fix không bao giờ vào container.
- **Chi tiết:**
  - Mọi service build (`api-gateway`, `auth`, `catalog`, `geo`, `order`, `inventory`, `billing`, `report`, `web`) pin `image: ${IMAGE_PREFIX:-gas-tam-de}/<svc>:${IMAGE_TAG:-latest}`
  - Tên image giờ độc lập với `-p`, nên build stage và run stage luôn trỏ cùng một image
  - Nếu pipeline quên build, `up --no-build` fail ngay với `No such image` thay vì âm thầm chạy image cũ
  - `IMAGE_PREFIX` / `IMAGE_TAG` override được khi push registry hoặc pin version
- **Verify (Docker thật):**
  - Reproduce: image build từ commit trước fix ⇒ không có log `listening`, healthcheck `Connection refused` (exit 1) — đúng lỗi VPS
  - Image build từ stag hiện tại ⇒ `listening addr=0.0.0.0:8082 network=tcp4`, healthcheck exit 0
  - Build `-p buildstage` rồi chạy `docker compose -p ts-tamde-stag up -d --no-build --remove-orphans` (đúng lệnh VPS) ⇒ exit 0, **9/9 container healthy**, `/readyz` catalog = `{"nats":"ok"}`
- **Liên quan:** Deploy stag `ts-tamde-stag`; nối tiếp 2 fix healthcheck 2026-08-04

## [2026-08-04] Service serve HTTP ngay, không chờ NATS; thêm `/readyz`

- **Loại:** fix
- **Phạm vi:** `pkg/natsx`, `pkg/httpx`, `services/{catalog,order,inventory,billing,report}-service`
- **Tóm tắt:** Deploy stag fail `container ts-tamde-stag-catalog-service-1 is unhealthy`. `main()` chặn ở `ConnectJS` + `EnsureStreams` **trước khi** mount HTTP, nên khi NATS chậm thì `/healthz` chưa hề tồn tại → container unhealthy → cả `docker compose up` fail theo `depends_on`. Giờ HTTP serve ngay, NATS kết nối nền, dependency báo qua `/readyz`.
- **Chi tiết:**
  - `natsx.Background`: connect JetStream trong goroutine, retry vô hạn (backoff 1s → 30s); không `os.Exit` vì NATS
  - `natsx.JSProvider` / `natsx.Static`: publisher catalog/order/billing lấy JS lazy; publish khi chưa ready trả lỗi (call site vốn chỉ log, không fail request)
  - Consumer inventory + report attach qua `Start(onReady)` khi JetStream lên; lỗi → reconnect và thử lại
  - `httpx.MountReady` + `ReadyCheck`: `/readyz` 200 khi dependency OK, 503 + tên dependency lỗi; `/healthz` giữ nguyên là liveness cho healthcheck container
  - Test: `TestBackgroundBecomesReadyWhenBrokerStartsLate`, `TestBackgroundNotReadyBeforeBrokerArrives`, `TestStaticProvider`, `TestMountHealthIgnoresDependencies`, `TestMountReadyOKWhenDependenciesPass`
  - Verify: chạy catalog **không có NATS** → `healthy` + `/readyz` 503; bật NATS trễ → consumer start + `/readyz` ready; full stack 9/9 healthy
  - Lưu ý: deploy dùng `--no-build` ⇒ pipeline phải build lại image mới có fix
- **Workdocs:** `docs/workdocs_nats_background_readyz_04082026/`
- **Liên quan:** Deploy stag `ts-tamde-stag`; nối tiếp fix healthcheck 2026-08-03/04

## [2026-08-04] Fix geo-service unhealthy: bind IPv4 + wget + EXPOSE đúng cổng

- **Loại:** fix
- **Phạm vi:** `pkg/httpx`, `pkg/config`, `services/*`, `deploy`
- **Tóm tắt:** Deploy VPS fail `container ts-gas-tam-de-geo-service-1 is unhealthy`. Process có thể đang listen IPv6-only (`:8083` → `:::8083`) trong khi healthcheck probe `127.0.0.1` — connection refused. Chuẩn hóa bind `0.0.0.0`, cài `wget` trong image service, `EXPOSE` đúng port theo service, hỗ trợ env `PORT`.
- **Chi tiết:**
  - `httpx.NormalizeListenAddr`: `:8083` → `0.0.0.0:8083`; `ListenAndServe` dùng `tcp4` cho bind `0.0.0.0` (tránh fail trên host `net.ipv6.bindv6only=1`)
  - `config.ListenAddr`: `GEO_ADDR` (v.v.) → `PORT` → fallback; áp dụng 8 Go service
  - `Dockerfile.service`: `apk add wget` (healthcheck không phụ thuộc busybox applet); `ARG EXPOSE_PORT` thay hardcode `EXPOSE 8080`
  - compose: truyền `EXPOSE_PORT` (geo `8083`, …)
  - Test: `TestNormalizeListenAddr`, `TestListenAddrOrder`
- **Workdocs:** `docs/workdocs_geo_healthcheck_ipv4_04082026/`
- **Liên quan:** Deploy VPS `ts-gas-tam-de`; nối tiếp fix NATS unhealthy 2026-08-03

## [2026-08-03] Service chờ NATS thay vì chết khi deploy VPS

- **Loại:** fix
- **Phạm vi:** `pkg/natsx`, `deploy/docker-compose.yml`, `Makefile`, docs
- **Tóm tắt:** Deploy VPS fail `dependency failed to start: container …-billing-service-1 is unhealthy`. Các service dùng NATS gọi `ConnectJS`/`EnsureStreams` **một lần rồi `os.Exit(1)`**; NATS chấp nhận TCP trước khi JetStream restore xong store nên trên host chậm/cold start service chết ngay lúc boot. Giờ retry có backoff và healthcheck cho đủ thời gian.
- **Chi tiết:**
  - `natsx.Connect` / `ConnectJS` / `EnsureStreams`: retry backoff (500ms → 5s) trong `NATS_STARTUP_TIMEOUT_SEC` (mặc định 60s, `0` = thử 1 lần); log `WARN waiting for nats what=… attempt=…`
  - `ConnectJS` chờ `PingJS` OK ⇒ JetStream thật sự sẵn sàng, không chỉ TCP
  - compose: `start_period` của service Go `5s → 90s` (phải > NATS timeout, nếu không container bị kết luận unhealthy trong lúc còn đang đợi); truyền `NATS_STARTUP_TIMEOUT_SEC` vào 8 service
  - `make doctor`: chỉ in container **không** healthy kèm output probe cuối + 40 dòng log — trả lời được "vì sao unhealthy"
  - Test: `TestConnectJSWaitsForBrokerStartedLate` (broker bật trễ 1.5s), `TestConnectFailsAfterBudgetWhenBrokerNeverArrives`, `TestRetryUntil*`
  - Verify: stop NATS → billing `running/starting` + log `waiting for nats` (trước đây `Exited`); start NATS → tự `healthy` sau ~20s
  - README: mục "Khi deploy báo `container … is unhealthy`"
- **Workdocs:** `docs/workdocs_web_service_compose_observability_03082026/`
- **Liên quan:** Deploy VPS `ts-gas-tam-de` fail; nối tiếp thay đổi healthcheck cùng ngày

## [2026-08-03] Website vào docker-compose + healthcheck cho toàn stack

- **Loại:** fix
- **Phạm vi:** `deploy`, `pkg/httpx`, `Makefile`, `scripts/dev.ps1`, docs
- **Tóm tắt:** Chẩn đoán runtime báo "log chỉ có NATS, không thấy dịch vụ website". Nguyên nhân: website **chưa bao giờ là service trong compose** (Flutter Web chỉ chạy trên host), và các service Go không có healthcheck/restart nên chết im lặng. Thêm service `web` (Flutter Web release sau nginx, `:8090`) và bật healthcheck + restart cho mọi service. Trên VPS, mở `:8080/` trả **404 là đúng** (API Gateway) — website ở `:8090` (hoặc `WEB_PORT=80`).
- **Chi tiết:**
  - `deploy/Dockerfile.web` + `deploy/nginx.web.conf`: build Flutter Web release → nginx; SPA fallback, gzip, `/web-healthz`
  - nginx reverse-proxy `/v1/*` + `/healthz` → `api-gateway:8080` ⇒ website same-origin, không cần CORS
  - compose: service `web` (`WEB_PORT`, `WEB_API_BASE_URL`, `FLUTTER_VERSION=3.44.0`); `healthcheck` + `restart: unless-stopped` cho cả 8 service Go; `api-gateway` chờ `service_healthy` thay vì `service_started`
  - `pkg/httpx`: bỏ access log cho `/healthz` khi status 200 (healthcheck 10s × 8 service làm chìm log thật); `/healthz` lỗi vẫn log
  - Make + PowerShell: `compose-ps`, `compose-logs`, `web-up`, `web-logs`, `web-health`, `stack-health`; `compose-up` giờ `-d --wait` rồi in trạng thái; nạp `deploy/.env` nếu có
  - `.dockerignore`: loại `data/`, `.git`, build output khỏi build context (SQLite runtime hay bust cache)
  - README mục "Truy cập trên VPS" + `deploy/.env.example` ghi rõ `WEB_PORT` / 404 trên `:8080`
  - Lưu ý: image `flutter:3.35.4` fail `pub get` vì `google_fonts 8.2.1` cần Dart `^3.10` → pin `3.44.0`
- **Workdocs:** `docs/workdocs_web_service_compose_observability_03082026/`
- **Liên quan:** Chẩn đoán AI Runtime (log chỉ hiển thị NATS); VPS 404 khi mở host `:8080`

## [2026-08-03] Fix Docker build: Go image khớp go.mod 1.25

- **Loại:** fix
- **Phạm vi:** `deploy/Dockerfile.service`, docs
- **Tóm tắt:** `docker compose build` fail vì image `golang:1.22` không thỏa `go.mod` (`go >= 1.25.0`, `GOTOOLCHAIN=local`). Nâng base image lên `golang:1.25-bookworm`.
- **Chi tiết:**
  - `deploy/Dockerfile.service`: `FROM golang:1.25-bookworm`
  - Đồng bộ README / architecture → Go 1.25+
- **Workdocs:** n/a (sửa config 1 dòng + docs)
- **Liên quan:** Docker compose build failure
## [2026-08-03] Gửi SMS OTP thật qua Stringee SMS REST

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy`
- **Tóm tắt:** Thay seam production (luôn `ErrSMSNotConfigured`) bằng client Stringee thật: `POST /v1/auth/otp/request` giờ gửi được OTP qua `POST https://api.stringee.com/v1/sms` với JWT `X-STRINGEE-AUTH`. Mock vẫn là default local.
- **Chi tiết:**
  - `StringeeSMSSender` — JWT HS256 (`cty=stringee-api;v=1`, `iss`/`jti`/`exp`/`rest_api`), body `{"sms":[{"from","to","text"}]}`, `to` dạng `84…`
  - Chọn adapter: `SMS_PROVIDER=stringee` hoặc `production` + `SMS_VENDOR=stringee`; eSMS giữ seam cũ
  - Env: `SMS_API_SID` / `SMS_API_SECRET` / `SMS_SENDER` (+ `SMS_API_URL`, `SMS_TIMEOUT_SEC`, `SMS_JWT_TTL_SEC`); fallback `SMS_API_KEY="sid:secret"`
  - Lỗi vendor (`r != 0`, `smsSent < 1`, non-2xx) → `ErrSMSRejected` → `502 SMS_FAILED`; thiếu credential → fail-closed, không gọi vendor
  - Không retry (tránh SMS trùng/tốn phí); log chỉ `phone_masked`, không log OTP/token
  - `docker-compose.yml` truyền `SMS_*` + `OTP_DEV_REVEAL` vào `auth-service`
- **Workdocs:** `docs/workdocs_sms_stringee_client_03082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.3, architecture §2 + §9.7

## [2026-08-03] Guest chỉ Đăng nhập; đơn hàng trong hồ sơ
## [2026-08-03] Sync PRD: E10 Customer UX + T5.1.x desk + T7.1.3 stock reserve

- **Loại:** docs
- **Phạm vi:** `docs/prd.md`
- **Tóm tắt:** Bổ sung E10 (brand shop, profile, lịch sử đơn, prefill) và cập nhật các task còn thiếu trong E5 (wait badge, TTS, desk settings), T7.1.3 (reserve/release logic), T9.2.6 (design system), user flow §3.1, MoSCoW should-have, Sprint 5.
- **Chi tiết:**
  - E10 mới: US-10.1–10.4 brand shop / hồ sơ / lịch sử / prefill
  - E5: T5.1.5–5.1.7 badge/TTS; US-5.3 desk settings
  - T7.1.3 ghi rõ reserve on placed / release on cancelled
  - §3.1 user flow cập nhật flow mới (guest → Đăng nhập → shop)
  - Sprint 5 bổ sung Customer UX
- **Workdocs:** n/a (docs sync)
- **Liên quan:** E10 / PR #4

- **Loại:** feature
- **Phạm vi:** `apps/mobile`
- **Tóm tắt:** Màn đầu chỉ còn nút **Đăng nhập** (OTP). Quản lý «Đơn hàng của tôi» chỉ vào từ hồ sơ sau login; bỏ tab Đơn trên shop bottom nav.
- **Chi tiết:**
  - `HomePage`: một CTA «Đăng nhập» → `/auth/phone`
  - Shop nav: Cửa hàng | Hồ sơ; `/orders/history` back → `/profile`
- **Workdocs:** `docs/workdocs_customer_shop_profile_03082026/`
- **Liên quan:** UX Home / hồ sơ khách

## [2026-08-03] Customer shop brand + hồ sơ + admin redirect theo role

- **Loại:** feature
- **Phạm vi:** `apps/mobile`
- **Tóm tắt:** Sau OTP khách vào trang cửa hàng mang cảm giác brand (hero + danh mục + CTA), có màn hồ sơ cá nhân; bỏ CTA «Dành cho cửa hàng» trên Home — session `role=admin` tự điều hướng `/admin`.
- **Chi tiết:**
  - Guest landing full-bleed brand; OTP verify → shop thay vì nhảy thẳng form đặt hàng
  - `CustomerShopPage` + `CustomerProfilePage` (`GET/PATCH /v1/me`, đơn của tôi, đăng xuất)
  - Router: admin trên `/` → `/admin`; login admin qua `/#/admin/login`
  - Theme Be Vietnam Pro (`google_fonts`)
- **Workdocs:** `docs/workdocs_customer_shop_profile_03082026/`
- **Liên quan:** UX Home / US-1.1 / US-1.2 / US-2.2

## [2026-08-02] Cursor Cloud env setup + OrderCart.isNotEmpty

- **Loại:** chore / fix / docs
- **Phạm vi:** `AGENTS.md`, `apps/mobile`, Cloud Agent DX
- **Tóm tắt:** Chuẩn bị môi trường Cloud (Go modules, Flutter pub, NATS/Docker, chạy full stack) và ghi chú vận hành; sửa thiếu getter `OrderCart.isNotEmpty` khiến Flutter Web không compile.
- **Chi tiết:**
  - `AGENTS.md` — Cursor Cloud specific instructions (NATS, services, Chrome `--no-sandbox`, hash routes)
  - `order_cart.dart` — thêm `bool get isNotEmpty => !isEmpty`
  - Verify: `make test`, API hello-world (OTP + place + complete), Flutter admin login → dashboard
- **Workdocs:** `workdocs_cursor_cloud_env_02082026/`
- **Liên quan:** DevEx / Cursor Cloud
## [2026-08-02] Desk wait badge / TTS interval / stock reserve / hủy đơn

- **Loại:** feature
- **Phạm vi:** order-service, inventory-service, api-gateway, apps/mobile
- **Tóm tắt:** Badge thời gian chờ màu xanh/cam/đỏ (ngưỡng admin); TTS VI «Bạn có N đơn chưa giao» theo chu kỳ; trừ tồn lúc đặt / hoàn khi hủy; tạo SP kèm phiếu nhập; khách xem lịch sử + hủy đơn.
- **Chi tiết:**
  - `GET/PUT /v1/admin/desk-settings`; Flutter desk settings + badge + interval TTS
  - `POST /v1/internal/stock/reserve|release`, `GET /v1/stock/levels`; place trừ tồn; complete không trừ lại
  - `POST /v1/orders/{id}/cancel`; Home **Đơn của tôi**
- **Workdocs:** `docs/workdocs_desk_stock_cancel_history_02082026/`
- **Liên quan:** Order Desk / Inventory / Customer orders

## [2026-08-02] TTS đơn mới + nhớ tên/địa chỉ theo SĐT

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `services/order-service`, `services/api-gateway`, `apps/mobile`
- **Tóm tắt:** Order Desk phát âm to «Bạn có đơn hàng mới»; khách lần đầu nhập tên lưu `users.full_name`, lần sau prefill tên + địa chỉ đơn gần nhất.
- **Chi tiết:**
  - Auth `GET/PATCH /v1/me`; Order `GET /v1/orders/me/defaults`; gateway customer `/me`
  - Flutter `flutter_tts` trên desk poll; review/address prefill + «Dùng địa chỉ lần trước»
- **Workdocs:** `docs/workdocs_order_alert_customer_profile_02082026/`
- **Liên quan:** Order Desk / place-order UX

## [2026-08-02] CORS allow X-User-* / X-Phone-Masked (Flutter Web)

- **Loại:** fix
- **Phạm vi:** `services/api-gateway`
- **Tóm tắt:** Preflight order/quote từ Flutter Web bị chặn vì `X-Phone-Masked` không nằm trong `Access-Control-Allow-Headers`.
- **Chi tiết:** Bổ sung `X-User-Id`, `X-User-Role`, `X-Phone-Masked` vào CORS allow-headers; cập nhật test.
- **Workdocs:** n/a (fix nhỏ)
- **Liên quan:** Flutter Web quote / place order

## [2026-08-02] Persist session + admin vị trí cửa hàng

- **Loại:** feature
- **Phạm vi:** `apps/mobile` (auth, geo, dashboard)
- **Tóm tắt:** Lưu JWT session qua `shared_preferences` (bootstrap + refresh token); Home CTA nhận session sẵn; màn admin **Vị trí cửa hàng** (`GET/PUT` geo store) + đăng xuất.
- **Chi tiết:**
  - `AuthSessionNotifier` + store; `POST /v1/auth/refresh` lúc mở app nếu access hết hạn
  - Desk tile `/admin/store`: tên, địa chỉ, lat/lng, bán kính, map pin, search, GPS
  - Logout xóa session
- **Workdocs:** `docs/workdocs_session_persist_store_admin_02082026/`
- **Liên quan:** Auth UX / T3.2.1 store settings

## [2026-08-02] Fix OrderCart.isNotEmpty (Flutter Web compile)

- **Loại:** fix
- **Phạm vi:** `apps/mobile/lib/features/order/order_cart.dart`
- **Tóm tắt:** Thêm getter `isNotEmpty` trên `OrderCart` — `order_review_page` dùng nhưng thiếu, làm `flutter run -d chrome` fail compile.
- **Workdocs:** n/a (fix nhỏ)
- **Liên quan:** order review / place order flow

## [2026-08-02] Chuyển toàn bộ workdocs vào docs/

- **Loại:** chore / docs
- **Phạm vi:** `docs/`, `.cursor/skills/change-workdocs`, `.cursor/rules`, `CHANGESLOG.md`
- **Tóm tắt:** Di chuyển 65 thư mục `workdocs_*` từ root vào `docs/`; cập nhật skill/rule/templates và mọi link Workdocs sang `docs/workdocs_*`.
- **Chi tiết:**
  - `git mv workdocs_* docs/`
  - Prefix path trong CHANGESLOG + README workdocs cũ
  - Quy ước mới: tạo workdocs tại `docs/workdocs_<mo-ta>_<ddmmyyyy>/`
- **Workdocs:** `docs/workdocs_move_workdocs_into_docs_02082026/`
- **Liên quan:** n/a

## [2026-08-02] Track Flutter pubspec.lock

- **Loại:** chore
- **Phạm vi:** `apps/mobile`
- **Tóm tắt:** Commit `pubspec.lock` để khóa phiên bản dependency Flutter/Dart, build tái lập được giữa máy dev và CI.
- **Chi tiết:**
  - Thêm `apps/mobile/pubspec.lock` (trước đó untracked)
- **Workdocs:** n/a (chore nhỏ — chỉ lockfile)
- **Liên quan:** Sprint 0 / apps/mobile

## [2026-08-02] Platform checklist Web + Android + iOS (T9.2.5)

- **Loại:** docs / chore
- **Phạm vi:** `apps/mobile`, `.github/workflows`, `docs`
- **Tóm tắt:** Đóng Sprint 0 platform — checklist không dùng API single-OS thiếu fallback; audit geolocator / url_launcher / flutter_map; verify Web + Android Emulator + iOS Simulator hoặc CI macOS; workflow Flutter analyze/test/web + iOS no-codesign (không secret).
- **Chi tiết:**
  - `apps/mobile/PLATFORM_CHECKLIST.md` + cập nhật README
  - `.github/workflows/flutter-ci.yml` (ubuntu analyze/test/web; macos `flutter build ios --no-codesign`)
  - Mark `- [DONE] T9.2.5` — mọi task `T*.*.*` trong PRD đã DONE
- **Workdocs:** `docs/workdocs_platform_checklist_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.5

## [2026-08-02] Flutter CTA shell Web + Android + iOS (T9.2.4)

- **Loại:** feature / docs
- **Phạm vi:** `apps/mobile`, `Makefile`, `scripts`, `docs`, root README
- **Tóm tắt:** Hoàn thiện CTA shell multi-platform — Home khách vs admin, scaffold `android`/`ios`/`web` đủ để chạy cùng codebase, DX targets + README hướng dẫn 3 target.
- **Chi tiết:**
  - Bổ sung Android res (themes, mipmap), debug/profile manifests, gradle-wrapper.properties
  - iOS: Flutter xcconfig, storyboards, Assets, hand-authored `Runner.xcodeproj` + workspace/scheme, GeneratedPluginRegistrant stubs
  - Web icons/favicon (incl. maskable); `.metadata` android/ios/web
  - README bảng CTA + lệnh `flutter run` Web / Android emulator / iOS Simulator
  - `make` / `dev.ps1`: `flutter-create`, `flutter-android`, `flutter-ios`
  - Mark `- [DONE] T9.2.4`; T9.2.5 còn lại (checklist verify)
- **Workdocs:** `docs/workdocs_flutter_cta_shell_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.4

## [2026-08-02] Gateway audit log admin actions (T9.1.4)

- **Loại:** security
- **Phạm vi:** `services/api-gateway`, `deploy`, `docs`
- **Tóm tắt:** Ghi audit mọi request admin mutating qua gateway — actor (JWT sub), method/path, thời điểm, HTTP status — vào SQLite `gateway.db` và structured log `admin_audit`.
- **Chi tiết:**
  - Middleware `AuditAdminMutations` sau JWT + RBAC admin; chỉ `POST`/`PUT`/`PATCH`/`DELETE`
  - Bảng `admin_audit_logs`; env `GATEWAY_DB` (default `data/gateway.db`)
  - Unit tests memory + SQLite; mark `- [DONE] T9.1.4`
- **Workdocs:** `docs/workdocs_gateway_admin_audit_02082026/`
- **Liên quan:** US-9.1 / T9.1.4

## [2026-08-02] Gateway security headers + ẩn internal error (T9.1.3)

- **Loại:** security
- **Phạm vi:** `services/api-gateway`, `pkg/httpx`, `docs`
- **Tóm tắt:** Thêm security headers trên mọi response gateway và đảm bảo lỗi proxy/panic trả JSON generic — không lộ URL nội bộ, dial error hay stack trace cho client.
- **Chi tiết:**
  - Middleware `SecurityHeaders` (nosniff, frame DENY, referrer, permissions-policy, CSP frame-ancestors)
  - Proxy `ErrorHandler` log server-side; client luôn `502 BAD_GATEWAY` / `upstream unavailable`; strip `Server` / `X-Powered-By`
  - `httpx.SafeRecover` thay chi Recoverer → `500 INTERNAL_ERROR` JSON
  - Unit tests headers + no-leak; mark `- [DONE] T9.1.3`
- **Workdocs:** `docs/workdocs_gateway_security_headers_02082026/`
- **Liên quan:** US-9.1 / T9.1.3

## [2026-08-02] Gateway rate limit OTP / login / place-order (T9.1.2)

- **Loại:** security
- **Phạm vi:** `services/api-gateway`, `deploy`, `docs`
- **Tóm tắt:** Thêm rate limit cạnh gateway cho OTP request, admin login và place-order (IP/user), trả `429 RATE_LIMITED` + `Retry-After` — bổ sung limiter phone/IP sẵn có trên auth-service.
- **Chi tiết:**
  - Sliding-window per minute: OTP/login theo IP; place-order theo IP + JWT subject
  - Env `RATE_LIMIT_*`; CORS expose `Retry-After`
  - Unit tests limiter + endpoint 429; mark `- [DONE] T9.1.2`
- **Workdocs:** `docs/workdocs_gateway_rate_limit_02082026/`
- **Liên quan:** US-9.1 / T9.1.2

## [2026-08-02] Gateway routing, CORS, JWT validation (T9.1.1)

- **Loại:** feature / security
- **Phạm vi:** `services/api-gateway`, `deploy`, `apps/mobile`, `docs`
- **Tóm tắt:** Thay stub 501 bằng reverse-proxy thật tới mọi upstream, CORS cho Flutter Web, và cứng JWT (require `exp`, strip spoof headers) trên nền RBAC T1.2.4.
- **Chi tiết:**
  - Proxy giữ path `/v1/...`; admin split catalog/geo/order/inventory/billing/report
  - `CORS_ORIGINS` (default `localhost` / `127.0.0.1` wildcard port); OPTIONS preflight
  - Upstream down → `502 BAD_GATEWAY`; tests proxy/CORS/RBAC
  - Mark `- [DONE] T9.1.1`
- **Workdocs:** `docs/workdocs_gateway_routing_cors_jwt_02082026/`
- **Liên quan:** US-9.1 / T9.1.1

## [2026-08-02] Flutter dashboard widgets (T8.1.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile` (dashboard), `docs/prd.md`
- **Tóm tắt:** Admin desk `/admin` hiện widgets tổng quan doanh thu / lợi nhuận / phí giao / công nợ / số đơn từ `GET /v1/admin/dashboard/summary`, kèm filter Hôm nay / 7 ngày / Tháng này.
- **Chi tiết:**
  - `features/dashboard/` — models, API client, `AdminDashboardPage` (summary + nav tiles)
  - `ApiConfig` note report-service `:8087`; README verify
  - Mark `- [DONE] T8.1.3`
- **Workdocs:** `docs/workdocs_flutter_admin_dashboard_02082026/`
- **Liên quan:** US-8.1 / T8.1.3

## [2026-08-02] API dashboard summary (T8.1.2)

- **Loại:** feature
- **Phạm vi:** `services/report-service`, `services/api-gateway`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Admin đọc tổng hợp dashboard (doanh thu / COGS / phí / profit / đơn / công nợ) theo ngày hoặc khoảng ngày từ `daily_stats`, kèm debt snapshot từ `billing.debt.updated`.
- **Chi tiết:**
  - `GET /v1/admin/dashboard/summary` (`day` | `from`+`to` | mặc định hôm nay VN)
  - Consumer `report-billing-debt-updated` → `customer_debt_balances` + `dashboard_snapshot.debt_total`
  - Tests report-service + gateway RBAC path; mark `- [DONE] T8.1.2`
- **Workdocs:** `docs/workdocs_report_dashboard_summary_api_02082026/`
- **Liên quan:** US-8.1 / T8.1.2

## [2026-08-02] report-service subscribe events → daily_stats (T8.1.1)

- **Loại:** feature
- **Phạm vi:** `services/report-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** JetStream durable consumers upsert `daily_stats` từ `order.placed` / `order.completed` (idempotent), nền tảng dashboard summary T8.1.2.
- **Chi tiết:**
  - Consumers `report-order-placed`, `report-order-completed` + migrate schema + NATS wire-up
  - Upsert theo ngày VN; profit qua `BuildDailyStatsAmounts` / `ApplyProfit`; fee derive từ `total − revenue` khi thiếu `delivery_fee`
  - Tests handler + JetStream; mark `- [DONE] T8.1.1`
- **Workdocs:** `docs/workdocs_report_daily_stats_events_02082026/`
- **Liên quan:** US-8.1 / T8.1.1

## [2026-08-02] Công thức profit report-service (T7.2.2)

- **Loại:** feature
- **Phạm vi:** `services/report-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt công thức lợi nhuận MVP `profit = revenue − COGS` (fee ship tách riêng) thành helper tái sử dụng cho `daily_stats`, nền tảng dashboard T8.1.x.
- **Chi tiết:**
  - `SumSaleRevenue` / `SumCOGS` / `ComputeProfit` / `BuildDailyStatsAmounts` / `ApplyProfit`
  - Tests: multi-line, fee không trừ profit, profit âm khi COGS > revenue
  - Sync architecture §6.7 + schema comments; mark `- [DONE] T7.2.2`
- **Workdocs:** `docs/workdocs_report_profit_formula_02082026/`
- **Liên quan:** US-7.2 / T7.2.2

## [2026-08-02] COGS snapshot tại thời điểm xuất/bán (T7.2.1)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt contract lưu giá vốn trên mọi OUT (admin + `order.completed`): `unit_cost` snapshot từ `cost_price` lúc xuất, bất biến khi nhập sau đổi giá; nền tảng cho report profit.
- **Chi tiết:**
  - Helper `snapshotOUTCost`; bỏ qua `unit_cost` client trên OUT; insert OUT bắt buộc có snapshot
  - Tests freeze sau later IN (API + ORDER) + ignore client cost
  - Architecture §4.4 / §6.5 ghi COGS snapshot contract; mark `- [DONE] T7.2.1`
- **Workdocs:** `docs/workdocs_inventory_cogs_snapshot_02082026/`
- **Liên quan:** US-7.2 / T7.2.1

## [2026-08-02] Flutter màn tồn kho admin (T7.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Admin Flutter list tồn kho và tạo phiếu IN/OUT/ADJUST qua `GET/POST /v1/admin/inventory`; gắn tile từ desk `/admin`.
- **Chi tiết:**
  - `features/inventory/` — models, API client, `AdminInventoryPage` (list + dialog phiếu + FAB nhập mới)
  - Route `/admin/inventory`; `ApiConfig`/README note port `:8085`; mark `- [DONE] T7.1.4`
- **Workdocs:** `docs/workdocs_flutter_admin_inventory_02082026/`
- **Liên quan:** US-7.1 / T7.1.4

## [2026-08-02] Consumer order.completed trừ tồn (T7.1.3)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Durable JetStream consumer `inventory-order-completed` trừ tồn (OUT + snapshot COGS) khi `order.completed`; không trừ trên `order.placed`. Idempotent qua `processed_events`.
- **Chi tiết:**
  - Wire NATS trong inventory-service; Ack/Nak + transaction multi-line OUT (`ref_type=ORDER`)
  - MVP: thiếu stock → tạo placeholder, cho phép `on_hand` âm
  - Tests unit + embedded JetStream; sync architecture §6.5; mark `- [DONE] T7.1.3`
- **Workdocs:** `docs/workdocs_inventory_order_completed_02082026/`
- **Liên quan:** US-7.1 / T7.1.3

## [2026-08-02] APIs nhập/xuất/điều chỉnh tồn (T7.1.2)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `services/api-gateway`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Admin `GET/POST /v1/admin/inventory` list tồn và tạo phiếu IN/OUT/ADJUST: cập nhật `on_hand` + `cost_price`, ghi `stock_movements`. Gateway RBAC giữ `/v1/admin/*`.
- **Chi tiết:**
  - `IN` tạo stock nếu chưa có; `cost_price` = `unit_cost`; `OUT` snapshot COGS; `ADJUST` dùng `delta` signed
  - MVP cho phép `on_hand` âm; tests validation + persist movements
  - Sync architecture §4.4; mark `- [DONE] T7.1.2`
- **Workdocs:** `docs/workdocs_inventory_stock_apis_02082026/`
- **Liên quan:** US-7.1 / T7.1.2

## [2026-08-02] Schema stock + movements + cost (T7.1.1)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt `inventory.db`: `stock_items` / `stock_movements` / `processed_events` với CHECK giá vốn & qty; migrate-on-start; seed empty (không hardcode SP).
- **Chi tiết:**
  - Embed `schema.sql` + `seedInventoryDefaults` (`INVENTORY_SEED`)
  - Tests columns/indexes/constraints (âm `on_hand` OK; `qty > 0`)
  - Sync architecture §6.5; mark `- [DONE] T7.1.1`
- **Workdocs:** `docs/workdocs_inventory_stock_schema_02082026/`
- **Liên quan:** US-7.1 / T7.1.1

## [2026-08-02] Flutter UI công nợ admin (T6.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Admin desk thêm màn **Công nợ** gọi `GET /v1/admin/debts`: banner tổng + list khách còn nợ (SĐT masked, số tiền).
- **Chi tiết:**
  - `BillingApi` / models + `AdminDebtsPage` (refresh, empty state)
  - Route `/admin/debts` + tile từ `/admin`
  - `ApiConfig` + README verify local `:8086`
  - Mark `- [DONE] T6.2.2` trên PRD
- **Workdocs:** `docs/workdocs_flutter_admin_debts_02082026/`
- **Liên quan:** US-6.2 / T6.2.2

## [2026-08-02] API list/aggregate debts (T6.2.1)

- **Loại:** feature
- **Phạm vi:** `services/billing-service`, `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** Admin `GET /v1/admin/debts` trả danh sách khách còn nợ (`balance > 0`) kèm aggregate `total_balance` / `count`. Gateway giữ RBAC `/v1/admin/*` (role=admin).
- **Chi tiết:**
  - `handleListDebts` + query sort balance DESC
  - Tests empty / aggregate / omit zero-balance
  - Gateway RBAC coverage cho path debts (customer 403, admin pass-through stub)
  - Mark `- [DONE] T6.2.1` trên PRD
- **Workdocs:** `docs/workdocs_billing_admin_debts_api_02082026/`
- **Liên quan:** US-6.2 / T6.2.1

## [2026-08-02] Flutter dialog hoàn tất đơn (T6.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Order Desk chi tiết thêm dialog «Hoàn tất» chọn FULL / PARTIAL / UNPAID (+ amount_paid), gọi `POST /v1/admin/orders/{id}/complete`, rồi quay về list (PENDING biến mất).
- **Chi tiết:**
  - Models + `OrderApi.completeOrder`
  - Dialog radio + preview công nợ; validate PARTIAL local
  - SnackBar kết quả; `onCompleted` → `/admin/orders`
  - Mark `- [DONE] T6.1.4` trên PRD
- **Workdocs:** `docs/workdocs_flutter_order_complete_dialog_02082026/`
- **Liên quan:** US-6.1 / T6.1.4

## [2026-08-02] Events order.completed + billing.payment/debt (T6.1.3)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `services/billing-service`, `docs/prd.md`
- **Tóm tắt:** Publish JetStream `order.completed` sau admin complete; billing publish `billing.payment.recorded` + `billing.debt.updated` sau ghi payment (không republish khi idempotent). Payload theo architecture §5.1.
- **Chi tiết:**
  - Mở rộng `orderPublisher` + `PublishOrderCompleted` (items/total/payment_type/amount_paid)
  - `jsBillingPublisher` + wire billing `ConnectJS`/`EnsureStreams` (`billing.>` đã có sẵn)
  - Tests recording bus + embedded JetStream; lỗi publish chỉ log
  - Mark `- [DONE] T6.1.3` trên PRD
- **Workdocs:** `docs/workdocs_order_billing_events_02082026/`
- **Liên quan:** US-6.1 / T6.1.3

## [2026-08-02] Billing ghi payments + cập nhật debts (T6.1.2)

- **Loại:** feature
- **Phạm vi:** `services/billing-service`, `services/order-service`, `deploy`, `docs/prd.md`
- **Tóm tắt:** Sau admin complete, billing-service ghi `payments` và tăng `debts`/`debt_ledger` theo FULL/PARTIAL/UNPAID. Sync HTTP từ order-service; events JetStream để T6.1.3.
- **Chi tiết:**
  - Migrate `billing.db`; `POST /v1/internal/payments` + `recordPayment` (idempotent `order_id`)
  - Order complete gọi billing với `customer_key=phone_hash`; lỗi billing chỉ log
  - Tests AC PARTIAL 100k/450k→debt 350k, FULL (không tạo nợ), UNPAID, accumulate, idempotent
  - Compose/env: `BILLING_SERVICE_URL` cho order-service
  - Mark `- [DONE] T6.1.2` trên PRD
- **Workdocs:** `docs/workdocs_billing_payments_debts_02082026/`
- **Liên quan:** US-6.1 / T6.1.2

## [2026-08-02] API hoàn tất đơn + payment payload (T6.1.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** Admin `POST /v1/admin/orders/{id}/complete` nhận FULL/PARTIAL/UNPAID + `amount_paid`, chuyển PENDING→COMPLETED và trả debt settlement. Billing write / events để T6.1.2–T6.1.3.
- **Chi tiết:**
  - `handleCompleteOrder` + `settlePayment` (PRD M6 rules)
  - Snapshot `payment_type` / `amount_paid` trên `orders`; `completed_at`
  - Tests PARTIAL AC (100k/450k→debt 350k), FULL, UNPAID, validation, 404/409
  - Gateway RBAC assert cho path complete dưới `/v1/admin/*`
  - Mark `- [DONE] T6.1.1` trên PRD
- **Workdocs:** `docs/workdocs_order_complete_api_02082026/`
- **Liên quan:** US-6.1 / T6.1.1

## [2026-08-02] Flutter nút «Dẫn đường» chi tiết đơn (T5.2.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Order Desk chi tiết đơn có nút «Dẫn đường» gọi `openNavigationTo(lat,lng)`; SnackBar nếu thiếu toạ độ hoặc không mở được Maps. Không đụng E6 hoàn tất/thanh toán.
- **Chi tiết:**
  - `AdminOrderDetailPage`: `FilledButton.icon` «Dẫn đường» sau khối địa chỉ
  - Guard `lat/lng == 0,0` (API null → model default) + hiện lỗi launch từ helper
  - Mark `- [DONE] T5.2.3` trên PRD; README verify
- **Workdocs:** `docs/workdocs_flutter_order_nav_button_02082026/`
- **Liên quan:** Sprint 3 / US-5.2 / T5.2.3

## [2026-08-02] Flutter deep-link Maps / geo intent (T5.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Helper `openNavigationTo(lat,lng)` mở chỉ đường tới điểm giao qua Google Maps / `geo:` / Apple Maps + HTTPS fallback (Web/Android/iOS). Chưa nút UI (T5.2.3).
- **Chi tiết:**
  - `navigation_link.dart`: candidate URIs theo platform; omit `origin` → Maps dùng vị trí thiết bị
  - Android `<queries>` + iOS `LSApplicationQueriesSchemes`; `platform_config` fragments
  - Unit tests URI builders; README verify
  - Mark `- [DONE] T5.2.2` trên PRD
- **Workdocs:** `docs/workdocs_flutter_maps_deeplink_02082026/`
- **Liên quan:** Sprint 3 / US-5.2 / T5.2.2

## [2026-08-02] Admin order lat/lng + GET by id (T5.2.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Expose `lat`/`lng` điểm giao trên admin list/detail; implement `GET /v1/admin/orders/{id}` (trước đó stub) để CCH lấy toạ độ cho dẫn đường. Chưa deep-link Maps / nút Flutter.
- **Chi tiết:**
  - `handleGetAdminOrder` + `loadOrderByID`; wire thay `notImplemented`
  - `orderView.lat`/`lng` documented as delivery destination (WGS84)
  - Unit tests get-by-id coords, 404, list exposes lat/lng
  - Mark `- [DONE] T5.2.1` trên PRD
- **Workdocs:** `docs/workdocs_order_admin_lat_lng_02082026/`
- **Liên quan:** Sprint 3 / US-5.2 / T5.2.1

## [2026-08-02] Flutter Order Desk polling báo đơn mới (T5.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Order Desk tự poll `GET /v1/admin/orders` mỗi 10s (pause khi app background), giữ pull-to-refresh; SnackBar khi phát hiện đơn id mới. Chọn polling thay SSE/NATS cho MVP Web+mobile.
- **Chi tiết:**
  - `AdminOrdersPage`: `Timer.periodic` + `WidgetsBindingObserver`; silent refresh không che list
  - Empty state cũng kéo-refresh; copy nhắc chu kỳ poll
  - Mark `- [DONE] T5.1.4` trên PRD
- **Workdocs:** `docs/workdocs_flutter_order_desk_polling_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.4

## [2026-08-02] Flutter Order Desk UI (T5.1.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn admin Order Desk gọi `GET /v1/admin/orders`, hiển thị STT | tên | SĐT masked | địa chỉ | km | thời gian (FIFO cũ nhất trước); link từ `/admin`. Chi tiết đọc-only từ payload list (chưa polling / maps).
- **Chi tiết:**
  - `AdminOrder` + `OrderApi.listAdminOrders`
  - `AdminOrdersPage` / `AdminOrderDetailPage`; routes `/admin/orders`, `/admin/orders/detail`
  - Tile **Order Desk** trên admin home
  - Mark `- [DONE] T5.1.3` trên PRD
- **Workdocs:** `docs/workdocs_flutter_order_desk_ui_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.3

## [2026-08-02] Admin Order Desk columns STT + fields (T5.1.2)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Làm giàu `GET /v1/admin/orders` với `stt` (FIFO 1-based) và khẳng định contract cột desk: tên, SĐT masked, địa chỉ, km, thời gian. Chưa Flutter UI.
- **Chi tiết:**
  - `orderView.stt` (omitempty); `adminOrderViewsFromRows` gán STT
  - SĐT admin = `phone_masked` (orders không có plaintext phone)
  - Unit tests desk columns + khách không lộ `stt`
  - Mark `- [DONE] T5.1.2` trên PRD
- **Workdocs:** `docs/workdocs_order_admin_desk_columns_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.2

## [2026-08-02] API admin list orders FIFO (T5.1.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** `GET /v1/admin/orders` danh sách đơn FIFO (`created_at ASC`); mặc định `PENDING`, optional `?status=`. Path sẵn dưới gateway admin RBAC. Chưa Flutter desk / cột STT.
- **Chi tiết:**
  - Handler thay stub; response basic `orderView` + items
  - Unit tests FIFO (A trước B), filter COMPLETED, status invalid
  - Mark `- [DONE] T5.1.1` trên PRD
- **Workdocs:** `docs/workdocs_order_admin_list_fifo_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.1

## [2026-08-02] Flutter review: hiển thị quote phí giao (T4.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn xác nhận đơn gọi `POST /v1/orders/quote` khi mở / trước place; hiển thị khoảng cách, phí giao, tạm tính, tổng; chặn đặt khi ngoài phạm vi hoặc quote lỗi. Bỏ stub phí = 0.
- **Chi tiết:**
  - Models `QuoteOrderRequest` / `OrderQuote`; `OrderApi.quoteOrder`
  - Review: loading báo giá, refresh, distance + fee + totals; re-quote trước place
  - Mark `- [DONE] T4.2.2` trên PRD
- **Workdocs:** `docs/workdocs_flutter_order_review_quote_02082026/`
- **Liên quan:** Sprint 2 / US-4.2 / T4.2.2

## [2026-08-02] API quote: distance + fee + total (T4.2.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** `POST /v1/orders/quote` preview khoảng cách + phí + tổng từ geo/catalog/fee engine; không persist. Ngoài bán kính vẫn 200 kèm `in_range=false` (place giữ 422). Chưa Flutter review (T4.2.2).
- **Chi tiết:**
  - Body `{ items, lat, lng }`; response `distance_km`, `in_range`, `max_radius_km`, `delivery_fee`, `subtotal`, `total`
  - Reuse `computeDeliveryFee` + catalog prices; customer identity headers
  - Unit tests happy / OOR preview / fee off / auth / validation; mark `- [DONE] T4.2.1`
- **Workdocs:** `docs/workdocs_order_quote_api_02082026/`
- **Liên quan:** Sprint 2 / US-4.2 / T4.2.1

## [2026-08-02] Flutter admin: màn phí giao hàng (T4.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn admin bật/tắt phí ship + chỉnh bậc km; gọi `GET/PUT /v1/admin/delivery-fee` với JWT session; tile từ desk `/admin`. Không làm quote khách (T4.2.x).
- **Chi tiết:**
  - `DeliveryFeeApi` + models; toggle `enabled` PUT ngay; **Lưu bậc** replace `rules`
  - Validate local overlap / open-ended; Material 3 khớp admin products
  - Route `/admin/delivery-fee`; README + ApiConfig note order `:8084`
  - Mark `- [DONE] T4.1.4` trên PRD
- **Workdocs:** `docs/workdocs_flutter_admin_delivery_fee_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.4

## [2026-08-02] Engine tính phí giao khi place order (T4.1.3)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Thay stub phí = 0 bằng engine reusable: tắt → 0; bật → khớp bậc `[min_km, max_km)` theo `distance_km`; wire vào `POST /v1/orders`. Quote API để T4.2.1.
- **Chi tiết:**
  - `matchDeliveryFee` (pure) + `computeDeliveryFee` (load DB); missing settings = fee 0
  - Unit tests band / disabled / inactive / gap; place-order tests enabled vs disabled
  - Mark `- [DONE] T4.1.3` trên PRD
- **Workdocs:** `docs/workdocs_delivery_fee_engine_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.3

## [2026-08-02] Admin APIs cấu hình phí giao (T4.1.2)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** `GET/PUT /v1/admin/delivery-fee` đọc/cập nhật toggle phí + bậc km trên order-service; validate band không overlap; path nằm dưới gateway RBAC admin — chưa fee engine / Flutter UI.
- **Chi tiết:**
  - PUT partial: `enabled` và/hoặc replace toàn bộ `rules` (transaction delete+insert)
  - Validate min/max/fee; open-ended `max_km=null` chỉ được là band active cuối
  - Tests order-service + assert customer bị FORBIDDEN trên `/v1/admin/delivery-fee`
  - Mark `- [DONE] T4.1.2` trên PRD
- **Workdocs:** `docs/workdocs_admin_delivery_fee_apis_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.2

## [2026-08-02] Schema delivery_fee_settings + delivery_fee_rules (T4.1.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`, `docs/architecture.md`
- **Tóm tắt:** Chốt schema phí giao trên `order.db`: toggle singleton + bậc khoảng cách; migrate-on-start; seed local (fee off + 3 bậc architecture); tests schema/seed — chưa admin API / fee engine.
- **Chi tiết:**
  - CHECK `enabled`/`active`/`fee_vnd`/`min_km`/`max_km`; index `idx_delivery_fee_rules_active`
  - Seed idempotent settings `default` + rules 0–5/5–10/10–∞ (10k/20k/30k); env `DELIVERY_FEE_SEED` / `DELIVERY_FEE_ENABLED`
  - Sync architecture §6.4; Mark `- [DONE] T4.1.1` trên PRD
- **Workdocs:** `docs/workdocs_delivery_fee_schema_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.1

## [2026-08-02] Mask PII trong order response (T3.3.4)

- **Loại:** security
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Response khách của order-service chỉ trả SĐT đã mask (`090***4567`, cùng style auth); remask defense-in-depth khi header vô tình chứa số thật; `GET /v1/orders/me` dùng cùng policy.
- **Chi tiết:**
  - `maskPhoneDisplay` / `ensurePhoneMasked` / `customerOrderView` — không trả `phone_hash` / `phone_e164`
  - Create order remask trước persist + JSON
  - `GET /v1/orders/me` (own orders, masked); buffer rows trước nested item query (tránh SQLite deadlock)
  - Tests remask + list + 401; Mark `- [DONE] T3.3.4` trên PRD
- **Workdocs:** `docs/workdocs_order_mask_pii_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.4

## [2026-08-02] Flutter review + success place order (T3.3.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Hoàn tất bước xác nhận đơn trên Flutter: màn review (giỏ + địa chỉ + tổng), gọi `POST /v1/orders` với JWT/session, rồi màn thành công — nối từ địa chỉ in-range.
- **Chi tiết:**
  - `OrderApi` + models; local gắn `X-User-*` từ session (gateway proxy còn stub)
  - `OrderReviewPage` / `OrderSuccessPage`; routes `/order/review`, `/order/success`
  - Phí giao preview stub 0 (E4); clear cart sau place thành công
  - Mark `- [DONE] T3.3.3` trên PRD
- **Workdocs:** `docs/workdocs_flutter_order_review_success_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.3

## [2026-08-02] Persist order + publish order.placed (T3.3.2)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `deploy`, `docs/prd.md`
- **Tóm tắt:** Sau place order thành công, polish persist (CHECK/index) và publish `order.placed` lên JetStream (payload `order_id`/`total`/`distance_km`/`created_at`) cho report — theo pattern catalog product.updated.
- **Chi tiết:**
  - Schema: CHECK money/qty; index `orders(created_at)`, `order_items(product_id)`
  - `jsOrderPublisher` + `natsx.PublishEnvelope` sau commit (lỗi chỉ log)
  - Startup: `ConnectJS` + `EnsureStreams`
  - Tests mock recorder + embedded JetStream; assert cột persisted
  - Mark `- [DONE] T3.3.2` trên PRD
- **Workdocs:** `docs/workdocs_order_placed_event_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.2

## [2026-08-02] API POST /orders validate + thin persist (T3.3.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `deploy`, `docs/prd.md`
- **Tóm tắt:** `POST /v1/orders` kiểm tra identity gateway (`X-User-*`), items qua catalog, geo in-range qua geo-service, fee stub 0 (TODO E4); thin insert order + items `PENDING` (chưa `order.placed`).
- **Chi tiết:**
  - Validate body + coords; `422 OUT_OF_RANGE` kèm distance/max_radius
  - Giá lấy từ catalog active; merge qty trùng `product_id`
  - Env `GEO_SERVICE_URL` / `CATALOG_SERVICE_URL`; migrate schema khi boot
  - Unit tests happy / auth / out-of-range / product missing
  - Mark `- [DONE] T3.3.1` trên PRD
- **Workdocs:** `docs/workdocs_order_post_create_api_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.1

## [2026-08-02] Flutter UI ngoài phạm vi giao (T3.2.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Sau khi chọn địa chỉ, app gọi `POST /v1/geo/check`; ngoài bán kính hiện message VN rõ và chặn Tiếp tục; trong phạm vi mới cho đi tiếp (chưa place order).
- **Chi tiết:**
  - `GeoApi.check` + `GeoCheckResult` (distance / in_range / max_radius)
  - Banner đỏ ngoài phạm vi (kèm km) / xanh trong phạm vi; nút disable khi `in_range=false`
  - `orderGeoCheckProvider` giữ kết quả cho T3.3; `onContinue` stub SnackBar
  - Mark `- [DONE] T3.2.3` trên PRD
- **Workdocs:** `docs/workdocs_flutter_out_of_range_ui_02082026/`
- **Liên quan:** Sprint 2 / US-3.2 / T3.2.3

## [2026-08-02] Haversine geo check + in_range (T3.2.2)

- **Loại:** feature
- **Phạm vi:** `services/geo-service`, `docs/prd.md`
- **Tóm tắt:** `POST /v1/geo/check` tính khoảng cách Haversine từ cửa hàng tới lat/lng khách và trả `distance_km` (2dp), `in_range`, `max_radius_km`.
- **Chi tiết:**
  - Haversine R=6371 km; `in_range` khi `distance_km <= max_radius_km`
  - Validate coords; 404 nếu chưa seed `store_settings`
  - Unit tests math + handler (trong/ngoài bán kính, boundary, invalid)
  - Mark `- [DONE] T3.2.2` trên PRD
- **Workdocs:** `docs/workdocs_geo_haversine_check_02082026/`
- **Liên quan:** Sprint 2 / US-3.2 / T3.2.2

## [2026-08-02] Store settings lat/lng + max_radius_km (T3.2.1)

- **Loại:** feature
- **Phạm vi:** `services/geo-service`, `pkg/config`, `deploy/.env.example`, `docs/prd.md`
- **Tóm tắt:** Persist singleton `store_settings` (lat/lng, `max_radius_km`) với seed từ env cho local; expose `GET /v1/geo/store` (public) và `PUT /v1/admin/geo/store` để cập nhật — nền cho check bán kính T3.2.2.
- **Chi tiết:**
  - Seed idempotent `STORE_LAT` / `STORE_LNG` / `STORE_MAX_RADIUS_KM` (default ≈ Bến Thành HCMC, radius 10km)
  - Public GET trả name/lat/lng/max_radius_km/(address_text); không lộ `updated_by`
  - Admin PUT partial update + validate coords; unit tests seed/GET/PUT
  - `config.GetFloat`; mark `- [DONE] T3.2.1` trên PRD
- **Workdocs:** `docs/workdocs_geo_store_settings_02082026/`
- **Liên quan:** Sprint 2 / US-3.2 / T3.2.1

## [2026-08-02] Flutter map/picker + autocomplete địa chỉ (T3.1.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Bước địa chỉ đặt hàng có autocomplete (gọi geo-service proxy, không gọi OSM trực tiếp) và bản đồ ghim pin (`flutter_map`) trên Web / Android / iOS; lưu lat/lng/label.
- **Chi tiết:**
  - `GeoApi.search` → `GET /v1/geo/search?q=` + debounce autocomplete
  - `flutter_map` + OSM tiles; chạm bản đồ / chọn gợi ý / GPS → pin + `orderAddressProvider`
  - README verify geo `:8083`; mark `- [DONE] T3.1.3` trên PRD
- **Workdocs:** `docs/workdocs_flutter_map_autocomplete_02082026/`
- **Liên quan:** Sprint 2 / US-3.1 / T3.1.3

## [2026-08-02] Proxy search geocode Photon/Nominatim (T3.1.2)

- **Loại:** feature
- **Phạm vi:** `services/geo-service`, `docs/prd.md`, `deploy/.env.example`
- **Tóm tắt:** Geo-service proxy `GET /v1/geo/search?q=` tới Photon (mặc định) hoặc Nominatim — User-Agent, rate limit IP, cache SQLite; Flutter không gọi OSM trực tiếp.
- **Chi tiết:**
  - Provider `GEOCODE_PROVIDER=photon|nominatim`; chuẩn hóa `{items:[{label,lat,lng,source}]}`
  - `geocode_cache` + migrate schema; IP rate limit; Nominatim min 1 req/s
  - Unit tests mock upstream; ghi chú `ApiConfig` `:8083`
  - Mark `- [DONE] T3.1.2` trên PRD
- **Workdocs:** `docs/workdocs_geo_search_proxy_02082026/`
- **Liên quan:** Sprint 2 / US-3.1 / T3.1.2

## [2026-08-02] Xin quyền location Web/Android/iOS (T3.1.1)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Khách xin quyền vị trí và lấy lat/lng trên bước địa chỉ («Dùng vị trí hiện tại») với message lỗi tiếng Việt; cấu hình permission cho 3 platform.
- **Chi tiết:**
  - Helper `location_permission.dart` (geolocator: denied / deniedForever / serviceDisabled / timeout; Web fallback khi Permissions API thiếu)
  - Nút «Dùng vị trí hiện tại» trên `/order/address`; hiện lat/lng hoặc lỗi VN
  - Android `ACCESS_FINE/COARSE_LOCATION`; iOS `NSLocationWhenInUseUsageDescription`; Web HTTPS/localhost note
  - Bỏ `permission_handler` (thừa — `geolocator` đủ 3 target)
  - Mark `- [DONE] T3.1.1` trên PRD
- **Workdocs:** `docs/workdocs_location_permission_02082026/`
- **Liên quan:** Sprint 2 / US-3.1 / T3.1.1

## [2026-08-02] Flutter bước chọn SP đặt hàng (T2.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Thay order placeholder bằng màn chọn sản phẩm active (`GET /v1/products`) + giỏ local; Tiếp tục sang placeholder địa chỉ (E3).
- **Chi tiết:**
  - `CatalogApi.listActiveProducts`; `OrderCart` Riverpod; `SelectProductsPage` (+/− qty, tổng VND)
  - Routes `/order`, `/order/address` (placeholder)
  - README / `ApiConfig` ghi chú verify customer pick
  - Mark `- [DONE] T2.2.2` trên PRD
- **Workdocs:** `docs/workdocs_flutter_order_select_products_02082026/`
- **Liên quan:** Sprint 2 / US-2.2 / T2.2.2

## [2026-08-02] API list products active (T2.2.1)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `docs/prd.md`
- **Tóm tắt:** `GET /v1/products` trả danh sách sản phẩm `active` cho khách (public/authenticated); ẩn SP không còn lộ ra ngoài admin.
- **Chi tiết:**
  - Handler `handleListActiveProducts` (`WHERE active = 1`) thay stub rỗng
  - Refactor `collectProducts` dùng chung với admin list
  - Unit test empty / filter inactive / admin vẫn thấy đủ
  - Mark `- [DONE] T2.2.1` trên PRD
- **Workdocs:** `docs/workdocs_catalog_list_active_products_02082026/`
- **Liên quan:** Sprint 2 / US-2.2 / T2.2.1

## [2026-08-02] Flutter admin màn sản phẩm (T2.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn admin Flutter list / thêm / sửa / ẩn sản phẩm gọi catalog CRUD (`GET/POST/PATCH /v1/admin/products`); desk `/admin` có lối vào Sản phẩm.
- **Chi tiết:**
  - `CatalogApi` + models; list + form create/edit; toggle ẩn/hiện (`active`)
  - Routes `/admin/products`, `/admin/products/new`, `/admin/products/:id`
  - README / `ApiConfig` ghi chú local `API_BASE_URL` → catalog `:8082`
  - Mark `- [DONE] T2.1.4` trên PRD
- **Workdocs:** `docs/workdocs_flutter_admin_products_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.4

## [2026-08-02] Event catalog.product.updated (T2.1.3)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `pkg/natsx`, `docs/prd.md`
- **Tóm tắt:** Catalog publish `catalog.product.updated` (envelope JetStream) sau create/update/ẩn sản phẩm — payload `product_id`/`sku`/`sale_price`/`active` cho inventory & report.
- **Chi tiết:**
  - `natsx.PublishEnvelope` (MsgId = event_id) + EnsureStreams lúc start catalog
  - Hook publish sau commit create/patch; lỗi bus chỉ log
  - Tests mock recorder + embedded JetStream
  - Mark `- [DONE] T2.1.3` trên PRD
- **Workdocs:** `docs/workdocs_catalog_product_updated_event_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.3

## [2026-08-02] Schema products + product_prices (T2.1.2)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt schema catalog `products` + `product_price_history` (PRD `product_prices`) — indexes, CHECK/FK, comments migrate-on-start; tests assert schema.
- **Chi tiết:**
  - CHECK `sale_price >= 0`, `active IN (0,1)`; FK history → products
  - Index `idx_products_active`, `idx_price_history_product`
  - Unit tests columns/indexes + constraint/FK; sync architecture §6.2
  - Mark `- [DONE] T2.1.2` trên PRD
- **Workdocs:** `docs/workdocs_catalog_products_schema_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.2

## [2026-08-02] CRUD APIs catalog (T2.1.1)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `docs/prd.md`
- **Tóm tắt:** Admin CRUD sản phẩm/giá bán trên catalog-service (`GET/POST/PATCH /v1/admin/products`) với SQLite migrate và lịch sử giá; ẩn SP bằng `active=false`.
- **Chi tiết:**
  - Endpoints: list, create, get-by-id, patch (sku/name/desc/unit/sale_price/active/image_url)
  - Ghi `product_price_history` khi tạo hoặc đổi giá; `changed_by` từ `X-User-Id`
  - Unit tests validation / SKU conflict / 404; public `GET /v1/products` vẫn stub (T2.2.1)
  - Path trên catalog `:8082` — gateway admin proxy vẫn stub
  - Mark `- [DONE] T2.1.1` trên PRD
- **Workdocs:** `docs/workdocs_catalog_crud_apis_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.1

## [2026-08-02] Middleware RBAC trên gateway (T1.2.4)

- **Loại:** feature / security
- **Phạm vi:** `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** Gateway validate JWT access (HS256, cùng secret/claims auth-service) và enforce RBAC — `admin` cho `/v1/admin/**`, `customer` cho orders + `POST /geo/check`; auth/public routes bỏ qua.
- **Chi tiết:**
  - `RequireJWT` + `RequireRole`; forward `X-User-Id` / `X-User-Role` / `X-Session-Id`
  - Public: health, hello, `/v1/auth/*`, products, `GET /geo/store|search`
  - Unit tests 401/403/role split; upstream vẫn stub 501
  - Mark `- [DONE] T1.2.4` trên PRD
- **Workdocs:** `docs/workdocs_gateway_rbac_middleware_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.4

## [2026-08-02] Flutter admin login screen (T1.2.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Thêm màn đăng nhập admin (username/password) trên Flutter Web/Android/iOS — gọi `POST /v1/auth/admin/login`, lưu JWT session in-memory, CTA Home «Dành cho cửa hàng».
- **Chi tiết:**
  - `AdminLoginPage` + routes `/admin/login`, `/admin` (placeholder desk)
  - `AuthApi.adminLogin`; dùng chung `AuthTokenResult` / `authSessionProvider` với OTP
  - Map lỗi `INVALID_CREDENTIALS`; README seed `admin` / `admin-change-me`
  - Mark `- [DONE] T1.2.3` trên PRD
- **Workdocs:** `docs/workdocs_flutter_admin_login_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.3

## [2026-08-02] API login admin + refresh (T1.2.2)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `docs/prd.md`
- **Tóm tắt:** Implement `POST /v1/auth/admin/login` (bcrypt → JWT `role=admin`) và `POST /v1/auth/refresh` với rotation session — dùng chung pattern token từ OTP verify.
- **Chi tiết:**
  - Login admin: username/password → access + opaque refresh; session `role=admin`
  - Refresh xoay vòng (revoke cũ + session mới) cho admin và customer
  - Sai credentials / disabled / token hết hạn → 401 thống nhất; unit tests
  - Mark `- [DONE] T1.2.2` trên PRD
- **Workdocs:** `docs/workdocs_admin_login_refresh_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.2

## [2026-08-02] Seed admin account bcrypt (T1.2.1)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Bootstrap tài khoản admin mặc định vào `admin_accounts` với `password_hash` bcrypt khi process start — cấu hình qua env, không commit secret.
- **Chi tiết:**
  - Seed idempotent sau migrate: insert nếu username chưa có; không overwrite password hiện có
  - Env: `ADMIN_USERNAME` / `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_DISPLAY_NAME`, `ADMIN_SEED`
  - Unit tests hash≠plaintext, verify bcrypt, idempotent, disable seed
  - Mark `- [DONE] T1.2.1` trên PRD
- **Workdocs:** `docs/workdocs_seed_admin_account_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.1

## [2026-08-02] OTP challenges SQLite hash + expiry (T1.1.5)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt T1.1.5 — `otp_challenges` trên SQLite auth lưu OTP đã hash + expiry; siết schema/index và tests (persistence đã có từ T1.1.1/1.2).
- **Chi tiết:**
  - Comment schema rõ contract hash/expiry; thêm `idx_otp_expires`
  - Tests: migrate columns/indexes, `code_hash` ≠ plaintext, `expires_at` TTL, verify `OTP_EXPIRED`
  - Sync architecture §6.1; mark `- [DONE] T1.1.5` trên PRD
- **Workdocs:** `docs/workdocs_otp_challenges_sqlite_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.5

## [2026-08-02] Flutter màn SĐT + OTP (T1.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`
- **Tóm tắt:** Thêm màn nhập SĐT và OTP trên Flutter (Web/Android/iOS), gọi `POST /v1/auth/otp/request` + `verify` → JWT session in-memory; CTA Home đi qua auth trước order placeholder.
- **Chi tiết:**
  - `PhonePage` / `OtpPage` + validate SĐT VN, resend cooldown, hiện `dev_code` local
  - `AuthApi` (Dio) + `authSessionProvider` + interceptor gắn Bearer
  - Routes `/auth/phone`, `/auth/otp`; README hướng dẫn `API_BASE_URL` → auth `:8081` khi gateway chưa proxy
  - Mark `- [DONE] T1.1.4` trên PRD
- **Workdocs:** `docs/workdocs_flutter_otp_ui_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.4

## [2026-08-02] Adapter SMS mock + production seam (T1.1.3)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`
- **Tóm tắt:** Thêm `SMSSender` interface, mock adapter (default local), và production seam (eSMS/Stringee) — wire vào `POST /auth/otp/request`; chưa gọi vendor thật.
- **Chi tiết:**
  - `SMS_PROVIDER=mock|production` + env `SMS_API_KEY` / `SMS_VENDOR` / `SMS_API_URL` / `SMS_SENDER`
  - Mock ghi nhận send in-memory (tests); log chỉ `phone_masked`, không raw OTP
  - Production seam trả `ErrSMSNotConfigured` cho đến khi plug client vendor
  - OTP request → `502 SMS_FAILED` nếu gửi SMS lỗi; unit tests adapter + handler
  - Mark `- [DONE] T1.1.3` trên PRD
- **Workdocs:** `docs/workdocs_sms_adapter_mock_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.3

## [2026-08-02] OTP verify API → JWT (T1.1.2)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`
- **Tóm tắt:** Implement `POST /v1/auth/otp/verify` — kiểm tra OTP, upsert khách (phone encrypt), phát JWT access + refresh session theo architecture §7.2.
- **Chi tiết:**
  - Validate phone/code; max attempts / expire / consume; invalidate challenge mở khác cùng SĐT
  - JWT HS256 (`sub`, `role`, `phone_masked`, `sid`); refresh opaque + `sessions.refresh_hash`
  - Env: `JWT_ACCESS_TTL_SEC`, `JWT_REFRESH_TTL_SEC`, `OTP_MAX_ATTEMPTS`
  - Unit tests + mark `- [DONE] T1.1.2` trên PRD
- **Workdocs:** `docs/workdocs_otp_verify_jwt_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.2

## [2026-08-02] OTP request API + rate limit (T1.1.1)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`
- **Tóm tắt:** Implement `POST /v1/auth/otp/request` với validate SĐT VN, cooldown/quota theo phone_hash + IP, và lưu OTP hash vào SQLite — nền cho verify JWT ở task tiếp theo.
- **Chi tiết:**
  - Handler trả `phone_masked` / `expires_in_sec` / `resend_after_sec`; `429 RATE_LIMITED` + `Retry-After`
  - Sinh OTP 6 số (TTL 5 phút); hash peppered; không log raw OTP; `OTP_DEV_REVEAL` cho local
  - Migrate `schema.sql` khi start; unit tests phone/rate-limit/handler
  - Mark `- [DONE] T1.1.1` trên PRD
- **Workdocs:** `docs/workdocs_otp_request_ratelimit_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.1

## [2026-08-02] Makefile / scripts chạy dev (T9.2.3)

- **Loại:** feature
- **Phạm vi:** `Makefile`, `scripts/`, `README.md`
- **Tóm tắt:** Hoàn thiện DX local Sprint 0 — Makefile targets (nats, services, flutter, health) và mirror PowerShell `scripts/dev.ps1` cho Windows không có GNU Make.
- **Chi tiết:**
  - `Makefile`: `help` mặc định, `nats` (up+wait+init), compose, per-service `go run`, `build`/`test`, flutter helpers
  - `scripts/dev.ps1`: cùng tên lệnh để chạy trên PowerShell
  - README hướng dẫn Make + PS1; mark `- [DONE] T9.2.3` trên PRD
- **Workdocs:** `docs/workdocs_makefile_dev_scripts_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.3

## [2026-08-02] NATS JetStream local (T9.2.2)

- **Loại:** feature
- **Phạm vi:** `deploy/`, `pkg/natsx`, `cmd/nats-init`
- **Tóm tắt:** Bật NATS JetStream local với config + volume + healthcheck, bootstrap stream theo bounded context (architecture §5.1), và CLI/`go test` để verify — nền event bus cho Sprint 0.
- **Chi tiết:**
  - `deploy/nats.conf` + compose healthcheck / `nats-data` volume; services `depends_on` NATS healthy
  - `pkg/natsx`: `ConnectJS`, `EnsureStreams`, `PingJS` + embedded JetStream test
  - `cmd/nats-init` đảm bảo 6 stream (AUTH…BILLING) và in trạng thái
  - Mark `- [DONE] T9.2.2` trong `docs/prd.md`
- **Workdocs:** `docs/workdocs_nats_jetstream_local_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.2

## [2026-08-02] Accept T9.2.1 monorepo layout (Sprint 0)

- **Loại:** chore
- **Phạm vi:** `docs/prd.md`, monorepo root
- **Tóm tắt:** Xác nhận scaffold hiện có đã đủ AC của T9.2.1 (cây `apps/mobile` + `services/*` + `pkg/` + `deploy/` khớp architecture §2.1; `go build ./services/...` OK) và đánh dấu task DONE trên PRD — không scaffold lại.
- **Chi tiết:**
  - Verify layout: 8 Go services, Flutter `apps/mobile`, shared `pkg`, `deploy/docker-compose.yml`
  - Mark `- [DONE] T9.2.1` trong `docs/prd.md`
  - Bổ sung workdocs acceptance note
- **Workdocs:** `docs/workdocs_scaffold_monorepo_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.1

## [2026-08-02] Scaffold monorepo boilerplate theo architecture

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `services/*`, `pkg/*`, `deploy/`
- **Tóm tắt:** Dựng skeleton monorepo khớp `docs/architecture.md` để Sprint 0 có gateway hello/healthz, NATS compose, Flutter home CTA, và stub API/schema cho từng bounded context.
- **Chi tiết:**
  - Go module `gas-tam-de` + shared `pkg/{config,httpx,sqlite,events,natsx}` (HTTP: Chi)
  - Stub 8 services (gateway + auth/catalog/geo/order/inventory/billing/report) kèm `schema.sql` từ architecture §6
  - `deploy/docker-compose.yml` (NATS JetStream + services), Dockerfile, `.env.example`
  - Flutter `apps/mobile`: brand Gas Tam Đệ + CTA Đặt giao gas (placeholder order flow)
  - Root `README.md`, `Makefile`, `.gitignore`
- **Workdocs:** `docs/workdocs_scaffold_monorepo_02082026/`
- **Liên quan:** Sprint 0 / architecture §2.1

## [2026-08-02] Đa nền tảng Web + Android + iOS song song

- **Loại:** docs
- **Phạm vi:** `docs/prd.md`, `docs/architecture.md`
- **Tóm tắt:** Đổi kế hoạch từ ưu tiên Android sang phát triển Flutter Web, Android và iOS cùng lúc; bổ sung chiến lược test bằng Web/emulator/CI macOS khi không có máy thật.
- **Chi tiết:**
  - PRD: §1.2 target platforms, cập nhật MoSCoW/NFR/DoD/sprint/rủi ro
  - Architecture: §8.4–8.5 multi-platform matrix; CI build iOS no-codesign; deploy IPA
  - iOS không còn “sau MVP”; store publish vẫn out of scope
- **Workdocs:** `docs/workdocs_multiplatform_web_android_ios_02082026/`
- **Liên quan:** Sprint 0 / T9.2.4–T9.2.5

## [2026-08-02] Skill change-workdocs + quy trình CHANGESLOG/workdocs

- **Loại:** chore
- **Phạm vi:** `.cursor/skills/change-workdocs`, root docs process
- **Tóm tắt:** Thêm Agent Skill bắt buộc ghi mọi change vào `CHANGESLOG.md` và tạo thư mục `docs/workdocs_<mo-ta>_<ddmmyyyy>` khi implement chức năng.
- **Chi tiết:**
  - Tạo skill `change-workdocs` kèm templates changelog/workdoc
  - Seed `CHANGESLOG.md` tại root
  - Ghi nhận lịch sử tài liệu PRD/architecture đã có
- **Workdocs:** `docs/workdocs_skill_change_workdocs_02082026/`
- **Liên quan:** n/a

## [2026-08-02] Tài liệu khởi tạo PRD + Architecture Gas Tam Đệ

- **Loại:** docs
- **Phạm vi:** `docs/`
- **Tóm tắt:** Viết PRD (requirement, epic/story/task, sprint) và architecture (microservice, EDA, schema, security, deploy/monorepo).
- **Chi tiết:**
  - Thêm `docs/prd.md`
  - Thêm `docs/architecture.md` (gồm §9 Deploy & Repo strategy)
- **Workdocs:** `docs/workdocs_docs_prd_architecture_02082026/`
- **Liên quan:** Sprint 0 / nền tảng tài liệu
