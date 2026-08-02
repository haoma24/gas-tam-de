# CHANGESLOG — Gas Tam Đệ

Nhật ký thay đổi của repo. Entry mới nhất ở **trên cùng**.  
Quy trình: skill `.cursor/skills/change-workdocs`.

---

## [2026-08-02] Platform checklist Web + Android + iOS (T9.2.5)

- **Loại:** docs / chore
- **Phạm vi:** `apps/mobile`, `.github/workflows`, `docs`
- **Tóm tắt:** Đóng Sprint 0 platform — checklist không dùng API single-OS thiếu fallback; audit geolocator / url_launcher / flutter_map; verify Web + Android Emulator + iOS Simulator hoặc CI macOS; workflow Flutter analyze/test/web + iOS no-codesign (không secret).
- **Chi tiết:**
  - `apps/mobile/PLATFORM_CHECKLIST.md` + cập nhật README
  - `.github/workflows/flutter-ci.yml` (ubuntu analyze/test/web; macos `flutter build ios --no-codesign`)
  - Mark `- [DONE] T9.2.5` — mọi task `T*.*.*` trong PRD đã DONE
- **Workdocs:** `workdocs_platform_checklist_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.5

## [2026-08-02] Flutter CTA shell Web + Android + iOS (T9.2.4)

- **Loại:** feature / docs
- **Phạm vi:** `apps/mobile`, `Makefile`, `scripts`, `docs`, root README
- **Tóm tắt:** Hoàn thiện CTA shell multi-platform — Home khách vs admin, scaffold `android`/`ios`/`web` đủ để chạy cùng codebase, DX targets + README hướng dẫn 3 target.
- **Chi tiết:**
  - Bổ sung Android res (themes, mipmap), debug/profile manifests, gradle-wrapper.properties
  - iOS: Flutter xcconfig, storyboards, Assets, hand-authored `Runner.xcodeproj` + workspace/scheme, GeneratedPluginRegistrant stubs
  - Web icons/favicon (incl. maskable); `.metadata` android/ios/web
  - README bảng CTA + lệnh `flutter run` Web / Android emulator / iOS Simulator
  - `make` / `dev.ps1`: `flutter-create`, `flutter-android`, `flutter-ios`
  - Mark `- [DONE] T9.2.4`; T9.2.5 còn lại (checklist verify)
- **Workdocs:** `workdocs_flutter_cta_shell_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.4

## [2026-08-02] Gateway audit log admin actions (T9.1.4)

- **Loại:** security
- **Phạm vi:** `services/api-gateway`, `deploy`, `docs`
- **Tóm tắt:** Ghi audit mọi request admin mutating qua gateway — actor (JWT sub), method/path, thời điểm, HTTP status — vào SQLite `gateway.db` và structured log `admin_audit`.
- **Chi tiết:**
  - Middleware `AuditAdminMutations` sau JWT + RBAC admin; chỉ `POST`/`PUT`/`PATCH`/`DELETE`
  - Bảng `admin_audit_logs`; env `GATEWAY_DB` (default `data/gateway.db`)
  - Unit tests memory + SQLite; mark `- [DONE] T9.1.4`
- **Workdocs:** `workdocs_gateway_admin_audit_02082026/`
- **Liên quan:** US-9.1 / T9.1.4

## [2026-08-02] Gateway security headers + ẩn internal error (T9.1.3)

- **Loại:** security
- **Phạm vi:** `services/api-gateway`, `pkg/httpx`, `docs`
- **Tóm tắt:** Thêm security headers trên mọi response gateway và đảm bảo lỗi proxy/panic trả JSON generic — không lộ URL nội bộ, dial error hay stack trace cho client.
- **Chi tiết:**
  - Middleware `SecurityHeaders` (nosniff, frame DENY, referrer, permissions-policy, CSP frame-ancestors)
  - Proxy `ErrorHandler` log server-side; client luôn `502 BAD_GATEWAY` / `upstream unavailable`; strip `Server` / `X-Powered-By`
  - `httpx.SafeRecover` thay chi Recoverer → `500 INTERNAL_ERROR` JSON
  - Unit tests headers + no-leak; mark `- [DONE] T9.1.3`
- **Workdocs:** `workdocs_gateway_security_headers_02082026/`
- **Liên quan:** US-9.1 / T9.1.3

## [2026-08-02] Gateway rate limit OTP / login / place-order (T9.1.2)

- **Loại:** security
- **Phạm vi:** `services/api-gateway`, `deploy`, `docs`
- **Tóm tắt:** Thêm rate limit cạnh gateway cho OTP request, admin login và place-order (IP/user), trả `429 RATE_LIMITED` + `Retry-After` — bổ sung limiter phone/IP sẵn có trên auth-service.
- **Chi tiết:**
  - Sliding-window per minute: OTP/login theo IP; place-order theo IP + JWT subject
  - Env `RATE_LIMIT_*`; CORS expose `Retry-After`
  - Unit tests limiter + endpoint 429; mark `- [DONE] T9.1.2`
- **Workdocs:** `workdocs_gateway_rate_limit_02082026/`
- **Liên quan:** US-9.1 / T9.1.2

## [2026-08-02] Gateway routing, CORS, JWT validation (T9.1.1)

- **Loại:** feature / security
- **Phạm vi:** `services/api-gateway`, `deploy`, `apps/mobile`, `docs`
- **Tóm tắt:** Thay stub 501 bằng reverse-proxy thật tới mọi upstream, CORS cho Flutter Web, và cứng JWT (require `exp`, strip spoof headers) trên nền RBAC T1.2.4.
- **Chi tiết:**
  - Proxy giữ path `/v1/...`; admin split catalog/geo/order/inventory/billing/report
  - `CORS_ORIGINS` (default `localhost` / `127.0.0.1` wildcard port); OPTIONS preflight
  - Upstream down → `502 BAD_GATEWAY`; tests proxy/CORS/RBAC
  - Mark `- [DONE] T9.1.1`
- **Workdocs:** `workdocs_gateway_routing_cors_jwt_02082026/`
- **Liên quan:** US-9.1 / T9.1.1

## [2026-08-02] Flutter dashboard widgets (T8.1.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile` (dashboard), `docs/prd.md`
- **Tóm tắt:** Admin desk `/admin` hiện widgets tổng quan doanh thu / lợi nhuận / phí giao / công nợ / số đơn từ `GET /v1/admin/dashboard/summary`, kèm filter Hôm nay / 7 ngày / Tháng này.
- **Chi tiết:**
  - `features/dashboard/` — models, API client, `AdminDashboardPage` (summary + nav tiles)
  - `ApiConfig` note report-service `:8087`; README verify
  - Mark `- [DONE] T8.1.3`
- **Workdocs:** `workdocs_flutter_admin_dashboard_02082026/`
- **Liên quan:** US-8.1 / T8.1.3

## [2026-08-02] API dashboard summary (T8.1.2)

