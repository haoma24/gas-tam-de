# OTP challenges SQLite (T1.1.5)

- **Thư mục:** `workdocs_otp_challenges_sqlite_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.5

## Mục tiêu

Chốt lưu `otp_challenges` trên SQLite auth: **hash OTP** (không plaintext), **expiry**, schema/index vững — task PRD T1.1.5. Phần lớn đã có từ T1.1.1/T1.1.2; task này siết gap và kiểm chứng.

## Phạm vi

- Trong scope:
  - Schema `otp_challenges` + index (phone lookup, expires)
  - Persist `code_hash` + `expires_at` khi request OTP
  - Tests: migrate/columns/indexes, hash≠plaintext, expiry window, verify `OTP_EXPIRED`
  - Đồng bộ comment schema / architecture index
- Ngoài scope:
  - T1.2.x admin login
  - Cleanup job / purge expired rows
  - Đổi API request/verify / SMS adapter / Flutter

## Quyết định chính

- Giữ hash peppered `SHA-256(pepper:challenge_id:code)` như T1.1.1 — không đổi thuật toán.
- Thêm `idx_otp_expires` hỗ trợ quét/cleanup theo `expires_at` sau này; lookup verify vẫn dùng `idx_otp_phone`.
- Không tách repository mới — `insertChallenge` + migrate embed `schema.sql` đủ cho T1.1.5.

## Đã làm

- [x] Audit schema + insert/verify path (đã đủ từ T1.1.1/1.2)
- [x] Siết comment schema; thêm `idx_otp_expires`
- [x] Tests persistence hash/expiry + migrate + expired verify
- [x] Sync `docs/architecture.md` index
- [x] Mark `[DONE] T1.1.5` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/auth-service/schema.sql` | modified | Comments T1.1.5 + `idx_otp_expires` |
| `services/auth-service/otp_challenges_test.go` | added | Tests schema/hash/expiry |
| `docs/architecture.md` | modified | Thêm `idx_otp_expires` |
| `docs/prd.md` | modified | `[DONE] T1.1.5` |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_otp_challenges_sqlite_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/auth-service/ -count=1`
2. Confirm PRD: `- [DONE] T1.1.5 Lưu otp_challenges...`

## Ghi chú / blocker

- Insert challenge vẫn xảy ra trước SMS send (hành vi T1.1.1/1.3); không đổi trong task này.
- Next unfinished: **T1.2.1** Seed admin account (password hash).
