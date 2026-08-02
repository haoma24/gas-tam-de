# Skill change-workdocs + CHANGESLOG

- **Thư mục:** `docs/workdocs_skill_change_workdocs_02082026`
- **Ngày:** 02/08/2026
- **Loại:** chore
- **Liên quan:** n/a

## Mục tiêu

Đảm bảo mỗi change được ghi vào `CHANGESLOG.md` và mỗi lần implement chức năng có thư mục `docs/workdocs_*` lưu lịch sử.

## Phạm vi

- Trong scope: project skill, templates, seed CHANGESLOG, workdocs mẫu
- Ngoài scope: implement code ứng dụng

## Quyết định chính

- Tên file changelog đúng theo yêu cầu: `CHANGESLOG.md` (root)
- Format thư mục: `docs/workdocs_<mo-ta>_<ddmmyyyy>`
- Skill đặt tại `.cursor/skills/change-workdocs` (project scope, auto-invoke qua description)

## Đã làm

- [x] Tạo `.cursor/skills/change-workdocs/SKILL.md`
- [x] Templates changelog + workdoc README
- [x] Seed `CHANGESLOG.md`
- [x] Workdocs cho đợt docs PRD/architecture trước đó

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `.cursor/skills/change-workdocs/SKILL.md` | added | Skill chính |
| `.cursor/skills/change-workdocs/templates/*` | added | Templates |
| `CHANGESLOG.md` | added | Nhật ký root |
| `docs/workdocs_docs_prd_architecture_02082026/` | added | Lịch sử docs |
| `docs/workdocs_skill_change_workdocs_02082026/` | added | Lịch sử skill |

## Cách verify

1. Mở skill trong Cursor và xác nhận description có trigger CHANGESLOG/workdocs
2. Xem `CHANGESLOG.md` có entry mới nhất ở trên
3. Có đủ hai thư mục workdocs ngày 02082026

## Ghi chú / blocker

- Không
