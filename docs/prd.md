# Gas Tam Đệ — Product Requirements Document (PRD)

> Tài liệu tổng hợp: phân tích requirement, PRD, Epic → Story → Task, và Sprint planning.  
> Kiến trúc kỹ thuật: xem [architecture.md](./architecture.md).

---

## 1. Tổng quan sản phẩm

| Mục | Nội dung |
|-----|----------|
| Tên sản phẩm | **Gas Tam Đệ** |
| Loại hình | Website + Mobile app (**Web + Android + iOS** phát triển song song, một codebase Flutter) |
| Đối tượng | Cửa hàng gas gia đình: bán gas dân dụng, bếp gas, phụ kiện |
| Vấn đề giải quyết | Khách đặt giao gas nhanh; chủ cửa hàng theo dõi và giao theo thứ tự, quản lý phí ship, tồn kho, công nợ, lợi nhuận |
| Chức năng mũi nhọn | **Đặt giao gas** — CTA dễ nhìn thấy ngay khi mở web/app |

### 1.1 Mục tiêu kinh doanh

1. Giảm thời gian đặt hàng qua điện thoại / tin nhắn rời rạc.
2. Chuẩn hóa quy trình giao: FIFO (đơn cũ nhất trước), có địa chỉ + khoảng cách + dẫn đường.
3. Minh bạch phí giao hàng theo khoảng cách (admin cấu hình).
4. Theo dõi thanh toán (đủ / một phần / nợ) và tồn kho để tính lợi nhuận.

### 1.2 Target platforms (đã chốt)

| Platform | MVP | Ghi chú test |
|----------|-----|--------------|
| **Web** | Có | Chrome/Edge — kênh dev/UAT chính khi chưa có máy thật |
| **Android** | Có | Android Emulator (không bắt buộc máy thật) |
| **iOS** | Có (cùng sprint với Android) | iOS Simulator cần macOS; nếu không có Mac dùng CI macOS (GitHub Actions / Codemagic) |

Không trì hoãn iOS sang sau MVP: mọi feature Flutter phải giữ tương thích 3 target từ đầu (tránh plugin/Web-only hoặc Android-only nếu chưa có fallback).

**Ngoài scope MVP:** nộp / duyệt store (Play Store, App Store) — có thể phân phối nội bộ (APK / TestFlight / ad hoc) trước.

### 1.3 Out of scope (MVP)

- Thanh toán online (VNPay, MoMo, thẻ).
- Chat realtime khách–cửa hàng.
- Đa chi nhánh / đa cửa hàng.
- Phát hành chính thức lên App Store / Play Store (build iOS/Android vẫn nằm trong scope).
- Chương trình khuyến mãi phức tạp, loyalty điểm thưởng.
- Định tuyến tối ưu nhiều điểm giao trong một chuyến (route optimization).

---

## 2. Phân tích requirement

### 2.1 Personas

#### P1 — Khách hàng dân dụng

- Cần thay bình gas / mua phụ kiện nhanh, thường gấp.
- Không muốn form dài; cần biết có giao được tới nhà hay không.
- Cung cấp: họ tên, SĐT (xác thực OTP), địa chỉ (GPS hoặc search).

#### P2 — Admin / Chủ cửa hàng (CCH)

- Nhận đơn liên tục trong ngày; cần danh sách rõ, sắp xếp cũ → mới.
- Ra đường giao: cần map dẫn đường từ vị trí hiện tại tới điểm giao.
- Sau giao: đánh dấu hoàn tất + trạng thái thu tiền.
- Setup: sản phẩm, bán kính giao, bậc phí ship, nhập/xuất tồn, xem dashboard.

### 2.2 Jobs-to-be-done

