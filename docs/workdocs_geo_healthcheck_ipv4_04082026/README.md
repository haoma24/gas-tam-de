# Fix geo-service Docker healthcheck (IPv4 bind + wget + EXPOSE)

- **Thư mục:** `docs/workdocs_geo_healthcheck_ipv4_04082026`
- **Ngày:** 04/08/2026
- **Loại:** fix
- **Liên quan:** Deploy VPS `ts-gas-tam-de-geo-service-1` unhealthy

## Mục tiêu

Deploy `docker compose` trên VPS fail vì container `geo-service` bị đánh dấu
`unhealthy`, kéo theo `api-gateway` / `web` không lên được (`depends_on:
service_healthy`).

## Phạm vi

- Trong scope: listen address IPv4-safe, healthcheck tooling trong image,
  `EXPOSE` đúng cổng theo service, fallback `PORT`, docs/changelog
- Ngoài scope: đổi endpoint `/healthz`, schema/DB geo, geocode upstream

## Quyết định chính

- Nguyên nhân khả dĩ trên VPS: `http.ListenAndServe(":8083")` bind IPv6-only
  khi `net.ipv6.bindv6only=1` → probe `wget http://127.0.0.1:8083/healthz`
  bị connection refused dù process vẫn chạy.
- Chuẩn hóa mọi `*:port` → `0.0.0.0:port` trong `httpx.ListenAndServe`, và
  dùng network `tcp4` khi bind `0.0.0.0` (chắc chắn probe `127.0.0.1` OK).
- Cài `wget` tường minh trong `Dockerfile.service` (giống `Dockerfile.web`).
- `EXPOSE` theo `EXPOSE_PORT` build-arg (geo = 8083, không còn hardcode 8080).
- `config.ListenAddr` hỗ trợ fallback env `PORT` (PaaS / diagnostic note).

## Đã làm

- [x] `httpx.NormalizeListenAddr` + test
- [x] `config.ListenAddr` (primary → PORT → fallback) + test; dùng ở 8 service
- [x] `Dockerfile.service`: `apk add wget`, `ARG EXPOSE_PORT`
- [x] `docker-compose.yml`: truyền `EXPOSE_PORT` cho từng Go service
- [x] Verify rebuild geo → listen `0.0.0.0:8083`, healthcheck healthy

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `pkg/httpx/httpx.go` | modified | normalize listen addr |
| `pkg/httpx/listen_addr_test.go` | added | unit test |
| `pkg/config/config.go` | modified | `ListenAddr` |
| `pkg/config/listen_addr_test.go` | added | unit test |
| `services/*/main.go` | modified | dùng `config.ListenAddr` |
| `deploy/Dockerfile.service` | modified | wget + EXPOSE_PORT |
| `deploy/docker-compose.yml` | modified | EXPOSE_PORT per service |
| `CHANGESLOG.md` | modified | entry |
| `docs/workdocs_geo_healthcheck_ipv4_04082026/` | added | workdoc này |

## Cách verify

1. `go test ./pkg/httpx/ ./pkg/config/`
2. `COMPOSE_PROJECT_NAME=ts-gas-tam-de docker compose -f deploy/docker-compose.yml up -d --build geo-service`
3. Log có `listening ... addr=0.0.0.0:8083`
4. `docker inspect … --format '{{.State.Health.Status}}'` → `healthy`
5. `docker exec … wget -qO- http://127.0.0.1:8083/healthz` → JSON ok

## Ghi chú / blocker

- Trên host dev (`bindv6only=0`) dual-stack vẫn healthy trước fix — bug chỉ lộ
  khi kernel bind IPv6-only hoặc probe IPv4 loopback.
- Geo **không** dùng NATS; `depends_on: nats` giữ nguyên (không phải root cause).