- **Loại:** feature
- **Phạm vi:** `services/report-service`, `services/api-gateway`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Admin đọc tổng hợp dashboard (doanh thu / COGS / phí / profit / đơn / công nợ) theo ngày hoặc khoảng ngày từ `daily_stats`, kèm debt snapshot từ `billing.debt.updated`.
- **Chi tiết:**
  - `GET /v1/admin/dashboard/summary` (`day` | `from`+`to` | mặc định hôm nay VN)
  - Consumer `report-billing-debt-updated` → `customer_debt_balances` + `dashboard_snapshot.debt_total`
  - Tests report-service + gateway RBAC path; mark `- [DONE] T8.1.2`
- **Workdocs:** `workdocs_report_dashboard_summary_api_02082026/`
- **Liên quan:** US-8.1 / T8.1.2

## [2026-08-02] report-service subscribe events → daily_stats (T8.1.1)

- **Loại:** feature
- **Phạm vi:** `services/report-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** JetStream durable consumers upsert `daily_stats` từ `order.placed` / `order.completed` (idempotent), nền tảng dashboard summary T8.1.2.
- **Chi tiết:**
  - Consumers `report-order-placed`, `report-order-completed` + migrate schema + NATS wire-up
  - Upsert theo ngày VN; profit qua `BuildDailyStatsAmounts` / `ApplyProfit`; fee derive từ `total − revenue` khi thiếu `delivery_fee`
  - Tests handler + JetStream; mark `- [DONE] T8.1.1`
- **Workdocs:** `workdocs_report_daily_stats_events_02082026/`
- **Liên quan:** US-8.1 / T8.1.1

## [2026-08-02] Công thức profit report-service (T7.2.2)

- **Loại:** feature
- **Phạm vi:** `services/report-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt công thức lợi nhuận MVP `profit = revenue − COGS` (fee ship tách riêng) thành helper tái sử dụng cho `daily_stats`, nền tảng dashboard T8.1.x.
- **Chi tiết:**
  - `SumSaleRevenue` / `SumCOGS` / `ComputeProfit` / `BuildDailyStatsAmounts` / `ApplyProfit`
  - Tests: multi-line, fee không trừ profit, profit âm khi COGS > revenue
  - Sync architecture §6.7 + schema comments; mark `- [DONE] T7.2.2`
- **Workdocs:** `workdocs_report_profit_formula_02082026/`
- **Liên quan:** US-7.2 / T7.2.2

## [2026-08-02] COGS snapshot tại thời điểm xuất/bán (T7.2.1)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt contract lưu giá vốn trên mọi OUT (admin + `order.completed`): `unit_cost` snapshot từ `cost_price` lúc xuất, bất biến khi nhập sau đổi giá; nền tảng cho report profit.
- **Chi tiết:**
  - Helper `snapshotOUTCost`; bỏ qua `unit_cost` client trên OUT; insert OUT bắt buộc có snapshot
  - Tests freeze sau later IN (API + ORDER) + ignore client cost
  - Architecture §4.4 / §6.5 ghi COGS snapshot contract; mark `- [DONE] T7.2.1`
- **Workdocs:** `workdocs_inventory_cogs_snapshot_02082026/`
- **Liên quan:** US-7.2 / T7.2.1

## [2026-08-02] Flutter màn tồn kho admin (T7.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Admin Flutter list tồn kho và tạo phiếu IN/OUT/ADJUST qua `GET/POST /v1/admin/inventory`; gắn tile từ desk `/admin`.
- **Chi tiết:**
  - `features/inventory/` — models, API client, `AdminInventoryPage` (list + dialog phiếu + FAB nhập mới)
  - Route `/admin/inventory`; `ApiConfig`/README note port `:8085`; mark `- [DONE] T7.1.4`
- **Workdocs:** `workdocs_flutter_admin_inventory_02082026/`
- **Liên quan:** US-7.1 / T7.1.4

## [2026-08-02] Consumer order.completed trừ tồn (T7.1.3)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Durable JetStream consumer `inventory-order-completed` trừ tồn (OUT + snapshot COGS) khi `order.completed`; không trừ trên `order.placed`. Idempotent qua `processed_events`.
- **Chi tiết:**
  - Wire NATS trong inventory-service; Ack/Nak + transaction multi-line OUT (`ref_type=ORDER`)
  - MVP: thiếu stock → tạo placeholder, cho phép `on_hand` âm
  - Tests unit + embedded JetStream; sync architecture §6.5; mark `- [DONE] T7.1.3`
- **Workdocs:** `workdocs_inventory_order_completed_02082026/`
- **Liên quan:** US-7.1 / T7.1.3

## [2026-08-02] APIs nhập/xuất/điều chỉnh tồn (T7.1.2)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `services/api-gateway`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Admin `GET/POST /v1/admin/inventory` list tồn và tạo phiếu IN/OUT/ADJUST: cập nhật `on_hand` + `cost_price`, ghi `stock_movements`. Gateway RBAC giữ `/v1/admin/*`.
- **Chi tiết:**
  - `IN` tạo stock nếu chưa có; `cost_price` = `unit_cost`; `OUT` snapshot COGS; `ADJUST` dùng `delta` signed
  - MVP cho phép `on_hand` âm; tests validation + persist movements
  - Sync architecture §4.4; mark `- [DONE] T7.1.2`
- **Workdocs:** `workdocs_inventory_stock_apis_02082026/`
- **Liên quan:** US-7.1 / T7.1.2

## [2026-08-02] Schema stock + movements + cost (T7.1.1)

- **Loại:** feature
- **Phạm vi:** `services/inventory-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt `inventory.db`: `stock_items` / `stock_movements` / `processed_events` với CHECK giá vốn & qty; migrate-on-start; seed empty (không hardcode SP).
- **Chi tiết:**
  - Embed `schema.sql` + `seedInventoryDefaults` (`INVENTORY_SEED`)
  - Tests columns/indexes/constraints (âm `on_hand` OK; `qty > 0`)
  - Sync architecture §6.5; mark `- [DONE] T7.1.1`
- **Workdocs:** `workdocs_inventory_stock_schema_02082026/`
- **Liên quan:** US-7.1 / T7.1.1

## [2026-08-02] Flutter UI công nợ admin (T6.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Admin desk thêm màn **Công nợ** gọi `GET /v1/admin/debts`: banner tổng + list khách còn nợ (SĐT masked, số tiền).
- **Chi tiết:**
  - `BillingApi` / models + `AdminDebtsPage` (refresh, empty state)
  - Route `/admin/debts` + tile từ `/admin`
  - `ApiConfig` + README verify local `:8086`
  - Mark `- [DONE] T6.2.2` trên PRD
- **Workdocs:** `workdocs_flutter_admin_debts_02082026/`
- **Liên quan:** US-6.2 / T6.2.2

## [2026-08-02] API list/aggregate debts (T6.2.1)

- **Loại:** feature
- **Phạm vi:** `services/billing-service`, `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** Admin `GET /v1/admin/debts` trả danh sách khách còn nợ (`balance > 0`) kèm aggregate `total_balance` / `count`. Gateway giữ RBAC `/v1/admin/*` (role=admin).
- **Chi tiết:**
  - `handleListDebts` + query sort balance DESC
  - Tests empty / aggregate / omit zero-balance
  - Gateway RBAC coverage cho path debts (customer 403, admin pass-through stub)
  - Mark `- [DONE] T6.2.1` trên PRD
