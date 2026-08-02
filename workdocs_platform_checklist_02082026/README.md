# Platform checklist Web + Android + iOS (T9.2.5)

- **Thư mục:** `workdocs_platform_checklist_02082026`
- **Ngày:** 02/08/2026
- **Loại:** docs / chore
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.5; architecture §8.4–8.5, §9.5

## Mục tiêu

Đóng task PRD **T9.2.5**: checklist đa nền tảng — không dùng API chỉ có trên một OS nếu chưa có fallback; verify Web + Android Emulator; iOS Simulator hoặc CI macOS. Flutter có thể chưa có trên PATH máy Windows — ưu tiên docs + CI thay vì fail task.

## Phạm vi

- Trong scope:
  - Checklist + audit deps/code (`geolocator`, `url_launcher`, `flutter_map` / maps deep-link)
  - Workdocs + cập nhật `apps/mobile` README / PLATFORM_CHECKLIST
  - GitHub Actions nhẹ: analyze + test + web build; job macOS `flutter build ios --no-codesign` (không secret)
  - Mark `- [DONE] T9.2.5` trong PRD
- Ngoài scope:
  - Chạy thật Flutter local (SDK có thể thiếu trên PATH)
  - Signing / TestFlight / Play Store
  - `flutter build apk` trên CI (có thể bổ sung sau; APK cần SDK Android đầy đủ hơn web)

## Quyết định chính

- Canonical checklist: `apps/mobile/PLATFORM_CHECKLIST.md` (workdocs link tới đó).
- Fallback Maps: native schemes trước, HTTPS Google Maps luôn là candidate cuối (`navigation_link.dart`).
- Location: `geolocator` đa nền tảng; Web có nhánh Permissions API quirks.
- Bản đồ: `flutter_map` + OSM tiles — không Google Maps SDK (tránh single-OS).
- CI: Linux cover analyze/test/web; macOS cover compile iOS khi không có Mac local (khớp architecture §9.5).

## Đã làm

- [x] Audit `pubspec` + `lib/` (không `dart:io` / Google Maps SDK)
- [x] Viết `apps/mobile/PLATFORM_CHECKLIST.md`
- [x] Workflow `.github/workflows/flutter-ci.yml`
- [x] Cập nhật README mobile (thay mục T9.2.5 chưa làm)
- [x] Mark DONE T9.2.5 trong `docs/prd.md`
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/PLATFORM_CHECKLIST.md` | added | Checklist + audit + verify |
| `apps/mobile/README.md` | modified | Link checklist; bỏ “T9.2.5 chưa làm” |
| `.github/workflows/flutter-ci.yml` | added | analyze / test / web + iOS no-codesign |
| `docs/prd.md` | modified | `- [DONE] T9.2.5` |
| `CHANGESLOG.md` | modified | Entry T9.2.5 |
| `workdocs_platform_checklist_02082026/README.md` | added | Lịch sử |

## Cách verify

1. Đọc `apps/mobile/PLATFORM_CHECKLIST.md` — đủ 3 mục Web / Android / iOS-or-CI.
2. `rg "dart:io|google_maps_flutter" apps/mobile/lib` → không match.
3. Khi có Flutter: `cd apps/mobile && flutter analyze && flutter test && flutter build web`.
4. PR mở → Actions job `flutter-analyze-web` (+ `ios-build` trên macOS) chạy xanh.
5. `docs/prd.md`: mọi dòng `- [DONE] T*.*.*` ; không còn `- T9.2.5` chưa DONE.

## Ghi chú / blocker

- Máy agent Windows lúc implement: `flutter` không trên PATH → không chạy analyze/build local; dựa CI + checklist thủ công.
- Job `ios-build` cần scaffold iOS đủ (đã có từ T9.2.4); nếu `flutter create` chưa chạy trên runner trước đó, job có thể cần `flutter create . --platforms=ios` — workflow gọi `pub get` trên tree hiện có.
