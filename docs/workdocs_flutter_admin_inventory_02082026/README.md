# Flutter admin: màn tồn kho (T7.1.4)

- **Thư mục:** `docs/workdocs_flutter_admin_inventory_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-7.1 / T7.1.4 / PRD M7 / architecture §4.4 `GET/POST /v1/admin/inventory`

## Mục tiêu

CCH (admin) xem tồn kho và tạo phiếu nhập (`IN`) / xuất (`OUT`) / điều chỉnh (`ADJUST`) trên Flutter, gắn từ desk `/admin`.

## Phạm vi

- Trong scope:
  - `InventoryApi` → `GET/POST /v1/admin/inventory`
  - Màn list: tồn, giá vốn, chip «Sắp hết»; menu phiếu kho; FAB nhập mới
  - Tile **Tồn kho** trên `/admin` → `/admin/inventory`
  - Style Material 3 khớp admin products / debts
- Ngoài scope:
  - T7.2.x profit / cost snapshot report
  - Gateway reverse-proxy inventory (local trỏ thẳng `:8085`)
  - Đồng bộ tên/SKU từ catalog events
  - Lịch sử movements UI (chỉ snackbar sau POST)

## Quyết định chính

- Feature folder `features/inventory/` (tách billing/catalog).
- Tái dùng `formatVnd` từ `catalog_models`.
- Dialog theo `movement_type`: IN cần `qty`+`unit_cost` (+ sku/name khi tạo mới); OUT chỉ `qty`; ADJUST dùng `delta` signed.
- Local: `API_BASE_URL=http://127.0.0.1:8085`.

## Đã làm

- [x] `inventory_models` / `inventory_api` + `InventoryApiException`
- [x] `AdminInventoryPage` (list, refresh, empty, movement dialogs)
- [x] Admin desk tile **Tồn kho** + route `/admin/inventory`
- [x] `ApiConfig` note `:8085` + README verify
- [x] Mark `[DONE] T7.1.4` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/inventory/inventory_models.dart` | added | StockItem / movement types |
| `apps/mobile/lib/features/inventory/inventory_api.dart` | added | listStock + postMovement |
| `apps/mobile/lib/features/inventory/admin_inventory_page.dart` | added | UI tồn kho |
| `apps/mobile/lib/main.dart` | modified | route + desk tile |
| `apps/mobile/lib/core/api_config.dart` | modified | inventory `:8085` |
| `apps/mobile/README.md` | modified | verify T7.1.4 |
| `docs/prd.md` | modified | `[DONE] T7.1.4` |
| `CHANGESLOG.md` | modified | entry mới |
| `docs/workdocs_flutter_admin_inventory_02082026/` | added | workdoc này |

## Cách verify

1. Inventory: `go run ./services/inventory-service` (port 8085).
2. Flutter (nếu có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8085
```

3. Mở `/admin/inventory` (hoặc desk → **Tồn kho**).
4. **Nhập mới** → product_id / SKU / tên / qty / giá nhập → thấy trong list.
5. Menu ⋮ → Xuất / Điều chỉnh → tồn cập nhật.

API nhanh:

```bash
curl -s http://127.0.0.1:8085/v1/admin/inventory
```

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — UI theo style admin debts / products.
- Gateway `/v1/admin/inventory` chưa proxy inventory → E2E qua `:8080` cần wire sau.
- Next unfinished PRD: **T7.2.1** Lưu cost tại thời điểm xuất/bán (snapshot).