- **Workdocs:** `workdocs_billing_admin_debts_api_02082026/`
- **Liên quan:** US-6.2 / T6.2.1

## [2026-08-02] Flutter dialog hoàn tất đơn (T6.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Order Desk chi tiết thêm dialog «Hoàn tất» chọn FULL / PARTIAL / UNPAID (+ amount_paid), gọi `POST /v1/admin/orders/{id}/complete`, rồi quay về list (PENDING biến mất).
- **Chi tiết:**
  - Models + `OrderApi.completeOrder`
  - Dialog radio + preview công nợ; validate PARTIAL local
  - SnackBar kết quả; `onCompleted` → `/admin/orders`
  - Mark `- [DONE] T6.1.4` trên PRD
- **Workdocs:** `workdocs_flutter_order_complete_dialog_02082026/`
- **Liên quan:** US-6.1 / T6.1.4

## [2026-08-02] Events order.completed + billing.payment/debt (T6.1.3)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `services/billing-service`, `docs/prd.md`
- **Tóm tắt:** Publish JetStream `order.completed` sau admin complete; billing publish `billing.payment.recorded` + `billing.debt.updated` sau ghi payment (không republish khi idempotent). Payload theo architecture §5.1.
- **Chi tiết:**
  - Mở rộng `orderPublisher` + `PublishOrderCompleted` (items/total/payment_type/amount_paid)
  - `jsBillingPublisher` + wire billing `ConnectJS`/`EnsureStreams` (`billing.>` đã có sẵn)
  - Tests recording bus + embedded JetStream; lỗi publish chỉ log
  - Mark `- [DONE] T6.1.3` trên PRD
- **Workdocs:** `workdocs_order_billing_events_02082026/`
- **Liên quan:** US-6.1 / T6.1.3

## [2026-08-02] Billing ghi payments + cập nhật debts (T6.1.2)

- **Loại:** feature
- **Phạm vi:** `services/billing-service`, `services/order-service`, `deploy`, `docs/prd.md`
- **Tóm tắt:** Sau admin complete, billing-service ghi `payments` và tăng `debts`/`debt_ledger` theo FULL/PARTIAL/UNPAID. Sync HTTP từ order-service; events JetStream để T6.1.3.
- **Chi tiết:**
  - Migrate `billing.db`; `POST /v1/internal/payments` + `recordPayment` (idempotent `order_id`)
  - Order complete gọi billing với `customer_key=phone_hash`; lỗi billing chỉ log
  - Tests AC PARTIAL 100k/450k→debt 350k, FULL (không tạo nợ), UNPAID, accumulate, idempotent
  - Compose/env: `BILLING_SERVICE_URL` cho order-service
  - Mark `- [DONE] T6.1.2` trên PRD
- **Workdocs:** `workdocs_billing_payments_debts_02082026/`
- **Liên quan:** US-6.1 / T6.1.2

## [2026-08-02] API hoàn tất đơn + payment payload (T6.1.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** Admin `POST /v1/admin/orders/{id}/complete` nhận FULL/PARTIAL/UNPAID + `amount_paid`, chuyển PENDING→COMPLETED và trả debt settlement. Billing write / events để T6.1.2–T6.1.3.
- **Chi tiết:**
  - `handleCompleteOrder` + `settlePayment` (PRD M6 rules)
  - Snapshot `payment_type` / `amount_paid` trên `orders`; `completed_at`
  - Tests PARTIAL AC (100k/450k→debt 350k), FULL, UNPAID, validation, 404/409
  - Gateway RBAC assert cho path complete dưới `/v1/admin/*`
  - Mark `- [DONE] T6.1.1` trên PRD
- **Workdocs:** `workdocs_order_complete_api_02082026/`
- **Liên quan:** US-6.1 / T6.1.1

## [2026-08-02] Flutter nút «Dẫn đường» chi tiết đơn (T5.2.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Order Desk chi tiết đơn có nút «Dẫn đường» gọi `openNavigationTo(lat,lng)`; SnackBar nếu thiếu toạ độ hoặc không mở được Maps. Không đụng E6 hoàn tất/thanh toán.
- **Chi tiết:**
  - `AdminOrderDetailPage`: `FilledButton.icon` «Dẫn đường» sau khối địa chỉ
  - Guard `lat/lng == 0,0` (API null → model default) + hiện lỗi launch từ helper
  - Mark `- [DONE] T5.2.3` trên PRD; README verify
- **Workdocs:** `workdocs_flutter_order_nav_button_02082026/`
- **Liên quan:** Sprint 3 / US-5.2 / T5.2.3

## [2026-08-02] Flutter deep-link Maps / geo intent (T5.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Helper `openNavigationTo(lat,lng)` mở chỉ đường tới điểm giao qua Google Maps / `geo:` / Apple Maps + HTTPS fallback (Web/Android/iOS). Chưa nút UI (T5.2.3).
- **Chi tiết:**
  - `navigation_link.dart`: candidate URIs theo platform; omit `origin` → Maps dùng vị trí thiết bị
  - Android `<queries>` + iOS `LSApplicationQueriesSchemes`; `platform_config` fragments
  - Unit tests URI builders; README verify
  - Mark `- [DONE] T5.2.2` trên PRD
- **Workdocs:** `workdocs_flutter_maps_deeplink_02082026/`
- **Liên quan:** Sprint 3 / US-5.2 / T5.2.2

## [2026-08-02] Admin order lat/lng + GET by id (T5.2.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Expose `lat`/`lng` điểm giao trên admin list/detail; implement `GET /v1/admin/orders/{id}` (trước đó stub) để CCH lấy toạ độ cho dẫn đường. Chưa deep-link Maps / nút Flutter.
- **Chi tiết:**
  - `handleGetAdminOrder` + `loadOrderByID`; wire thay `notImplemented`
  - `orderView.lat`/`lng` documented as delivery destination (WGS84)
  - Unit tests get-by-id coords, 404, list exposes lat/lng
  - Mark `- [DONE] T5.2.1` trên PRD
- **Workdocs:** `workdocs_order_admin_lat_lng_02082026/`
- **Liên quan:** Sprint 3 / US-5.2 / T5.2.1

