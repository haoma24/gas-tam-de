---
name: change-workdocs
description: >-
  Records every change in CHANGESLOG.md and creates docs/workdocs_<mo-ta>_<ngaythangnam>
  folders for feature implementation history on Gas Tam Đệ. Use when implementing
  a feature, finishing a change, writing docs that alter the system, scaffolding
  code, fixing bugs, or when the user mentions changelog, CHANGESLOG, workdocs,
  lịch sử thay đổi, or ghi nhận change.
---

# Gas Tam Đệ — CHANGESLOG & workdocs

Bắt buộc tuân thủ skill này cho mọi thay đổi có ý nghĩa trong repo **gas-tam-de**.

## Quy tắc vàng

1. **Mỗi change đều sẽ được ghi vào CHANGESLOG.md.**
2. **Mỗi khi implement 1 chức năng sẽ tạo một thư mục `docs/workdocs_mô_tả_ngày_tháng_năm` để lưu lại lịch sử.**
3. Không kết thúc task implement nếu thiếu một trong hai mục trên (trừ khi user nói rõ “bỏ qua changelog”).

## Khi nào áp dụng

| Tình huống | CHANGESLOG.md | Thư mục workdocs |
|------------|---------------|------------------|
| Implement chức năng / epic / story | Có | **Có — bắt buộc** |
| Bugfix, refactor, chỉnh docs kiến trúc/PRD, config deploy | Có | Nên có nếu thay đổi lớn; optional nếu sửa nhỏ 1–2 dòng |
| Chỉ trả lời / không sửa file | Không | Không |

## Đặt tên thư mục workdocs

Format:

```text
docs/workdocs_<mo-ta>_<ddmmyyyy>
```

- `<mo-ta>`: tiếng Việt không dấu, chữ thường, nối bằng `_` hoặc `-`, ngắn (≤ 40 ký tự).
- `<ddmmyyyy>`: ngày **bắt đầu** (hoặc ngày hoàn thành nếu làm trong một ngày) theo lịch local user.
- Đặt tại **`docs/`**: `docs/workdocs_.../` (không đặt ở root repo).

Ví dụ:

- `docs/workdocs_scaffold_monorepo_02082026`
- `docs/workdocs_otp_auth_15082026`
- `docs/workdocs_dat_giao_gas_geo_01092026`

Nếu thư mục cùng mô tả+ngày đã tồn tại: **tái sử dụng** thư mục đó, bổ sung file bên trong; không tạo bản trùng.

## Workflow khi implement chức năng

Copy checklist và đánh dấu khi làm:

```text
Change Progress:
- [ ] 1. Tạo / mở thư mục docs/workdocs_<mo-ta>_<ddmmyyyy>
- [ ] 2. Viết hoặc cập nhật workdocs (mục tiêu, phạm vi, quyết định, file đụng tới)
- [ ] 3. Implement code/docs theo yêu cầu
- [ ] 4. Cập nhật workdocs phần "Đã làm" + ghi chú lệch so với plan (nếu có)
- [ ] 5. Thêm entry mới nhất vào đầu CHANGESLOG.md
- [ ] 6. Link CHANGESLOG entry → đường dẫn workdocs (nếu có)
```

### Bước 1 — Tạo workdocs

Tạo thư mục và các file tối thiểu:

```text
docs/workdocs_<mo-ta>_<ddmmyyyy>/
  README.md          # bắt buộc
  decisions.md       # optional — quyết định kỹ thuật
  notes.md           # optional — nhật ký / blocker
```

`README.md` dùng template trong [templates/workdoc-readme.md](templates/workdoc-readme.md).

### Bước 2 — Implement

Làm việc bình thường. Trong workdocs ghi:

- File/thư mục đã tạo hoặc sửa
- API / event / schema liên quan (nếu có)
- Cách verify thủ công

### Bước 3 — Ghi CHANGESLOG.md

File tại **root repo**: `CHANGESLOG.md`.

- Nếu chưa có: tạo mới theo [templates/changelog.md](templates/changelog.md).
- **Thêm entry mới ở trên cùng** (ngay dưới tiêu đề / hướng dẫn), không append cuối.
- Dùng format [templates/changelog-entry.md](templates/changelog-entry.md).

## Format CHANGESLOG entry

```markdown
## [YYYY-MM-DD] <tiêu đề ngắn>

- **Loại:** feature | fix | docs | refactor | chore | security
- **Phạm vi:** <module / service / app>
- **Tóm tắt:** <1–3 câu>
- **Chi tiết:**
  - ...
- **Workdocs:** `docs/workdocs_<mo-ta>_<ddmmyyyy>/` (hoặc `n/a` nếu thay đổi nhỏ)
- **Liên quan:** <PRD story / sprint / issue nếu có>
```

## Thay đổi nhỏ (không tạo workdocs)

Vẫn **bắt buộc** một entry CHANGESLOG ngắn. Ghi `Workdocs: n/a` và lý do một cụm từ (ví dụ: typo docs).

## Không làm

- Không ghi changelog mơ hồ kiểu “update files”.
- Không để workdocs trống chỉ có tên thư mục.
- Không commit secret vào workdocs / CHANGESLOG.
- Không sửa entry cũ để giấu lịch sử — chỉ amend cùng ngày nếu chưa push và đang cùng một phiên làm việc liên tục.

## Ví dụ nhanh

Implement OTP:

1. Tạo `docs/workdocs_otp_auth_02082026/README.md`
2. Code `services/auth-service/...` + Flutter màn OTP
3. Prefixed entry trong `CHANGESLOG.md` loại `feature`, link workdocs
