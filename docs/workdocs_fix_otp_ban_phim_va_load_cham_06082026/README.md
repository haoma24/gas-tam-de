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
  gstatic là một giả thuyết hợp lý cho 30s).
- **Có brotli, nên self-host không tốn thêm byte nào.** Đo thực tế:
  gstatic trả `chromium/canvaskit.wasm` **br 1.65 MB** (chỉ gzip thì nó trả
  **nguyên 5.76 MB không nén**), còn nén sẵn `brotli -q 11` của mình ra
  **1.63 MB** — nhỏ hơn CDN. Nếu chỉ có gzip -9 (2.13 MB) thì self-host sẽ đắt
  hơn CDN ~480 KB, nên brotli là điều kiện để quyết định này đúng.
- **Runtime image đổi sang nginx của Alpine** (`apk add nginx nginx-mod-http-brotli`):
  image `nginx:1.27-alpine` chính thức **không** build kèm brotli, còn package
  Alpine có module cùng version nên chắc chắn tương thích (không phải tự compile
  module, tránh lỗi “module is not binary compatible” khi base image bump).
  Lưu ý: trên Alpine, `conf.d` được include ở **root context** → server block
  phải đặt ở `/etc/nginx/http.d/default.conf`.
- **Nén sẵn khi build + `brotli_static` / `gzip_static`** thay vì nén động mức 1
  mỗi request. Vẫn giữ `.gz` vì browser chỉ gửi `Accept-Encoding: br` trên
  secure context — local `http://127.0.0.1:8090` sẽ rơi về gzip.
- **Splash trong `index.html`**: người dùng thấy logo + spinner ngay từ HTML đầu
  tiên, xoá khi Flutter bắn `flutter-first-frame`.
- **Không preload CanvasKit**: engine chọn `canvaskit/` hoặc `canvaskit/chromium/`
  theo browser; preload cố định làm Chrome tải sai file (đã thấy warning
  “preloaded but not used” khi thử).

## Follow-up 2026-08-07 (PR sau #31)

Feedback staging: bàn phím OTP vẫn không mở.

**Nguyên nhân thêm:** PR #31 vẫn `await requestOtp()` trên màn SĐT rồi mới
`context.go('/auth/otp')` — trên mobile web user-gesture hết hạn sau await, nên
`autofocus` / `requestFocus` trên màn mới không mở bàn phím. Input trong suốt
phủ ô số vẫn không đủ trên Safari.

**Fix:**

- `CustomerAuthFlowPage` + `IndexedStack`: hai bước luôn trong cây widget; bấm
  «Gửi mã OTP» đặt `_index = otp` và `_otpFocus.requestFocus()` **đồng bộ** trong
  cùng handler nút, rồi mới `unawaited(_sendOtp())`.
- `OtpEntryBlock`: ô «Nhập 6 số OTP» hiển thị (DarkTextField-style), không dùng
  `TextField` trong suốt size 0.
- Router dùng flow mới; dòng CI/CD đã xóa trong source từ 2026-08-06 — cần image
  `:stag` mới trên VPS.

## Đã làm

- [x] `OtpBoxRow` nhận `focused`, export `kOtpBoxHeight`
- [x] `AuthScrollBody` + áp cho `otp_page`, `phone_page`, `admin_login_page`
- [x] Field OTP thật, phủ kín ô số, `autofocus`, auto-verify khi đủ 6 số
- [x] Hint “Chạm vào ô để mở bàn phím” (mờ dần khi đã focus)
- [x] `AuthSessionNotifier.bootstrap()` không chặn UI quá 4s
- [x] `--no-web-resources-cdn` + nén sẵn `brotli -q 11` / `gzip -9` +
      `brotli_static` / `gzip_static` (runtime image: nginx của Alpine)
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
| `deploy/Dockerfile.web` | modified | `--no-web-resources-cdn`, stage `compress`, runtime nginx Alpine + brotli |
| `deploy/nginx.web.conf` | modified | `brotli_static`, `gzip_static`, `gzip_vary`, level 6, `default_type` cho healthz |
| `README.md` | modified | ghi chú image `web`: nginx Alpine + brotli, asset nén sẵn |

## Số liệu (build release, Flutter 3.44.8)

| Asset | Raw | gzip -9 | brotli -q 11 |
|-------|-----|---------|--------------|
| `main.dart.js` | 3.2 MB | 958 KB | **739 KB** |
| `canvaskit/chromium/canvaskit.wasm` (Chrome) | 5.6 MB | 2.13 MB | **1.63 MB** |
| `canvaskit/canvaskit.wasm` (Safari/Firefox) | 7.1 MB | 2.83 MB | 2.18 MB |

Tổng payload first load (Chrome, cache trống, qua đúng image production):

| | Byte tải về | First frame @10 Mbps | @4 Mbps | @1.5 Mbps |
|---|---|---|---|---|
| gzip (bước trung gian) | 3.26 MB | 3.4s | 7.3s | 18.2s |
| brotli (bản cuối) | **2.48 MB** | **2.8s** | **5.8s** | **14.1s** |

Đo bằng CDP (`Network.emulateNetworkConditions`, cache tắt, viewport 390×780).
`main.dart.js` vẫn 739 KB — muốn giảm tiếp phải tách route bằng deferred import
(chưa làm, xem “Việc còn lại”). Lần vào sau nhanh hơn nhiều vì Flutter service
worker đã cache toàn bộ.

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

- Đã `docker build -f deploy/Dockerfile.web` (đúng lệnh CI) và chạy image thật:
  embedded api-gateway healthy, `/healthz` + `/gateway-healthz` 200, `/v1/*`
  proxy tới auth-service OK, access log ra stdout.
- Chrome headless (CDP, viewport 390×780, touch emulation) chạy hết luồng
  SĐT → OTP → đăng nhập thành công **trên image production đó**.
- Sau khi vào màn OTP: `document.activeElement` = `INPUT#one-time-code`
  (trước đây là `FLUTTER-VIEW`, tức không có input nào được focus → không có
  bàn phím). Chạm vào ô số cũng cho `INPUT#one-time-code`.
- `Content-Encoding: br` cho `index.html`, `main.dart.js` (739 KB),
  `canvaskit/chromium/canvaskit.wasm` (1.63 MB); client chỉ nhận gzip vẫn được
  `.gz` (958 KB). `/healthz` chỉ còn một `Content-Type`.
- Splash bị xoá sau `flutter-first-frame`, 0 JS exception.

## Việc còn lại / rủi ro

- **Runtime image đổi base** (`nginx:1.27-alpine` → `alpine` + package nginx).
  Nếu Alpine bump nginx thì module brotli bump cùng nên vẫn khớp, nhưng khi
  sửa image `web` sau này phải nhớ server block nằm ở `http.d/`, và
  `/var/log/nginx/*.log` là symlink sang stdout/stderr.
- **`main.dart.js` 739 KB nén.** Tách deferred import theo route (admin vs khách)
  là bước tiếp theo nếu vẫn thấy chậm.
- **Image `web` nặng thêm ~20 MB** vì giữ cả `.br` và `.gz` bên cạnh file gốc.
  Chỉ tốn lúc VPS pull, không ảnh hưởng người dùng.
- **Font runtime.** `google_fonts` vẫn tải Be Vietnam Pro từ `fonts.gstatic.com`
  (đã `preconnect`). Nếu mạng chặn gstatic thì chữ rơi về font hệ thống — không
  chặn first frame.
- **Session hết hạn khi mạng chậm.** Sau 4s UI đi tiếp với access token cũ, có thể
  có 1 request 401 trước khi refresh nền xong. Chấp nhận được so với 30s spinner.