## [2026-08-02] Flutter Order Desk polling báo đơn mới (T5.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Order Desk tự poll `GET /v1/admin/orders` mỗi 10s (pause khi app background), giữ pull-to-refresh; SnackBar khi phát hiện đơn id mới. Chọn polling thay SSE/NATS cho MVP Web+mobile.
- **Chi tiết:**
  - `AdminOrdersPage`: `Timer.periodic` + `WidgetsBindingObserver`; silent refresh không che list
  - Empty state cũng kéo-refresh; copy nhắc chu kỳ poll
  - Mark `- [DONE] T5.1.4` trên PRD
- **Workdocs:** `workdocs_flutter_order_desk_polling_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.4

## [2026-08-02] Flutter Order Desk UI (T5.1.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn admin Order Desk gọi `GET /v1/admin/orders`, hiển thị STT | tên | SĐT masked | địa chỉ | km | thời gian (FIFO cũ nhất trước); link từ `/admin`. Chi tiết đọc-only từ payload list (chưa polling / maps).
- **Chi tiết:**
  - `AdminOrder` + `OrderApi.listAdminOrders`
  - `AdminOrdersPage` / `AdminOrderDetailPage`; routes `/admin/orders`, `/admin/orders/detail`
  - Tile **Order Desk** trên admin home
  - Mark `- [DONE] T5.1.3` trên PRD
- **Workdocs:** `workdocs_flutter_order_desk_ui_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.3

## [2026-08-02] Admin Order Desk columns STT + fields (T5.1.2)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Làm giàu `GET /v1/admin/orders` với `stt` (FIFO 1-based) và khẳng định contract cột desk: tên, SĐT masked, địa chỉ, km, thời gian. Chưa Flutter UI.
- **Chi tiết:**
  - `orderView.stt` (omitempty); `adminOrderViewsFromRows` gán STT
  - SĐT admin = `phone_masked` (orders không có plaintext phone)
  - Unit tests desk columns + khách không lộ `stt`
  - Mark `- [DONE] T5.1.2` trên PRD
- **Workdocs:** `workdocs_order_admin_desk_columns_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.2

## [2026-08-02] API admin list orders FIFO (T5.1.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** `GET /v1/admin/orders` danh sách đơn FIFO (`created_at ASC`); mặc định `PENDING`, optional `?status=`. Path sẵn dưới gateway admin RBAC. Chưa Flutter desk / cột STT.
- **Chi tiết:**
  - Handler thay stub; response basic `orderView` + items
  - Unit tests FIFO (A trước B), filter COMPLETED, status invalid
  - Mark `- [DONE] T5.1.1` trên PRD
- **Workdocs:** `workdocs_order_admin_list_fifo_02082026/`
- **Liên quan:** Sprint 3 / US-5.1 / T5.1.1

## [2026-08-02] Flutter review: hiển thị quote phí giao (T4.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn xác nhận đơn gọi `POST /v1/orders/quote` khi mở / trước place; hiển thị khoảng cách, phí giao, tạm tính, tổng; chặn đặt khi ngoài phạm vi hoặc quote lỗi. Bỏ stub phí = 0.
- **Chi tiết:**
  - Models `QuoteOrderRequest` / `OrderQuote`; `OrderApi.quoteOrder`
  - Review: loading báo giá, refresh, distance + fee + totals; re-quote trước place
  - Mark `- [DONE] T4.2.2` trên PRD
- **Workdocs:** `workdocs_flutter_order_review_quote_02082026/`
- **Liên quan:** Sprint 2 / US-4.2 / T4.2.2

## [2026-08-02] API quote: distance + fee + total (T4.2.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** `POST /v1/orders/quote` preview khoảng cách + phí + tổng từ geo/catalog/fee engine; không persist. Ngoài bán kính vẫn 200 kèm `in_range=false` (place giữ 422). Chưa Flutter review (T4.2.2).
- **Chi tiết:**
  - Body `{ items, lat, lng }`; response `distance_km`, `in_range`, `max_radius_km`, `delivery_fee`, `subtotal`, `total`
  - Reuse `computeDeliveryFee` + catalog prices; customer identity headers
  - Unit tests happy / OOR preview / fee off / auth / validation; mark `- [DONE] T4.2.1`
- **Workdocs:** `workdocs_order_quote_api_02082026/`
- **Liên quan:** Sprint 2 / US-4.2 / T4.2.1

## [2026-08-02] Flutter admin: màn phí giao hàng (T4.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn admin bật/tắt phí ship + chỉnh bậc km; gọi `GET/PUT /v1/admin/delivery-fee` với JWT session; tile từ desk `/admin`. Không làm quote khách (T4.2.x).
- **Chi tiết:**
  - `DeliveryFeeApi` + models; toggle `enabled` PUT ngay; **Lưu bậc** replace `rules`
  - Validate local overlap / open-ended; Material 3 khớp admin products
  - Route `/admin/delivery-fee`; README + ApiConfig note order `:8084`
  - Mark `- [DONE] T4.1.4` trên PRD
- **Workdocs:** `workdocs_flutter_admin_delivery_fee_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.4

## [2026-08-02] Engine tính phí giao khi place order (T4.1.3)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Thay stub phí = 0 bằng engine reusable: tắt → 0; bật → khớp bậc `[min_km, max_km)` theo `distance_km`; wire vào `POST /v1/orders`. Quote API để T4.2.1.
- **Chi tiết:**
  - `matchDeliveryFee` (pure) + `computeDeliveryFee` (load DB); missing settings = fee 0
  - Unit tests band / disabled / inactive / gap; place-order tests enabled vs disabled
  - Mark `- [DONE] T4.1.3` trên PRD
- **Workdocs:** `workdocs_delivery_fee_engine_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.3

## [2026-08-02] Admin APIs cấu hình phí giao (T4.1.2)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** `GET/PUT /v1/admin/delivery-fee` đọc/cập nhật toggle phí + bậc km trên order-service; validate band không overlap; path nằm dưới gateway RBAC admin — chưa fee engine / Flutter UI.
- **Chi tiết:**
  - PUT partial: `enabled` và/hoặc replace toàn bộ `rules` (transaction delete+insert)
  - Validate min/max/fee; open-ended `max_km=null` chỉ được là band active cuối
  - Tests order-service + assert customer bị FORBIDDEN trên `/v1/admin/delivery-fee`
  - Mark `- [DONE] T4.1.2` trên PRD
- **Workdocs:** `workdocs_admin_delivery_fee_apis_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.2

## [2026-08-02] Schema delivery_fee_settings + delivery_fee_rules (T4.1.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `docs/prd.md`, `docs/architecture.md`
- **Tóm tắt:** Chốt schema phí giao trên `order.db`: toggle singleton + bậc khoảng cách; migrate-on-start; seed local (fee off + 3 bậc architecture); tests schema/seed — chưa admin API / fee engine.
- **Chi tiết:**
  - CHECK `enabled`/`active`/`fee_vnd`/`min_km`/`max_km`; index `idx_delivery_fee_rules_active`
  - Seed idempotent settings `default` + rules 0–5/5–10/10–∞ (10k/20k/30k); env `DELIVERY_FEE_SEED` / `DELIVERY_FEE_ENABLED`
  - Sync architecture §6.4; Mark `- [DONE] T4.1.1` trên PRD
