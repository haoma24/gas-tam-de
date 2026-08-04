# Fix Stage 8 Health Check — gateway listen 0.0.0.0:8080 sớm

- **Thư mục:** `docs/workdocs_stage8_gateway_listen_04082026`
- **Ngày:** 04/08/2026
- **Loại:** fix
- **Liên quan:** Cursor Cloud deploy Stage 8 — Health Check

## Mục tiêu

Sau khi qua Stage 5, Stage 8 fail với phân tích AI: ứng dụng không listen
`:8080` / bind `127.0.0.1`. Đảm bảo api-gateway bind `0.0.0.0:8080` và
**start sớm** sau `docker compose up -d` (platform không dùng `--wait`).

## Phạm vi

- Trong scope: `pkg/httpx` normalize loopback → `0.0.0.0`, compose gateway
  `PORT`/`GATEWAY_ADDR`/`depends_on`, Dockerfile `ENV PORT`, `.env*.example`
- Ngoài scope: đổi port website (`8090`), healthcheck từng backend service

## Quyết định chính

1. Stage 5 = `up -d --no-build` (không `--wait`). Gateway trước đây
   `depends_on: service_healthy` cho 8 backend → có thể trễ vài phút trước khi
   process listen → Stage 8 probe `host:8080` bị connection refused.
2. Đổi depends_on backend → `service_started` (gateway `/healthz` là liveness,
   không cần upstream sẵn sàng).
3. Pin `PORT=8080` + `GATEWAY_ADDR=0.0.0.0:8080`; Dockerfile `ENV PORT`.
4. `NormalizeListenAddr` rewrite `127.0.0.1`/`localhost` → `0.0.0.0`.

## Đã làm

- [x] httpx normalize loopback + tests
- [x] compose api-gateway PORT/ADDR/depends_on/healthcheck timing
- [x] Dockerfile.service `ENV PORT=${EXPOSE_PORT}`
- [x] `.env.vps.example` / `.env.example` thêm `PORT=8080`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `pkg/httpx/httpx.go` | modified | rewrite loopback |
| `pkg/httpx/listen_addr_test.go` | modified | cases mới |
| `deploy/docker-compose.yml` | modified | gateway listen sớm |
| `deploy/Dockerfile.service` | modified | `ENV PORT` |
| `deploy/.env.vps.example` | modified | `PORT=8080` |
| `deploy/.env.example` | modified | `PORT` + `0.0.0.0` |

## Cách verify

1. `go test ./pkg/httpx/ ./pkg/config/`
2. `docker compose -f deploy/docker-compose.yml config` chứa
   `GATEWAY_ADDR: 0.0.0.0:8080` và `PORT: "8080"`
3. Local: `docker compose ... up -d --no-build` rồi ngay lập tức
   `curl -sf http://127.0.0.1:8080/healthz` (sau khi image gateway mới build)
4. Redeploy Cursor Cloud — Stage 8 Health Check phải xanh

## Ghi chú / blocker

- Cần CI rebuild/push image `api-gateway:stag` (và ideally mọi service vì
  Dockerfile `ENV PORT`) trước khi VPS `up --no-build` lấy image mới.
- Trên VPS Environment: thêm `PORT=8080`; **không** set
  `GATEWAY_ADDR=127.0.0.1:8080`.
- Follow-up Traefik Unreachable: xem [coolify-traefik.md](./coolify-traefik.md)
  (bỏ `networks.gastamde` + label port 8080).
- Follow-up Unreachable sau NotOnNet: xem [traefik-no-publish.md](./traefik-no-publish.md)
  (bỏ `ports:` publish gateway — Traefik chọn sai IP).
