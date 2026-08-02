# Platform location permissions (T3.1.1)

Repo có sẵn permission trong `android/` `ios/` `web/` (khi đã generate).  
Thư mục này giữ **fragment + script** để áp lại sau `flutter create .` nếu Manifest/Info.plist bị scaffold mặc định ghi đè.

## Sau `flutter create .`

```powershell
cd apps/mobile
.\platform_config\apply_location_permissions.ps1
```

Hoặc merge thủ công theo các file bên dưới.

## Android — `android/app/src/main/AndroidManifest.xml`

Thêm **trong** thẻ `<manifest>` (cùng cấp `<application>`), **không** cần background location cho MVP (chỉ when-in-use):

```xml
<!-- Location — Gas Tam Đệ T3.1.1 (when-in-use) -->
<uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION" />
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
```

Xem `android/AndroidManifest.permissions.xml`.

## iOS — `ios/Runner/Info.plist`

Thêm keys (when-in-use only). Tránh Always trừ khi cần background:

```xml
<key>NSLocationWhenInUseUsageDescription</key>
<string>Gas Tam Đệ cần vị trí để xác định địa chỉ giao gas.</string>
```

Với `geolocator_apple`, nếu App Store hỏi Always: set preprocessor `BYPASS_PERMISSION_LOCATION_ALWAYS=1` trên pod `geolocator_apple` (xem README geolocator) — MVP không dùng Always.

Xem `ios/Info.plist.location.keys`.

## Web

Không cần meta đặc biệt: `geolocator_web` dùng Browser Geolocation API.  
Chạy trên **localhost** hoặc **HTTPS**. User sẽ thấy prompt trình duyệt khi bấm «Dùng vị trí hiện tại».

Nếu trình duyệt không hỗ trợ Permissions API, `checkPermission` có thể báo denied — app vẫn gọi `getCurrentPosition` (fallback trong `location_permission.dart`).

## Deep-link Maps (T5.2.2)

Helper: `lib/features/order/navigation_link.dart` → `openNavigationTo(lat, lng)`.

| Platform | Ưu tiên |
|----------|---------|
| Web | HTTPS Google Maps directions |
| Android | `google.navigation:` → `geo:` → `comgooglemaps://` → HTTPS |
| iOS | `comgooglemaps://` → Apple `maps:` → HTTPS |

### Android — thêm vào `<queries>` trong Manifest

Xem `android/AndroidManifest.maps_queries.xml` (package visibility API 30+).

### iOS — `LSApplicationQueriesSchemes`

Xem `ios/Info.plist.maps_queries_schemes`.

Nút «Dẫn đường» trên Order Desk = **T5.2.3** (chưa wire ở T5.2.2).
