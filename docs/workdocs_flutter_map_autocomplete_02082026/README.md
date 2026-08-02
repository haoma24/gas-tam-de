# Flutter map/picker + autocomplete địa chỉ

- **Thư mục:** `docs/workdocs_flutter_map_autocomplete_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.1 / Sprint 2 / T3.1.3

## Mục tiêu

Khách trên Web / Android / iOS chọn địa chỉ giao bằng tìm kiếm có gợi ý (qua geo-service) và/hoặc ghim trên bản đồ; lưu `lat` / `lng` / `label` cho bước sau (bán kính, đặt đơn).

## Phạm vi

- Trong scope:
  - Flutter client `GET /v1/geo/search` (proxy geo-service — không gọi Photon/Nominatim/OSM search từ app)
  - Autocomplete debounce trên `/order/address`
  - Map/picker multi-platform (`flutter_map` + OSM raster tiles)
  - Giữ «Dùng vị trí hiện tại» (T3.1.1); state `orderAddressProvider`
  - PRD mark DONE T3.1.3
- Ngoài scope:
  - Kiểm tra bán kính giao (T3.2.x)
  - Place order (T3.3.x)
  - Gateway reverse-proxy thật (vẫn stub; local dùng `--dart-define=API_BASE_URL=…:8083`)

## Quyết định chính

- **Bản đồ:** dùng `flutter_map` + `latlong2` (đã có trong `pubspec.yaml`) với tile OSM — chạy được Web/Android/iOS không cần API key Google/Mapbox. Đủ cho pin picker; không nhúng SDK bản đồ nặng.
- **Autocomplete:** gọi geo-service proxy (`GeoApi`), debounce 400ms, `q` ≥ 2 ký tự.
- **Chọn vị trí:** (1) tap suggestion → move map + pin; (2) tap map → cập nhật lat/lng; (3) GPS → pin «Vị trí hiện tại».
- State in-memory `orderAddressProvider` để US-3.2 / US-3.3 đọc sau.

## Đã làm

- [x] `geo_models.dart` / `geo_api.dart` / `order_address_selection.dart`
- [x] Nâng `OrderAddressPage`: search + suggestions + `FlutterMap` pin
- [x] README verify geo search + map
- [x] Mark `[DONE] T3.1.3` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/geo_models.dart` | added | `GeoPlace`, `SelectedAddress` |
| `apps/mobile/lib/features/order/geo_api.dart` | added | `GET /v1/geo/search` client |
| `apps/mobile/lib/features/order/order_address_selection.dart` | added | Riverpod selection |
| `apps/mobile/lib/features/order/order_address_page.dart` | modified | autocomplete + map |
| `apps/mobile/README.md` | modified | verify T3.1.3 |
| `docs/prd.md` | modified | `[DONE] T3.1.3` |
| `CHANGESLOG.md` | modified | entry mới |
| `docs/workdocs_flutter_map_autocomplete_02082026/` | added | this folder |

## Cách verify

1. Chạy geo-service: `go run ./services/geo-service` (`:8083`).
2. `cd apps/mobile` → `flutter pub get` →  
   `flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8083`
3. Vào `/order/address` → gõ ≥2 ký tự (vd. `ben thanh`) → chọn gợi ý → pin + label.
4. Chạm bản đồ → cập nhật lat/lng; **Dùng vị trí hiện tại** → pin GPS.
5. Android emulator: `API_BASE_URL=http://10.0.2.2:8083`.

## Ghi chú / blocker

- Agent **không có Flutter trên PATH** — chưa `flutter pub get` / run tại đây.
- Tile OSM cần mạng; User-Agent package `vn.gastamde.gas_tam_de`.
- Next candidate: **T3.2.1** Store settings: lat/lng, `max_radius_km`.