| Actor | Job | Kết quả mong muốn |
|-------|-----|-------------------|
| Khách | Khi hết gas, tôi muốn đặt giao trong vài phút | Đơn được xác nhận, biết trong/ngoài phạm vi |
| Admin | Khi có đơn mới, tôi muốn biết ai cần giao trước | Danh sách FIFO + khoảng cách + địa chỉ |
| Admin | Khi đang trên đường, tôi muốn dẫn đường nhanh | Mở map navigation tới tọa độ đơn |
| Admin | Khi giao xong, tôi muốn ghi nhận tiền đã thu | Đủ / một phần / nợ → công nợ cập nhật |
| Admin | Cuối ngày/tháng, tôi muốn biết lãi lỗ | Dashboard doanh thu, công nợ, lợi nhuận, tồn |

### 2.3 MoSCoW

#### Must have

- CTA “Đặt giao gas” nổi bật trên màn hình đầu.
- Chọn sản phẩm (admin setup).
- Nhập họ tên, SĐT + xác thực OTP.
- Địa chỉ: xin quyền vị trí **hoặc** search vị trí (gợi ý giống Google Maps).
- Kiểm tra bán kính giao (mặc định 10 km, admin chỉnh); ngoài phạm vi → thông báo rõ.
- Phí giao: bật/tắt + bậc theo km (admin).
- Admin: danh sách đơn (STT, Tên, SĐT, địa chỉ, khoảng cách, thời gian đặt) — sort cũ nhất trước.
- Admin: xác nhận giao hoàn tất + option thanh toán (đủ / một phần / nợ).
- Admin: dẫn đường tới điểm giao.
- Tồn kho cơ bản: nhập/xuất, giá nhập, giá bán theo mã SP → phục vụ tính lợi nhuận.
- Bảo mật: cô lập dữ liệu giữa user; mask PII trên API response.
- Flutter chạy được trên **Web + Android + iOS** (cùng feature set MVP; test bằng emulator/simulator/web).

#### Should have

- Dashboard nhỏ: doanh thu, công nợ, lợi nhuận, tồn.
- Lịch sử đơn của khách (chỉ đơn của chính họ).
- Audit log hành động admin quan trọng.
- CI build kiểm tra `flutter build` cho web/android/ios (ios trên runner macOS).

#### Could have

- Push notification đơn mới cho admin.
- Ước tính thời gian giao.
- In hóa đơn / chia sẻ đơn qua Zalo.
- Phát hành store công khai.

#### Won’t have (giai đoạn này)

- Thanh toán cổng online.
- Multi-tenant / nhiều cửa hàng.
- AI chatbot.

### 2.4 Ràng buộc nghiệp vụ

| Ràng buộc | Chi tiết |
|-----------|----------|
| Bán kính giao | Mặc định **10 km**; so với tọa độ cửa hàng đã cấu hình; admin được chỉnh |
| Phí giao (ví dụ) | Bật/tắt toàn cục; ví dụ: 10k khi &lt; 5 km; 20k khi ≥ 5 km và ≤ 10 km; có thể cấu hình thêm bậc &gt; 10 km nếu nới bán kính |
| Thứ tự đơn admin | **Oldest first** (thời gian đặt tăng dần) trong các đơn chưa hoàn tất |
| OTP | Bắt buộc trước khi đặt đơn thành công |
| Sản phẩm | Chỉ đặt được SP đang `active` và còn đủ tồn (nếu inventory bật trừ tồn) |

### 2.5 Giả định

- Một cửa hàng duy nhất (Gas Tam Đệ).
- Tiền tệ: VND, số nguyên.
- Admin đăng nhập bằng tài khoản riêng (không OTP khách); khách dùng SĐT + OTP.
- Deploy quy mô nhỏ (VPS / máy tại quán); không cần Kubernetes.

---

## 3. PRD chi tiết

### 3.1 User flow — Khách hàng (Đặt giao gas)

```text
[Home] CTA "Đặt giao gas"
    → Chọn sản phẩm (+ số lượng)
    → Nhập SĐT → Gửi OTP → Nhập OTP → Session khách
    → Nhập họ tên
    → Địa chỉ: [Dùng vị trí hiện tại] hoặc [Search + chọn gợi ý]
    → Hệ thống tính khoảng cách vs cửa hàng
         ├─ Ngoài phạm vi → Thông báo "Ngoài phạm vi giao hàng" (không cho đặt)
         └─ Trong phạm vi → Hiển thị phí ship (nếu bật) + tổng tiền
    → Xác nhận đặt hàng
    → Màn hình thành công (mã đơn, tóm tắt; SĐT mask nếu hiển thị lại)
```

