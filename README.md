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
| API Gateway | <http://127.0.0.1:8080> | truy cập API trực tiếp |
| NATS monitoring | <http://127.0.0.1:8222/jsz> | |

Website gọi API **same-origin**: nginx proxy `/v1/*` và `/healthz` sang
`api-gateway:8080`, nên trình duyệt không cần CORS. Các service `auth`, `catalog`,
`geo`, `order`, `inventory`, `billing`, `report` **cố ý không publish cổng ra host**
— chúng đi qua gateway. Tất cả service đều có `healthcheck` + `restart: unless-stopped`,
nên container chết sẽ hiện `unhealthy` trong `make compose-ps` thay vì im lặng.

Biến môi trường build của website:

| Biến | Mặc định | Ý nghĩa |
|------|----------|---------|
| `WEB_PORT` | `8090` | cổng host của nginx |
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
