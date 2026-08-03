# Platform checklist — Web + Android + iOS (T9.2.5)

Một codebase Flutter (`apps/mobile`) phải chạy được trên **Web**, **Android Emulator**, và **iOS Simulator** (hoặc chứng minh qua CI macOS). Không dùng API chỉ có trên một OS nếu chưa có **fallback** chung.

Flutter SDK **không bắt buộc** trên PATH của máy Windows dev — checklist này mô tả bước verify thủ công khi đã cài Flutter.

---

## 1. Quy tắc “không single-OS API”

| Được phép | Không được phép (MVP) |
|-----------|------------------------|
| Plugin đa nền tảng (`geolocator`, `url_launcher`, `flutter_map`) với nhánh `kIsWeb` / `defaultTargetPlatform` + fallback | `dart:io` File/Socket trong `lib/` (gãy Web) |
| Deep-link native (`geo:`, `maps:`, `comgooglemaps:`) **kèm** HTTPS fallback | `google_maps_flutter` / MapKit-only UI không có Web |
| Permission when-in-use với message tiếng Việt khi deny | Background location / Always |
| Tile OSM qua `flutter_map` (không API key) | Gọi OSM/Photon trực tiếp từ client (phải qua geo-service) |

**Review gate mỗi PR Flutter:** grep `dart:io`, `dart:html`, `Platform.is`, plugin Android/iOS-only — nếu có, bắt buộc có nhánh Web + fallback UX.

---

## 2. Audit phụ thuộc & fallback hiện có

### 2.1 `pubspec.yaml` (platform-sensitive)

| Package | Vai trò | Web | Android | iOS | Fallback / ghi chú |
|---------|---------|-----|---------|-----|-------------------|
| `geolocator` | GPS when-in-use | Browser Geolocation (`localhost`/HTTPS) | FINE/COARSE | When-in-use Info.plist | `location_permission.dart`: message TV; Web vẫn gọi `getCurrentPosition` khi Permissions API báo denied |
| `url_launcher` | Mở Maps / deep-link | Tab HTTPS | Intent + schemes | URL schemes | `navigation_link.dart`: thử native → luôn kết thúc bằng HTTPS Google Maps |
| `flutter_map` + `latlong2` | Bản đồ pin / search | OSM raster | OSM raster | OSM raster | Không cần API key; không dùng Google Maps SDK |
| `dio` / `go_router` / `flutter_riverpod` | HTTP + routing | OK | OK | OK | Thuần Dart/Flutter |

**Không có trong deps:** `permission_handler` riêng, `google_maps_flutter`, `map_launcher`, `dart:io` trong `lib/`.

### 2.2 Code paths

| File | Hành vi đa nền tảng |
|------|---------------------|
| `lib/features/order/location_permission.dart` | `Geolocator` chung 3 target; `kIsWeb` cho phép tiếp tục sau `denied` (prompt trình duyệt) |
| `lib/features/order/navigation_link.dart` | Web → HTTPS only; Android → `google.navigation` → `geo:` → `comgooglemaps` → HTTPS; iOS → `comgooglemaps` → Apple `maps:` → HTTPS; desktop/default → HTTPS |
| `lib/features/order/order_address_page.dart` | `flutter_map` + geo-service API (search/check) — không gọi Maps SDK native |
| `platform_config/` | Fragment Manifest / Info.plist + script re-apply sau `flutter create .` |

Unit: `test/navigation_link_test.dart` (URI helpers + HTTPS luôn là candidate cuối).

### 2.3 Cấu hình native tối thiểu

| Platform | Location | Maps queries |
|----------|----------|--------------|
| Android | `ACCESS_FINE_LOCATION` + `ACCESS_COARSE_LOCATION` | `<queries>` cho Maps packages (API 30+) |
| iOS | `NSLocationWhenInUseUsageDescription` | `LSApplicationQueriesSchemes` (`comgooglemaps`, `maps`, …) |
| Web | Không meta đặc biệt | HTTPS Google Maps trong tab mới |

Chi tiết merge: `platform_config/README.md`.

---

## 3. Verify thủ công

**Điều kiện:** Flutter 3.x trên PATH (`flutter doctor`). Nếu chưa cài — cài SDK rồi chạy các bước dưới; CI (mục 4) cover analyze + web (+ iOS no-codesign) khi không có máy local.

```powershell
cd apps/mobile
flutter pub get
flutter analyze
flutter test
```

### 3.1 Web (bắt buộc local)

```powershell
flutter run -d chrome
# hoặc từ root: make flutter-web / .\scripts\dev.ps1 flutter-web
```

Checklist:

- [ ] Home (guest): brand **Gas Tam Đệ** + chỉ CTA **Đăng nhập**
- [ ] Sau OTP: shop + hồ sơ (đơn hàng trong hồ sơ); admin session `/` → `/admin`
- [ ] Flow địa chỉ: **Dùng vị trí hiện tại** → prompt trình duyệt → lat/lng hoặc message từ chối
- [ ] Bản đồ `flutter_map` hiển thị (cần mạng cho tile OSM)
- [ ] Order Desk → **Dẫn đường** → tab Google Maps directions (HTTPS)

API local: `--dart-define=API_BASE_URL=http://127.0.0.1:<port>` (gateway `8080` hoặc service trực tiếp — xem README).

### 3.2 Android Emulator (bắt buộc local)

```powershell
# Bật emulator trước
flutter devices
flutter run -d android --dart-define=API_BASE_URL=http://10.0.2.2:8080
```

Checklist:

- [ ] Cùng Home CTA như Web
- [ ] Location: Extended Controls → Location (mock) → **Dùng vị trí hiện tại**
- [ ] Deny / Deny forever → message tiếng Việt
- [ ] **Dẫn đường** → mở Maps / chooser (nếu app Maps cài) hoặc HTTPS fallback

### 3.3 iOS Simulator **hoặc** CI macOS

**A — Simulator (macOS + Xcode):**

```bash
flutter run -d ios
# Features → Location → Custom Location khi test GPS
```

- [ ] Cùng CTA / location / maps deep-link (Apple Maps hoặc Google Maps nếu cài)

**B — Không có Mac:** dựa vào GitHub Actions job `ios-build` trong `.github/workflows/flutter-ci.yml`:

```text
flutter build ios --no-codesign
```

Job xanh = compile iOS OK (không thay thế UAT Simulator, nhưng đủ DoD Sprint khi không có Mac local).

---

## 4. CI (không secret)

Workflow: `.github/workflows/flutter-ci.yml`

| Job | Runner | Lệnh chính |
|-----|--------|------------|
| `flutter-analyze-web` | `ubuntu-latest` | `flutter analyze`, `flutter test`, `flutter build web --release` |
| `ios-build` | `macos-latest` | `flutter build ios --no-codesign` |

Không dùng GitHub Secrets. Trigger: PR + push `main` khi đổi `apps/mobile/**` hoặc chính workflow.

---

## 5. Sign-off T9.2.5

| Hạng mục | Trạng thái |
|----------|------------|
| Không API single-OS thiếu fallback | Đạt (audit mục 2) |
| Checklist verify Web | Mục 3.1 |
| Checklist verify Android Emulator | Mục 3.2 |
| iOS Simulator **hoặc** CI macOS | Mục 3.3 + workflow `ios-build` |
| Docs workdocs | `docs/workdocs_platform_checklist_02082026/` |

Khi Flutter chưa trên PATH máy hiện tại: audit + docs + CI đủ để đóng task; chạy tay mục 3 sau khi cài SDK / trên máy có emulator.