**Yêu cầu UX**

- Màn hình đầu: một composition rõ; brand **Gas Tam Đệ** + CTA đặt gas là tín hiệu chính.
- Không bắt buộc đăng ký form dài trước khi thấy CTA.
- Lỗi ngoài phạm vi: message rõ, không silent fail.

### 3.2 User flow — Admin / CCH

```text
[Đăng nhập Admin]
    → Order Desk: danh sách đơn chờ / đang giao
         cột: STT | Tên | SĐT (mask hoặc full theo role) | Địa chỉ | Khoảng cách | Thời gian đặt
         sort: thời gian đặt ASC (cũ nhất trước)
    → Mở chi tiết đơn
         → [Dẫn đường] mở map từ vị trí hiện tại → điểm giao
         → [Hoàn tất giao]
              → Chọn thanh toán: Đủ | Một phần (nhập số đã thu) | Nợ tất cả
              → Xác nhận → đơn = COMPLETED; cập nhật công nợ / doanh thu
    → Setup: Sản phẩm | Phí giao | Bán kính & vị trí CH | Tồn kho | Dashboard
```

### 3.3 Chức năng theo module

#### M1 — Đặt hàng & Geo

- Tạo đơn với: `customer_name`, `phone` (đã verify), `address_text`, `lat`, `lng`, `items[]`, `distance_km`, `delivery_fee`, `subtotal`, `total`.
- Từ chối nếu `distance_km > max_radius_km`.
- Trạng thái đơn tối thiểu: `PENDING` → `COMPLETED` | `CANCELLED` (cancel admin optional MVP).

#### M2 — OTP & phiên

- Gửi OTP tới SĐT; giới hạn số lần / cooldown.
- Verify OTP → JWT access (ngắn) + refresh.
- Chỉ đặt hàng khi session hợp lệ và SĐT khớp.

#### M3 — Catalog

- Admin CRUD sản phẩm: mã, tên, mô tả ngắn, giá bán, đơn vị, ảnh (optional), active/inactive.
- Khách chỉ thấy sản phẩm `active`.

#### M4 — Phí giao hàng

- Toggle `delivery_fee_enabled`.
- Danh sách bậc: `min_km`, `max_km` (nullable = ∞), `fee_vnd`.
- Khi đặt: nếu tắt → phí = 0; nếu bật → chọn bậc khớp `distance_km`.

#### M5 — Order desk & navigation

- List FIFO; filter theo trạng thái.
- Deep-link / intent Google Maps (hoặc OSM) navigation: origin = vị trí thiết bị admin, destination = lat/lng đơn.

#### M6 — Thanh toán & công nợ

- Khi hoàn tất:
  - `FULL`: paid = total, debt = 0
  - `PARTIAL`: nhập `amount_paid` (0 &lt; paid &lt; total), debt = total − paid
  - `UNPAID`: paid = 0, debt = total
- Công nợ gắn theo khách (SĐT đã verify / user_id).

#### M7 — Tồn kho

- Theo `product_sku` / `product_id`: số tồn, giá nhập hiện tại (hoặc trung bình gia quyền đơn giản), giá bán (đồng bộ catalog).
- Phiếu nhập (`IN`), xuất (`OUT` — gồm xuất khi giao hoặc xuất tay).
- Lợi nhuận ước tính: doanh thu bán − giá vốn hàng bán (COGS) − (optional) không trừ phí ship vào COGS; phí ship có thể tách doanh thu phụ.

#### M8 — Dashboard

- Doanh thu (theo ngày/tuần/tháng).
- Tổng công nợ còn lại.
- Lợi nhuận ước tính (doanh thu − COGS).
- Tồn kho theo SP (cảnh báo thấp nếu cấu hình `reorder_level`).

