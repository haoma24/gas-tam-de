# Flutter UI thông báo ngoài phạm vi giao

- **Thư mục:** `docs/workdocs_flutter_out_of_range_ui_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.2 / Sprint 2 / T3.2.3

## Mục tiêu

Khi khách chọn địa chỉ giao, app gọi `POST /v1/geo/check`; nếu `in_range=false` hiện message tiếng Việt rõ ràng và chặn **Tiếp tục**; nếu trong phạm vi thì cho đi tiếp (chưa place order — T3.3).

## Phạm vi

- Trong scope:
  - Flutter `GeoApi.check` → `POST /v1/geo/check`
  - Gọi check mỗi lần chọn địa chỉ (search / map / GPS)
  - Banner ngoài phạm vi (kèm `distance_km` / `max_radius_km`) + disable continue
  - Banner trong phạm vi + enable **Tiếp tục**
  - Lưu `GeoCheckResult` vào `orderGeoCheckProvider` cho bước sau
- Ngoài scope:
  - Place order / review / success (T3.3.x)
  - Gateway reverse-proxy thật (local: `API_BASE_URL` → geo-service `:8083`)
  - Admin chỉnh bán kính UI

## Quyết định chính

- Check ngay khi chọn pin — không đợi bấm Tiếp tục — để khách thấy lỗi sớm.
- Response 200 + `in_range=false` → UI message (không coi là HTTP error).
- Nút continue: disabled khi đang check / chưa có kết quả / `in_range=false` / lỗi API.
- `onContinue` tạm SnackBar (T3.3 chưa có màn review).

## Đã làm

- [x] `GeoCheckResult` + `GeoApi.check`
- [x] `OrderAddressPage`: check + banners + block continue
- [x] `orderGeoCheckProvider`
- [x] README verify + mark `[DONE] T3.2.3` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/geo_models.dart` | modified | `GeoCheckResult` |
| `apps/mobile/lib/features/order/geo_api.dart` | modified | `check()` + error copy |
| `apps/mobile/lib/features/order/order_address_selection.dart` | modified | `orderGeoCheckProvider` |
| `apps/mobile/lib/features/order/order_address_page.dart` | modified | UI T3.2.3 |
| `apps/mobile/lib/main.dart` | modified | `onContinue` stub |
| `apps/mobile/README.md` | modified | verify T3.2.3 |
| `docs/prd.md` | modified | `[DONE] T3.2.3` |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_flutter_out_of_range_ui_02082026/` | added | this folder |

## Cách verify

1. `go run ./services/geo-service` (`:8083`, đã seed store).
2. `cd apps/mobile` → `flutter pub get` →  
   `flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8083`
3. `/order/address` → điểm gần cửa hàng → banner xanh → **Tiếp tục** bật.
4. Điểm xa (~12 km) → banner đỏ «Địa chỉ ngoài phạm vi…» → nút disabled «Ngoài phạm vi giao».

## Ghi chú / blocker

- Agent **không có Flutter trên PATH** — chưa `flutter analyze` / run tại đây.
- Next candidate: **T3.3.1** API `POST /orders` (validate JWT, items, geo, fee).
