# Chuyển workdocs vào docs/

- **Thư mục:** `docs/workdocs_move_workdocs_into_docs_02082026`
- **Ngày:** 02/08/2026
- **Loại:** chore / docs
- **Liên quan:** skill change-workdocs

## Mục tiêu

Gom toàn bộ thư mục `workdocs_*` từ root repo vào `docs/` để root gọn hơn và lịch sử implement nằm cùng chỗ với tài liệu dự án.

## Phạm vi

- Trong scope: di chuyển 65 thư mục `workdocs_*`, cập nhật link trong CHANGESLOG / skill / rule / templates / PRD / checklist
- Ngoài scope: đổi nội dung kỹ thuật bên trong từng workdoc cũ (chỉ cập nhật path)

## Quyết định chính

- Vị trí chuẩn mới: `docs/workdocs_<mo-ta>_<ddmmyyyy>/`
- Không đặt workdocs ở root nữa
- Dùng `git mv` để giữ lịch sử rename

## Đã làm

- [x] `git mv` toàn bộ `workdocs_*` → `docs/workdocs_*`
- [x] Cập nhật path trong `CHANGESLOG.md` và file markdown bên trong workdocs
- [x] Cập nhật skill, rule, templates, `docs/prd.md`, `apps/mobile/PLATFORM_CHECKLIST.md`
- [x] Workdoc + entry CHANGESLOG cho thay đổi này

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `docs/workdocs_*/` | renamed | từ root → `docs/` |
| `CHANGESLOG.md` | modified | prefix `docs/` cho Workdocs links |
| `.cursor/skills/change-workdocs/*` | modified | quy ước path mới |
| `.cursor/rules/change-workdocs.mdc` | modified | path `docs/workdocs_*` |
| `docs/prd.md` | modified | checklist path |
| `apps/mobile/PLATFORM_CHECKLIST.md` | modified | link workdocs |
| `docs/workdocs_move_workdocs_into_docs_02082026/` | added | workdoc này |

## Cách verify

1. Root không còn thư mục `workdocs_*`
2. `docs/` chứa ≥ 65 thư mục `workdocs_*` + workdoc mới
3. `CHANGESLOG.md` mọi dòng Workdocs bắt đầu bằng `` `docs/workdocs_ ``

## Ghi chú / blocker

- Không
