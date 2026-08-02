# Flutter deep-link Google Maps / geo intent (T5.2.2)

- **Thư mục:** `docs/workdocs_flutter_maps_deeplink_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.2 / Sprint 3 / T5.2.2

## Mục tiêu

CCH mở chỉ đường từ vị trí thiết bị tới lat/lng điểm giao đơn — qua deep-link Google Maps / `geo:` / Apple Maps, dùng chung Web + Android + iOS (`url_launcher`).

## Phạm vi

- Trong scope:
  - Helper `openNavigationTo(lat, lng)` + URI builders (testable)
  - Platform config: Android `<queries>`, iOS `LSApplicationQueriesSchemes`
  - Unit test URI builders
- Ngoài scope:
  - Nút «Dẫn đường» trên chi tiết đơn (**T5.2.3**)
  - Routing engine / OSM directions server-side
  - Lấy origin lat/lng trong app (Maps app tự dùng vị trí hiện tại khi omit `origin`)

## Quyết định chính

- HTTPS Google Maps Directions API URL (`destination` only, `travelmode=driving`) = Web primary + fallback mọi platform.
- Android ưu tiên: `google.navigation:q=…` → `geo:` → `comgooglemaps://` → HTTPS.
- iOS ưu tiên: `comgooglemaps://` → Apple `maps:` → HTTPS.
- Không wire UI button (PRD tách T5.2.3).

## Đã làm

- [x] `navigation_link.dart` — `openNavigationTo` + candidate URIs
- [x] Android Manifest `<queries>` + iOS `LSApplicationQueriesSchemes`
- [x] `platform_config` fragments + README
- [x] `test/navigation_link_test.dart`
- [x] Mark `[DONE] T5.2.2` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/navigation_link.dart` | added | helper |
| `apps/mobile/test/navigation_link_test.dart` | added | URI unit tests |
| `apps/mobile/android/.../AndroidManifest.xml` | modified | maps queries |
| `apps/mobile/ios/Runner/Info.plist` | modified | LSApplicationQueriesSchemes |
| `apps/mobile/platform_config/**` | added/modified | fragments + docs |
| `apps/mobile/README.md` | modified | verify T5.2.2 |
| `docs/prd.md` | modified | `[DONE] T5.2.2` |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_flutter_maps_deeplink_02082026/` | added | this folder |

## Cách verify

```powershell
cd apps/mobile
flutter test test/navigation_link_test.dart
```

Hoặc gọi `openNavigationTo(10.776889, 106.700897)` từ debug — Web mở Google Maps directions; mobile mở app Maps nếu có.

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — logic URI thuần + docs đủ để T5.2.3 gắn nút.
- Next candidate: **T5.2.3** Nút «Dẫn đường» trên chi tiết đơn.