- **Workdocs:** `workdocs_delivery_fee_schema_02082026/`
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.1

## [2026-08-02] Mask PII trong order response (T3.3.4)

- **Loại:** security
- **Phạm vi:** `services/order-service`, `docs/prd.md`
- **Tóm tắt:** Response khách của order-service chỉ trả SĐT đã mask (`090***4567`, cùng style auth); remask defense-in-depth khi header vô tình chứa số thật; `GET /v1/orders/me` dùng cùng policy.
- **Chi tiết:**
  - `maskPhoneDisplay` / `ensurePhoneMasked` / `customerOrderView` — không trả `phone_hash` / `phone_e164`
  - Create order remask trước persist + JSON
  - `GET /v1/orders/me` (own orders, masked); buffer rows trước nested item query (tránh SQLite deadlock)
  - Tests remask + list + 401; Mark `- [DONE] T3.3.4` trên PRD
- **Workdocs:** `workdocs_order_mask_pii_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.4

## [2026-08-02] Flutter review + success place order (T3.3.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Hoàn tất bước xác nhận đơn trên Flutter: màn review (giỏ + địa chỉ + tổng), gọi `POST /v1/orders` với JWT/session, rồi màn thành công — nối từ địa chỉ in-range.
- **Chi tiết:**
  - `OrderApi` + models; local gắn `X-User-*` từ session (gateway proxy còn stub)
  - `OrderReviewPage` / `OrderSuccessPage`; routes `/order/review`, `/order/success`
  - Phí giao preview stub 0 (E4); clear cart sau place thành công
  - Mark `- [DONE] T3.3.3` trên PRD
- **Workdocs:** `workdocs_flutter_order_review_success_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.3

## [2026-08-02] Persist order + publish order.placed (T3.3.2)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `deploy`, `docs/prd.md`
- **Tóm tắt:** Sau place order thành công, polish persist (CHECK/index) và publish `order.placed` lên JetStream (payload `order_id`/`total`/`distance_km`/`created_at`) cho report — theo pattern catalog product.updated.
- **Chi tiết:**
  - Schema: CHECK money/qty; index `orders(created_at)`, `order_items(product_id)`
  - `jsOrderPublisher` + `natsx.PublishEnvelope` sau commit (lỗi chỉ log)
  - Startup: `ConnectJS` + `EnsureStreams`
  - Tests mock recorder + embedded JetStream; assert cột persisted
  - Mark `- [DONE] T3.3.2` trên PRD
