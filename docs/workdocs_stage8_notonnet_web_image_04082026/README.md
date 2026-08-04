# Fix Stage 8 `cause=NotOnNet` — nginx crash loop + thiếu image `web`

- **Thư mục:** `docs/workdocs_stage8_notonnet_web_image_04082026`
- **Ngày:** 04/08/2026
- **Loại:** fix
- **Liên quan:** Cursor Cloud / Tensorship deploy Stage 8 — Health Check

## Mục tiêu

Deploy fail với:

```text
[07:38:47] [HEALTHCHECK FAILED] cause=NotOnNet labeledPort=8080
[07:38:47] → Container chưa lên mạng tensorship-net — Traefik không với tới.
[07:36:52] Warning: failed to connect container 34998b37ad01 to tensorship-net
```

kèm log nginx không phân giải được hostname `api-gateway`. Mục tiêu: container
website không còn crash loop, image `web` thật sự tồn tại trên registry, và
Traefik chọn đúng IP khi container nằm trên 2 network.

## Phạm vi

- Trong scope: `deploy/nginx.web.conf`, `deploy/docker-compose.yml`,
  workflow build/push image `web`, script chẩn đoán network trên VPS.
- Ngoài scope: đổi cổng public (vẫn `8080` = api-gateway), sửa platform, đổi
  storage SQLite.

## Nguyên nhân

### 1. nginx chết ⇒ container ở trạng thái `restarting` ⇒ không attach được net

`proxy_pass http://api-gateway:8080` là **hostname literal**: nginx resolve
**một lần lúc parse config**. Nếu container `api-gateway` chưa chạy, nginx thoát
ngay:

```text
[emerg] host not found in upstream "api-gateway" in /etc/nginx/conf.d/default.conf:27
```

Đã reproduce đúng lỗi này bằng nginx 1.27-alpine (container `Exited (1)`).

`depends_on` chỉ có tác dụng ở lần `compose up`; sau khi host/docker daemon
restart, `restart: unless-stopped` bật container lại theo thứ tự bất kỳ → nginx
có thể lên trước gateway → crash loop. Container đang `restarting` thì
`docker network connect` **fail** (`container is restarting, wait until the
container is running`) — đúng warning `failed to connect container ... to
tensorship-net`, và Stage 8 báo `NotOnNet`.

### 2. Image `ghcr.io/haoma24/gas-tam-de/web:stag` chưa từng được push

`backend-ci.yml` chỉ build 8 service Go. Kiểm tra registry:

| Image | Manifest `:stag` |
|-------|------------------|
| `gas-tam-de/api-gateway` | `200` (public) |
| `gas-tam-de/web` | `403` (không tồn tại / private) |

VPS deploy bằng `docker compose up -d --no-build` ⇒ không pull được `web` ⇒
service website không lên (và pull lỗi có thể chặn cả `up`).

### 3. Container nằm trên 2 network nhưng thiếu `traefik.docker.network`

Platform attach container vào `tensorship-net` **sau khi** compose đã tạo chúng
trên network default của project. Traefik thấy 2 IP; không có
`traefik.docker.network` thì có thể chọn IP không định tuyến được (chính là
`cause=Unreachable` của lần fix trước).

## Quyết định chính

1. nginx resolve **runtime** qua DNS nội bộ của Docker
   (`resolver 127.0.0.11`) + biến `$api_gateway` → không còn chết khi gateway
   chưa sẵn sàng; gateway down chỉ còn là `503` JSON cho `/v1/*`.
2. `web.depends_on.api-gateway` → `service_started` (không cần healthy nữa).
3. Thêm workflow riêng `web-image.yml` để build/push image website (Flutter
   build chậm, path filter riêng, có `workflow_dispatch` để rebuild tay).
4. Label `traefik.docker.network=${PROXY_NETWORK:-tensorship-net}` cho gateway.
5. `scripts/vps-net-check.sh` để chẩn đoán/khắc phục `NotOnNet` ngay trên VPS.

## Đã làm

- [x] `deploy/nginx.web.conf`: `resolver` + `proxy_pass $api_gateway$request_uri`
      + `@api_unavailable` trả 503 JSON
- [x] `deploy/docker-compose.yml`: label `traefik.docker.network`,
      `web` depends_on `service_started`, ghi chú NotOnNet
- [x] `.github/workflows/web-image.yml`: build & push `web:stag` / `:latest`
- [x] `scripts/vps-net-check.sh` + `make vps-net-check` / `make vps-net-fix`
- [x] README: mục xử lý `NotOnNet` và yêu cầu image `web`

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/nginx.web.conf` | modified | runtime DNS + 503 fallback |
| `deploy/docker-compose.yml` | modified | traefik network label, depends_on |
| `.github/workflows/web-image.yml` | added | push image website lên GHCR |
| `scripts/vps-net-check.sh` | added | báo cáo / attach lại proxy network |
| `Makefile` | modified | `vps-net-check`, `vps-net-fix` |
| `README.md` | modified | troubleshooting NotOnNet |

## Cách verify

Đã chạy thật với Docker trong Cloud VM:

1. Config cũ, không có gateway:
   `docker run ... nginx:1.27-alpine` → `Exited (1)`,
   `host not found in upstream "api-gateway"` (reproduce lỗi).
2. Config mới, không có gateway: container `Up`, `/` trả HTML,
   `/web-healthz` trả `{"status":"ok","service":"web"}`, `/v1/hello` trả `503`.
3. Bật container alias `api-gateway`: **không restart nginx**, `/v1/hello` →
   `{"path":"/v1/hello"}`, `/v1/orders?limit=2` giữ nguyên query, `/healthz` OK.
4. Xoá gateway: nginx vẫn `Up`, `/v1/*` quay lại `503`.
5. `docker compose -f deploy/docker-compose.yml config` → exit 0, thấy
   `traefik.docker.network: tensorship-net`.
6. `scripts/vps-net-check.sh` trên container giả: báo `no` → `--fix` attach →
   re-check `yes`, exit 0.

## Việc cần làm trên VPS

1. Merge/push để CI chạy **cả hai** workflow (`backend-ci`, `web image`) —
   `web:stag` phải tồn tại trước khi deploy `--no-build`.
2. Nếu package `web` là private: đặt public (GitHub → Packages) hoặc
   `docker login ghcr.io` trên VPS.
3. Redeploy. Nếu vẫn `NotOnNet`:
   ```bash
   ./scripts/vps-net-check.sh          # container nào thiếu / đang crash loop
   ./scripts/vps-net-check.sh --fix    # attach lại rồi kiểm tra Traefik
   docker logs <container>             # nếu cột STATE là restarting
   ```

## Ghi chú / blocker

- Public port vẫn là **8080 = api-gateway**; website `:8090` chưa được Traefik
  route. Muốn mở website qua domain thì phải trỏ labeled port sang service
  `web` (nginx đã proxy `/v1` same-origin nên chỉ cần 1 cổng) — thay đổi này
  chạm cấu hình platform nên để riêng.
- `PROXY_NETWORK` chỉ dùng cho label Traefik và script; không tạo custom
  network trong compose (vẫn giữ khuyến nghị của Coolify).
