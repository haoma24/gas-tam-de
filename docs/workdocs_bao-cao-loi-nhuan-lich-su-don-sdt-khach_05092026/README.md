# Workdocs — Lợi nhuận đúng, lịch sử đơn, thống kê khách, SĐT khách cho admin

**Ngày:** 05/09/2026
**Nhánh:** `refactor/ui-minimalism`
**Yêu cầu gốc (chủ shop):**

1. Báo cáo tính lợi nhuận sai — lợi nhuận đang bằng doanh thu, phải là `giá bán − giá nhập`.
2. Đơn đã hoàn tất không xem lại được. Cần bộ lọc trạng thái ở tab «Đơn» (Chưa giao / Đã giao / Bị hủy…).
3. Báo cáo cần thống kê theo từng khách hàng — khách nào đã đặt bao nhiêu đơn.
4. Admin đang bị che số điện thoại khách — không gọi được cho khách. Đây là lỗi nghiêm trọng.

---

## 1. Chẩn đoán (vì sao đang sai)

### 1.1 Lợi nhuận = doanh thu

Công thức trong `services/report-service/profit.go` **đã đúng**:
`profit_vnd = revenue_vnd − cogs_vnd`.

Lỗi nằm ở **dữ liệu đầu vào**: `cogs_vnd` luôn bằng `0`.

Chuỗi đứt ở đây:

| Bước | File | Trạng thái |
|---|---|---|
| Kho lưu giá nhập | `inventory-service/schema.sql` → `stock_items.cost_price` | ✅ có |
| Kho chốt giá vốn khi trừ kho | `inventory-service/reserve.go` → `stock_movements.unit_cost` | ✅ có |
| Đơn lưu giá vốn từng dòng | `order-service/schema.sql` → `order_items` | ❌ **không có cột `unit_cost`** |
| Sự kiện `order.completed` mang giá vốn | `order-service/order_events.go` | ❌ **payload chỉ có `unit_price`** |
| Báo cáo cộng COGS | `report-service/order_stats.go:parseOrderCompletedItems` | ✅ đọc `m["unit_cost"]` — nhưng luôn `nil` → `0` |

⇒ `cogs = 0` ⇒ `profit = revenue − 0 = revenue`. Đúng như chủ shop thấy.

**Điểm quan trọng:** tồn kho bị trừ ở lúc **đặt đơn** (`reserve.go`, `POST /v1/internal/stock/reserve`),
không phải lúc hoàn tất. Nên bản chụp giá vốn (`unit_cost`) đã tồn tại sẵn trong `stock_movements`
ngay từ lúc đặt — chỉ là order-service không nhận lại giá trị đó. Lấy nó về ở đúng chỗ này
là đường ngắn nhất, không cần thêm sự kiện NATS mới.

### 1.2 Không xem lại được đơn đã xong

`GET /v1/admin/orders?status=` **đã** nhận `PENDING|COMPLETED|CANCELLED`
(`list_orders.go:parseAdminOrderStatusFilter`), nhưng:

- Không có giá trị `ALL`.
- Flutter `OrderApi.listAdminOrders()` được gọi ở `admin_orders_page.dart:125` **không truyền `status`**
  ⇒ luôn mặc định `PENDING`. Đơn hoàn tất biến mất khỏi màn hình, không có đường nào xem lại.
- Response không có `completed_at`, `cancelled_at`, `payment_type`, `amount_paid`
  ⇒ dù lọc được cũng không thấy đơn đã thu bao nhiêu, nợ bao nhiêu.
- Sắp xếp cứng `created_at ASC` (FIFO) — đúng cho hàng chờ, sai cho lịch sử (phải mới nhất trước).

### 1.3 Không có thống kê theo khách

Không tồn tại endpoint nào. `report-service` không giữ dữ liệu khách (chỉ `daily_stats` +
`customer_debt_balances` khoá theo `customer_key`). Dữ liệu đơn theo khách nằm ở
`order-service.orders(user_id, customer_name, …)`.

### 1.4 Admin không thấy SĐT khách

Chuỗi che số là **cố ý** theo thiết kế cũ, và che ở **mọi tầng**:

- JWT chỉ mang `phone_masked` (`auth-service/tokens.go:AccessClaims`).
- Gateway chỉ gắn header `X-Phone-Masked` (`api-gateway/rbac.go:40`).
- `order-service` nhận đúng header đó, `orders` **chỉ có cột `phone_masked`**, không có số thật
  (`phone_hash` là chuỗi giả `"uid:<user_id>"`, không giải ngược được).