- **Workdocs:** `workdocs_order_placed_event_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.2

## [2026-08-02] API POST /orders validate + thin persist (T3.3.1)

- **Loại:** feature
- **Phạm vi:** `services/order-service`, `deploy`, `docs/prd.md`
- **Tóm tắt:** `POST /v1/orders` kiểm tra identity gateway (`X-User-*`), items qua catalog, geo in-range qua geo-service, fee stub 0 (TODO E4); thin insert order + items `PENDING` (chưa `order.placed`).
- **Chi tiết:**
  - Validate body + coords; `422 OUT_OF_RANGE` kèm distance/max_radius
  - Giá lấy từ catalog active; merge qty trùng `product_id`
  - Env `GEO_SERVICE_URL` / `CATALOG_SERVICE_URL`; migrate schema khi boot
  - Unit tests happy / auth / out-of-range / product missing
  - Mark `- [DONE] T3.3.1` trên PRD
- **Workdocs:** `workdocs_order_post_create_api_02082026/`
- **Liên quan:** Sprint 2 / US-3.3 / T3.3.1

## [2026-08-02] Flutter UI ngoài phạm vi giao (T3.2.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Sau khi chọn địa chỉ, app gọi `POST /v1/geo/check`; ngoài bán kính hiện message VN rõ và chặn Tiếp tục; trong phạm vi mới cho đi tiếp (chưa place order).
- **Chi tiết:**
  - `GeoApi.check` + `GeoCheckResult` (distance / in_range / max_radius)
  - Banner đỏ ngoài phạm vi (kèm km) / xanh trong phạm vi; nút disable khi `in_range=false`
  - `orderGeoCheckProvider` giữ kết quả cho T3.3; `onContinue` stub SnackBar
  - Mark `- [DONE] T3.2.3` trên PRD
- **Workdocs:** `workdocs_flutter_out_of_range_ui_02082026/`
- **Liên quan:** Sprint 2 / US-3.2 / T3.2.3

## [2026-08-02] Haversine geo check + in_range (T3.2.2)

- **Loại:** feature
- **Phạm vi:** `services/geo-service`, `docs/prd.md`
- **Tóm tắt:** `POST /v1/geo/check` tính khoảng cách Haversine từ cửa hàng tới lat/lng khách và trả `distance_km` (2dp), `in_range`, `max_radius_km`.
- **Chi tiết:**
  - Haversine R=6371 km; `in_range` khi `distance_km <= max_radius_km`
  - Validate coords; 404 nếu chưa seed `store_settings`
  - Unit tests math + handler (trong/ngoài bán kính, boundary, invalid)
  - Mark `- [DONE] T3.2.2` trên PRD
- **Workdocs:** `workdocs_geo_haversine_check_02082026/`
- **Liên quan:** Sprint 2 / US-3.2 / T3.2.2

## [2026-08-02] Store settings lat/lng + max_radius_km (T3.2.1)

- **Loại:** feature
- **Phạm vi:** `services/geo-service`, `pkg/config`, `deploy/.env.example`, `docs/prd.md`
- **Tóm tắt:** Persist singleton `store_settings` (lat/lng, `max_radius_km`) với seed từ env cho local; expose `GET /v1/geo/store` (public) và `PUT /v1/admin/geo/store` để cập nhật — nền cho check bán kính T3.2.2.
- **Chi tiết:**
  - Seed idempotent `STORE_LAT` / `STORE_LNG` / `STORE_MAX_RADIUS_KM` (default ≈ Bến Thành HCMC, radius 10km)
  - Public GET trả name/lat/lng/max_radius_km/(address_text); không lộ `updated_by`
  - Admin PUT partial update + validate coords; unit tests seed/GET/PUT
  - `config.GetFloat`; mark `- [DONE] T3.2.1` trên PRD
- **Workdocs:** `workdocs_geo_store_settings_02082026/`
- **Liên quan:** Sprint 2 / US-3.2 / T3.2.1

## [2026-08-02] Flutter map/picker + autocomplete địa chỉ (T3.1.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Bước địa chỉ đặt hàng có autocomplete (gọi geo-service proxy, không gọi OSM trực tiếp) và bản đồ ghim pin (`flutter_map`) trên Web / Android / iOS; lưu lat/lng/label.
- **Chi tiết:**
  - `GeoApi.search` → `GET /v1/geo/search?q=` + debounce autocomplete
  - `flutter_map` + OSM tiles; chạm bản đồ / chọn gợi ý / GPS → pin + `orderAddressProvider`
  - README verify geo `:8083`; mark `- [DONE] T3.1.3` trên PRD
- **Workdocs:** `workdocs_flutter_map_autocomplete_02082026/`
- **Liên quan:** Sprint 2 / US-3.1 / T3.1.3

## [2026-08-02] Proxy search geocode Photon/Nominatim (T3.1.2)

- **Loại:** feature
- **Phạm vi:** `services/geo-service`, `docs/prd.md`, `deploy/.env.example`
- **Tóm tắt:** Geo-service proxy `GET /v1/geo/search?q=` tới Photon (mặc định) hoặc Nominatim — User-Agent, rate limit IP, cache SQLite; Flutter không gọi OSM trực tiếp.
- **Chi tiết:**
  - Provider `GEOCODE_PROVIDER=photon|nominatim`; chuẩn hóa `{items:[{label,lat,lng,source}]}`
  - `geocode_cache` + migrate schema; IP rate limit; Nominatim min 1 req/s
  - Unit tests mock upstream; ghi chú `ApiConfig` `:8083`
  - Mark `- [DONE] T3.1.2` trên PRD
- **Workdocs:** `workdocs_geo_search_proxy_02082026/`
- **Liên quan:** Sprint 2 / US-3.1 / T3.1.2

## [2026-08-02] Xin quyền location Web/Android/iOS (T3.1.1)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Khách xin quyền vị trí và lấy lat/lng trên bước địa chỉ («Dùng vị trí hiện tại») với message lỗi tiếng Việt; cấu hình permission cho 3 platform.
- **Chi tiết:**
  - Helper `location_permission.dart` (geolocator: denied / deniedForever / serviceDisabled / timeout; Web fallback khi Permissions API thiếu)
  - Nút «Dùng vị trí hiện tại» trên `/order/address`; hiện lat/lng hoặc lỗi VN
  - Android `ACCESS_FINE/COARSE_LOCATION`; iOS `NSLocationWhenInUseUsageDescription`; Web HTTPS/localhost note
  - Bỏ `permission_handler` (thừa — `geolocator` đủ 3 target)
  - Mark `- [DONE] T3.1.1` trên PRD
- **Workdocs:** `workdocs_location_permission_02082026/`
- **Liên quan:** Sprint 2 / US-3.1 / T3.1.1

## [2026-08-02] Flutter bước chọn SP đặt hàng (T2.2.2)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Thay order placeholder bằng màn chọn sản phẩm active (`GET /v1/products`) + giỏ local; Tiếp tục sang placeholder địa chỉ (E3).
- **Chi tiết:**
  - `CatalogApi.listActiveProducts`; `OrderCart` Riverpod; `SelectProductsPage` (+/− qty, tổng VND)
  - Routes `/order`, `/order/address` (placeholder)
  - README / `ApiConfig` ghi chú verify customer pick
  - Mark `- [DONE] T2.2.2` trên PRD
- **Workdocs:** `workdocs_flutter_order_select_products_02082026/`
- **Liên quan:** Sprint 2 / US-2.2 / T2.2.2

## [2026-08-02] API list products active (T2.2.1)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `docs/prd.md`
- **Tóm tắt:** `GET /v1/products` trả danh sách sản phẩm `active` cho khách (public/authenticated); ẩn SP không còn lộ ra ngoài admin.
- **Chi tiết:**
  - Handler `handleListActiveProducts` (`WHERE active = 1`) thay stub rỗng
  - Refactor `collectProducts` dùng chung với admin list
  - Unit test empty / filter inactive / admin vẫn thấy đủ
  - Mark `- [DONE] T2.2.1` trên PRD
- **Workdocs:** `workdocs_catalog_list_active_products_02082026/`
- **Liên quan:** Sprint 2 / US-2.2 / T2.2.1

## [2026-08-02] Flutter admin màn sản phẩm (T2.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Màn admin Flutter list / thêm / sửa / ẩn sản phẩm gọi catalog CRUD (`GET/POST/PATCH /v1/admin/products`); desk `/admin` có lối vào Sản phẩm.
- **Chi tiết:**
  - `CatalogApi` + models; list + form create/edit; toggle ẩn/hiện (`active`)
  - Routes `/admin/products`, `/admin/products/new`, `/admin/products/:id`
  - README / `ApiConfig` ghi chú local `API_BASE_URL` → catalog `:8082`
  - Mark `- [DONE] T2.1.4` trên PRD
- **Workdocs:** `workdocs_flutter_admin_products_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.4

## [2026-08-02] Event catalog.product.updated (T2.1.3)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `pkg/natsx`, `docs/prd.md`
- **Tóm tắt:** Catalog publish `catalog.product.updated` (envelope JetStream) sau create/update/ẩn sản phẩm — payload `product_id`/`sku`/`sale_price`/`active` cho inventory & report.
- **Chi tiết:**
  - `natsx.PublishEnvelope` (MsgId = event_id) + EnsureStreams lúc start catalog
  - Hook publish sau commit create/patch; lỗi bus chỉ log
  - Tests mock recorder + embedded JetStream
  - Mark `- [DONE] T2.1.3` trên PRD
- **Workdocs:** `workdocs_catalog_product_updated_event_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.3

## [2026-08-02] Schema products + product_prices (T2.1.2)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt schema catalog `products` + `product_price_history` (PRD `product_prices`) — indexes, CHECK/FK, comments migrate-on-start; tests assert schema.
- **Chi tiết:**
  - CHECK `sale_price >= 0`, `active IN (0,1)`; FK history → products
  - Index `idx_products_active`, `idx_price_history_product`
  - Unit tests columns/indexes + constraint/FK; sync architecture §6.2
  - Mark `- [DONE] T2.1.2` trên PRD
- **Workdocs:** `workdocs_catalog_products_schema_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.2

## [2026-08-02] CRUD APIs catalog (T2.1.1)

- **Loại:** feature
- **Phạm vi:** `services/catalog-service`, `docs/prd.md`
- **Tóm tắt:** Admin CRUD sản phẩm/giá bán trên catalog-service (`GET/POST/PATCH /v1/admin/products`) với SQLite migrate và lịch sử giá; ẩn SP bằng `active=false`.
- **Chi tiết:**
  - Endpoints: list, create, get-by-id, patch (sku/name/desc/unit/sale_price/active/image_url)
  - Ghi `product_price_history` khi tạo hoặc đổi giá; `changed_by` từ `X-User-Id`
  - Unit tests validation / SKU conflict / 404; public `GET /v1/products` vẫn stub (T2.2.1)
  - Path trên catalog `:8082` — gateway admin proxy vẫn stub
  - Mark `- [DONE] T2.1.1` trên PRD
