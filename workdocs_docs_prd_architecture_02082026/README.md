# Tài liệu PRD + Architecture

- **Thư mục:** `workdocs_docs_prd_architecture_02082026`
- **Ngày:** 02/08/2026
- **Loại:** docs
- **Liên quan:** Nền tảng tài liệu / trước Sprint 0

## Mục tiêu

Chốt requirement, backlog, sprint và kiến trúc (Go microservices, Flutter, SQLite, NATS) cho Gas Tam Đệ trước khi code.

## Phạm vi

- Trong scope: `docs/prd.md`, `docs/architecture.md`
- Ngoài scope: scaffold code, CI/CD thật

## Quyết định chính

- Frontend: Flutter (Web + Android)
- Backend: Go microservices + NATS JetStream + SQLite per service
- Git: monorepo; Deploy: 1 VPS
- Fee rules nằm trong `order-service`; trừ tồn khi `order.completed`

## Đã làm

- [x] Phân tích requirement + PRD + Epic/Story/Task + Sprint S0–S5
- [x] Architecture: services, EDA, schema, security, Flutter overview
- [x] Bổ sung §9 Deploy & Repo strategy (monorepo, 1 VPS)

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `docs/prd.md` | added | PRD tổng hợp |
| `docs/architecture.md` | added | Kiến trúc + deploy |

## Cách verify

1. Đọc cross-link giữa `docs/prd.md` ↔ `docs/architecture.md`
2. Kiểm tra thương hiệu thống nhất: **Gas Tam Đệ**

## Ghi chú / blocker

- Workdocs này được tạo hồi tố khi thiết lập skill change-workdocs.
