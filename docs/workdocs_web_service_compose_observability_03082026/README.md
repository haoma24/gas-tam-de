# Website vào docker-compose + healthcheck/observability cho cả stack

- **Thư mục:** `docs/workdocs_web_service_compose_observability_03082026`
- **Ngày:** 03/08/2026
- **Loại:** fix
- **Liên quan:** Chẩn đoán AI Runtime — "log chỉ hiển thị NATS, không có log của dịch vụ website"

## Mục tiêu

Chẩn đoán runtime báo: log chỉ thấy NATS khởi động thành công, không có bất kỳ
thông tin nào về **dịch vụ website**. Mục tiêu là làm cho website thực sự tồn tại
trong stack Docker và làm mọi lỗi service hiện ra thay vì im lặng.

## Nguyên nhân gốc

Không phải website crash — **website chưa bao giờ là một service trong
`deploy/docker-compose.yml`**. File compose chỉ có `nats` + 8 service Go; Flutter
Web được chạy trên host bằng `flutter run -d chrome`. Vì vậy
`docker compose logs <web>` không thể có output: service đó không tồn tại.

Hai yếu tố khuếch đại triệu chứng:

1. `make nats` chỉ chạy `docker compose up nats -d`. Ai chạy target này rồi xem
   log sẽ chỉ thấy NATS — đúng như mô tả trong chẩn đoán.
2. Các service Go **không có `healthcheck` và không có `restart`**, còn
   `api-gateway` chỉ `depends_on: … condition: service_started`. Container chết
   sẽ nằm im ở trạng thái `Exited` mà không có tín hiệu nào.

Xác nhận trên VM: `docker compose ps` cho thấy `deploy-nats-1` được tạo **34 giờ
trước** trong khi tất cả service khác chưa từng chạy — đúng trạng thái "chỉ có
NATS đang chạy".

## Phạm vi

- Trong scope:
  - Thêm service `web` (Flutter Web release + nginx) vào compose
  - `healthcheck` + `restart: unless-stopped` cho mọi service
  - Giảm nhiễu log `/healthz` để log còn đọc được
  - Make/PowerShell target xem trạng thái + log; cập nhật README/AGENTS
- Ngoài scope:
  - Thay đổi code nghiệp vụ của Flutter app hoặc service Go
  - Cấu hình production (TLS, domain, CDN)

## Quyết định chính

- **nginx proxy `/v1` + `/healthz` sang `api-gateway:8080`** thay vì build web
  với `API_BASE_URL=http://127.0.0.1:8080`. Website và API cùng origin nên trình
  duyệt không cần CORS preflight, và `WEB_PORT` đổi được mà không phải build lại.
  Muốn trỏ ra gateway khác thì set `WEB_API_BASE_URL`.
- **Pin `FLUTTER_VERSION=3.44.0`.** Bản `3.35.4` fail ở `flutter pub get`:
  `google_fonts 8.2.1` cần Dart `^3.10.0` còn image đó chỉ có Dart 3.9.2.
- **Không publish cổng 8081–8087 ra host.** Giữ nguyên chủ đích "đi qua gateway",
  đồng thời tránh đụng cổng khi dev chạy song song `make gateway` trên host.
- **`depends_on: service_healthy`** thay cho `service_started`, để gateway không
  khởi động trước khi upstream thực sự trả lời được.
- **Bỏ log `/healthz` khi status 200** trong `pkg/httpx`. Healthcheck 10s × 8
  service sẽ tạo ~48 dòng/phút và nhấn chìm request thật. `/healthz` lỗi
  (non-200) vẫn được log.

## Đã làm

- [x] `deploy/Dockerfile.web` — multi-stage Flutter build → nginx
- [x] `deploy/nginx.web.conf` — SPA fallback, proxy API, gzip, `/web-healthz`
- [x] Service `web` trong compose (cổng `${WEB_PORT:-8090}`)
- [x] `healthcheck` + `restart: unless-stopped` cho 8 service Go
- [x] `api-gateway` chờ `service_healthy` của mọi upstream
- [x] `pkg/httpx` bỏ log `/healthz` thành công
- [x] Make + `scripts/dev.ps1`: `compose-ps`, `compose-logs`, `web-up`,
      `web-logs`, `web-health`, `stack-health`
- [x] `.dockerignore` giữ `data/` và build output ngoài build context
- [x] README + AGENTS.md

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/Dockerfile.web` | added | Flutter 3.44.0 → nginx 1.27-alpine |
| `deploy/nginx.web.conf` | added | SPA + reverse proxy `/v1`, `/healthz` |
| `deploy/docker-compose.yml` | modified | service `web`, healthcheck, restart, `service_healthy` |
| `.dockerignore` | added | loại `data/`, `.git`, build output |
| `pkg/httpx/httpx.go` | modified | không log `/healthz` khi 200 |
| `Makefile` | modified | target trạng thái + log + web |
| `scripts/dev.ps1` | modified | parity Windows |
| `README.md` | modified | mục "Chạy toàn bộ stack bằng Docker" |
| `AGENTS.md` | modified | bảng DX + ghi chú healthcheck |

## Cách verify

```bash
make compose-up      # build tất cả, chờ healthy
make compose-ps      # 10 container, tất cả (healthy)
```

Website + proxy API:

```bash
curl -s http://127.0.0.1:8090/ | grep -o "<title>[^<]*</title>"   # <title>Gas Tam Đệ</title>
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8090/main.dart.js   # 200
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8090/admin/login    # 200 (SPA fallback)
curl -sf http://127.0.0.1:8090/web-healthz      # {"status":"ok","service":"web"}
curl -sf http://127.0.0.1:8090/v1/hello         # qua nginx → gateway
curl -s -X POST http://127.0.0.1:8090/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"0912345678"}'   # POST proxy OK
```

Triệu chứng gốc đã hết — website giờ có log riêng:

```bash
docker compose -f deploy/docker-compose.yml logs web
# web-1 | 172.18.0.1 - - [...] "GET / HTTP/1.1" 200 754 "-" "curl/8.5.0"
```

Lỗi không còn im lặng:

```bash
docker compose -f deploy/docker-compose.yml exec catalog-service kill 1
docker compose -f deploy/docker-compose.yml ps catalog-service   # restart + (healthy)
```

Log sạch khi idle (không còn spam `/healthz`):

```bash
docker compose -f deploy/docker-compose.yml logs --since=30s | wc -l   # 0
```

## Ghi chú / blocker

- **VPS 404:** mở `http://<IP>:8080/` trả `404 page not found` là **đúng hành vi** của API
  Gateway (không phục vụ HTML). Website nằm ở **`http://<IP>:8090/`**. Muốn cổng 80:
  `cp deploy/.env.example deploy/.env` rồi `WEB_PORT=80` và `make compose-up`.
  Makefile / `dev.ps1` tự `--env-file deploy/.env` khi file tồn tại.
- Build `web` lần đầu mất khá lâu (kéo image Flutter ~vài GB + compile release).
  Các lần sau dùng layer cache; `pubspec.*` được COPY trước source nên sửa code
  Dart không phải chạy lại `flutter pub get`.
- `make flutter-web` (hot reload, trỏ `:8080`) vẫn là cách dev hằng ngày. Service
  `web` là bản release để smoke test/demo.
- `FLUTTER_VERSION` phải theo kịp `pubspec.lock`. Nếu sau này bump dependency cần
  Dart mới hơn, nhớ bump luôn arg này (và tag phải tồn tại trên
  `ghcr.io/cirruslabs/flutter`; `3.44.8` chưa có tag khi làm thay đổi này).
