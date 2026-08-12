# Validate số điện thoại theo đầu số di động Việt Nam

- **Thư mục:** `docs/workdocs_validate_dau_so_di_dong_vn_12082026`
- **Ngày:** 12/08/2026
- **Loại:** fix
- **Liên quan:** rà soát luồng đăng nhập OTP 12/08/2026

## Mục tiêu

Luật cũ chỉ kiểm tra **hình dạng**: `^0\d{9}$` (local) và `^\+84\d{9}$` (E.164).
Đúng độ dài là qua, bất kể đầu số. Những số sau đều lọt và đi thẳng tới
Stringee — mỗi lần là **một tin SMS thật tốn tiền** rồi mới thất bại:

| Input lọt luật cũ | Vấn đề |
|-------------------|--------|
| `0123456789` | đầu số `012` đã bị xoá từ đợt chuyển đổi 2018 |
| `0000000000` | số giả |
| `0212345678` | `02x` là đầu số cố định |
| `+84012345678` | E.164 Việt Nam không bao giờ có số `0` sau `+84` |

Rate limit theo IP (`RATE_LIMIT_OTP_PER_IP_MINUTE`) không đỡ được việc gõ nhầm
hoặc rải số rác đúng dạng. Chặn ở khâu validate rẻ hơn nhiều.

## Phạm vi

- Trong scope: luật đầu số ở auth-service + client, thông báo lỗi, test
- Ngoài scope: rà dữ liệu `auth.db` hiện có (chủ repo quyết định bỏ qua),
  kiểm tra số có thật đang hoạt động hay không (chỉ nhà mạng biết)

## Quyết định chính

- **Đầu số theo đợt chuyển đổi 2018**, viết không kèm số 0 đứng đầu:

  ```
  3[2-9]    Viettel          032-039
  5[25689]  Vietnamobile 052/056/058, Reddi 055, Gmobile 059
  7[06-9]   MobiFone         070, 076-079
  8[1-9]    VinaPhone 081-085/088, Viettel 086, Itelecom 087, MobiFone 089
  9[0-9]    mọi nhà mạng     090-099
  ```

- **Giữ nguyên toàn bộ phần chuẩn hoá.** Vẫn nhận `0…`, `+84…`, `84…`, `0084…`,
  bỏ khoảng trắng/gạch/chấm. Chỉ siết bộ đầu số, không đụng cách người dùng gõ.
- **`080` bị loại** (dải dịch vụ đặc biệt, không phải di động công cộng) trong
  khi `081-089` được nhận.
- **`9[0-9]` để rộng**: `095` hiện gần như không phát hành nhưng dải `09x` vốn
  thuộc di động, chặn thêm chỉ tăng rủi ro từ chối khách thật.
- **Hai nơi phải khớp nhau.** `phone_utils.dart` là bản soi gương của
  `phone.go`; client lỏng hơn thì khách nhận lỗi từ API thay vì gợi ý tại chỗ,
  client chặt hơn thì khách thật không đăng nhập được. Comment ở cả hai file trỏ
  sang nhau.
- **Thông báo lỗi kèm ví dụ** («SĐT di động không hợp lệ (VD: 0901234567)») —
  người gõ nhầm số bàn cần biết vì sao bị từ chối.

## Đã làm

- [x] `vnMobilePrefix` trong `phone.go`, dùng cho cả `reVNLocal` và `reE164VN`
- [x] `_vnMobilePrefix` tương ứng trong `phone_utils.dart`
- [x] Sửa copy lỗi ở `auth_models.dart`, `phone_page.dart`, `customer_auth_flow_page.dart`
- [x] 5 test Go: mọi đầu số nhà mạng, các cách gõ, số không phải di động, input hỏng, số admin seed
- [x] 6 test Dart soi gương lại

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/phone.go` | modified | `vnMobilePrefix` thay `\d{9}` |
| `services/auth-service/phone_test.go` | added | 5 test |
| `apps/mobile/lib/features/auth/phone_utils.dart` | modified | bản soi gương |
| `apps/mobile/test/phone_utils_test.dart` | added | 6 test |
| `apps/mobile/lib/features/auth/auth_models.dart` | modified | copy lỗi `INVALID_PHONE` |
| `apps/mobile/lib/features/auth/phone_page.dart` | modified | copy validator |
| `apps/mobile/lib/features/auth/customer_auth_flow_page.dart` | modified | copy validator |

Validate chạy ở **cả 4 điểm vào** vốn đã gọi `normalizePhoneVN`:
`otp_request.go:51`, `otp_verify.go:41`, `admin_phones_api.go:69`,
`admin_phones.go:156` — không cần sửa thêm chỗ nào.

## Cách verify

```bash
go test ./services/auth-service/ -run TestNormalizePhoneVN -v
cd apps/mobile && flutter test test/phone_utils_test.dart
```

Thủ công: nhập `0123456789` ở màn đăng nhập → báo lỗi ngay tại ô, **không** gửi
SMS. Nhập `0901234567` → nhận OTP như cũ.

## Ghi chú / blocker

- **Rủi ro đã được chấp nhận:** nếu `auth.db` trên staging/production có user
  đăng ký bằng số sai đầu số, họ sẽ **không đăng nhập lại được** — validate chạy
  cả ở `otp_verify`. Chủ repo quyết định không rà dữ liệu trước khi triển khai.
  Nếu có khiếu nại, đối chiếu `phone_masked` (3 số đầu) trong `auth.db`:

  ```sql
  SELECT phone_masked, created_at FROM users
  WHERE substr(phone_masked,1,3) NOT IN
    ('032','033','034','035','036','037','038','039','086','096','097','098',
     '052','055','056','058','059','092','099',
     '070','076','077','078','079','089','090','093',
     '081','082','083','084','085','087','088','091','094');
  ```

  `phone_hash` là HMAC nên không dò ngược được số gốc.
- Đầu số do Bộ TT&TT cấp phát và có thể thêm dải mới. Khi đó sửa
  `vnMobilePrefix` ở cả hai file và thêm ca test — test đầu số nhà mạng sẽ là
  chỗ nhắc.
- Validate này chỉ khẳng định số **đúng dạng di động VN**, không khẳng định số
  **có thật / đang hoạt động**. Việc đó vẫn do OTP xác nhận.
