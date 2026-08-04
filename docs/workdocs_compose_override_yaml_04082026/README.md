# Fix Stage 5: docker-compose.override.yml YAML parse error

- **Thư mục:** `docs/workdocs_compose_override_yaml_04082026`
- **Ngày:** 04/08/2026
- **Loại:** fix
- **Liên quan:** Cursor Cloud deploy Stage 5 — Run Container

## Mục tiêu

Deploy VPS/Cursor Cloud fail ở Stage 5 với lỗi parse
`docker-compose.override.yml` (`did not find expected key` / mapping values).
Làm giá trị env an toàn với generator YAML không quote của platform, và thêm
check tự động để lỗi không tái diễn.

## Phạm vi

- Trong scope:
  - `GEOCODE_USER_AGENT` trong `deploy/.env.example` + default geo-service
  - Script + Make/CI guardrail chống `: ` trong giá trị env
  - Ghi chú deploy trong `.env.example`
- Ngoài scope:
  - Sửa generator YAML phía Cursor Cloud platform
  - Đổi schema compose / image names (đã hardcode GHCR ở commit trước)

## Quyết định chính

- Cursor Cloud đọc `deploy/.env.example` → ghi unquoted `KEY: VALUE` vào
  `docker-compose.override.yml`. Giá trị chứa `: ` (colon + space) bị YAML
  hiểu là nested mapping.
- Thay `contact: local` → `contact=local` (tránh mọi dạng colon-space).
- Thêm `scripts/check-env-yaml-safe.sh` (Make + Backend CI) để fail sớm.

## Đã làm

- [x] Harden `GEOCODE_USER_AGENT` / `defaultUserAgent` → `contact=local`
- [x] Ghi chú Stage 5 / YAML constraint trong `deploy/.env.example`
- [x] Script `scripts/check-env-yaml-safe.sh` + `make check-env-yaml`
- [x] CI step trong `backend-ci.yml` test job
- [x] Mirror target trong `scripts/dev.ps1`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/.env.example` | modified | note Stage 5 + `contact=local` |
| `services/geo-service/geocode.go` | modified | `defaultUserAgent` |
| `scripts/check-env-yaml-safe.sh` | added | guardrail |
| `Makefile` | modified | `check-env-yaml` |
| `scripts/dev.ps1` | modified | `check-env-yaml` |
| `.github/workflows/backend-ci.yml` | modified | CI check + path triggers |

## Cách verify

1. `make check-env-yaml` → `OK: ... is safe...`
2. Tạm sửa `.env.example` thành `contact: local` rồi chạy lại → script FAIL
3. `docker compose -f deploy/docker-compose.yml config` → exit 0
4. Redeploy Cursor Cloud Stage 5 — không còn lỗi parse override

## Ghi chú / blocker

- Nếu project settings trên Cursor Cloud vẫn inject env cũ có `: `
  (ví dụ `GEOCODE_USER_AGENT=... contact: local` hoặc `IMAGE_PREFIX=...`),
  hãy xóa/sửa biến đó ở UI project — override generator ưu tiên giá trị đó.
- Image names đã hardcode `ghcr.io/haoma24/gas-tam-de/<svc>:stag` trong
  `deploy/docker-compose.yml` (không cần `IMAGE_PREFIX`/`IMAGE_TAG`).
- Checklist sửa `.env` VPS thật: [vps-env.md](./vps-env.md) + template
  `deploy/.env.vps.example`.