### 3.4 Non-functional requirements (NFR)

| ID | Hạng mục | Yêu cầu |
|----|----------|---------|
| NFR-1 | Bảo mật | TLS; JWT; RBAC `customer` \| `admin`; rate limit OTP/đặt hàng |
| NFR-2 | Privacy | Response mask SĐT; không IDOR giữa khách; admin mới xem full list |
| NFR-3 | Hiệu năng | Place order p95 &lt; 2s trên mạng tốt (không tính SMS OTP) |
| NFR-4 | Sẵn sàng | MVP chấp nhận single-node; backup file SQLite định kỳ |
| NFR-5 | Khả dụng | Flutter Web + Android + iOS; UI tiếng Việt; responsive web |
| NFR-6 | Quan sát | Log có correlation id; không log OTP plaintext / full PII |

### 3.5 Acceptance criteria (luồng chính)

#### AC — Đặt giao trong phạm vi

1. Given cửa hàng tại (lat0, lng0), `max_radius_km = 10`, phí bật với bậc hợp lệ.  
2. When khách verify OTP, chọn SP, chọn địa chỉ cách 3 km.  
3. Then hệ thống cho đặt; hiển thị phí đúng bậc; đơn xuất hiện ở admin list với distance ≈ 3 km.

#### AC — Ngoài phạm vi

1. Given `max_radius_km = 10`.  
2. When địa chỉ cách 12 km.  
3. Then không tạo đơn; UI báo ngoài phạm vi giao hàng.

#### AC — OTP bắt buộc

1. When khách chưa verify OTP.  
2. Then API place-order trả 401/403; UI chặn bước xác nhận.

#### AC — Admin FIFO

1. Given 2 đơn PENDING, A đặt trước B.  
2. When admin mở danh sách.  
3. Then A đứng trước B.

#### AC — Hoàn tất + công nợ

1. When admin hoàn tất với `PARTIAL` paid = 100_000, total = 450_000.  
2. Then đơn COMPLETED; debt = 350_000; dashboard công nợ tăng tương ứng.

#### AC — Cô lập dữ liệu

1. Given khách X và Y mỗi người một đơn.  
2. When X gọi API lấy đơn của Y.  
3. Then 404 hoặc 403; không trả body đơn Y.

---

## 4. Epic → Story → Task

### E1 — Identity & OTP

#### US-1.1 Gửi và xác thực OTP khách
**Story:** Là khách, tôi muốn xác thực SĐT bằng OTP để đặt hàng an toàn.  
**Tasks:**
- [DONE] T1.1.1 API `POST /auth/otp/request` + rate limit
- [DONE] T1.1.2 API `POST /auth/otp/verify` → JWT
- [DONE] T1.1.3 Adapter SMS (mock + interface production)
- [DONE] T1.1.4 Flutter: màn nhập SĐT + OTP
- [DONE] T1.1.5 Lưu `otp_challenges` (hash OTP, expiry) trên SQLite auth

#### US-1.2 Đăng nhập Admin
**Story:** Là CCH, tôi muốn đăng nhập bằng tài khoản admin.  
**Tasks:**
- [DONE] T1.2.1 Seed admin account (password hash)
- [DONE] T1.2.2 API login admin + refresh
- [DONE] T1.2.3 Flutter admin login screen
- [DONE] T1.2.4 Middleware RBAC trên gateway

---

### E2 — Catalog & Pricing

#### US-2.1 Admin quản lý sản phẩm
**Story:** Là admin, tôi muốn thêm/sửa/ẩn sản phẩm và giá bán.  
**Tasks:**
- [DONE] T2.1.1 CRUD APIs catalog
- [DONE] T2.1.2 Schema `products`, `product_prices`
- [DONE] T2.1.3 Event `catalog.product.updated`
- [DONE] T2.1.4 Flutter admin: màn sản phẩm

#### US-2.2 Khách xem sản phẩm để đặt
**Story:** Là khách, tôi muốn chọn sản phẩm đang bán khi đặt giao gas.  
**Tasks:**
- [DONE] T2.2.1 API list products `active` (public/authenticated)
- [DONE] T2.2.2 Flutter: bước chọn SP trong flow đặt hàng

