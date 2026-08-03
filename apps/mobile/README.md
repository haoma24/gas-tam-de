# Gas Tam Đệ Flutter app — CTA shell (T9.2.4)

Một codebase **Web + Android + iOS** (architecture §8.4).

| Lối vào | Vai trò | Route |
|---------|---------|--------|
| **Đăng nhập** | Khách (guest) | `/` → `/auth/phone` → OTP → **shop brand** (`/`) |
| Shop sau OTP | Khách đã login | Hero brand + danh mục + CTA đặt hàng; **Hồ sơ** `/profile` (gồm đơn của tôi) |
| Admin | CCH | Deep link `/admin/login` → desk; session `role=admin` mở `/` sẽ **tự vào** `/admin` |

Cùng `lib/` trên mọi target; chỉ khác artifact build (`web` / `apk` / `ipa`).

## Yêu cầu

- Flutter **3.x** trên PATH (`flutter doctor`)
- Web: Chrome
- Android: Android Studio + emulator (API 30+ khuyến nghị)
- iOS: macOS + Xcode + Simulator (hoặc CI `macos-latest` — xem T9.2.5)

## Bootstrap platforms (lần đầu / máy mới)

Repo đã có scaffold tay: `android/` (Manifest, Gradle, icons, themes), `ios/` (Info.plist, Podfile, AppDelegate, storyboards, Flutter xcconfig), `web/` (index.html, manifest, icons).

Nếu thiếu **Xcode project** / **Gradle wrapper** (file generate bởi Flutter tooling), chạy **một lần**:

```powershell
cd apps/mobile
flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=web,android,ios
flutter pub get
```

`flutter create .` **giữ** `lib/` / `pubspec.yaml` và **không ghi đè** file platform đã chỉnh (AndroidManifest, Info.plist, …). Chỉ bổ sung phần còn thiếu (`.xcodeproj`, `gradlew`, …).

Sau create, nếu Manifest/Info.plist bị reset quyền vị trí / maps queries:

```powershell
.\platform_config\apply_location_permissions.ps1
```

## Chạy cùng codebase trên 3 target

Từ repo root (cần Flutter trên PATH):

```powershell
# Web (dev loop chính)
make flutter-web
# hoặc: .\scripts\dev.ps1 flutter-web
# hoặc:
cd apps/mobile
flutter run -d chrome

# Android emulator (bật emulator trước: flutter devices)
make flutter-android
# hoặc:
flutter run -d android
# API từ host machine → emulator:
flutter run -d android --dart-define=API_BASE_URL=http://10.0.2.2:8080

# iOS Simulator (macOS + Xcode)
make flutter-ios
# hoặc:
flutter run -d ios
```

Liệt kê thiết bị:

```powershell
make flutter-devices
# .\scripts\dev.ps1 flutter-devices
```

### Shortcut Makefile / scripts (T9.2.3 + T9.2.4)

| Target | Lệnh |
|--------|------|
| `flutter-get` | `flutter pub get` |
| `flutter-web` | `flutter run -d chrome` |
| `flutter-android` | `flutter run -d android` |
| `flutter-ios` | `flutter run -d ios` |
| `flutter-devices` | `flutter devices` |
| `flutter-create` | bootstrap `flutter create . --platforms=web,android,ios` |

Windows không có GNU Make: `.\scripts\dev.ps1 <same-name>`.

### Quyền vị trí (T3.1.1)

| Platform | Cấu hình |
|----------|----------|
| Android | `ACCESS_FINE_LOCATION` + `ACCESS_COARSE_LOCATION` trong `android/app/src/main/AndroidManifest.xml` |
| iOS | `NSLocationWhenInUseUsageDescription` trong `ios/Runner/Info.plist` |
| Web | Browser Geolocation API qua `geolocator` — **chỉ HTTPS hoặc `localhost`** |

## Sprint 0–2 UI

