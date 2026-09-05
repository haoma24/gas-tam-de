# CODEMAP — Bản đồ chức năng ⇄ file

> **File này trả lời một câu hỏi duy nhất: "muốn sửa X thì mở file nào?"**
>
> Nguyên tắc phân vai giữa ba loại tài liệu trong repo — đừng trộn lẫn:
>
> | File | Trả lời | Thì |
> |---|---|---|
> | `docs/codemap.md` (file này) | **Ở đâu** — chức năng nào nằm ở file nào | Hiện tại |
> | `CHANGESLOG.md` + `docs/workdocs_*/` | **Vì sao** — quyết định, đánh đổi, lịch sử | Quá khứ |
> | `docs/prd.md`, `docs/architecture.md` | **Cái gì / như thế nào** — nghiệp vụ và kiến trúc | Định hướng |
>
> Codemap **không** chép lại lý do hay lịch sử. Muốn biết vì sao một thứ được làm như vậy
> thì `git log` file đó rồi tra workdocs tương ứng.

---

## 1. Tra cứu nhanh theo chức năng

Mỗi dòng là một **lát cắt dọc** đầy đủ: màn hình → route → file UI → file gọi API →
endpoint → service backend. Sửa một chức năng thường phải đụng cả dòng, không chỉ một ô.

Đường dẫn Flutter viết tắt từ `apps/mobile/lib/`, backend tính từ gốc repo.

### 1.1 Phía khách hàng

| Chức năng | Route | Trang (UI) | Gọi API | Endpoint | Service |
|---|---|---|---|---|---|
| Màn chào (chưa đăng nhập) | `/welcome` | `features/home/welcome_page.dart` | — | — | — |
| Đăng nhập (Google + OTP) | `/auth/login` | `features/auth/google_login_page.dart` | `features/auth/auth_api.dart` | `/v1/auth/google`, `/v1/auth/otp/request`, `/v1/auth/otp/verify` | auth |
| Cửa hàng (trang chủ) | `/` | `features/home/customer_shop_page.dart` | `features/catalog/catalog_api.dart`, `features/inventory/stock_levels_api.dart` | `GET /v1/products`, `GET /v1/stock/levels` | catalog, inventory |
| Chi tiết sản phẩm | `/products/:id` | `features/catalog/product_detail_page.dart` | `features/catalog/catalog_api.dart` | `GET /v1/products` | catalog |
| Đặt hàng (1 màn: giỏ + địa chỉ + người nhận + thanh toán) | `/order` | `features/order/order_page.dart` | `features/order/order_api.dart` | `POST /v1/orders/quote`, `POST /v1/orders` | order |
| Chọn / tìm địa chỉ, GPS, sổ địa chỉ | `/order/address` | `features/order/order_address_page.dart` | `features/order/geo_api.dart` | `GET /v1/geo/search`, `GET /v1/geo/store`, `POST /v1/geo/check` | geo |
| Đặt hàng thành công | `/order/success` | `features/order/order_success_page.dart` | — | — | — |
| Đơn hàng của tôi | `/orders` | `features/order/my_orders_page.dart` | `features/order/order_api.dart` | `GET /v1/orders/me`, `POST /v1/orders/{id}/cancel` | order |
| Hồ sơ + chọn giao diện Sáng/Tối | `/profile` | `features/auth/customer_profile_page.dart` | `features/auth/me_api.dart` | `GET /v1/me`, `PATCH /v1/me` | auth |

**Vỏ điều hướng khách:** `features/shell/customer_shell.dart` — bottom nav 3 tab
(Cửa hàng `/` | Đơn hàng `/orders` | Hồ sơ `/profile`), khai báo bằng
`StatefulShellRoute.indexedStack` trong `core/router.dart`.

**Trạng thái cục bộ của luồng đặt hàng** — không có API riêng, đều là provider trong `features/order/`:

| Việc | File |
|---|---|
| Giỏ hàng | `order_cart.dart` |
| Địa chỉ đang chọn | `order_address_selection.dart` |
| Sổ địa chỉ đã lưu | `saved_addresses.dart` |
| «Đặt lại đơn trước» — nạp sẵn giỏ + địa chỉ | `last_order.dart`, `customer_order_prefill.dart` |
| Xin quyền vị trí | `location_permission.dart` |
| Mở Google Maps chỉ đường | `navigation_link.dart` |
| Badge thời gian chờ | `wait_time_badge.dart` |
| Chuông báo đơn chờ + popup Báo lại / Không hiển thị lại (admin) | `new_order_alarm.dart` |

