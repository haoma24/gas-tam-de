# Gas Tam Đệ

Monorepo cửa hàng gia đình **Gas Tam Đệ**: Flutter (Web + Android + iOS) + Go microservices + SQLite + NATS JetStream.

Tài liệu:

- [docs/prd.md](docs/prd.md) — sản phẩm & backlog
- [docs/architecture.md](docs/architecture.md) — kiến trúc kỹ thuật

## Cấu trúc

```text
apps/mobile/          Flutter app (customer + admin theo role)
services/             Go services (api-gateway + bounded contexts)
pkg/                  Shared Go (config, httpx, sqlite, events)
deploy/               docker-compose + Dockerfile (services + web/nginx)
docs/                 PRD + Architecture
data/                 SQLite files (local; gitignored)
```

## Yêu cầu

- Go 1.25+
- Docker (NATS + optional full stack)
- Flutter 3.x (cho `apps/mobile`) — nếu chưa cài, xem `apps/mobile/README.md`

## Chạy nhanh (local)

Ưu tiên dùng shortcut DX (T9.2.3). Trên Windows (không cần GNU Make):

```powershell
.\scripts\dev.ps1 help
.\scripts\dev.ps1 nats       # NATS up + wait + bootstrap streams
.\scripts\dev.ps1 gateway    # API gateway :8080
.\scripts\dev.ps1 health     # /healthz + /v1/hello
.\scripts\dev.ps1 flutter-web
```

Trên Linux/macOS hoặc khi có GNU Make — cùng tên target:

```bash
make help
make nats
make gateway
make health
make flutter-web
```

## Chạy toàn bộ stack bằng Docker (kèm website)

`make nats` **chỉ** khởi động NATS — đó là lý do `docker compose logs` khi đó chỉ
hiện log NATS. Muốn cả website + API chạy trong Docker:

```bash
make compose-up      # build + start tất cả, chờ healthy, rồi in trạng thái
make stack-health    # trạng thái container + health gateway + health web
make compose-logs    # tail log của TẤT CẢ service
make web-logs        # chỉ log nginx của website
make compose-down
```

| Thành phần | URL | Ghi chú |
|------------|-----|---------|
| Website (Flutter Web + nginx) | <http://127.0.0.1:8090> | đổi cổng bằng `WEB_PORT` |
| API Gateway | <http://127.0.0.1:8080> | truy cập API trực tiếp — **`/` trả 404 là đúng** |
| NATS monitoring | <http://127.0.0.1:8222/jsz> | |

### Truy cập trên VPS

**Coolify / Cursor Cloud (Traefik):** cổng public `labeledPort=8080` trỏ vào
service **`web`** (nginx lắng nghe `0.0.0.0:8080`). Domain của app phục vụ HTML
+ `/v1/*` same-origin. Không publish host port trong compose chính — Traefik
nối qua `tensorship-net`.

**Local / `make compose-up`** (merge `docker-compose.local.yml`):

1. Website: **`http://127.0.0.1:8090/`** (`WEB_PORT`, map → container `:8080`).
2. API trực tiếp: **`http://127.0.0.1:8080/`** (gateway; `/` trả 404 là đúng).

Muốn mở website local bằng cổng 80:

```bash
# trong deploy/.env (copy từ deploy/.env.example)
cp deploy/.env.example deploy/.env
# sửa WEB_PORT=80 (và JWT_SECRET / mật khẩu admin trước khi public)
WEB_PORT=80
make compose-up
# → http://127.0.0.1/
```

`make` / `scripts/dev.ps1` tự nạp `deploy/.env` nếu file đó tồn tại.

Hoặc giữ web ở 8090 và để nginx/Caddy trên host reverse-proxy `80/443 → 127.0.0.1:8090`.

### Khi deploy báo `container ... is unhealthy`

```bash
make doctor    # chỉ in container KHÔNG healthy: probe cuối + 40 dòng log
```

Hoặc trực tiếp (thay `-p` bằng project name của bạn):

```bash
docker compose -p <project> logs billing-service --tail=50
```

Các service `catalog`, `order`, `inventory`, `billing`, `report` dùng **NATS
JetStream**, nhưng **không còn chờ NATS mới serve HTTP**: chúng lên `/healthz`
ngay rồi kết nối broker ở nền (retry vô hạn, log `WARN nats not ready; retrying
in background`). NATS down ⇒ container vẫn `healthy`, chỉ readiness đỏ:

| Endpoint | Ý nghĩa | Dùng cho |
|----------|---------|----------|
| `/healthz` | liveness — process đang serve, không phụ thuộc broker | `healthcheck` compose, `depends_on` |
| `/readyz` | readiness — 200 khi dependency OK, 503 + tên dependency lỗi | debug, LB gate traffic |

```bash
docker compose -p <project> exec catalog-service wget -qO- http://127.0.0.1:8082/readyz
# {"dependencies":{"nats":"ok"},"service":"catalog-service","status":"ready"}
```

`NATS_STARTUP_TIMEOUT_SEC` (mặc định 60s) giờ chỉ giới hạn **một vòng** thử kết
nối; hết budget thì vòng sau retry tiếp, service không thoát.

Nếu deploy chạy `docker compose up --no-build`, nhớ **build lại image** — image
cũ vẫn chặn ở NATS trước khi mount `/healthz`.

### Khi deploy báo `HEALTHCHECK FAILED cause=NotOnNet`

