# Flutter: màn nhập SĐT + OTP (T1.1.4)

- **Thư mục:** `workdocs_flutter_otp_ui_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.4 / architecture §8.2

## Mục tiêu

Khách nhập SĐT Việt Nam → nhận OTP → nhập mã 6 số → nhận JWT session trên Flutter (Web + Android + iOS), gọi API auth đã có (T1.1.1–T1.1.3).

## Phạm vi

- Trong scope:
  - Màn nhập SĐT + màn OTP
  - `AuthApi` Dio → `POST /v1/auth/otp/request` và `POST /v1/auth/otp/verify`
  - Session in-memory (Riverpod) sau verify
  - Resend cooldown; hiện `dev_code` khi server trả (local)
  - Wire CTA Home → auth → order placeholder
- Ngoài scope:
  - T1.1.5 (đã có challenge trên SQLite backend; task PRD còn lại là ghi nhận/đóng)
  - Admin login (US-1.2)
  - Persist token (secure storage)
  - Gateway reverse-proxy auth (vẫn stub 501)

## Quyết định chính

- Packages sẵn có: `dio`, `go_router`, `flutter_riverpod` — không thêm plugin platform-only.
- Default `API_BASE_URL` = gateway `:8080`; local OTP: `--dart-define=API_BASE_URL=http://127.0.0.1:8081` (auth-service) cho đến khi gateway proxy xong.
- Session chỉ giữ RAM (đủ cho demo OTP); refresh lưu để dùng khi có `/auth/refresh`.
- Validate SĐT client-side mirror rule VN của auth-service; normalize thật ở server.

## Đã làm

- [x] `PhonePage` + `OtpPage` (UI tiếng Việt, Material 3 theo theme app)
- [x] `AuthApi` + models + error mapping (RATE_LIMITED, OTP_INVALID, …)
- [x] `authSessionProvider` + Dio JWT interceptor
- [x] Routes `/auth/phone`, `/auth/otp`; CTA → OTP → `/order`
- [x] README mobile hướng dẫn `API_BASE_URL`
- [x] Mark T1.1.4 DONE trên PRD
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/auth/*` | added | API, models, session, phone/OTP UI |
| `apps/mobile/lib/core/api_client.dart` | added | Dio + JWT header |
| `apps/mobile/lib/core/api_config.dart` | modified | Ghi chú auth-service override |
| `apps/mobile/lib/main.dart` | modified | Routes auth |
| `apps/mobile/README.md` | modified | Hướng dẫn OTP local |
| `docs/prd.md` | modified | T1.1.4 DONE |
| `CHANGESLOG.md` | modified | Entry |
| `workdocs_flutter_otp_ui_02082026/` | added | this folder |

## Cách verify

1. Chạy auth-service: `go run ./services/auth-service` (port 8081, `OTP_DEV_REVEAL` default on).
2. Flutter (có SDK):

```powershell
cd apps/mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://127.0.0.1:8081
```

3. Home → **Đặt giao gas** → nhập `0901234567` → Gửi OTP → dùng `dev_code` hiện trên màn → Xác nhận → vào order placeholder.
4. Thử SĐT sai / OTP sai → thấy message tiếng Việt.

## Ghi chú / blocker

- Máy này có thể chưa có Flutter trên PATH — UI viết theo style hiện có; cần `flutter analyze` khi có SDK.
- Flutter Web → API local cần CORS trên auth/gateway (chưa có); Chrome có thể bị block; Android/iOS emulator không bị CORS.
- Gateway `/v1/auth/*` vẫn 501 — bắt buộc `API_BASE_URL` trỏ auth-service khi test OTP.
