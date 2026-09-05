# Nút bật/tắt light mode + sửa mất icon «Báo cáo» / «Cài đặt» trên admin

- **Thư mục:** `docs/workdocs_theme_toggle_va_icon_admin_05092026`
- **Ngày:** 05/09/2026
- **Loại:** feat + fix
- **Liên quan:** `docs/workdocs_refactor_ui_minimalism_05092026/` (bản refactor đưa dark mode vào app)

## Hai vấn đề được báo

1. **Không tìm thấy nút bật/tắt light mode.** Đúng — nút đó chưa từng tồn tại.
2. **Tab «Báo cáo» và «Cài đặt» của admin mất icon**, chỉ còn ô trống, trong khi
   «Đơn» và «Kho» vẫn hiện bình thường.

---

## 1. Light mode: vì sao "mất"

Bản refactor minimalism thêm dark theme và `main.dart` để cứng:

```dart
themeMode: ThemeMode.system,
```

Không có chỗ nào ghi đè. Máy đang để dark ⇒ app **luôn** tối, không có đường ra.
Bản trước đó (`8d8294c`) chỉ có light nên trông như app "tự dưng đổi màu".

### Cách sửa

Thêm `apps/mobile/lib/core/ui/app_theme_mode.dart`:

- `ThemeModeController extends StateNotifier<ThemeMode>` đọc/ghi
  `SharedPreferences` với khoá `gas_tam_de.theme_mode.v1` (giá trị `system` /
  `light` / `dark`). Dùng lại `sharedPreferencesProvider` sẵn có trong
  `features/auth/auth_session.dart` nên state khôi phục ngay ở lần build đầu,
  không nhấp nháy theme khi mở app.
- `AppThemeModeSection` — một `AppSection` tiêu đề «Giao diện» chứa
  `SegmentedButton<ThemeMode>` ba lựa chọn **Hệ thống / Sáng / Tối**, kèm dòng
  chú thích. Widget dùng chung để hai vai không lệch nhau.

`main.dart` đổi `themeMode: ThemeMode.system` → `themeMode: ref.watch(themeModeProvider)`.

### Nút nằm ở đâu

| Vai | Đường đi |
|---|---|
| Admin | Tab **Cài đặt** → khối «Giao diện» (ngay dưới thẻ tài khoản đang đăng nhập) |
| Khách | Tab **Hồ sơ** → khối «Giao diện» (ngay trên «Đơn hàng của tôi») |

Lựa chọn được lưu trên máy, giữ nguyên sau khi tải lại trang / mở lại app.

---

## 2. Icon admin bị trống: nguyên nhân thật là **cache font**, không phải code

### Điều tra

Giả thuyết đầu tiên là `const_finder` (bộ tree-shake icon của Flutter) bỏ sót
`IconData` nằm trong const *record* — `admin_shell.dart` khai báo tab bằng
`<({IconData icon, IconData selected, String label})>[...]`. Đã **loại giả thuyết
này**: parse bảng `cmap` của `MaterialIcons-Regular.otf` vừa build ra thì cả 8
codepoint (`receipt_long`, `inventory_2`, `bar_chart`, `settings` — bản outline và
rounded) đều **có mặt**. Build lại `flutter build web --release` và mở Chrome:
bốn icon hiện đủ.

### Nguyên nhân

```
git grep "Icons\.\(bar_chart\|settings_\)" 8d8294c -- 'apps/mobile/lib'   # → rỗng
```

Bản deploy **trước** không dùng `bar_chart` và `settings` ở bất cứ đâu, nên font
tree-shaken của bản đó **không chứa glyph** cho hai icon này. Ghép ba điều kiện:

1. Flutter phục vụ `main.dart.js` và `assets/**` ở **URL cố định không hash** —
   `assets/fonts/MaterialIcons-Regular.otf` của bản mới trùng URL với bản cũ.
2. Service worker của SDK này đã deprecated và **tự huỷ đăng ký**, không cache
   thay, nên độ tươi phụ thuộc hoàn toàn vào HTTP header.
3. `deploy/nginx.web.conf` **không gửi `Cache-Control`** cho hai nhóm đó. Không
   có header ⇒ trình duyệt áp *heuristic freshness* và tự bịa hạn dùng.

Kết quả: trình duyệt tải JS mới nhưng **dùng lại font cũ**, và font cũ thiếu đúng
hai glyph vừa được thêm ⇒ «Báo cáo» và «Cài đặt» vẽ ra ô trống, phần còn lại của
UI vẫn đúng.

### Cách sửa

`deploy/nginx.web.conf` — ép revalidate bằng ETag cho các đường dẫn không hash:

```nginx
location ~ ^/(main\.dart\.js|flutter\.js|flutter_bootstrap\.js)$ {
    add_header Cache-Control "no-cache";
}

location /assets/ {
    add_header Cache-Control "no-cache";
    try_files $uri =404;
}

location /canvaskit/ {
    add_header Cache-Control "no-cache";
    try_files $uri =404;
}
```

`no-cache` **không** tắt cache — nó chỉ buộc hỏi lại server trước khi dùng. File
không đổi trả `304`, tốn đúng một round trip; file đã đổi được tải mới. `etag on`
đã bật sẵn từ trước ở đầu file.

### Máy đang bị lỗi cần làm gì

Header mới chỉ áp cho **lần tải sau**. Máy đang giữ font cũ trong cache phải
**hard reload một lần** (`Ctrl+Shift+R`, hoặc `Cmd+Shift+R` trên macOS) để bỏ
font cũ. Từ sau lần đó, mọi deploy tự lấy đúng asset.

---

## Kiểm chứng

- `flutter analyze lib` — 7 info, **đều tồn tại từ trước** (dangling doc comment,
  deprecation của `Radio`); không phát sinh mới.
- `flutter test` — 56/56 pass (54 cũ + 2 mới).
- `apps/mobile/test/theme_mode_test.dart` (mới):
  - mặc định là `ThemeMode.system`; `set()` ghi xuống prefs; `ProviderContainer`
    mới mồi sẵn `{'gas_tam_de.theme_mode.v1': 'light'}` khôi phục đúng light.
  - dựng `AppThemeModeSection`, kiểm tiêu đề + ba nhãn, chạm «Tối» rồi «Sáng» và
    xác nhận state provider đổi theo.
- `flutter build web --release` — thành công, bốn icon admin hiện đủ khi mở thật.

### Chưa kiểm được

- **`nginx -t`** cho file config: Docker daemon không chạy trên máy này. Cú pháp
  theo đúng khuôn các `location` đã có trong cùng file; CI `web-image.yml` sẽ
  build image và nginx sẽ tự báo nếu sai.
- **Rà mắt admin nav ở chế độ light**: extension Chrome mất kết nối giữa chừng.
  Nav dùng token `context.palette` chung nên không có nhánh màu riêng, nhưng vẫn
  nên liếc lại sau khi deploy.

## Ảnh hưởng

- Không đổi API, không đổi schema, không đổi hợp đồng service nào.
- Người dùng cũ chưa từng chọn gì ⇒ vẫn `ThemeMode.system`, hành vi y như trước.