- `pii.go:ensurePhoneMasked` còn chuẩn hoá lại lần nữa để "số đầy đủ không bao giờ lọt ra".

⇒ Số thật **chưa từng được lưu** trong `order.db`. Chỉ `auth-service` có, ở dạng mã hoá
(`users.phone_e164_enc` / `contact_phone_e164_enc`, AES-GCM, `phone_crypto.go`).
Không có cách nào sửa việc này chỉ ở tầng UI — phải mở một đường lấy số thật từ auth-service.

---

## 2. Quyết định thiết kế

| Quyết định | Chọn | Vì sao không chọn cách khác |
|---|---|---|
| Lấy giá vốn vào đơn | Cho `POST /v1/internal/stock/reserve` **trả về** `unit_cost` từng dòng; order-service lưu vào `order_items.unit_cost` | Sự kiện NATS mới (`inventory.cogs.recorded`) tốn thêm stream + consumer + bảng idempotent, trong khi giá vốn đã có sẵn ngay trong lời gọi đồng bộ đang tồn tại |
| Chốt giá vốn lúc nào | Lúc **đặt đơn** (khi kho thực trừ) | Ledger kho đã ghi OUT ở đúng thời điểm đó; chốt lúc hoàn tất sẽ lệch với `stock_movements` nếu giá nhập đổi giữa chừng |
| Thống kê theo khách | Query gộp trong `order-service` (`GET /v1/admin/orders/customers`) | Đẩy qua NATS vào report-service cần bảng đọc mới + backfill; số liệu khách là truy vấn `GROUP BY` một bảng, không cần read-model |
| Lấy SĐT thật | `auth-service` mở `POST /v1/internal/users/phones` (batch, **không** qua gateway); order-service lưu snapshot vào `orders.customer_phone` | Nhét số thật vào JWT làm số nằm trong localStorage trình duyệt của mọi phiên — rộng hơn mức cần. Endpoint nội bộ giữ auth là nguồn sự thật duy nhất và cho phép **vá ngược đơn cũ** |
| Đơn cũ (đã có trong DB) | Vá ngược lười: admin list/detail thấy `customer_phone` rỗng thì gọi auth một lần theo lô rồi `UPDATE` luôn | Không cần script migration thủ công trên VPS |
| Ai thấy số thật | **Chỉ** endpoint `/v1/admin/*`. `GET /v1/orders/me` của khách vẫn masked | Giữ nguyên nguyên tắc PII cho phía khách |

---

## 3. Các phần thực hiện

- [x] **Phần 1 — Lợi nhuận đúng (COGS)**
- [x] **Phần 2 — Lọc đơn theo trạng thái + xem lại đơn đã xong**
- [x] **Phần 3 — Thống kê theo khách hàng**
- [x] **Phần 4 — SĐT thật cho admin**
- [x] **Phần 5 — Test, codemap, changeslog**

(Ô đánh dấu được cập nhật khi từng phần xong — chi tiết ở mục 4.)

---

## 4. Nhật ký thực hiện

### Phần 1 — Lợi nhuận đúng (COGS) — ✅ xong

**Backend**

| File | Thay đổi |
|---|---|
| `services/inventory-service/reserve.go` | `POST /v1/internal/stock/reserve` trả thêm `items: [{product_id, unit_cost}]` — giá vốn vừa chốt vào `stock_movements`. `release` giữ nguyên hình dạng cũ (`items` rỗng). |
| `services/order-service/schema.sql` | Thêm `order_items.unit_cost INTEGER NOT NULL DEFAULT 0` (bảng mới) |
| `services/order-service/main.go` | Thêm `ensureColumn` (copy khuôn `auth-service`) → vá `order_items.unit_cost` cho DB đã deploy; migration idempotent, không có công cụ migration trong repo |
| `services/order-service/inventory_client.go` | `Reserve` đổi chữ ký → trả `map[productID]unitCost`; parse `items[].unit_cost` từ response |
| `services/order-service/create_order.go` | Sau khi reserve thành công → `UPDATE order_items SET unit_cost = ?`; best-effort, lỗi chỉ log (đơn đã commit) |
| `services/order-service/complete_order.go` + `order_events.go` | `orderItemView` thêm `UnitCost`; payload `order.completed` thêm `unit_cost` mỗi dòng |
| `services/order-service/list_orders.go` | `loadOrderItems` đọc thêm `unit_cost` |