---

### E3 — Ordering & Geo fence

#### US-3.1 Chọn địa chỉ (GPS / search)
**Story:** Là khách, tôi muốn dùng vị trí hiện tại hoặc search địa chỉ có gợi ý.  
**Tasks:**
- [DONE] T3.1.1 Xin quyền location (Web / Android / iOS)
- [DONE] T3.1.2 Proxy search geocode (Photon/Nominatim) qua geo-service
- [DONE] T3.1.3 Flutter: map/picker + autocomplete

#### US-3.2 Kiểm tra bán kính giao
**Story:** Là khách, tôi muốn biết địa chỉ có nằm trong phạm vi giao không.  
**Tasks:**
- [DONE] T3.2.1 Store settings: lat/lng, `max_radius_km`
- [DONE] T3.2.2 API tính Haversine distance + `in_range`
- [DONE] T3.2.3 UI thông báo ngoài phạm vi

#### US-3.3 Đặt đơn giao gas
**Story:** Là khách, tôi muốn xác nhận đơn sau khi đủ thông tin.  
**Tasks:**
- [DONE] T3.3.1 API `POST /orders` (validate JWT, items, geo, fee)
- [DONE] T3.3.2 Persist order + items; publish `order.placed`
- [DONE] T3.3.3 Flutter: review + success screen
- [DONE] T3.3.4 Mask PII trong response

---

### E4 — Delivery Fee Rules

#### US-4.1 Cấu hình bậc phí và bật/tắt
**Story:** Là admin, tôi muốn bật/tắt phí ship và set bậc theo km.  
**Tasks:**
- [DONE] T4.1.1 Schema `delivery_fee_settings`, `delivery_fee_rules`
- [DONE] T4.1.2 Admin APIs cấu hình phí
- [DONE] T4.1.3 Engine tính phí khi preview/place order
- [DONE] T4.1.4 Flutter admin: màn phí giao hàng

#### US-4.2 Preview phí trước khi đặt
**Story:** Là khách, tôi muốn thấy phí giao trước khi xác nhận.  
**Tasks:**
- [DONE] T4.2.1 API quote: distance + fee + total
- [DONE] T4.2.2 Hiển thị trên Flutter review step

---

### E5 — Admin Order Desk + Navigation

#### US-5.1 Danh sách đơn FIFO
**Story:** Là admin, tôi muốn xem đơn chờ giao theo thứ tự cũ nhất trước.  
**Tasks:**
- [DONE] T5.1.1 API list orders (admin), sort `created_at ASC`
- [DONE] T5.1.2 Cột STT, tên, SĐT, địa chỉ, km, thời gian
- [DONE] T5.1.3 Flutter Order Desk UI
- [DONE] T5.1.4 (Should) Polling hoặc SSE/NATS bridge báo đơn mới

#### US-5.2 Dẫn đường tới điểm giao
**Story:** Là CCH, tôi muốn mở chỉ đường từ vị trí hiện tại tới khách.  
**Tasks:**
- [DONE] T5.2.1 Lấy lat/lng đơn
- [DONE] T5.2.2 Deep-link Google Maps / geo intent
- [DONE] T5.2.3 Nút “Dẫn đường” trên chi tiết đơn

---

### E6 — Payment status & Debt

#### US-6.1 Hoàn tất giao + ghi nhận thanh toán
**Story:** Là admin, khi giao xong tôi muốn chọn đã thu đủ / một phần / nợ.  
**Tasks:**
- [DONE] T6.1.1 API `POST /orders/{id}/complete` + payment payload
- [DONE] T6.1.2 Billing ghi `payments` + cập nhật `debts`
- [DONE] T6.1.3 Events `order.completed`, `billing.payment.recorded`, `billing.debt.updated`
- [DONE] T6.1.4 Flutter dialog hoàn tất