- **Workdocs:** `workdocs_catalog_crud_apis_02082026/`
- **Liên quan:** Sprint 2 / US-2.1 / T2.1.1

## [2026-08-02] Middleware RBAC trên gateway (T1.2.4)

- **Loại:** feature / security
- **Phạm vi:** `services/api-gateway`, `docs/prd.md`
- **Tóm tắt:** Gateway validate JWT access (HS256, cùng secret/claims auth-service) và enforce RBAC — `admin` cho `/v1/admin/**`, `customer` cho orders + `POST /geo/check`; auth/public routes bỏ qua.
- **Chi tiết:**
  - `RequireJWT` + `RequireRole`; forward `X-User-Id` / `X-User-Role` / `X-Session-Id`
  - Public: health, hello, `/v1/auth/*`, products, `GET /geo/store|search`
  - Unit tests 401/403/role split; upstream vẫn stub 501
  - Mark `- [DONE] T1.2.4` trên PRD
- **Workdocs:** `workdocs_gateway_rbac_middleware_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.4

## [2026-08-02] Flutter admin login screen (T1.2.3)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `docs/prd.md`
- **Tóm tắt:** Thêm màn đăng nhập admin (username/password) trên Flutter Web/Android/iOS — gọi `POST /v1/auth/admin/login`, lưu JWT session in-memory, CTA Home «Dành cho cửa hàng».
- **Chi tiết:**
  - `AdminLoginPage` + routes `/admin/login`, `/admin` (placeholder desk)
  - `AuthApi.adminLogin`; dùng chung `AuthTokenResult` / `authSessionProvider` với OTP
  - Map lỗi `INVALID_CREDENTIALS`; README seed `admin` / `admin-change-me`
  - Mark `- [DONE] T1.2.3` trên PRD
- **Workdocs:** `workdocs_flutter_admin_login_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.3

## [2026-08-02] API login admin + refresh (T1.2.2)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `docs/prd.md`
- **Tóm tắt:** Implement `POST /v1/auth/admin/login` (bcrypt → JWT `role=admin`) và `POST /v1/auth/refresh` với rotation session — dùng chung pattern token từ OTP verify.
- **Chi tiết:**
  - Login admin: username/password → access + opaque refresh; session `role=admin`
  - Refresh xoay vòng (revoke cũ + session mới) cho admin và customer
  - Sai credentials / disabled / token hết hạn → 401 thống nhất; unit tests
  - Mark `- [DONE] T1.2.2` trên PRD
- **Workdocs:** `workdocs_admin_login_refresh_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.2

## [2026-08-02] Seed admin account bcrypt (T1.2.1)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Bootstrap tài khoản admin mặc định vào `admin_accounts` với `password_hash` bcrypt khi process start — cấu hình qua env, không commit secret.
- **Chi tiết:**
  - Seed idempotent sau migrate: insert nếu username chưa có; không overwrite password hiện có
  - Env: `ADMIN_USERNAME` / `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_DISPLAY_NAME`, `ADMIN_SEED`
  - Unit tests hash≠plaintext, verify bcrypt, idempotent, disable seed
  - Mark `- [DONE] T1.2.1` trên PRD
- **Workdocs:** `workdocs_seed_admin_account_02082026/`
- **Liên quan:** Sprint 1 / US-1.2 / T1.2.1

## [2026-08-02] OTP challenges SQLite hash + expiry (T1.1.5)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `docs/architecture.md`, `docs/prd.md`
- **Tóm tắt:** Chốt T1.1.5 — `otp_challenges` trên SQLite auth lưu OTP đã hash + expiry; siết schema/index và tests (persistence đã có từ T1.1.1/1.2).
- **Chi tiết:**
  - Comment schema rõ contract hash/expiry; thêm `idx_otp_expires`
  - Tests: migrate columns/indexes, `code_hash` ≠ plaintext, `expires_at` TTL, verify `OTP_EXPIRED`
  - Sync architecture §6.1; mark `- [DONE] T1.1.5` trên PRD
- **Workdocs:** `workdocs_otp_challenges_sqlite_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.5

## [2026-08-02] Flutter màn SĐT + OTP (T1.1.4)

- **Loại:** feature
- **Phạm vi:** `apps/mobile`
- **Tóm tắt:** Thêm màn nhập SĐT và OTP trên Flutter (Web/Android/iOS), gọi `POST /v1/auth/otp/request` + `verify` → JWT session in-memory; CTA Home đi qua auth trước order placeholder.
- **Chi tiết:**
  - `PhonePage` / `OtpPage` + validate SĐT VN, resend cooldown, hiện `dev_code` local
  - `AuthApi` (Dio) + `authSessionProvider` + interceptor gắn Bearer
  - Routes `/auth/phone`, `/auth/otp`; README hướng dẫn `API_BASE_URL` → auth `:8081` khi gateway chưa proxy
  - Mark `- [DONE] T1.1.4` trên PRD
- **Workdocs:** `workdocs_flutter_otp_ui_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.4

## [2026-08-02] Adapter SMS mock + production seam (T1.1.3)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`
- **Tóm tắt:** Thêm `SMSSender` interface, mock adapter (default local), và production seam (eSMS/Stringee) — wire vào `POST /auth/otp/request`; chưa gọi vendor thật.
- **Chi tiết:**
  - `SMS_PROVIDER=mock|production` + env `SMS_API_KEY` / `SMS_VENDOR` / `SMS_API_URL` / `SMS_SENDER`
  - Mock ghi nhận send in-memory (tests); log chỉ `phone_masked`, không raw OTP
  - Production seam trả `ErrSMSNotConfigured` cho đến khi plug client vendor
  - OTP request → `502 SMS_FAILED` nếu gửi SMS lỗi; unit tests adapter + handler
  - Mark `- [DONE] T1.1.3` trên PRD
- **Workdocs:** `workdocs_sms_adapter_mock_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.3

## [2026-08-02] OTP verify API → JWT (T1.1.2)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`
- **Tóm tắt:** Implement `POST /v1/auth/otp/verify` — kiểm tra OTP, upsert khách (phone encrypt), phát JWT access + refresh session theo architecture §7.2.
- **Chi tiết:**
  - Validate phone/code; max attempts / expire / consume; invalidate challenge mở khác cùng SĐT
  - JWT HS256 (`sub`, `role`, `phone_masked`, `sid`); refresh opaque + `sessions.refresh_hash`
  - Env: `JWT_ACCESS_TTL_SEC`, `JWT_REFRESH_TTL_SEC`, `OTP_MAX_ATTEMPTS`
  - Unit tests + mark `- [DONE] T1.1.2` trên PRD