**Kết quả:** `report-service/order_stats.go:parseOrderCompletedItems` vốn đã đọc `m["unit_cost"]`
— không phải sửa gì bên report. `cogs_vnd` bắt đầu có giá trị thật, `profit_vnd = revenue − cogs`.

**Cảnh báo dữ liệu:** đơn đặt **trước** thay đổi này có `unit_cost = 0` ⇒ lợi nhuận của chúng
vẫn bằng doanh thu. Và nếu admin chưa nhập giá nhập trong tab «Kho» thì `cost_price = 0`
⇒ COGS vẫn 0. Tab «Báo cáo» nay hiện cảnh báo rõ khi `cogs = 0` mà `revenue > 0`
(xem Phần 3) thay vì im lặng báo lãi ảo.

**Flutter:** `OrderItemView.unitCost` (`order_models.dart`) + `unit_cost` trong JSON.

### Phần 2 — Lọc đơn theo trạng thái + xem lại đơn đã xong — ✅ xong

**Backend — `services/order-service/list_orders.go`**

- `parseAdminOrderStatusFilter` nhận thêm `ALL` (rỗng vẫn = `PENDING`, giữ hành vi Order Desk cũ).
- Query dựng động: `ALL` bỏ mệnh đề `WHERE status = ?`.
- **Sắp xếp theo ngữ cảnh:** `PENDING` → `created_at ASC` (FIFO, hàng chờ);
  mọi trạng thái khác → `created_at DESC` (lịch sử, mới nhất trước).
  `stt` chỉ đánh số khi FIFO — với lịch sử số thứ tự FIFO vô nghĩa.
- `orderRow` + response admin thêm `completed_at`, `cancelled_at`, `payment_type`, `amount_paid`.
- `GET /v1/admin/orders/{id}` dùng chung view admin ⇒ mở lại đơn đã hoàn tất thấy đủ thông tin
  thanh toán và công nợ.

**Flutter — `admin_orders_page.dart`**

- Hàng chip lọc dính đầu danh sách: **Chờ giao · Đã giao · Đã hủy · Tất cả**
  (`AdminOrderFilter` trong `order_models.dart`).
- Chỉ tự động poll 10s khi đang ở «Chờ giao» — lịch sử không cần tự làm mới, và
  thông báo/đọc giọng đơn mới cũng chỉ chạy ở tab đó (trước đây poll luôn kéo về PENDING).
- Tile hiện badge trạng thái khi không phải hàng chờ; badge thời gian chờ chỉ hiện cho đơn PENDING.
- Trang chi tiết: khối «Thanh toán» (loại, đã thu, còn nợ), mốc «Hoàn tất lúc» / «Hủy lúc»;
  nút «Hoàn tất» tự ẩn với đơn không còn PENDING.

### Phần 3 — Thống kê theo khách hàng — ✅ xong

**Backend — `services/order-service/customer_stats.go` (mới)**

`GET /v1/admin/orders/customers?from=&to=&limit=` (mặc định 30 ngày gần nhất, `limit` 200):

```json
{ "from":"2026-08-06","to":"2026-09-05","count":12,
  "customers":[{ "user_id":"…","customer_name":"…","customer_phone":"0909…","phone_masked":"090***7020",
    "orders_total":9,"orders_completed":7,"orders_cancelled":1,"orders_pending":1,
    "spent_vnd":2450000,"paid_vnd":2100000,"debt_vnd":350000,
    "first_order_at":"…","last_order_at":"…","address_text":"…" }] }
```

- Một câu `GROUP BY user_id` trên `orders` — không thêm bảng, không thêm sự kiện.
- Doanh thu/nợ chỉ cộng đơn `COMPLETED` (đúng luật «chỉ tính khi hoàn tất» của kiến trúc).
- Sắp xếp: chi tiêu giảm dần, rồi số đơn giảm dần.
- Chi tiết bộ lọc ngày là **ngày VN (UTC+7)**, khớp với `daily_stats.day` của report-service
  để hai con số trên cùng màn hình không lệch múi giờ.
- Route đặt **trước** `/v1/admin/orders/{id}` — chi ưu tiên route tĩnh nên không đụng nhau,
  và gateway đã có sẵn `admin.Handle("/admin/orders/*", orderProxy)` nên không cần sửa phân quyền.

