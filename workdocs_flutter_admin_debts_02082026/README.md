# Flutter admin: màn Công nợ (T6.2.2)

- **Thư mục:** `workdocs_flutter_admin_debts_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-6.2 / T6.2.2 / PRD M6 / architecture §4.4 `GET /v1/admin/debts`

## Mục tiêu

CCH (admin) xem danh sách khách còn nợ và tổng công nợ trên Flutter, gắn từ desk `/admin`.

## Phạm vi

- Trong scope:
  - `BillingApi` → `GET /v1/admin/debts`
  - Màn list đơn giản: banner tổng + dòng khách (phone masked / balance)
  - Tile **Công nợ** trên `/admin` → `/admin/debts`
  - Style Material 3 khớp các màn admin hiện có
- Ngoài scope:
  - E7 inventory
  - Dashboard analytics E8
  - Gateway reverse-proxy billing (local trỏ thẳng `:8086`)
  - Chi tiết ledger / thu nợ từ màn này

## Quyết định chính

- Feature folder `features/billing/` (không nhét vào order) vì API thuộc billing-service.
- Tái dùng `formatVnd` từ catalog_models (cùng pattern Order Desk).
- Local: `API_BASE_URL=http://127.0.0.1:8086`; Bearer JWT qua Dio interceptor nếu đã login (billing không enforce JWT khi gọi thẳng).

## Đã làm

- [x] `billing_models` / `billing_api` + `BillingApiException`
- [x] `AdminDebtsPage` (total banner, list, refresh, empty)
- [x] Admin desk tile **Công nợ** + route `/admin/debts`
- [x] `ApiConfig` note `:8086` + README verify
- [x] Mark `[DONE] T6.2.2` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/billing/billing_models.dart` | added | DebtItem + DebtsList |
| `apps/mobile/lib/features/billing/billing_api.dart` | added | listDebts |
| `apps/mobile/lib/features/billing/admin_debts_page.dart` | added | UI Công nợ |
| `apps/mobile/lib/main.dart` | modified | route + desk tile |
| `apps/mobile/lib/core/api_config.dart` | modified | billing `:8086` |
| `apps/mobile/README.md` | modified | verify T6.2.2 |
| `docs/prd.md` | modified | `[DONE] T6.2.2` |
| `CHANGESLOG.md` | modified | entry mới |
| `workdocs_flutter_admin_debts_02082026/` | added | workdoc này |

## Cách verify

1. Billing: `go run ./services/billing-service` (port 8086), có debts `balance > 0`.
2. Flutter (nếu có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8086
```

3. Mở `/admin/debts` (hoặc Home → cửa hàng → desk → **Công nợ**).
4. Banner tổng khớp `total_balance`; list SĐT + số tiền; pull-to-refresh.

API nhanh:

```bash
curl -s http://127.0.0.1:8086/v1/admin/debts
```

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — UI theo style admin products / delivery fee.
- Gateway `/v1/admin/debts` chưa proxy billing → E2E qua `:8080` cần wire sau.
- Next unfinished PRD: **T7.1.1** Schema stock + movements + cost (E7 Inventory).
