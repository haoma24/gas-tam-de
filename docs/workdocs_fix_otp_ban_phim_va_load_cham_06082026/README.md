# Fix bàn phím màn OTP + load trang chậm (~30s)

- **Thư mục:** `docs/workdocs_fix_otp_ban_phim_va_load_cham_06082026`
- **Ngày:** 06/08/2026
- **Loại:** fix
- **Liên quan:** báo cáo staging `tamde-stag.tinhgon.xyz` (nhấn “Gửi mã OTP” → màn OTP
  không mở bàn phím; chờ ~30s mới thấy trang đăng nhập)

## Mục tiêu

1. Người dùng vào màn OTP là nhập được ngay — bàn phím trên mobile web phải mở.
2. Trang đăng nhập hiện nhanh, không còn màn hình đen/spinner ~30s.

## Phạm vi

- Trong scope:
  - `apps/mobile` màn OTP + màn SĐT + màn admin login (layout an toàn với bàn phím)
  - Startup: `AuthSessionNotifier.bootstrap()` không chặn UI vì network
  - Web shell + build/serve: `web/index.html`, `deploy/Dockerfile.web`, `deploy/nginx.web.conf`
- Ngoài scope:
  - Code splitting / deferred import cho `main.dart.js`
  - Bundle font Be Vietnam Pro vào assets (vẫn dùng `google_fonts` runtime)
  - Brotli (image `nginx:1.27-alpine` không có module brotli)

## Nguyên nhân

### 1. Không mở được bàn phím ở màn OTP

`otp_page.dart` vẽ 6 ô số bằng `OtpBoxRow` (chỉ là decoration) và đặt `TextFormField`
thật trong `SizedBox(height: 0)` với `fontSize: 0`. Trên Flutter Web, element
`<input>` trong DOM được đặt đúng vị trí/kích thước của widget → input cao 0px.
Browser mobile không mở bàn phím cho một input kích thước 0, và `requestFocus()`
trong `addPostFrameCallback` không có user gesture nên cũng không mở được.

Thêm nữa: `Column` + `Spacer` với `Scaffold.resizeToAvoidBottomInset = true` sẽ
**overflow** khi bàn phím thu nhỏ viewport, thay vì đẩy ô nhập lên.

### 2. Chờ ~30s mới thấy trang đăng nhập

- `authBootstrapProvider` chặn toàn bộ router bằng spinner cho tới khi
  `bootstrap()` xong. `bootstrap()` `await` refresh token khi access token hết hạn,
  còn Dio cấu hình `connectTimeout 15s` + `receiveTimeout 15s` → **đúng ~30s** khi
  API chậm/không tới được.
- First load tải CanvasKit từ `www.gstatic.com` (~2.1 MB nén) + `main.dart.js`
  (~1.05 MB với gzip mức 1 của nginx), trong khi `index.html` trống trơn nên người
  dùng chỉ thấy màn đen suốt thời gian đó.

## Quyết định chính

- **Ô OTP = decoration, input phủ lên trên.** `Stack` cao `kOtpBoxHeight` (58):
  `OtpBoxRow` bọc `IgnorePointer` ở dưới, `TextField` trong suốt phủ kín ở trên.
  Tap rơi trực tiếp vào field → Flutter focus DOM input trong chính gesture đó →
  bàn phím mở. Bỏ `GestureDetector` + field 0px.
- **`autofocus: true`** cho field OTP: giữ bàn phím đang mở từ bước 1 (browser
  không cho mở lại bằng focus programmatic sau `await` mạng).
- **Tự xác nhận khi đủ 6 số**, bỏ `Form`/`validator` trả `''` (hack cũ), lỗi thiếu
  số báo bằng `_error` như các lỗi khác.
- **`AuthScrollBody`** (widget mới, dùng cho cả 3 màn auth): `SingleChildScrollView`
  + `ConstrainedBox(minHeight: viewport)` + `MainAxisAlignment.spaceBetween`.
  Dùng `spaceBetween` chứ **không** dùng `Spacer`/`Expanded` vì trong scroll view
  chiều cao là unbounded, flex sẽ throw.
- **Startup không chờ mạng:** `bootstrap()` publish session đã lưu trước (routing
  chỉ cần `role`), refresh chỉ được **4s** (`bootstrapRefreshTimeout`) rồi UI đi
  tiếp; `Future.timeout` không cancel nên refresh vẫn chạy nền và cập nhật session.
- **CanvasKit chạy từ origin của mình** (`--no-web-resources-cdn`): bỏ một
  DNS + TLS handshake sang gstatic và bỏ phụ thuộc bên thứ ba (ISP chặn/chậm
  gstatic là một giả thuyết hợp lý cho 30s). Đổi lại mất brotli của CDN, nên bù
  bằng gzip -9 dựng sẵn.
- **Nén sẵn khi build + `gzip_static on`** thay vì nén động mức 1 mỗi request.
- **Splash trong `index.html`**: người dùng thấy logo + spinner ngay từ HTML đầu
  tiên, xoá khi Flutter bắn `flutter-first-frame`.
- **Không preload CanvasKit**: engine chọn `canvaskit/` hoặc `canvaskit/chromium/`
  theo browser; preload cố định làm Chrome tải sai file (đã thấy warning
  “preloaded but not used” khi thử).

## Đã làm