**Flutter**

- `features/order/customer_stats_models.dart` (mới). Không tạo `customer_stats_api.dart` riêng —
  endpoint nằm dưới `/v1/admin/orders/*` nên `OrderApi.listCustomerStats()` là chỗ đúng của nó.
- `dashboard_models.dart`: thêm `rangeForPeriod()` — cùng kỳ với summary, đổi sang dạng `from`/`to`.
- Tab «Báo cáo» thêm mục «Khách hàng» (top `kCustomerPreviewCount` = 10 + nút «Xem tất cả»):
  tên, SĐT bấm gọi được, số đơn (hoàn tất / chờ / hủy), tổng chi, công nợ.
- Thêm cảnh báo COGS (`_CogsWarning`): khi `cogs_vnd == 0` và `revenue_vnd > 0`, thay dòng
  chú thích cũ bằng khối cảnh báo «Lợi nhuận đang bằng doanh thu — vào tab Kho nhập giá nhập».

### Phần 4 — SĐT thật cho admin — ✅ xong

**auth-service — `internal_users.go` (mới)**

`POST /v1/internal/users/phones` — body `{"user_ids":["…"]}` (tối đa 500),
trả `{"phones":{"<user_id>":"0909…"}}`. Giải mã `COALESCE(contact_phone_e164_enc, phone_e164_enc)`
bằng khoá sẵn có (`phone_crypto.go`), trả về dạng nội địa `0…` cho tiện bấm gọi.
**Không** mount ở gateway — cùng lớp với `/v1/internal/stock/*` và `/v1/internal/payments`.
User không có số (tài khoản Google chưa thêm SĐT liên hệ) bị bỏ khỏi map, không báo lỗi.

**order-service**

| File | Thay đổi |
|---|---|
| `schema.sql` + `main.go` | Thêm cột `orders.customer_phone TEXT NOT NULL DEFAULT ''` (+ `ensureColumn` cho DB cũ) |
| `auth_client.go` (mới) | `PhonesByUserID(ctx, ids)` → gọi endpoint nội bộ trên; `AUTH_SERVICE_URL` (mặc định `http://127.0.0.1:8081`) |
| `create_order.go` | `lookupCustomerPhone` lấy số thật lúc đặt đơn, lưu vào `orders.customer_phone`. Best-effort: auth chết thì đơn **vẫn đặt được**, số vá sau |
| `auth_client.go` → `fillCustomerPhones` | Vá ngược lười ở `handleListAdminOrders` / `handleGetAdminOrder`: gom `user_id` của các đơn còn thiếu số → **một** lời gọi theo lô → `UPDATE orders` để lần sau khỏi gọi lại |
| `customer_stats.go` → `fillStatPhones` | Tương tự cho danh sách khách (không ghi lại — dòng đơn tự có khi Order Desk liệt kê) |
| `pii.go` | Thêm `adminOrderView` — giống `customerOrderView` nhưng có `customer_phone`; `displayPhone` đổi `+84…` → `0…`. `customerOrderView` **không đổi**: khách vẫn chỉ thấy masked |
| `complete_order.go` | Response complete thêm `customer_phone` |
| `deploy/docker-compose.yml` | order-service nhận `AUTH_SERVICE_URL: http://auth-service:8081` |

**Không thêm** `auth` vào `/readyz` của order-service, cũng **không** thêm `depends_on: auth-service`:
lời gọi này là best-effort, order-service vẫn sẵn sàng khi auth chết. Ràng thêm phụ thuộc cứng
chính là cái bẫy «cả stack hỏng vì một service chậm» mà `CLAUDE.md` cảnh báo.
URL đích được in trong log `upstream urls` lúc khởi động để chẩn đoán.

**Gateway:** không đổi. `/v1/internal/*` không đi qua gateway (giống stock/payments).

**Flutter**

- `AdminOrder.customerPhone`, `.displayPhone` (số thật → masked → `—`), `.dialablePhone`,
  `.debt`, `.completedAt`, `.cancelledAt`, `.paymentType`, `.amountPaid`.
- Tile Order Desk: hiện **số thật**, fallback về masked rồi `—`.
- Trang chi tiết: dòng «SĐT» là `_PhoneField` — chạm để gọi (`tel:`), giữ để copy;
  không có số thật thì không hiện nút gọi mà nói rõ «Khách chưa có SĐT liên hệ trong hồ sơ».