- Brand **Gas Tam Đệ** (guest landing + shop sau OTP)
- Guest: chỉ CTA **Đăng nhập** → SĐT → OTP → **shop**
- Shop: catalogue + CTA đặt hàng; bottom nav Cửa hàng / Hồ sơ
- **Hồ sơ** (`/profile`): SĐT ẩn, họ tên (`GET/PATCH /v1/me`), **Đơn hàng của tôi**, đăng xuất
- Admin: mở `/#/admin/login` (không CTA trên Home); session admin tự điều hướng `/admin`
- OTP: `POST /v1/auth/otp/request` + `verify`
- Admin: `POST /v1/auth/admin/login`
- Session: persist local (`shared_preferences`) + `POST /v1/auth/refresh` lúc mở app
- Catalog: `GET /v1/products` (khách) · `GET/POST/PATCH /v1/admin/products` (admin)
- Order Desk (admin): `GET /v1/admin/orders` (FIFO, PENDING)
- Delivery fee (admin): `GET/PUT /v1/admin/delivery-fee`
- Vị trí cửa hàng (admin): `GET /v1/geo/store` · `PUT /v1/admin/geo/store`
- Công nợ (admin): `GET /v1/admin/debts`
- Tồn kho (admin): `GET/POST /v1/admin/inventory` (IN / OUT / ADJUST)
- Dashboard (admin): `GET /v1/admin/dashboard/summary`
- Place order: `POST /v1/orders` (JWT customer + cart + địa chỉ)

## Config API

Mặc định: `http://127.0.0.1:8080` (gateway — xem `lib/core/api_config.dart`).

Gateway chưa proxy đầy đủ upstream. Local tách theo service:

**Auth (login / OTP):**

```powershell
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8081
```

**Catalog (admin + khách chọn SP):**

```powershell
# catalog-service phải chạy: go run ./services/catalog-service
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8082
```

Android emulator → host: `--dart-define=API_BASE_URL=http://10.0.2.2:8082`.

Seed admin mặc định (auth-service): username `admin` / password `admin-change-me` (đổi qua env khi deploy).

### Verify nhanh CTA shell (T9.2.4)

1. `flutter pub get` trong `apps/mobile`.
2. Web: `flutter run -d chrome` → Home thấy **Gas Tam Đệ** + **Đăng nhập** only.
3. OTP xong → shop; **Hồ sơ** → Đơn hàng của tôi / sửa tên / đăng xuất.
4. Admin: mở `/#/admin/login` → desk; refresh `/` vẫn vào admin khi còn session.

### Verify nhanh màn sản phẩm (admin)

1. Chạy catalog: `go run ./services/catalog-service`
2. Flutter với `API_BASE_URL=…:8082` → mở `/admin/products` (sau `/#/admin/login` → desk → Sản phẩm; session có thể trống nếu chưa login — list API không bắt JWT trên catalog).
3. **Thêm** → nhập SKU / tên / giá → **Tạo sản phẩm** → thấy trong list → tap để sửa hoặc icon mắt để ẩn/hiện.

### Verify Order Desk (admin) — T5.1.3 / T5.1.4

Cần **order-service** (`GET /v1/admin/orders`) và ít nhất 1 đơn `PENDING`:

```powershell
# terminal 1
go run ./services/order-service

# terminal 2
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

1. Mở `/admin/orders` (hoặc desk → **Order Desk**).
2. List hiện STT | tên | SĐT masked | địa chỉ | km | thời gian — cũ nhất trước.
3. Tap một dòng → chi tiết đọc-only (từ payload list; chưa gọi `GET /admin/orders/{id}`).
4. Pull-to-refresh / icon tải lại.
5. **Polling (T5.1.4):** để desk mở ~10s; tạo thêm đơn PENDING từ client khác — list tự cập nhật + SnackBar «Có N đơn mới». (MVP dùng poll `GET /v1/admin/orders`, không SSE/NATS bridge.)

### Verify deep-link Maps helper — T5.2.2

Helper `openNavigationTo(lat, lng)` trong `lib/features/order/navigation_link.dart`.

```powershell
cd apps/mobile
flutter test test/navigation_link_test.dart
```

### Verify nút «Dẫn đường» Order Desk — T5.2.3

1. Admin login → Order Desk → mở chi tiết đơn có `lat`/`lng`.
2. Bấm **Dẫn đường** → Web mở Google Maps directions; Android/iOS mở Maps app (nếu có) tới điểm giao (origin = vị trí thiết bị).
3. Đơn thiếu toạ độ (`0,0` / null parse) → SnackBar «Đơn không có toạ độ điểm giao.»

### Verify dialog hoàn tất đơn — T6.1.4

Cần **order-service** (`POST /v1/admin/orders/{id}/complete`) + admin JWT:

```powershell
# local: trỏ thẳng order-service
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

