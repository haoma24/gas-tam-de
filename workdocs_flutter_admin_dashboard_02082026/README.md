# Flutter admin: dashboard widgets (T8.1.3)

- **Thư mục:** `workdocs_flutter_admin_dashboard_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-8.1 / T8.1.3 / PRD E8 / architecture §4 `GET /v1/admin/dashboard/summary`

## Mục tiêu

CCH (admin) xem doanh thu, lợi nhuận, phí giao, công nợ và số đơn ngay trên desk `/admin` sau khi login.

## Phạm vi

- Trong scope:
  - `DashboardApi` → `GET /v1/admin/dashboard/summary`
  - Widgets metric trên `/admin` (enhance admin home)
  - Filter kỳ: Hôm nay / 7 ngày / Tháng này (VN UTC+7)
  - Nav tiles desk giữ nguyên; tap Công nợ trên widget → `/admin/debts`
  - Style Material 3 khớp màn admin hiện có
- Ngoài scope:
  - E9 gateway hardening / reverse-proxy report (local trỏ thẳng `:8087`)
  - Tồn kho trong summary (vẫn dùng `/admin/inventory`)
  - Biểu đồ / drill-down theo ngày

## Quyết định chính

- Enhance `/admin` thay vì route riêng — CCH thấy dashboard ngay sau login.
- Feature folder `features/dashboard/` (API thuộc report-service).
- Period presets map sang `day=` hoặc `from`+`to`; mặc định hôm nay.
- Tái dùng `formatVnd` từ catalog_models.
- Local: `API_BASE_URL=http://127.0.0.1:8087`.

## Đã làm

- [x] `dashboard_models` / `dashboard_api` + `DashboardApiException`
- [x] `AdminDashboardPage` (metric tiles, period chips, refresh, nav)
- [x] Wire `/admin` trong `main.dart` (thay `_AdminHomePage`)
- [x] `ApiConfig` note `:8087` + README verify
- [x] Mark `[DONE] T8.1.3` trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/dashboard/dashboard_models.dart` | added | Summary + period helpers |
| `apps/mobile/lib/features/dashboard/dashboard_api.dart` | added | fetchSummary |
| `apps/mobile/lib/features/dashboard/admin_dashboard_page.dart` | added | UI desk + widgets |
| `apps/mobile/test/dashboard_models_test.dart` | added | fromJson + period query |
| `apps/mobile/lib/main.dart` | modified | `/admin` → AdminDashboardPage |
| `apps/mobile/lib/core/api_config.dart` | modified | report `:8087` |
| `apps/mobile/README.md` | modified | verify T8.1.3 |
| `docs/prd.md` | modified | `[DONE] T8.1.3` |
| `CHANGESLOG.md` | modified | entry mới |
| `workdocs_flutter_admin_dashboard_02082026/` | added | workdoc này |

## Cách verify

1. Report: `go run ./services/report-service` (port 8087).
2. Flutter (nếu có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8087
```

3. Admin login → `/admin`: widgets Doanh thu / Lợi nhuận / Phí giao / Công nợ / Đơn; chips kỳ; pull-to-refresh.
4. API nhanh:

```bash
curl -s "http://127.0.0.1:8087/v1/admin/dashboard/summary"
```

## Ghi chú / blocker

- Máy có thể chưa có Flutter trên PATH — UI theo style admin debts / inventory.
- Gateway `/v1/admin/dashboard/summary` chưa proxy report → E2E qua `:8080` cần wire sau (E9).
- Next unfinished PRD: **T9.1.1** Routing, CORS, JWT validation (E9).