### 1.2 Phía admin

| Chức năng | Route | Trang (UI) | Gọi API | Endpoint | Service |
|---|---|---|---|---|---|
| Đăng nhập admin | `/admin/login` | `features/auth/admin_login_page.dart` | `features/auth/auth_api.dart` | `POST /v1/auth/admin/login` | auth |
| **Đơn** — hàng chờ giao + lịch sử, lọc theo trạng thái (màn đích sau đăng nhập) | `/admin` | `features/order/admin_orders_page.dart` | `features/order/order_api.dart` | `GET /v1/admin/orders?status=PENDING\|COMPLETED\|CANCELLED\|ALL`, `POST /v1/admin/orders/{id}/complete` | order |
| Chi tiết một đơn (toàn màn, dưới 900px) — SĐT bấm gọi, khối thanh toán | `/admin/orders/detail` | `features/order/admin_orders_page.dart` | ↑ | `GET /v1/admin/orders/{id}` | order |
| **Kho** — tồn kho, nhập / xuất | `/admin/inventory` | `features/inventory/admin_inventory_page.dart` | `features/inventory/inventory_api.dart` | `GET/POST /v1/admin/inventory` | inventory |
| **Báo cáo** — metrics kỳ + thống kê theo khách + công nợ | `/admin/reports` | `features/dashboard/admin_reports_page.dart` | `features/dashboard/dashboard_api.dart`, `features/order/order_api.dart`, `features/billing/billing_api.dart` | `GET /v1/admin/dashboard/summary`, `GET /v1/admin/orders/customers`, `GET /v1/admin/debts` | report, order, billing |
| **Cài đặt** — hub + chọn giao diện | `/admin/settings` | `features/dashboard/admin_settings_page.dart` | — | — | — |
| ├ Sản phẩm (danh sách) | `/admin/products` | `features/catalog/admin_products_page.dart` | `features/catalog/catalog_api.dart` | `GET /v1/admin/products` | catalog |
| ├ Sản phẩm (thêm / sửa) | `/admin/products/new`, `/admin/products/:id` | `features/catalog/admin_product_form_page.dart` | ↑ | `POST /v1/admin/products`, `PATCH /v1/admin/products/{id}` | catalog |
| ├ Phí giao hàng | `/admin/delivery-fee` | `features/order/admin_delivery_fee_page.dart` | `features/order/delivery_fee_api.dart` | `GET/PUT /v1/admin/delivery-fee` | order |
| ├ Cửa hàng (toạ độ, bán kính) | `/admin/store` | `features/order/admin_store_page.dart` | `features/order/geo_api.dart` | `PUT /v1/admin/geo/store` | geo |
| ├ Cài đặt quầy (giờ mở, thời gian chờ) | `/admin/desk-settings` | `features/order/admin_desk_settings_page.dart` | `features/order/desk_settings_api.dart` | `GET/PUT /v1/admin/desk-settings` | order |
| ├ Số điện thoại admin | `/admin/admin-phones` | `features/auth/admin_admin_phones_page.dart` | `features/auth/admin_phones_api.dart` | `GET/POST/DELETE /v1/admin/admin-phones` | auth |
| └ Tài khoản quản lý | `/admin/admin-accounts` | `features/auth/admin_admin_accounts_page.dart` | `features/auth/admin_accounts_api.dart` | `GET/POST/PATCH /v1/admin/admin-accounts` | auth |

**Vỏ điều hướng admin:** `features/shell/admin_shell.dart` — 4 tab
(Đơn | Kho | Báo cáo | Cài đặt). Responsive: `NavigationBar` dưới 900px,
`NavigationRail` + bố cục 2 cột từ 900px (`AppBreakpoints.expanded`).
Bảy màn trong nhóm «Cài đặt» là route đẩy chồng lên shell, không phải tab.

---

## 2. Tầng chung phía Flutter

Sửa những file này là **ảnh hưởng toàn app** — đọc kỹ trước khi đụng.

| Việc | File |
|---|---|
| Điểm vào, `MaterialApp.router`, `themeMode` | `main.dart` |
| Toàn bộ route + **một** redirect guard đăng nhập | `core/router.dart` |
| `Dio` provider, interceptor refresh token (pre-emptive + retry 401) | `core/api_client.dart` |
| `API_BASE_URL` từ `--dart-define` | `core/api_config.dart` |
| Format tiền, ngày, số | `core/format.dart` |
| Phiên đăng nhập, `sharedPreferencesProvider`, `session.isAdmin` | `features/auth/auth_session.dart`, `features/auth/auth_session_store.dart` |
| Chuẩn hoá / che số điện thoại VN | `features/auth/phone_utils.dart` |
| Mở trình quay số (`tel:`) từ màn admin | `core/phone_link.dart` |
| Nút Google Sign-In (web / stub theo nền tảng) | `features/auth/google_auth.dart`, `features/auth/google_sign_in_button*.dart` |