1. Admin login → Order Desk → mở chi tiết đơn PENDING.
2. Bấm **Hoàn tất** → chọn **Đã thu đủ** / **Thu một phần** / **Chưa thu (nợ)**.
3. PARTIAL: nhập số tiền đã thu (`0 < paid < total`) → **Xác nhận**.
4. Kỳ vọng: SnackBar thu/nợ; quay về list; đơn biến mất khỏi PENDING.
5. PARTIAL sai (0 / ≥ total) → lỗi trên dialog, không gọi API.

### Verify màn Dashboard (admin) — T8.1.3

Cần **report-service** (`GET /v1/admin/dashboard/summary`) có `daily_stats` (và tùy chọn debt snapshot):

```powershell
# terminal 1
go run ./services/report-service

# terminal 2
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8087
```

1. Đăng nhập admin → `/admin` (desk).
2. Widgets **Doanh thu / Lợi nhuận / Phí giao / Công nợ / Đơn hoàn tất / Đơn đặt**.
3. Chips **Hôm nay / 7 ngày / Tháng này**; pull-to-refresh / icon tải lại.
4. Tap **Công nợ** trên widget → `/admin/debts`.

### Verify màn Công nợ (admin) — T6.2.2

Cần **billing-service** (`GET /v1/admin/debts`) có ít nhất 1 dòng `balance > 0`:

```powershell
# terminal 1
go run ./services/billing-service

# terminal 2
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8086
```

1. Mở `/admin/debts` (hoặc desk → **Công nợ**).
2. Banner **Tổng công nợ** + số khách; list SĐT masked / customer_key + số tiền nợ (cao → thấp).
3. Pull-to-refresh / icon tải lại; empty → «Không có công nợ».

### Verify màn Tồn kho (admin) — T7.1.4

Cần **inventory-service** (`GET/POST /v1/admin/inventory`):

```powershell
# terminal 1
go run ./services/inventory-service

# terminal 2
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8085
```

1. Mở `/admin/inventory` (hoặc desk → **Tồn kho**).
2. Empty → **Nhập kho mới** (product_id / SKU / tên / qty / giá nhập).
3. List: tên · SKU · tồn · giá vốn; chip «Sắp hết» khi `on_hand ≤ reorder_level`.
4. Menu ⋮ trên dòng → **Nhập** / **Xuất** / **Điều chỉnh** → SnackBar + list cập nhật.
5. Pull-to-refresh / icon tải lại.

### Verify màn phí giao hàng (admin) — T4.1.4

Cần **order-service** đã seed `delivery_fee_*` (`GET/PUT /v1/admin/delivery-fee`):

