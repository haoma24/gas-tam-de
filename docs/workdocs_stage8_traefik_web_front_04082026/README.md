# Fix Stage 8 Unreachable — Traefik target = web:8080

- **Thư mục:** `docs/workdocs_stage8_traefik_web_front_04082026`
- **Ngày:** 04/08/2026
- **Loại:** fix
- **Liên quan:** Cursor Cloud / Coolify Stage 8 — Health Check Unreachable labeledPort=8080

## Mục tiêu

Sau chuỗi fix bind/`ports`/network, Stage 8 vẫn fail:

```text
[08:33:43] FAILED: Health check failed: Unreachable — App KHÔNG lắng nghe
(không nối được) trên cổng Traefik (8080).
```

Traefik cần một process **chắc chắn** lắng nghe `0.0.0.0:8080` ngay khi
container start. Đưa **web (nginx)** làm public target thay vì api-gateway
(gateway còn mở SQLite trước khi sẵn sàng đầy đủ).

## Phạm vi

- Trong scope: nginx listen 8080, Traefik labels trên `web`, bỏ publish host
  trên compose chính, gateway listen-before-DB.
- Ngoài scope: đổi domain Coolify UI (vẫn Ports Exposes = **8080**).

## Quyết định chính

1. **Public Traefik service = `web`**, `loadbalancer.server.port=8080`,
   `traefik.docker.network=tensorship-net`.
2. nginx `listen 8080`; `/healthz` trả JSON từ nginx (không phụ thuộc gateway).
3. api-gateway bỏ Traefik labels (chỉ internal); vẫn bind sớm với `/healthz`
   trước khi SQLite xong (atomic handler swap).
4. Không `ports:` trên main compose (nats/gateway/web) — local map ở
   `docker-compose.local.yml`.

## Đã làm

- [x] `deploy/nginx.web.conf` listen 8080 + `/healthz` local + `/gateway-healthz`
- [x] `deploy/Dockerfile.web` `EXPOSE 8080`
- [x] compose: Traefik labels trên `web`; bỏ labels gateway; bỏ host ports
- [x] gateway `atomicHandler` listen trước DB
- [x] CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/nginx.web.conf` | modified | listen 8080, healthz local |
| `deploy/Dockerfile.web` | modified | EXPOSE 8080 |
| `deploy/docker-compose.yml` | modified | Traefik → web |
| `deploy/docker-compose.local.yml` | modified | `8090:8080` |
| `services/api-gateway/main.go` | modified | listen-before-DB |
| `README.md`, `.env*.example` | modified | docs VPS/Traefik |

## Cách verify

```bash
docker compose -f deploy/docker-compose.yml config
# web: labels port 8080 + tensorship-net; không ports publish
# api-gateway: không traefik labels

# nginx alone (no gateway):
docker run --rm -d --name web-hc -p 18080:8080 <web-image>
curl -sf http://127.0.0.1:18080/healthz   # {"status":"ok","service":"web"}
curl -sf http://127.0.0.1:18080/web-healthz
```

Coolify UI: **Ports Exposes = 8080** (không đổi). Redeploy sau khi CI push
`web:stag` + `api-gateway:stag`.

## Ghi chú / blocker

- Fix `ports`-publish trước đó (08:26) **không đủ** — Unreachable vẫn ở 08:33.
- Lý thuyết còn lại: Traefik đang probe đúng cổng 8080 nhưng process đích
  (gateway) chưa listen / IP lệch; chuyển sang nginx loại bỏ race DB.
