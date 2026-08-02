# Xin quyền location (Web / Android / iOS)

- **Thư mục:** `docs/workdocs_location_permission_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.1 / Sprint 2 / T3.1.1

## Mục tiêu

Khách trên Web / Android / iOS xin được quyền vị trí và lấy lat/lng hiện tại trong bước địa chỉ đặt hàng («Dùng vị trí hiện tại»), với message lỗi tiếng Việt rõ khi denied / deniedForever / serviceDisabled.

## Phạm vi

- Trong scope:
  - Platform permission config (AndroidManifest, Info.plist, ghi chú Web HTTPS/localhost)
  - Helper `location_permission.dart` (geolocator)
  - Wire `/order/address` + nút «Dùng vị trí hiện tại» hiển thị lat/lng hoặc lỗi
  - PRD mark DONE T3.1.1
- Ngoài scope:
  - Geocode / reverse geocode (T3.1.2)
  - Map picker / autocomplete (T3.1.3)
  - Kiểm tra bán kính giao (US-3.2)

## Quyết định chính

- Dùng **`geolocator`** cho check/request permission + `getCurrentPosition` trên cả 3 target. Bỏ `permission_handler` khỏi `pubspec` — thừa cho location; geolocator cover Web tốt hơn.
- Fragment + script áp quyền: `apps/mobile/platform_config/` (sau `flutter create .` nếu cần).
- Tạo sẵn permission trong `android/` `ios/` `web/`; máy chưa có Flutter SDK vẫn cần `flutter create .` để bổ sung Gradle/Xcode đầy đủ (không ghi đè Manifest/Info.plist đã có).
- Web: Geolocation chỉ chạy trên HTTPS hoặc localhost; nếu Permissions API thiếu, vẫn gọi `getCurrentPosition` để hiện dialog trình duyệt.

## Đã làm

- [x] Android: `ACCESS_FINE_LOCATION` / `ACCESS_COARSE_LOCATION` trong AndroidManifest
- [x] iOS: `NSLocationWhenInUseUsageDescription` trong Info.plist
- [x] Web: `index.html` + README note HTTPS/localhost
- [x] `location_permission.dart` — request + Position; VN messages; Web fallback khi Permissions API thiếu
- [x] `OrderAddressPage` + wire `/order/address` («Dùng vị trí hiện tại»)
- [x] Mark `[DONE] T3.1.1` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/location_permission.dart` | added | Helper xin quyền + get Position (+ Web fallback) |
| `apps/mobile/lib/features/order/order_address_page.dart` | added | UI «Dùng vị trí hiện tại» (sau mở rộng T3.1.3) |
| `apps/mobile/lib/main.dart` | modified | Wire `/order/address` |
| `apps/mobile/pubspec.yaml` | modified | `geolocator`; bỏ `permission_handler` |
| `apps/mobile/platform_config/` | added | Fragments + apply script |
| `apps/mobile/android/app/src/main/AndroidManifest.xml` | added | Location permissions |
| `apps/mobile/android/app/src/main/kotlin/.../MainActivity.kt` | added | FlutterActivity |
| `apps/mobile/android/*.gradle` / `gradle.properties` | added | Stub Gradle (org `vn.gastamde`) |
| `apps/mobile/ios/Runner/Info.plist` | added | NSLocationWhenInUseUsageDescription |
| `apps/mobile/ios/Runner/AppDelegate.swift` | added | App entry |
| `apps/mobile/ios/Podfile` | added | CocoaPods |
| `apps/mobile/web/index.html` | added | Geolocator web note |
| `apps/mobile/web/manifest.json` | added | PWA manifest |
| `apps/mobile/README.md` | modified | Location + verify T3.1.1 |
| `docs/prd.md` | modified | `[DONE] T3.1.1` |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_location_permission_02082026/` | added | this folder |

## Cách verify

1. Cài Flutter 3.x → `cd apps/mobile` → `flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=web,android,ios` (bổ sung scaffold thiếu) → `flutter pub get`.
2. `flutter run -d chrome` → vào `/order/address` → **Dùng vị trí hiện tại** → Allow → thấy lat/lng; Deny → message VN.
3. Android Emulator / iOS Simulator: cùng flow; tắt Location Services → message serviceDisabled; deny forever → message tương ứng.

## Ghi chú / blocker

- Môi trường agent **không có Flutter trên PATH** — chưa chạy được `flutter pub get` / `flutter run` tại đây. Code + platform permission đã sẵn; verify runtime trên máy có Flutter.
- Next unfinished PRD (sau US-3.1 + T3.2.1–2): **T3.2.3** UI thông báo ngoài phạm vi.