#### US-6.2 Xem công nợ khách (admin)
**Story:** Là admin, tôi muốn xem khách còn nợ bao nhiêu.  
**Tasks:**
- [DONE] T6.2.1 API list/aggregate debts
- [DONE] T6.2.2 UI đơn giản trong dashboard hoặc tab Công nợ

---

### E7 — Inventory

#### US-7.1 Nhập / xuất tồn theo mã SP
**Story:** Là admin, tôi muốn nhập xuất kho và giữ giá nhập/giá bán.  
**Tasks:**
- [DONE] T7.1.1 Schema stock + movements + cost
- [DONE] T7.1.2 APIs nhập/xuất/điều chỉnh
- [DONE] T7.1.3 Consumer `order.placed` / `order.completed` trừ tồn (chốt: trừ khi placed hoặc completed — mặc định **trừ khi complete** để tránh giữ tồn ảo nếu hủy; document trong architecture)
- [DONE] T7.1.4 Flutter màn tồn kho

#### US-7.2 Cơ sở tính lợi nhuận
**Story:** Là admin, tôi muốn hệ thống có giá vốn để tính lãi.  
**Tasks:**
- [DONE] T7.2.1 Lưu cost tại thời điểm xuất/bán (snapshot)
- [DONE] T7.2.2 Công thức profit cho report-service

---

### E8 — Dashboard Analytics

#### US-8.1 Dashboard kinh doanh nhỏ
**Story:** Là CCH, tôi muốn xem doanh thu, công nợ, lợi nhuận, tồn.  
**Tasks:**
- [DONE] T8.1.1 report-service subscribe events → `daily_stats`
- [DONE] T8.1.2 API dashboard summary
- [DONE] T8.1.3 Flutter dashboard widgets

---

### E9 — Platform / Security / Gateway

#### US-9.1 API Gateway & cứng hóa
**Story:** Là hệ thống, mọi request đi qua gateway an toàn.  
**Tasks:**
- [DONE] T9.1.1 Routing, CORS, JWT validation
- [DONE] T9.1.2 Rate limit OTP / login / place-order
- [DONE] T9.1.3 Security headers; không lộ internal error
- [DONE] T9.1.4 Audit log admin actions

#### US-9.2 Scaffold & DX
**Tasks:**
- [DONE] T9.2.1 Monorepo layout (Go services + Flutter app)
- [DONE] T9.2.2 NATS JetStream local
- [DONE] T9.2.3 Makefile / scripts chạy dev
- [DONE] T9.2.4 CTA shell Flutter cho Web + Android + iOS (cùng lúc)
- [DONE] T9.2.5 Checklist platform: không dùng API chỉ có trên một OS nếu chưa có fallback; verify Web + emulator Android; iOS Simulator hoặc CI macOS

---

## 5. Sprint planning

Đề xuất **6 sprint**, mỗi sprint **1–2 tuần** (1–2 người: vừa học Go vừa làm Flutter).

### Definition of Done (chung)

- [ ] API có authn/authz đúng role
- [ ] Response không lộ PII thừa; SĐT được mask ở các API khách
- [ ] Happy-path test (ít nhất manual checklist; ưu tiên unit cho fee/distance)
- [ ] Flutter **Web + Android + iOS** chạy được phần việc của sprint (Web + Android Emulator bắt buộc local; iOS Simulator hoặc artifact CI macOS)
- [ ] Event publish/consume (nếu story liên quan) có log xác nhận
- [ ] Cập nhật ngắn trong PR / note liên kết PRD AC
- [ ] Ghi `CHANGESLOG.md` + `workdocs_*` theo skill change-workdocs

### Chiến lược test không máy Android thật

1. **Hàng ngày:** Flutter Web (Chrome) — flow UX/API nhanh nhất.
2. **Mỗi story có quyền location / deep-link / permission:** Android Emulator (Android Studio).
3. **iOS:** Simulator trên Mac nếu có; nếu không → GitHub Actions `macos-latest` build + (optional) screenshot/test; không để plugin Android-only lọt vào `main`.
4. **UAT cửa hàng:** ưu tiên Web + máy thật bất kỳ (Android/iOS của CCH) khi có.

