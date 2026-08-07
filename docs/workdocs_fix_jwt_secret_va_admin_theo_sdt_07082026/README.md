# Sửa 401 toàn bộ route có xác thực + đăng nhập admin bằng số điện thoại

- **Thư mục:** `docs/workdocs_fix_jwt_secret_va_admin_theo_sdt_07082026`
- **Ngày:** 07/08/2026
- **Loại:** fix + feature
- **Liên quan:** báo lỗi staging 07/08/2026 (sửa tên trong hồ sơ lỗi "Invalid access token", màn Đơn hàng báo "Phiên đăng nhập hết hạn")

## Mục tiêu

1. Hồ sơ người dùng lưu được tên và màn Đơn hàng của tôi tải được dữ liệu.
2. Số `0909777020` đăng nhập bằng OTP là vào thẳng trang Admin, và admin tự thêm/bớt được số khác trong danh sách.

## Nguyên nhân gốc (phần 1)

`auth-service` **ký** access JWT, `api-gateway` **xác minh** nó. Trong
`deploy/docker-compose.yml`, chỉ `api-gateway` và `web` (gateway nhúng) nhận
`JWT_SECRET`; `auth-service` không nhận nên rơi về default biên dịch sẵn
`dev-jwt-secret-change-me`.

Trên deploy có `JWT_SECRET` thật trong `deploy/.env`, gateway từ chối **mọi**
token mà auth-service phát hành — kể cả token vừa lấy xong sau khi verify OTP:

| Màn hình | Lỗi thấy được | Nguồn |
|----------|---------------|-------|
| Hồ sơ (`PATCH /v1/me`) | `invalid or expired access token` | `RequireJWT` ở gateway → `AuthApiException` fallback in `message` |
| Đơn hàng (`GET /v1/orders/me`) | `Phiên đăng nhập hết hạn…` | cùng 401, map từ code `UNAUTHORIZED` |

Đây là lý do bản fix trước (refresh token phía client) không giúp được gì:
refresh chạy đúng, nhưng token mới cũng bị ký bằng secret sai.

Tái hiện cục bộ: chạy gateway với `JWT_SECRET=a-real-production-secret` còn
auth-service để trống → cả `PATCH /v1/me` lẫn `GET /v1/orders/me` trả 401 với
đúng hai thông báo trên, kể cả sau khi refresh.

## Phạm vi

- Trong scope: env compose của `auth-service`, log chẩn đoán, allow-list admin theo SĐT (API + UI), xử lý 401 chết phía client.
- Ngoài scope: đổi `PHONE_HASH_PEPPER` / `PHONE_ENC_KEY` (xem "Quyết định chính"), đổi TTL token.

## Quyết định chính

- **Không** truyền `PHONE_ENC_KEY` / `PHONE_HASH_PEPPER` vào `auth-service`. `phone_hash`
  là khoá định danh khách; đổi pepper trên `auth.db` đang chạy sẽ tạo user mới cho
  mọi số cũ và mất liên kết lịch sử đơn. Có test `TestPhoneSecretsNotWired` giữ
  chủ ý này, kèm comment trong compose.
- Log `jwt_secret_fp` (8 hex đầu của SHA-256) ở cả bên ký và bên xác minh: so log
  là biết ngay lệch secret mà không lộ secret.
- Test `deploy/compose_env_test.go` chạy trong `make test`, chặn tái diễn.
- Admin theo SĐT dùng bảng `admin_phones` khoá bằng **cùng peppered hash** với
  `users.phone_hash` — không lưu số thật, chỉ lưu thêm `phone_masked` để UI hiển thị.
- `refresh` đọc lại allow-list mỗi lần xoay token thay vì tin `sessions.role`, nên
  thêm/bớt số có hiệu lực ở lần refresh kế tiếp, không phải chờ đăng nhập lại.
- Không cho xoá entry cuối cùng: nếu không, không còn số nào đăng nhập admin bằng
  OTP được nữa (tài khoản username/password vẫn còn, nhưng dễ gây hoảng).
- Client: 401 vẫn còn sau khi refresh + retry ⇒ session vô dụng ⇒ xoá session để
  router đưa về màn đăng nhập, thay vì đứng lại trên màn báo lỗi tiếng Anh.

## Đã làm

