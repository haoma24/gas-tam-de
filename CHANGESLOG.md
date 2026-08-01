# CHANGESLOG — Gas Tam Đệ

Nhật ký thay đổi của repo. Entry mới nhất ở **trên cùng**.  
Quy trình: skill `.cursor/skills/change-workdocs`.

---

## [2026-08-02] Skill change-workdocs + quy trình CHANGESLOG/workdocs

- **Loại:** chore
- **Phạm vi:** `.cursor/skills/change-workdocs`, root docs process
- **Tóm tắt:** Thêm Agent Skill bắt buộc ghi mọi change vào `CHANGESLOG.md` và tạo thư mục `workdocs_<mo-ta>_<ddmmyyyy>` khi implement chức năng.
- **Chi tiết:**
  - Tạo skill `change-workdocs` kèm templates changelog/workdoc
  - Seed `CHANGESLOG.md` tại root
  - Ghi nhận lịch sử tài liệu PRD/architecture đã có
- **Workdocs:** `workdocs_skill_change_workdocs_02082026/`
- **Liên quan:** n/a

## [2026-08-02] Tài liệu khởi tạo PRD + Architecture Gas Tam Đệ

- **Loại:** docs
- **Phạm vi:** `docs/`
- **Tóm tắt:** Viết PRD (requirement, epic/story/task, sprint) và architecture (microservice, EDA, schema, security, deploy/monorepo).
- **Chi tiết:**
  - Thêm `docs/prd.md`
  - Thêm `docs/architecture.md` (gồm §9 Deploy & Repo strategy)
- **Workdocs:** `workdocs_docs_prd_architecture_02082026/`
- **Liên quan:** Sprint 0 / nền tảng tài liệu