### Design system — `core/ui/`

Import một cửa qua `core/ui/ui.dart`; **không** import lẻ từng file, và **không**
viết `TextStyle` hay mã hex trực tiếp trong màn hình.

| File | Nội dung |
|---|---|
| `app_tokens.dart` | `AppPalette` (ThemeExtension: `ink` / `inkMuted` / `surface` / `border` / `primary` / `secondary`, `accent` là alias của `primary`), `AppSpacing`, `AppRadius`, extension `context.palette`, `context.text` |
| `app_theme_data.dart` | Dựng `ThemeData` light + dark từ token, theme hoá mọi component |
| `app_theme_mode.dart` | `themeModeProvider` (lưu `SharedPreferences`, mặc định **Sáng**) + khối UI «Giao diện» dùng chung |
| `app_breakpoints.dart` | `AppBreakpoints`, `context.isExpanded` (mốc 900px) |
| `app_scaffold.dart` | `AppScaffold`, `AppSectionTitle`, `VGap` |
| `app_section.dart` | `AppSection` — card có tiêu đề + icon |
| `app_button.dart` | Nút chính / phụ / nguy hiểm |
| `app_field.dart` | Ô nhập, ô tìm kiếm |
| `app_states.dart` | Empty state, error state, spinner |
| `app_badge.dart` | Badge trạng thái, badge khẩn |
| `app_money.dart` | `MoneyRow`, `QtyStepper` |
| `app_tile.dart` | `AppNavTile` — dòng điều hướng trong trang Cài đặt / Hồ sơ |
| `auth_layout.dart` | Bố cục dùng chung cho các màn đăng nhập |

---

## 3. Backend — service, cổng, dữ liệu

Mỗi service là một `package main`, một file SQLite riêng, test nằm cùng package.
Khuôn dựng giống hệt nhau ở mọi `services/*/main.go`:
`config.ListenAddr` → `sqlite.Open` → `migrate()` từ `//go:embed schema.sql` →
(tuỳ chọn) `natsx.NewBackground(url).Start(...)` → `httpx.NewRouter` + `MountHealth`
+ `MountReady` → `httpx.ListenAndServe`.

| Service | Cổng | Endpoint | Bảng SQLite | Sự kiện |
|---|---|---|---|---|
| `services/api-gateway` | 8080 | reverse-proxy toàn bộ `/v1/*` | `gateway.db` (audit) | — |
| `services/auth-service` | 8081 | `/v1/auth/{otp/request,otp/verify,refresh,logout,google}`, `/v1/auth/admin/login`, `/v1/me`, `/v1/admin/admin-phones*`, `/v1/admin/admin-accounts*`, `/v1/internal/users/phones` | `users`, `otp_challenges`, `sessions`, `admin_accounts`, `admin_phones`, `audit_logs` | — |
| `services/catalog-service` | 8082 | `/v1/products`, `/v1/admin/products*` | `products`, `product_price_history` | phát `catalog.product.updated` |
| `services/geo-service` | 8083 | `/v1/geo/{store,search,check}`, `/v1/admin/geo/store` | `store_settings`, `geocode_cache` | — |
| `services/order-service` | 8084 | `/v1/orders*`, `/v1/orders/quote`, `/v1/orders/me{,/defaults}`, `/v1/admin/orders*`, `/v1/admin/orders/customers`, `/v1/admin/{delivery-fee,desk-settings}` | `orders` (+ `customer_phone`), `order_items` (+ `unit_cost`), `delivery_fee_settings`, `delivery_fee_rules`, `desk_settings`, `processed_events` | phát `order.placed`, `order.completed` (kèm `unit_cost`), `order.cancelled` |
| `services/inventory-service` | 8085 | `/v1/stock/levels`, `/v1/admin/inventory`, `/v1/internal/stock/{reserve,release}` | `stock_items`, `stock_movements`, `processed_events` | nghe `order.completed`, `catalog.product.updated` |
| `services/billing-service` | 8086 | `/v1/admin/debts`, `/v1/internal/payments` | `payments`, `debts`, `debt_ledger`, `processed_events` | phát `billing.payment.recorded`, `billing.debt.updated` |
| `services/report-service` | 8087 | `/v1/admin/dashboard/summary` | `daily_stats`, `dashboard_snapshot`, `customer_debt_balances`, `processed_events` | nghe `order.placed`, `order.completed`, `billing.debt.updated` |