- [x] `OtpBoxRow` nhận `focused`, export `kOtpBoxHeight`
- [x] `AuthScrollBody` + áp cho `otp_page`, `phone_page`, `admin_login_page`
- [x] Field OTP thật, phủ kín ô số, `autofocus`, auto-verify khi đủ 6 số
- [x] Hint “Chạm vào ô để mở bàn phím” (mờ dần khi đã focus)
- [x] `AuthSessionNotifier.bootstrap()` không chặn UI quá 4s
- [x] `--no-web-resources-cdn` + gzip -9 build-time + `gzip_static`
- [x] Splash + preload `main.dart.js` + preconnect `fonts.gstatic.com`
- [x] `/healthz` dùng `default_type` (trước đó trả 2 header `Content-Type`)
- [x] Bỏ dòng test “🚀 Test CI/CD tự động — GCP stag v2” trên màn đăng nhập
- [x] `test/otp_page_test.dart` (3 test) + verify thật trên Chrome headless

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/auth/otp_page.dart` | modified | field thật phủ ô số, auto-verify, layout scroll |
| `apps/mobile/lib/features/auth/_auth_widgets.dart` | modified | `AuthScrollBody`, `kOtpBoxHeight`, `OtpBoxRow.focused` |
| `apps/mobile/lib/features/auth/phone_page.dart` | modified | layout scroll, bỏ dòng test CI/CD |
| `apps/mobile/lib/features/auth/admin_login_page.dart` | modified | layout scroll (bàn phím che nút Đăng nhập) |
| `apps/mobile/lib/features/auth/auth_session.dart` | modified | bootstrap không chặn UI, timeout 4s |
| `apps/mobile/web/index.html` | modified | splash, preload, preconnect, viewport meta |
| `apps/mobile/test/otp_page_test.dart` | added | regression: field có kích thước, tap focus, auto-verify, bàn phím |
| `deploy/Dockerfile.web` | modified | `--no-web-resources-cdn`, gzip -9 |
| `deploy/nginx.web.conf` | modified | `gzip_static`, `gzip_vary`, level 6, `default_type` cho healthz |

## Số liệu (build release, Flutter 3.44.8)

| Asset | Raw | gzip -9 |
|-------|-----|---------|
| `main.dart.js` | 3.2 MB | 958 KB (nginx mức 1: ~1.05 MB) |
| `canvaskit/chromium/canvaskit.wasm` (Chrome) | 5.6 MB | 2.13 MB |
| `canvaskit/canvaskit.wasm` (Safari/Firefox) | 7.1 MB | 2.83 MB |

`main.dart.js` vẫn ~1 MB — muốn giảm tiếp phải tách route bằng deferred import
(chưa làm, xem “Việc còn lại”).

## Cách verify

### Unit / widget

```bash
cd apps/mobile && flutter analyze && flutter test
```

`test/otp_page_test.dart` chốt lại: field OTP cao `kOtpBoxHeight` (không còn 0px),
tap vào ô số thì field nhận focus, nhập 6 số là gọi verify, và layout không
overflow khi `viewInsets.bottom = 420`.

### Thủ công (browser thật, nên dùng điện thoại)

1. `make nats` (hoặc chỉ cần auth-service + api-gateway), `OTP_DEV_REVEAL=1`.
2. Mở `#/auth/phone`, nhập SĐT → “Gửi mã OTP”.
3. Màn OTP: bàn phím số phải đang mở; nếu đã đóng, chạm vào dãy ô → mở lại.
4. Nhập đủ 6 số → tự xác nhận, không cần bấm “Xác nhận”.
5. Mở bàn phím rồi kéo trang: không có dải overflow vàng/đen, ô nhập không bị che.

### Đã verify trong môi trường agent

- Chrome headless (CDP, viewport 390×780, touch emulation) chạy hết luồng
  SĐT → OTP → đăng nhập thành công trên **build release** phục vụ bởi đúng
  `deploy/nginx.web.conf` (container `nginx:1.27-alpine`, asset nén sẵn).
- Sau khi vào màn OTP: `document.activeElement` = `INPUT#one-time-code`
  (trước đây là `FLUTTER-VIEW`, tức không có input nào được focus → không có
  bàn phím). Chạm vào ô số cũng cho `INPUT#one-time-code`.
- `nginx -t` OK; `Content-Encoding: gzip` + `Vary` cho `main.dart.js`,
  `canvaskit/**.wasm`, `index.html`; `/healthz` chỉ còn một `Content-Type`.
- Splash bị xoá sau `flutter-first-frame`, 0 JS exception.

## Việc còn lại / rủi ro

- **Không có brotli.** `nginx:1.27-alpine` không build kèm module brotli; nếu muốn
  ~15–20% nữa thì phải đổi base image hoặc build module.
- **`main.dart.js` ~1 MB nén.** Tách deferred import theo route (admin vs khách)
  là bước tiếp theo nếu vẫn thấy chậm.
- **Font runtime.** `google_fonts` vẫn tải Be Vietnam Pro từ `fonts.gstatic.com`
  (đã `preconnect`). Nếu mạng chặn gstatic thì chữ rơi về font hệ thống — không
  chặn first frame.
- **Session hết hạn khi mạng chậm.** Sau 4s UI đi tiếp với access token cũ, có thể
  có 1 request 401 trước khi refresh nền xong. Chấp nhận được so với 30s spinner.
