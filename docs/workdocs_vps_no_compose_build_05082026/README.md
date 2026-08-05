# VPS: tắt compose build — pull GHCR :stag

- **Thư mục:** `docs/workdocs_vps_no_compose_build_05082026`
- **Ngày:** 05/08/2026
- **Loại:** fix deploy

## Mục tiêu

Stage 5 trên VPS (`ts-tamde-stag`) fail:

```text
failed to export layer: CreateDiff: mount callback failed ...
rename .../ingest/.../data .../blobs/sha256/...: no such file or directory
```

khi chạy `docker compose build` với mọi biến env thành `--build-arg` (kể cả
Flutter **web**).

## Nguyên nhân

- VPS không đủ ổn định để build image lớn (Flutter web + 8 Go service).
- Lỗi containerd thường do export layer / cache / disk, không phải logic app.
- Thiết kế đúng: CI push `ghcr.io/.../:stag`, VPS `pull` + `up --no-build`.

## Quyết định

- Gỡ toàn bộ `build:` khỏi `deploy/docker-compose.yml` (file VPS load).
- Chuyển `build:` sang `deploy/docker-compose.local.yml` (Make merge khi dev local).
- Script `scripts/vps-compose-up.sh` + `make vps-up`.

## Verify

```bash
docker compose -f deploy/docker-compose.yml build
# → No services to build

docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.local.yml config | rg 'build:' | wc -l
# → 9 (local vẫn build được)
```

## File đụng tới

| Path | Ghi chú |
|------|---------|
| `deploy/docker-compose.yml` | image-only VPS |
| `deploy/docker-compose.local.yml` | build + ports |
| `scripts/vps-compose-up.sh` | pull + up |
| `Makefile` | `vps-up` |
| `README.md` | troubleshooting CreateDiff |