- **Workdocs:** `workdocs_otp_verify_jwt_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.2

## [2026-08-02] OTP request API + rate limit (T1.1.1)

- **Loại:** feature
- **Phạm vi:** `services/auth-service`, `deploy/.env.example`
- **Tóm tắt:** Implement `POST /v1/auth/otp/request` với validate SĐT VN, cooldown/quota theo phone_hash + IP, và lưu OTP hash vào SQLite — nền cho verify JWT ở task tiếp theo.
- **Chi tiết:**
  - Handler trả `phone_masked` / `expires_in_sec` / `resend_after_sec`; `429 RATE_LIMITED` + `Retry-After`
  - Sinh OTP 6 số (TTL 5 phút); hash peppered; không log raw OTP; `OTP_DEV_REVEAL` cho local
  - Migrate `schema.sql` khi start; unit tests phone/rate-limit/handler
  - Mark `- [DONE] T1.1.1` trên PRD
- **Workdocs:** `workdocs_otp_request_ratelimit_02082026/`
- **Liên quan:** Sprint 1 / US-1.1 / T1.1.1

## [2026-08-02] Makefile / scripts chạy dev (T9.2.3)

- **Loại:** feature
- **Phạm vi:** `Makefile`, `scripts/`, `README.md`
- **Tóm tắt:** Hoàn thiện DX local Sprint 0 — Makefile targets (nats, services, flutter, health) và mirror PowerShell `scripts/dev.ps1` cho Windows không có GNU Make.
- **Chi tiết:**
  - `Makefile`: `help` mặc định, `nats` (up+wait+init), compose, per-service `go run`, `build`/`test`, flutter helpers
  - `scripts/dev.ps1`: cùng tên lệnh để chạy trên PowerShell
  - README hướng dẫn Make + PS1; mark `- [DONE] T9.2.3` trên PRD
- **Workdocs:** `workdocs_makefile_dev_scripts_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.3

## [2026-08-02] NATS JetStream local (T9.2.2)

- **Loại:** feature
- **Phạm vi:** `deploy/`, `pkg/natsx`, `cmd/nats-init`
- **Tóm tắt:** Bật NATS JetStream local với config + volume + healthcheck, bootstrap stream theo bounded context (architecture §5.1), và CLI/`go test` để verify — nền event bus cho Sprint 0.
- **Chi tiết:**
  - `deploy/nats.conf` + compose healthcheck / `nats-data` volume; services `depends_on` NATS healthy
  - `pkg/natsx`: `ConnectJS`, `EnsureStreams`, `PingJS` + embedded JetStream test
  - `cmd/nats-init` đảm bảo 6 stream (AUTH…BILLING) và in trạng thái
  - Mark `- [DONE] T9.2.2` trong `docs/prd.md`
- **Workdocs:** `workdocs_nats_jetstream_local_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.2

## [2026-08-02] Accept T9.2.1 monorepo layout (Sprint 0)

- **Loại:** chore
- **Phạm vi:** `docs/prd.md`, monorepo root
- **Tóm tắt:** Xác nhận scaffold hiện có đã đủ AC của T9.2.1 (cây `apps/mobile` + `services/*` + `pkg/` + `deploy/` khớp architecture §2.1; `go build ./services/...` OK) và đánh dấu task DONE trên PRD — không scaffold lại.
- **Chi tiết:**
  - Verify layout: 8 Go services, Flutter `apps/mobile`, shared `pkg`, `deploy/docker-compose.yml`
  - Mark `- [DONE] T9.2.1` trong `docs/prd.md`
  - Bổ sung workdocs acceptance note
- **Workdocs:** `workdocs_scaffold_monorepo_02082026/`
- **Liên quan:** Sprint 0 / US-9.2 / T9.2.1

## [2026-08-02] Scaffold monorepo boilerplate theo architecture

- **Loại:** feature
- **Phạm vi:** `apps/mobile`, `services/*`, `pkg/*`, `deploy/`
- **Tóm tắt:** Dựng skeleton monorepo khớp `docs/architecture.md` để Sprint 0 có gateway hello/healthz, NATS compose, Flutter home CTA, và stub API/schema cho từng bounded context.
- **Chi tiết:**
  - Go module `gas-tam-de` + shared `pkg/{config,httpx,sqlite,events,natsx}` (HTTP: Chi)
  - Stub 8 services (gateway + auth/catalog/geo/order/inventory/billing/report) kèm `schema.sql` từ architecture §6
  - `deploy/docker-compose.yml` (NATS JetStream + services), Dockerfile, `.env.example`
  - Flutter `apps/mobile`: brand Gas Tam Đệ + CTA Đặt giao gas (placeholder order flow)
  - Root `README.md`, `Makefile`, `.gitignore`
- **Workdocs:** `workdocs_scaffold_monorepo_02082026/`
- **Liên quan:** Sprint 0 / architecture §2.1

## [2026-08-02] Đa nền tảng Web + Android + iOS song song

- **Loại:** docs
- **Phạm vi:** `docs/prd.md`, `docs/architecture.md`
- **Tóm tắt:** Đổi kế hoạch từ ưu tiên Android sang phát triển Flutter Web, Android và iOS cùng lúc; bổ sung chiến lược test bằng Web/emulator/CI macOS khi không có máy thật.
- **Chi tiết:**
  - PRD: §1.2 target platforms, cập nhật MoSCoW/NFR/DoD/sprint/rủi ro
  - Architecture: §8.4–8.5 multi-platform matrix; CI build iOS no-codesign; deploy IPA
  - iOS không còn “sau MVP”; store publish vẫn out of scope
- **Workdocs:** `workdocs_multiplatform_web_android_ios_02082026/`
- **Liên quan:** Sprint 0 / T9.2.4–T9.2.5

## [2026-08-02] Skill change-workdocs + quy trình CHANGESLOG/workdocs

- **Loại:** chore
- **Phạm vi:** `.cursor/skills/change-workdocs`, root docs process
- **Tóm tắt:** Thêm Agent Skill bắt buộc ghi mọi change vào `CHANGESLOG.md` và tạo thư mục `workdocs_<mo-ta>_<ddmmyyyy>` khi implement chức năng.
- **Chi tiết:**
  - Tạo skill `change-workdocs` kèm templates changelog/workdoc
  - Seed `CHANGESLOG.md` tại root
  - Ghi nhận lịch sử tài liệu PRD/architecture đã có
- **Workdocs:** `workdocs_skill_change_workdocs_02082026/`
- **Liên quan:** n/a

## [2026-08-02] Tài liệu khởi tạo PRD + Architecture Gas Tam Đệ

- **Loại:** docs
- **Phạm vi:** `docs/`
- **Tóm tắt:** Viết PRD (requirement, epic/story/task, sprint) và architecture (microservice, EDA, schema, security, deploy/monorepo).
- **Chi tiết:**
  - Thêm `docs/prd.md`
  - Thêm `docs/architecture.md` (gồm §9 Deploy & Repo strategy)
- **Workdocs:** `workdocs_docs_prd_architecture_02082026/`
- **Liên quan:** Sprint 0 / nền tảng tài liệu