```powershell
# terminal 1
go run ./services/order-service

# terminal 2 — admin JWT: login auth trước hoặc mock session; local trỏ thẳng order
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

1. Mở `/admin/delivery-fee` (hoặc desk → **Phí giao hàng**).
2. Switch **Thu phí giao hàng** → bật/tắt (PUT `enabled` ngay).
3. Sửa min/max/phí các bậc → **Lưu bậc** → snackbar thành công; max trống = không giới hạn.
4. **Thêm bậc** / xóa / tắt «Đang áp dụng» → Lưu; overlap → lỗi tiếng Việt từ API hoặc validate local.

### Verify chọn SP khi đặt hàng (T2.2.2)

1. Catalog chạy (`:8082`) và đã có ít nhất 1 SP `active`.
2. Flutter `API_BASE_URL=…:8082` → mở `/order` (hoặc OTP xong → `/order`).
3. Tăng/giảm số lượng → footer hiện tổng → **Tiếp tục** → màn địa chỉ (giỏ vẫn giữ trong session).

### Verify quyền vị trí (T3.1.1)

1. `flutter pub get` trong `apps/mobile` (cần Flutter SDK).
2. Web: `flutter run -d chrome` → OTP/order → **Tiếp tục** → `/order/address` → **Dùng vị trí hiện tại** → cho phép trong trình duyệt → hiện lat/lng. Từ chối → message tiếng Việt.
3. Android Emulator: bật mock location → cùng flow; deny / deny forever → message tương ứng.
4. iOS Simulator: Features → Location → Custom Location → cùng flow.

### Verify map + autocomplete địa chỉ (T3.1.3)

Cần **geo-service** (`GET /v1/geo/search`) — Flutter không gọi OSM/Photon trực tiếp:

```powershell
# terminal 1
go run ./services/geo-service

# terminal 2
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8083
```

1. Mở `/order/address` → gõ ≥ 2 ký tự (vd. `ben thanh`) → danh sách gợi ý → chọn → pin + label trên bản đồ.
2. Chạm bản đồ → cập nhật lat/lng («Vị trí đã chọn trên bản đồ» nếu chưa có label search).
3. **Dùng vị trí hiện tại** → pin GPS (cần quyền vị trí như T3.1.1).
4. Android emulator: `--dart-define=API_BASE_URL=http://10.0.2.2:8083`.

Bản đồ dùng `flutter_map` + tile OSM (Web/Android/iOS, không API key). Sau địa chỉ in-range → review + place order (T3.3.3).

### Verify ngoài phạm vi giao (T3.2.3)

Cần **geo-service** đã seed `store_settings` (`POST /v1/geo/check`):

```powershell
# terminal 1
go run ./services/geo-service

# terminal 2
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8083
```

1. Mở `/order/address` → chọn điểm gần cửa hàng (vd. search `ben thanh` hoặc lat≈10.78, lng≈106.70) → banner xanh «Trong phạm vi giao» → **Tiếp tục** bật.
2. Chạm bản đồ / chọn điểm xa (vd. lat≈10.90 cùng lng) → banner đỏ rõ: ngoài phạm vi + khoảng cách / bán kính tối đa → nút **Ngoài phạm vi giao** (disabled).
3. Chọn lại điểm trong phạm vi → banner xanh, **Tiếp tục** bật lại.

### Verify review + đặt đơn (T3.3.3)

Cần **order-service** (`POST /v1/orders`) + catalog + geo đã chạy (order gọi upstream). Local Flutter trỏ thẳng order-service; app gửi `Authorization` + `X-User-*` từ session OTP:

```powershell
# terminals: catalog :8082, geo :8083, order :8084 (+ NATS nếu order publish)
go run ./services/catalog-service
go run ./services/geo-service
go run ./services/order-service

# Flutter — place order (session OTP cần auth :8081 trước, hoặc mock session)
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8084
```

1. Flow đầy đủ (khi gateway proxy sẵn): OTP → `/order` → chọn SP → địa chỉ in-range → **Tiếp tục** → `/order/review`.
2. Review: thấy giỏ + địa chỉ + tạm tính / phí 0 / tổng; nhập **tên người nhận** → **Đặt đơn**.
3. Thành công → `/order/success` (mã đơn ngắn, tổng từ API) → **Về trang chủ**.
4. Lỗi `OUT_OF_RANGE` / hết phiên → banner đỏ trên review, không điều hướng success.

## Platform checklist (T9.2.5)

Xem **[PLATFORM_CHECKLIST.md](./PLATFORM_CHECKLIST.md)** — audit deps (`geolocator` / `url_launcher` / `flutter_map`), fallback Maps & GPS, bước verify Web + Android Emulator + iOS Simulator (hoặc CI macOS), và workflow `.github/workflows/flutter-ci.yml`.