Website: cổng host `8090` → nginx `:8080` trong container. NATS monitoring `8222`.

**Thư viện dùng chung — `pkg/`**

| Gói | Việc |
|---|---|
| `pkg/config` | Đọc env, `ListenAddr`, `SecretFingerprint` (in `jwt_secret_fp` lúc khởi động) |
| `pkg/httpx` | Router chuẩn, `MountHealth` (`/healthz`), `MountReady` (`/readyz`), `ListenAndServe` (ép `0.0.0.0`, `tcp4`) |
| `pkg/sqlite` | Mở DB: WAL, `SetMaxOpenConns(1)` |
| `pkg/natsx` | `Background` (kết nối lại mãi, backoff 1s→30s), `JSProvider`, `Static(js)` cho test |
| `pkg/events` | Hằng số subject + `Envelope` (`event_id`, `occurred_at`, `schema_version`) |
| `cmd/nats-init` | Tạo stream JetStream lúc bootstrap (`make nats`) |

---

## 4. Phân quyền — chỉ nằm ở gateway

**Quan trọng:** service phía sau **không** kiểm tra role lần nữa. Thêm một endpoint
nghĩa là phải thêm nó vào đúng nhóm trong `newGatewayRouter` (`services/api-gateway/main.go`),
nếu không nó sẽ hoặc lộ công khai, hoặc không ai gọi được.

| Nhóm | Middleware | Đường dẫn |
|---|---|---|
| Công khai | rate limit OTP / login | `/v1/auth/*`, `/v1/products*`, `GET /v1/geo/store`, `GET /v1/geo/search`, `GET /v1/stock/levels` |
| Khách | `RequireJWT` + `RequireRole(customer)` + rate limit đặt đơn | `GET/PATCH /v1/me`, `POST /v1/geo/check`, `/v1/orders*` |
| Admin | `RequireJWT` + `RequireRole(admin)` + `AuditAdminMutations` | `/v1/admin/*` (admin-phones, admin-accounts, products, geo, orders, delivery-fee, desk-settings, inventory, debts, dashboard) |

Header nhận dạng do client gửi lên (`X-User-Id`, `X-User-Role`, …) bị xoá trong
`stripInboundIdentityHeaders` **trước** khi middleware JWT gắn lại giá trị tin cậy.

Gateway bind `:8080` bằng handler chỉ-health trước, đổi sang router thật khi SQLite
sẵn sàng (`atomicHandler`) — để Traefik không phải chờ DB khởi tạo.

---

## 5. Giao tiếp giữa các service

Có **hai** đường, đừng nhầm: gọi HTTP nội bộ (đồng bộ, chặn luồng đặt đơn) và sự kiện
NATS (bất đồng bộ, cho số liệu phái sinh).

### 5.1 Gọi HTTP nội bộ — đồng bộ

`order-service` là bên duy nhất gọi ra. Hỏng ở đây là **hỏng ngay luồng đặt / hoàn tất đơn**.

| Từ | Đến | Endpoint | Khi nào |
|---|---|---|---|
| `services/order-service/inventory_client.go` | inventory | `POST /v1/internal/stock/reserve` | Giữ hàng cho đơn `PENDING`; **trả về `unit_cost`** (giá vốn) từng dòng để lưu vào `order_items` |
| `services/order-service/inventory_client.go` | inventory | `POST /v1/internal/stock/release` | Trả hàng lại khi huỷ đơn |
| `services/order-service/clients.go` | billing | `POST /v1/internal/payments` | Ghi thanh toán / công nợ |
| `services/order-service/auth_client.go` | auth | `POST /v1/internal/users/phones` | Lấy SĐT thật của khách (lúc đặt đơn + vá ngược đơn cũ). **Best-effort**: auth chết thì đơn vẫn đặt được |

Phía nhận: `services/inventory-service/reserve.go`, `services/billing-service/record_payment.go`,
`services/auth-service/internal_users.go`.
Các endpoint `/v1/internal/*` này **không** đi qua gateway.

### 5.2 Sự kiện NATS JetStream

Subject là hằng số trong `pkg/events/events.go`, đặt theo `<context>.<entity>.<verb_past>`.

