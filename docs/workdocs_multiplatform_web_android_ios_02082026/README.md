# Đa nền tảng Web + Android + iOS song song

- **Thư mục:** `docs/workdocs_multiplatform_web_android_ios_02082026`
- **Ngày:** 02/08/2026
- **Loại:** docs
- **Liên quan:** PRD §1.2, NFR-5, Sprint DoD; architecture §8.4–8.5, §9, §10

## Mục tiêu

Điều chỉnh kế hoạch: không ưu tiên Android-only / trì hoãn iOS. Phát triển **Web + Android + iOS** cùng lúc vì chưa có máy Android thật để test.

## Phạm vi

- Trong scope: cập nhật `docs/prd.md`, `docs/architecture.md`
- Ngoài scope: scaffold Flutter project, cấu hình CI macOS thật

## Quyết định chính

- Ba target Flutter từ Sprint 0
- Test chính: Web + Android Emulator; iOS qua Simulator (Mac) hoặc CI `macos-latest` (`flutter build ios --no-codesign`)
- Store publish vẫn out of scope MVP; build/phân phối nội bộ nằm trong scope
- DoD sprint yêu cầu chứng minh trên 3 target (iOS có thể qua CI)

## Đã làm

- [x] PRD: target platforms, MoSCoW, NFR-5, tasks T9.2.4–5, DoD, chiến lược test, rủi ro
- [x] Architecture: stack, §8.4–8.5, deploy/CI iOS, bảng quyết định
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `docs/prd.md` | modified | Bỏ Android-first; thêm multi-platform |
| `docs/architecture.md` | modified | Flutter 3 target + CI macOS |
| `CHANGESLOG.md` | modified | Entry mới |
| `docs/workdocs_multiplatform_web_android_ios_02082026/` | added | Lịch sử |

## Cách verify

1. Search docs không còn “ưu tiên Android” / “iOS sau” như hướng chính
2. PRD có §1.2 Target platforms và mục chiến lược test

## Ghi chú / blocker

- Build iOS local vẫn cần macOS; Windows chỉ cover Web + Android Emulator trực tiếp.
