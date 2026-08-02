# Flutter CTA shell Web + Android + iOS (T9.2.4)

- **Thư mục:** `workdocs_flutter_cta_shell_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature / docs
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.4 / architecture §8.4

## Mục tiêu

Một CTA shell Flutter chạy **cùng lúc** trên Web + Android + iOS: brand Gas Tam Đệ, hai lối vào khách vs admin, scaffold platform đủ để `flutter run` sau khi có SDK.

## Phạm vi

- Trong scope:
  - Audit / bổ sung scaffold `apps/mobile/{android,ios,web}`
  - Home CTA customer (**Đặt giao gas**) vs admin (**Dành cho cửa hàng**)
  - README + Makefile/scripts targets chạy 3 platform
  - Mark T9.2.4 DONE trên PRD
- Ngoài scope:
  - **T9.2.5** checklist platform đầy đủ (API một-OS, verify emulator/simulator/CI)
  - Commit git
  - Cài Flutter SDK trên máy agent (có thể không có trên PATH)

## Quyết định chính

- Giữ hand-authored Manifest / Info.plist / index.html (permissions + maps queries) trong repo; `flutter create .` chỉ bổ sung phần tooling (xcodeproj, gradlew) khi thiếu.
- Icons placeholder (amber brand) cho web + Android mipmap; iOS AppIcon placeholder Contents.json — thay branding sau nếu cần.
- DX: thêm `flutter-android` / `flutter-ios` / `flutter-create` song song `flutter-web`.

## Đã làm

- [x] Audit `android/` `ios/` `web/` — bổ sung styles/drawable/mipmap, Flutter xcconfig, storyboards, Assets, web icons/favicon
- [x] Hand-authored `ios/Runner.xcodeproj` + workspace/scheme + `GeneratedPluginRegistrant` stubs; Android `gradle/wrapper` + debug/profile manifests + `.metadata`
- [x] Home CTA documented as multi-platform shell (comment + README bảng customer/admin)
- [x] README: chạy Web + Android emulator + iOS Simulator từ cùng codebase
- [x] Makefile + `scripts/dev.ps1`: `flutter-create` / `flutter-android` / `flutter-ios`
- [x] Mark `- [DONE] T9.2.4` trên `docs/prd.md`
- [x] CHANGESLOG + workdocs
- [x] Ghi chú T9.2.5 còn lại (không verify full checklist)

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/android/app/src/main/res/**` | added | styles, launch_background, colors, mipmap icons |
| `apps/mobile/android/app/src/{debug,profile}/AndroidManifest.xml` | added | INTERNET for debug/profile |
| `apps/mobile/android/gradle/wrapper/gradle-wrapper.properties` | added | Gradle 8.3 distribution |
| `apps/mobile/android/.gitignore` | added | local.properties / keystore |
| `apps/mobile/ios/Flutter/*` | added | Debug/Release.xcconfig, AppFrameworkInfo.plist |
| `apps/mobile/ios/Runner.xcodeproj/**` | added | project.pbxproj + scheme/workspace |
| `apps/mobile/ios/Runner.xcworkspace/**` | added | workspace |
| `apps/mobile/ios/Runner/Base.lproj/*` | added | Main + LaunchScreen storyboards |
| `apps/mobile/ios/Runner/Assets.xcassets/**` | added | AppIcon / LaunchImage placeholders |
| `apps/mobile/ios/Runner/GeneratedPluginRegistrant.*` | added | stubs until `flutter pub get` |
| `apps/mobile/web/favicon.png`, `web/icons/*` | added | PWA icons (incl. maskable) |
| `apps/mobile/.metadata` | added | platforms android/ios/web |
| `apps/mobile/lib/features/home/home_page.dart` | modified | T9.2.4 doc comment |
| `apps/mobile/README.md` | modified | multi-platform run + CTA table |
| `apps/mobile/android/README.md`, `ios/README.md` | added/modified | per-platform bootstrap notes |
| `Makefile`, `scripts/dev.ps1` | modified | flutter-android/ios/create |
| `README.md`, `.gitignore` | modified | Flutter DX + ignore local/Pods |
| `docs/prd.md` | modified | T9.2.4 DONE |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_flutter_cta_shell_02082026/` | added | this folder |

## Cách verify

Cần Flutter 3.x trên PATH (máy agent có thể chưa có):

```powershell
cd apps/mobile
flutter pub get
# nếu tooling báo thiếu file: flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=web,android,ios
flutter run -d chrome
flutter run -d android   # emulator đang chạy
flutter run -d ios       # macOS + Simulator
```

Kỳ vọng: Home hiện **Gas Tam Đệ** + **Đặt giao gas** + **Dành cho cửa hàng**; tap vào được flow tương ứng.

Hoặc: `make flutter-web` / `make flutter-android` / `make flutter-ios`.

## Ghi chú / blocker

- Flutter **không** trên PATH ở session agent → không chạy `flutter run` tại đây.
- `gradlew` binary và `Pods/` vẫn đến từ máy local / `flutter pub get` + CocoaPods.
- **Next:** T9.2.5 Checklist platform (fallback API một-OS; verify Web + Android emulator; iOS Simulator hoặc CI macOS).