| Subject | Phát bởi | Nghe bởi |
|---|---|---|
| `catalog.product.updated` | catalog (`product_events.go`) | inventory (`product_updated.go`) |
| `order.placed` | order (`order_events.go`) | report (`order_stats.go`) |
| `order.completed` | order (`order_events.go`) | inventory (`order_completed.go`), report (`order_stats.go`) |
| `order.cancelled` | order (`order_events.go`) | — |
| `billing.payment.recorded` | billing (`billing_events.go`) | — |
| `billing.debt.updated` | billing (`billing_events.go`) | report (`debt_stats.go`) |

**Đã khai báo trong `pkg/events/events.go` nhưng chưa ai phát và chưa ai nghe** — chỗ chừa sẵn,
đừng tưởng đang chạy: `auth.otp.verified`, `geo.store_config.updated`,
`inventory.stock.adjusted`, `inventory.low_stock`.

**Luật quan trọng:** tồn kho và báo cáo doanh thu chỉ cập nhật khi `order.completed`,
**không bao giờ** khi `order.placed` (report có nghe `order.placed` nhưng chỉ để đếm đơn đã đặt).

**Chuỗi tính lợi nhuận** (`profit = revenue − cogs`, `report-service/profit.go`) — đứt một mắt là
lợi nhuận bằng doanh thu, không báo lỗi ở đâu cả:
`stock_items.cost_price` (admin nhập ở tab Kho) → `reserve.go` chốt vào `stock_movements.unit_cost`
→ response reserve → `order_items.unit_cost` → payload `order.completed` → `daily_stats.cogs_vnd`.

Consumer là subscription JetStream bền, gắn từ callback `Start(onReady)`, dùng
`ManualAck` + `Nak` khi lỗi, và idempotent nhờ bảng `processed_events(event_id PRIMARY KEY)`
trong DB của từng service.

---

## 6. Hạ tầng, deploy, CI

| Việc | File |
|---|---|
| Compose cho VPS (không `build:`, không `ports:`, ảnh từ GHCR, mạng `tensorship-net`) | `deploy/docker-compose.yml` |
| Compose overlay cho máy local (host port + build context) | `deploy/docker-compose.local.yml` |
| Image web: Flutter build + nginx + gateway nhúng | `deploy/Dockerfile.web`, `deploy/docker-entrypoint.web.sh` |
| Image cho mọi Go service | `deploy/Dockerfile.service` |
| **nginx cho web** — proxy `/v1`, SPA fallback, `Cache-Control` cho asset không hash | `deploy/nginx.web.conf` |
| Cấu hình NATS | `deploy/nats.conf` |
| Kiểm `.env` an toàn YAML (chạy trong CI) | `scripts/check-env-yaml-safe.sh`, `deploy/compose_env_test.go` |
| Chẩn đoán VPS | `scripts/vps-net-check.sh`, `scripts/vps-api-diagnose.sh`, `scripts/vps-compose-up.sh` |
| Lệnh dev trên Windows | `scripts/dev.ps1` (`.\scripts\dev.ps1 help`) |
| CI Go: test → build + push ảnh GHCR `:stag` | `.github/workflows/backend-ci.yml` |
| CI Flutter: analyze + test | `.github/workflows/flutter-ci.yml` |
| CI web: Flutter Web + nginx image | `.github/workflows/web-image.yml` |

Nhánh đích là **`stag`**, không phải `main`. VPS deploy bằng
`docker compose up --no-build`, nên một service chỉ deploy được khi ảnh đã lên GHCR.

---

## 7. Sửa gì thì chạy test gì

| Đụng vào | Chạy |
|---|---|
| Bất kỳ service Go nào | `make test` (`go test ./...`) |
| Một service | `go test ./services/order-service` |
| Một test | `go test ./services/order-service -run TestCreateOrder -v` |
| Flutter | `cd apps/mobile && flutter analyze && flutter test` |
| Một test Flutter | `flutter test test/theme_mode_test.dart` |
| Toàn stack tại chỗ | `make compose-up` → `make stack-health`; hỏng thì `make doctor` |
| Chỉ website | `make web-up` → `make web-health` |

**Test Flutter hiện có** (`apps/mobile/test/`) — sửa vùng nào thì test tương ứng là lưới an toàn:

| Test | Bảo vệ |
|---|---|
| `app_theme_test.dart` | Theme dựng được cả light lẫn dark |
| `theme_mode_test.dart` | Lưu / khôi phục lựa chọn giao diện, đổi qua UI |
| `shell_responsive_test.dart` | Ngưỡng đổi bottom nav ↔ rail, số tab |
| `welcome_page_test.dart` | Màn chào chỉ có đúng một hành động |
| `auth_session_refresh_test.dart` | Luồng refresh token |
| `phone_utils_test.dart` | Đầu số di động VN, che số |
| `admin_accounts_test.dart`, `admin_phones_test.dart` | Quản lý tài khoản / SĐT admin |
| `admin_inventory_picker_test.dart` | Chọn sản phẩm khi chưa có dòng tồn |
| `saved_addresses_test.dart` | Sổ địa chỉ |
| `navigation_link_test.dart` | Link chỉ đường |
| `new_order_alarm_test.dart` | WAV chuông báo (header, 2 tiếng «ting», đuôi im để lặp mượt) + popup trả snooze / tắt hẳn |
| `product_image_test.dart` | Ảnh sản phẩm + fallback |
| `dashboard_models_test.dart` | Model báo cáo |
| `admin_order_filter_test.dart` | Chip lọc trạng thái đơn → `status` gửi lên API, lịch sử không tự làm mới |
| `admin_order_phone_test.dart` | SĐT thật / fallback masked, `tel:`, công nợ đơn, lãi gộp dòng hàng |
| `customer_stats_test.dart` | Model thống kê theo khách + khoảng ngày của kỳ báo cáo |

Nhiễu đã biết, **không** phải lỗi mới: `flutter analyze` còn vài info deprecation tồn tại từ trước;
`dashboard_models_test.dart` vướng ràng buộc SDK về dấu phân cách chữ số.

---

## 8. Bẫy hay dính

Chi tiết đầy đủ ở `CLAUDE.md` mục *Things that bite*. Bản rút gọn:

- `/healthz` là **liveness** (không chạm NATS, dùng cho healthcheck compose); `/readyz` là readiness. Đừng đổi chỗ.
- `JWT_SECRET` phải khớp giữa auth-service và **cả hai** gateway. So `jwt_secret_fp=<8 hex>` trong log trước tiên.
- Không bao giờ bind `127.0.0.1` — `httpx.ListenAndServe` ép `0.0.0.0` trên `tcp4`.
- Thêm `build:` hay `ports:` vào `deploy/docker-compose.yml` là hỏng deploy.
- Giá trị `.env` không được chứa `": "` (nền tảng deploy chép vào YAML không trích dẫn).
- Flutter Web dùng hash route (`/#/admin/login`); URL không có `#` sau khi reload trông như hỏng nhưng không phải.
- Không có công cụ migration: mọi thay đổi `schema.sql` phải idempotent (`CREATE TABLE IF NOT EXISTS` / chỉ thêm cột).
  DB đã deploy **không** nhận cột mới từ `CREATE TABLE IF NOT EXISTS` — phải khai báo trong bảng
  `ensureColumn` ở `migrate()` (`auth-service/main.go`, `order-service/main.go`).
- Lợi nhuận bằng doanh thu ⇒ đứt chuỗi COGS ở §5.2, hoặc admin chưa nhập giá nhập trong tab «Kho».
  Tab «Báo cáo» hiện cảnh báo khi `cogs = 0` mà `revenue > 0`.
- `customer_phone` chỉ xuất hiện ở view admin (`order-service/pii.go:adminOrderView`).
  `customerOrderView` (API của khách) **không bao giờ** kèm số thật.
- Asset Flutter Web (`main.dart.js`, `assets/**`) dùng URL cố định không hash — `deploy/nginx.web.conf` phải giữ `Cache-Control: no-cache` cho chúng.

---

## 9. Cập nhật file này

Codemap chỉ có giá trị khi còn đúng. Cập nhật **cùng commit** với thay đổi, không để sau:

| Khi | Sửa mục |
|---|---|
| Thêm / xoá / đổi route Flutter | §1 |
| Thêm file trong `core/` hoặc `core/ui/` | §2 |
| Thêm / đổi endpoint | §1 (dòng chức năng) + §3 (bảng service) + **§4 (nhóm quyền ở gateway)** |
| Thêm bảng SQLite | §3 |
| Thêm subject NATS | §5 |
| Thêm file deploy / workflow / script | §6 |
| Thêm test | §7 |

Thay đổi lớn vẫn phải theo quy trình bắt buộc: workdocs + prepend `CHANGESLOG.md`
(xem `CLAUDE.md` và `.cursor/rules/change-workdocs.mdc`).