### Sprint 0 — Foundation & CTA

**Mục tiêu:** Skeleton chạy được trên 3 target; brand + CTA đặt gas hiện ngay.

| Hạng mục | Story / Task |
|----------|----------------|
| Platform | T9.2.1–T9.2.5 |
| UX | Home Flutter: Gas Tam Đệ + CTA “Đặt giao gas” (flow placeholder) |
| Infra | Gateway hello; NATS up; health checks |

**Demo:** Web + Android Emulator (+ iOS Simulator hoặc CI build) thấy CTA; ping gateway OK.

### Sprint 1 — OTP, Catalog, Order draft

| Hạng mục | Story |
|----------|--------|
| Auth | US-1.1, US-1.2 |
| Catalog | US-2.1, US-2.2 |
| Order | Draft model + API skeleton (chưa geo/fee đầy đủ) |

**Demo:** Admin thêm SP; khách OTP + chọn SP (chưa place thật hoặc place stub).

### Sprint 2 — Geo, Fee, Place order

| Hạng mục | Story |
|----------|--------|
| Geo | US-3.1, US-3.2 |
| Fee | US-4.1, US-4.2 |
| Order | US-3.3 |

**Demo:** Đặt đơn trong phạm vi thành công; ngoài phạm vi bị chặn; phí hiển thị đúng.

### Sprint 3 — Order desk, Complete, Navigation

| Hạng mục | Story |
|----------|--------|
| Admin desk | US-5.1, US-5.2 |
| Billing | US-6.1 |

**Demo:** Đơn FIFO; dẫn đường; hoàn tất với đủ/một phần/nợ.

### Sprint 4 — Inventory & COGS

| Hạng mục | Story |
|----------|--------|
| Inventory | US-7.1, US-7.2 |
| Debt view | US-6.2 (nếu chưa xong S3) |

**Demo:** Nhập kho; giao xong trừ tồn; có số liệu giá vốn.

### Sprint 5 — Dashboard, Security harden, UAT

| Hạng mục | Story |
|----------|--------|
| Report | US-8.1 |
| Security | US-9.1 + AC cô lập dữ liệu |
| UAT | Chạy thử tại cửa hàng Gas Tam Đệ với dữ liệu thật (cẩn trọng PII) |

**Demo:** Dashboard đủ 4 nhóm chỉ số; checklist AC đạt; sẵn sàng dùng nội bộ.

### Milestone sau MVP (không commit sprint)

- Push thông báo đơn mới
- Phát hành Play Store / App Store công khai
- Thanh toán online
- Tối ưu lộ trình nhiều điểm giao

---

## 6. Rủi ro & giảm thiểu

| Rủi ro | Mức | Giảm thiểu |
|--------|-----|------------|
| Chi phí SMS OTP | TB | Mock dev; chọn nhà cung cấp VN; rate limit; OTP cooldown |
| Geocode không chính xác | TB | Cho phép chỉnh pin trên map; admin gọi xác nhận địa chỉ khi giao |
| Microservices phức tạp cho shop nhỏ | Cao (DX) | Ranh giới rõ nhưng deploy 1 VPS; fee nằm trong order-service |
| SQLite concurrent write | TB | WAL mode; 1 writer/process; backup định kỳ |
| Lộ PII | Cao | Mask, encryption-at-rest cho phone, audit, không log full phone |
| Không có máy Android thật | TB | Emulator + Web; UAT trên máy CCH khi có |
| Không có Mac để test iOS local | TB | Enable iOS target từ Sprint 0; CI macOS build; mượn Mac/TestFlight khi UAT |

---

## 7. Liên kết

- Kiến trúc microservice, EDA, schema DB, security chi tiết: **[architecture.md](./architecture.md)**
- Deploy & chiến lược Git (monorepo, 1 VPS): **[architecture.md §9](./architecture.md#9-deploy--repo-strategy)**
- Tên thương hiệu bắt buộc trong UI/docs: **Gas Tam Đệ**