- [x] Truyền `JWT_SECRET`, TTL token, giới hạn OTP và biến seed admin cho `auth-service`.
- [x] Log `jwt_secret_fp` ở `auth-service` và `api-gateway`.
- [x] Test guard trên `deploy/docker-compose.yml`.
- [x] Bảng `admin_phones` + seed từ `ADMIN_PHONES` (mặc định `0909777020`).
- [x] OTP verify cấp `role=admin` cho số trong allow-list.
- [x] `refresh` phân giải admin theo SĐT, nâng/hạ quyền theo allow-list.
- [x] API `GET/POST/DELETE /v1/admin/admin-phones` + route gateway.
- [x] Màn **Quản trị → Số điện thoại admin** + entry dashboard + route `/admin/admin-phones`.
- [x] Client xoá session khi 401 sống sót qua refresh; thống nhất copy tiếng Việt.

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/docker-compose.yml` | modified | `auth-service` nhận `JWT_SECRET` + TTL + seed admin + `ADMIN_PHONES` |
| `deploy/compose_env_test.go` | added | Guard: signer/verifier cùng `JWT_SECRET`; phone pepper vẫn để trống |
| `deploy/.env.example`, `deploy/.env.vps.example` | modified | Ghi chú secret dùng chung + `ADMIN_PHONES` |
| `pkg/config/config.go` | modified | `SecretFingerprint` |
| `services/api-gateway/main.go` | modified | Log fingerprint + route `/v1/admin/admin-phones` |
| `services/auth-service/main.go` | modified | Log fingerprint, seed allow-list, mount API |
| `services/auth-service/schema.sql` | modified | Bảng `admin_phones` |
| `services/auth-service/admin_phones.go` | added | Store + seed allow-list |
| `services/auth-service/admin_phones_api.go` | added | Handler `/v1/admin/admin-phones` |
| `services/auth-service/otp_verify.go` | modified | Cấp `role` theo allow-list |
| `services/auth-service/refresh.go` | modified | `resolveRefreshPrincipal` cho admin username lẫn admin SĐT |
| `services/auth-service/tokens.go` | modified | Hằng `roleCustomer` / `roleAdmin` |
| `apps/mobile/lib/core/api_client.dart` | modified | Xoá session khi 401 sống sót qua refresh |
| `apps/mobile/lib/features/auth/auth_models.dart` | modified | Mapper Dio dùng chung + copy `UNAUTHORIZED` / `LAST_ADMIN_PHONE` |
| `apps/mobile/lib/features/auth/admin_phones_api.dart` | added | Client allow-list |
| `apps/mobile/lib/features/auth/admin_admin_phones_page.dart` | added | Màn quản lý số admin |
| `apps/mobile/lib/features/dashboard/admin_dashboard_page.dart` | modified | Tile mới + chào admin bằng số điện thoại |
| `apps/mobile/lib/main.dart` | modified | Route `/admin/admin-phones` |
| `apps/mobile/test/admin_phones_test.dart` | added | Parse JSON + widget test list/add/xoá |
| `apps/mobile/test/auth_session_refresh_test.dart` | modified | Test 401 sống sót qua refresh |

## Cách verify

### Lệch secret (phần 1)

```bash
# Trước fix: gateway và auth-service khác secret → 401 mọi route có JWT
go test ./deploy/          # TestJWTSecretWiredConsistently
docker compose -p gas-tamde-stag logs auth-service api-gateway web | grep jwt_secret_fp
# ba dòng phải cùng một giá trị
```

Trên staging sau khi deploy: đăng nhập OTP → Hồ sơ → **Sửa** tên → **Lưu** phải
hiện "Đã lưu hồ sơ."; Hồ sơ → **Đơn hàng của tôi** phải tải được danh sách.

### Admin theo số điện thoại (phần 2)

1. `go test ./...` và `cd apps/mobile && flutter test`.
2. Chạy `make gateway` + `make auth`, rồi:

```bash
# 0909777020 nhận role=admin ngay từ OTP verify
curl -s -X POST :8080/v1/auth/otp/request -d '{"phone":"0909777020"}' -H 'Content-Type: application/json'
curl -s -X POST :8080/v1/auth/otp/verify  -d '{"phone":"0909777020","code":"<dev_code>"}' -H 'Content-Type: application/json'
# → "user":{"role":"admin", ...}

curl -s :8080/v1/admin/admin-phones -H "Authorization: Bearer <access_token>"
curl -s -X POST :8080/v1/admin/admin-phones -H "Authorization: Bearer <access_token>" \
     -H 'Content-Type: application/json' -d '{"phone":"0912345678","label":"Nhan vien"}'
```

3. Trên app: đăng nhập bằng `0909777020` → vào thẳng `/admin` → **Số điện thoại
   admin** → thêm một số → số đó đăng nhập OTP cũng vào trang quản trị.

## Ghi chú / blocker

- Deploy phải chạy lại `docker compose up -d` để `auth-service` nhận biến mới; chỉ pull image là chưa đủ.
- Nếu staging đang có `JWT_SECRET` khác default, các refresh token cũ vẫn dùng được (chúng nằm trong DB, không phải JWT) — khách không bị đăng xuất hàng loạt sau khi fix.
- `flutter analyze` giữ nguyên 26 info/warning có sẵn, không phát sinh mới.