Platform (Cursor Cloud / Coolify) chạy compose xong mới `docker network connect`
container vào network của Traefik. Lệnh đó **fail nếu container đang
`restarting`** (crash loop) hoặc nếu container của lần deploy trước còn giữ tên
endpoint. Kết quả: `Warning: failed to connect container <id> to tensorship-net`
rồi `cause=NotOnNet`.

```bash
./scripts/vps-net-check.sh        # container nào chưa lên proxy network
./scripts/vps-net-check.sh --fix  # attach lại (make vps-net-fix)
# cột STATE = restarting ⇒ sửa crash loop trước: docker logs <container>
PROXY_NETWORK=<net> ./scripts/vps-net-check.sh   # proxy dùng network khác
```

Deploy bằng `docker compose up --no-build` cần **đủ image trên GHCR**: 8 service
Go do `backend-ci.yml` push, còn website do `web-image.yml` push
(`gas-tam-de/web:stag`). Thiếu image `web` thì bước pull hỏng và website không
bao giờ lên.

Website gọi API **same-origin**: nginx (container `:8080`) proxy `/v1/*` sang
`api-gateway:8080`; `/healthz` trả JSON từ nginx (liveness Traefik). Gateway
liveness riêng: `/gateway-healthz`. nginx resolve hostname `api-gateway`
**theo từng request** qua DNS nội bộ của Docker, nên gateway chưa lên chỉ làm
`/v1/*` trả `503` chứ không giết container website. Các service `auth`, `catalog`,
`geo`, `order`, `inventory`, `billing`, `report` **cố ý không publish cổng ra host**
— chúng đi qua gateway. Tất cả service đều có `healthcheck` + `restart: unless-stopped`,
nên container chết sẽ hiện `unhealthy` trong `make compose-ps` thay vì im lặng.

Biến môi trường build của website:

| Biến | Mặc định | Ý nghĩa |
|------|----------|---------|
| `WEB_PORT` | `8090` | cổng **host** map → nginx container `:8080` (chỉ local override) |
| `WEB_API_BASE_URL` | *(rỗng)* | `--dart-define=API_BASE_URL`; để rỗng = same-origin qua nginx |
| `FLUTTER_VERSION` | `3.44.0` | image `ghcr.io/cirruslabs/flutter` dùng để build web |

> Dev hằng ngày trên Flutter vẫn nên dùng `make flutter-web` (hot reload) và trỏ
> vào gateway `:8080`. Service `web` trong compose là bản build release để smoke
> test / demo giống production.

### 1. NATS JetStream (thủ công nếu không dùng shortcut)

```powershell
docker compose -f deploy/docker-compose.yml up nats -d
# chờ healthy rồi bootstrap stream theo bounded context (architecture §5.1)
go run ./cmd/nats-init
# monitoring: http://127.0.0.1:8222/jsz
```

### 2. API Gateway (smoke Sprint 0)

```powershell
go run ./services/api-gateway
# GET http://127.0.0.1:8080/healthz
# GET http://127.0.0.1:8080/v1/hello
```

### 3. Một service khác (ví dụ auth)

```powershell
go run ./services/auth-service
# GET http://127.0.0.1:8081/healthz
```

### 4. Flutter CTA shell (Web + Android + iOS — T9.2.4)

Cùng codebase `apps/mobile`: Home = brand **Gas Tam Đệ** + CTA khách (**Đặt giao gas**) / admin (**Dành cho cửa hàng**). Chi tiết: [`apps/mobile/README.md`](apps/mobile/README.md).

```powershell
cd apps/mobile
flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=web,android,ios   # lần đầu nếu thiếu xcodeproj/gradlew
flutter pub get
flutter run -d chrome      # Web
flutter run -d android     # Android emulator
flutter run -d ios         # iOS Simulator (macOS)
```

Shortcut: `make flutter-web` / `flutter-android` / `flutter-ios` (hoặc `.\scripts\dev.ps1 …`).

## DX shortcuts (Makefile + scripts)

| Mục đích | `make` | PowerShell |
|----------|--------|------------|
| Help | `make help` | `.\scripts\dev.ps1 help` |
| NATS + streams | `make nats` | `.\scripts\dev.ps1 nats` |
| Gateway | `make gateway` | `.\scripts\dev.ps1 gateway` |
| Health check | `make health` | `.\scripts\dev.ps1 health` |
| Full compose | `make compose-up` | `.\scripts\dev.ps1 compose-up` |
| Trạng thái container | `make compose-ps` | `.\scripts\dev.ps1 compose-ps` |
| Log tất cả service | `make compose-logs` | `.\scripts\dev.ps1 compose-logs` |
| Website trong Docker | `make web-up` | `.\scripts\dev.ps1 web-up` |
| Log website | `make web-logs` | `.\scripts\dev.ps1 web-logs` |
| Health cả stack | `make stack-health` | `.\scripts\dev.ps1 stack-health` |
| Flutter Web | `make flutter-web` | `.\scripts\dev.ps1 flutter-web` |
| Flutter Android | `make flutter-android` | `.\scripts\dev.ps1 flutter-android` |
| Flutter iOS | `make flutter-ios` | `.\scripts\dev.ps1 flutter-ios` |
| Flutter bootstrap | `make flutter-create` | `.\scripts\dev.ps1 flutter-create` |
| `go test ./...` | `make test` | `.\scripts\dev.ps1 test` |
