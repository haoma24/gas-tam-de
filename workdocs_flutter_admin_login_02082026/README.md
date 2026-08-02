# Flutter admin login screen (T1.2.3)

- **Thư mục:** `workdocs_flutter_admin_login_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.3 / architecture §8.2

## Mục tiêu

CCH đăng nhập bằng username/password trên Flutter (Web + Android + iOS) → nhận JWT `role=admin` từ auth-service (T1.2.2), session in-memory sẵn cho màn admin sau này.

## Phạm vi

- Trong scope:
  - Màn `AdminLoginPage` (username + password)
  - `AuthApi.adminLogin` → `POST /v1/auth/admin/login`
  - Session Riverpod dùng chung với OTP (`AuthTokenResult`)
  - Wire CTA Home **Dành cho cửa hàng** → `/admin/login` → `/admin` placeholder
- Ngoài scope:
  - T1.2.4 Middleware RBAC trên gateway
  - Persist token / secure storage
  - Gọi `POST /v1/auth/refresh` từ UI (API đã có; client chưa auto-refresh)
  - Order Desk / catalog admin screens

## Quyết định chính

- Tái dùng Dio + `authSessionProvider` từ T1.1.4; không thêm package.
- Gộp model token OTP/admin thành `AuthTokenResult` (typedef alias giữ tên cũ).
- Route `/admin/*` theo architecture §8 (cùng app, không subdomain).
- Local test: `API_BASE_URL` → auth-service `:8081` (gateway chưa proxy).

## Đã làm

- [x] `AdminLoginPage` (UI tiếng Việt, Material 3)
- [x] `AuthApi.adminLogin` + `AuthUser.username` / `displayName` + `INVALID_CREDENTIALS`
- [x] Routes `/admin/login`, `/admin`; Home CTA
- [x] README mobile ghi chú seed admin
- [x] Mark T1.2.3 DONE trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/auth/admin_login_page.dart` | added | UI login admin |
| `apps/mobile/lib/features/auth/auth_api.dart` | modified | `adminLogin` |
| `apps/mobile/lib/features/auth/auth_models.dart` | modified | `AuthTokenResult`, admin fields |
| `apps/mobile/lib/features/auth/auth_session.dart` | modified | `fromAdminLogin` / `fromTokens` |
| `apps/mobile/lib/features/home/home_page.dart` | modified | CTA `onAdminLogin` |
| `apps/mobile/lib/main.dart` | modified | Routes admin |
| `apps/mobile/README.md` | modified | Hướng dẫn admin local |
| `docs/prd.md` | modified | T1.2.3 DONE |
| `CHANGESLOG.md` | modified | Entry |
| `workdocs_flutter_admin_login_02082026/` | added | this folder |

## Cách verify

1. Chạy auth-service: `go run ./services/auth-service` (seed mặc định `admin` / `admin-change-me`).
2. Flutter (có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8081
```

3. Home → **Dành cho cửa hàng** → nhập `admin` / `admin-change-me` → **Đăng nhập** → thấy admin placeholder + tên hiển thị.
4. Sai mật khẩu → message «Tên đăng nhập hoặc mật khẩu không đúng.»

## Ghi chú / blocker

- Máy này có thể chưa có Flutter trên PATH — UI theo style OTP hiện có.
- Flutter Web → API local cần CORS; Android/iOS emulator không bị CORS.
- Next unfinished: **T1.2.4** Middleware RBAC trên gateway.