- `core/phone_link.dart` (mới): `telDigits` / `telUri` / `dialPhone()` bọc `url_launcher`,
  cùng khuôn `navigation_link.dart`. `telDigits` **từ chối** chuỗi chứa `*` — số đã che không
  phải số gọi được.

### Phần 5 — Test, codemap, changeslog — ✅ xong

Xem mục 5.

---

## 5. Test

**Go — mới**

| File | Bảo vệ |
|---|---|
| `services/order-service/customer_stats_test.go` | Gộp theo khách: đếm đơn theo trạng thái, chỉ cộng tiền đơn COMPLETED, **lọc theo ngày VN** (18:00Z là hôm sau), «tên mới nhất thắng», 400 cho khoảng ngày sai |
| `services/order-service/admin_orders_filter_test.go` | `status=ALL`, DESC cho lịch sử / ASC + `stt` cho PENDING, 400 cho status lạ, chi tiết đơn hiện thanh toán |
| `services/order-service/customer_phone_test.go` | Số thật ở view admin, **API của khách không lộ số**, vá ngược đúng **một** lời gọi theo lô rồi ghi lại, auth chết vẫn liệt kê được đơn, `displayPhone` |
| `services/order-service/order_cogs_test.go` | `unit_cost` từ reserve → `order_items` → payload `order.completed`; inventory bản cũ (không trả `items`) vẫn đặt đơn được |
| `services/auth-service/internal_users_test.go` | Giải mã đúng số + đổi sang `0…`, ưu tiên contact phone, bỏ qua user không có số, chặn lô rỗng / quá 500 id |
| `services/report-service/profit_test.go` | (bổ sung) có `unit_cost` thì lợi nhuận khác doanh thu; không có thì vẫn chạy, chỉ là `cogs = 0` |

**Flutter — mới**

| File | Bảo vệ |
|---|---|
| `apps/mobile/test/admin_order_filter_test.dart` | Chip lọc đổi `status` gửi lên API (widget test thật, fake Dio), chỉ «Chưa giao» là hàng chờ tự làm mới, badge trạng thái ở lịch sử |
| `apps/mobile/test/customer_stats_test.dart` | Parse JSON thống kê khách, fallback tên / SĐT, `rangeForPeriod` |
| `apps/mobile/test/admin_order_phone_test.dart` | Số thật → masked → `—`, `tel:` từ chối số đã che, công nợ đơn, `lineProfit` |

`AdminOrdersPage` chạy `Timer.periodic` nên `pumpAndSettle` sẽ treo — widget test bơm frame
thủ công rồi tháo cây widget để `dispose()` huỷ timer (xem `_settle` / `_teardown` trong file test).

**Chạy:** `make test` · `cd apps/mobile && flutter analyze && flutter test`

Kết quả lần này: `go test ./...` xanh toàn bộ; `flutter test` 73/73 xanh;
`flutter analyze` còn đúng 7 info deprecation vốn có từ trước (`RadioListTile.groupValue`,
`dangling_library_doc_comments`) — không phát sinh cái mới.

## 6. Rủi ro & việc còn lại

- **Đơn cũ không có giá vốn.** `order_items.unit_cost = 0` cho mọi đơn đặt trước thay đổi này;
  báo cáo kỳ cũ vẫn hiện lợi nhuận = doanh thu. Không vá ngược được (giá nhập tại thời điểm đó
  không còn ở đâu ngoài `stock_movements`, và chỉ khớp được theo `ref_id`). Chấp nhận:
  số liệu đúng dần từ ngày deploy.
- **Chưa nhập giá nhập.** Nếu `stock_items.cost_price = 0` thì COGS vẫn 0. Đã có cảnh báo trên UI,
  nhưng chủ shop cần vào tab «Kho» nhập giá nhập cho từng sản phẩm thì lợi nhuận mới có nghĩa.
- **SĐT khách đăng nhập bằng Google** chỉ có nếu khách đã tự thêm SĐT liên hệ ở «Hồ sơ».
  Chưa có thì admin thấy `—`. Có thể cân nhắc bắt buộc nhập SĐT khi đặt đơn — ngoài phạm vi lần này.
- `orders.customer_phone` là PII dạng thô trong `order.db`. File này nằm trên volume của VPS
  giống `auth.db`; không thêm bề mặt lộ mới nào ngoài các route `/v1/admin/*` vốn đã chặn bằng RBAC.
