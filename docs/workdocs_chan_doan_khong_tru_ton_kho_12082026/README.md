# Chẩn đoán được lỗi «Không trừ được tồn kho» khi đặt hàng

- **Thư mục:** `docs/workdocs_chan_doan_khong_tru_ton_kho_12082026`
- **Ngày:** 12/08/2026
- **Loại:** fix
- **Liên quan:** sự cố staging `tamde-stag`; nối tiếp `docs/workdocs_fix_ket_noi_tru_ton_kho_09082026/` (#38)

## Mục tiêu

Khách đặt hàng trên VPS staging nhận lỗi **«Không trừ được tồn kho. Thử lại.»**
(`INVENTORY_UNAVAILABLE`, HTTP 502). `order-service` gọi
`POST /v1/internal/stock/reserve` **đồng bộ** ngay sau khi commit đơn; gọi hỏng
thì đơn bị chuyển `CANCELLED` và khách thấy lỗi.

Nguyên nhân gốc đã được vá ở #38 (thêm `INVENTORY_SERVICE_URL` vào compose),
nhưng lớp vá đó **không tự hiển thị**: mọi container vẫn `healthy`, không có
endpoint nào cho biết order-service đang quay số vào đâu, và nếu container
`order-service` chưa được recreate thì nó vẫn giữ env cũ. Mục tiêu của lần này
là biến sự cố đó thành thứ trả lời được bằng một lệnh `curl`, thay vì phải chờ
một khách hàng thật đặt đơn hỏng.

## Phạm vi

- Trong scope:
  - `order-service` `/readyz` báo cáo 4 upstream đồng bộ (geo, catalog, billing, inventory)
  - Lỗi/log của inventory client kèm **base URL** thật đang cấu hình
  - Test cho nhánh reserve của checkout (trước đó không có test nào)
  - Bước chẩn đoán order → inventory trong `scripts/vps-api-diagnose.sh`
  - **Màn Nhập kho chọn sản phẩm từ catalog** thay vì gõ tay `product_id`
  - **Consumer `catalog.product.updated`** bên inventory-service tạo sẵn dòng tồn 0
- Ngoài scope:
  - Đổi `/healthz` (vẫn là liveness thuần, không phụ thuộc upstream)
  - Thao tác redeploy trên VPS (do maintainer chạy)
  - Dọn dòng `stock_items` cũ đã lệch id (xem "Ghi chú")

## Quyết định chính

- **Không** đưa upstream vào `/healthz`. Repo đã chốt `/healthz` = liveness cho
  compose healthcheck / `depends_on`; thêm dependency vào đó sẽ làm cả stack
  chết theo một service chậm. `/readyz` là chỗ đúng.
- **Lỗi phải kèm URL.** `dial tcp 127.0.0.1:8085` bên trong container chính là
  chữ ký của việc thiếu `INVENTORY_SERVICE_URL`; nếu không in URL thì nó trông
  y hệt trường hợp inventory-service chết thật.
- **Không thêm `INVENTORY_SERVICE_URL=http://127.0.0.1:8085` vào
  `deploy/.env.example`.** Giá trị `127.0.0.1` trong file này từng bị paste vào
  env của VPS và ghi đè DNS Docker — đúng cơ chế gây ra chính sự cố này. Default
  biên dịch sẵn đã là giá trị đó cho host process, nên chỉ để lại comment giải thích.
- Kiểm tra cả 4 upstream chứ không riêng inventory: cùng một kiểu hỏng, cùng một helper.
- **Nhập kho: chọn từ catalog, không cho gõ `product_id` khi catalog còn sống.**
  Tự do gõ tay chính là nguồn sai; `sku` và `name` cũng lấy từ sản phẩm đã chọn
  nên dòng tồn không thể lệch khỏi catalog.
- **Catalog hỏng thì không chặn nhập kho.** Nếu `GET /v1/admin/products` lỗi,
  dialog quay về ô nhập tay kèm cảnh báo đỏ — mất catalog không được làm cửa
  hàng không nhập được kho. Danh sách tồn vẫn hiện bình thường.
- **Ẩn sản phẩm đã có dòng tồn** khỏi picker để admin dùng «Nhập kho» trên dòng
  cũ, tránh tạo trùng. Hết lựa chọn thì nút Xác nhận bị khoá kèm giải thích.
- **Sản phẩm `active=false` vẫn nhập kho được**, chỉ gắn nhãn «(ngừng bán)»:
  hàng tồn thực tế không biến mất khi admin ẩn sản phẩm khỏi cửa hàng.
- **Consumer chỉ đồng bộ định danh, không đụng số lượng.** Catalog sở hữu
  `sku`/`name`; sổ kho sở hữu `on_hand`/`cost_price`. Đổi tên sản phẩm không
  được làm mất tồn, và tạo dòng mới không sinh `stock_movements` (không phải
  phiếu kho).
- **Thêm `name` vào payload `catalog.product.updated`.** §5.1 ghi consumer
  inventory để "đồng bộ tên/sku" nhưng payload lại không có `name`. Thêm field
  là tương thích ngược (consumer cũ bỏ qua field lạ) nên `schema_version` vẫn 1;
  consumer fallback về `sku` cho event cũ còn nằm trong stream.
- **`DeliverAll`** để lần đầu attach thì backfill sản phẩm đã tạo trước đó;
  `processed_events` giữ replay idempotent.
- **Xung đột SKU không được thành poison message.** `stock_items.sku` là UNIQUE;
  nếu event mang SKU của sản phẩm khác thì retry mãi cũng không hết. Consumer
  ghi log ERROR, đánh dấu đã xử lý và đi tiếp thay vì Nak vô hạn.
- **Ẩn nút «Nhập mới» khi mọi sản phẩm đã có dòng tồn.** Sau khi có consumer,
  trạng thái bình thường là sản phẩm nào cũng có dòng — nhập kho diễn ra trên
  dòng đó. Nút chỉ hiện khi thật sự làm được việc (còn sản phẩm thiếu dòng, hoặc
  catalog không tải được).

## Đã làm

- [x] `upstreamHealth` probe `GET <base>/healthz`, timeout 3s, lỗi kèm tên + URL
- [x] Đăng ký geo / catalog / billing / inventory vào `/readyz` của order-service
- [x] Inventory client in base URL trong mọi lỗi (dial lẫn status != 200)
- [x] 4 test cho nhánh reserve: happy path, 409 → `INSUFFICIENT_STOCK`, không gọi được → `INVENTORY_UNAVAILABLE`, lỗi kèm URL
- [x] 5 test cho probe + hợp đồng `/readyz`
- [x] `vps-api-diagnose.sh`: in env `*_SERVICE_URL` của order-service, gọi `/readyz`, lọc log `inventory reserve`
- [x] Comment cảnh báo trong `deploy/.env.example`
- [x] Màn Nhập kho: dropdown chọn sản phẩm từ `GET /v1/admin/products`, tự điền `product_id` / `sku` / `name`
- [x] Fallback nhập tay + cảnh báo khi catalog không tải được
- [x] Ẩn nút «Nhập mới» khi không còn sản phẩm nào thiếu dòng tồn
- [x] 6 widget test cho picker (chọn → POST đúng id, ẩn sản phẩm đã có tồn, nhãn ngừng bán, fallback, ẩn/hiện nút)
- [x] Consumer `inventory-catalog-product-updated` (stream `CATALOG`, `DeliverAll`, idempotent)
- [x] `name` thêm vào payload `catalog.product.updated` + assert trong test catalog
- [x] 9 test consumer: tạo dòng 0, đổi tên giữ tồn, sửa dòng placeholder, replay, xung đột SKU, parse, subject sai, JetStream thật

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/upstream_health.go` | added | probe `/healthz` + adapter `httpx.ReadyCheck` |
| `services/order-service/upstream_health_test.go` | added | ready / unreachable / 503 / URL rỗng / hợp đồng `/readyz` |
| `services/order-service/inventory_reserve_test.go` | added | phủ nhánh reserve của `handleCreateOrder` |
| `services/order-service/main.go` | modified | `MountReady` thêm 4 upstream |
| `services/order-service/inventory_client.go` | modified | lỗi kèm `c.baseURL` |
| `scripts/vps-api-diagnose.sh` | modified | khối chẩn đoán order → inventory |
| `deploy/.env.example` | modified | comment vì sao **không** set `INVENTORY_SERVICE_URL` |
| `apps/mobile/lib/features/inventory/admin_inventory_page.dart` | modified | picker sản phẩm + fallback nhập tay + ẩn nút «Nhập mới» |
| `apps/mobile/test/admin_inventory_picker_test.dart` | added | 6 widget test cho picker |
| `services/inventory-service/product_updated.go` | added | consumer `catalog.product.updated` → dòng tồn 0 / sync sku+name |
| `services/inventory-service/product_updated_test.go` | added | 9 test gồm JetStream nhúng |
| `services/inventory-service/main.go` | modified | attach cả hai consumer, unwind nếu một cái lỗi |
| `services/catalog-service/product_events.go` | modified | payload thêm `name` |
| `services/catalog-service/product_events_test.go` | modified | chốt `name` trong payload |

## Cách verify

Trên VPS:

```bash
COMPOSE_PROJECT_NAME=ts-tamde-stag make vps-api-diagnose
```

Khối `order-service upstream env` phải in `INVENTORY_SERVICE_URL=http://inventory-service:8085`.
Nếu in `NONE` hoặc `127.0.0.1` ⇒ container còn env cũ, recreate:

```bash
COMPOSE_PROJECT_NAME=ts-tamde-stag make vps-up
```

Sau đó:

```bash
docker exec ts-tamde-stag-order-service-1 wget -qO- http://127.0.0.1:8084/readyz
# {"dependencies":{"billing":"ok","catalog":"ok","geo":"ok","inventory":"ok","nats":"ok"},...,"status":"ready"}
```

Local: `go test ./services/order-service/`.

Màn Nhập kho (admin → Tồn kho → «Nhập mới»):

1. Danh sách sổ xuống phải liệt kê sản phẩm từ danh mục, **không** còn ô
   «Mã sản phẩm (product_id)».
2. Sản phẩm đã có dòng tồn không xuất hiện trong danh sách.
3. Nhập số lượng + giá nhập → dòng tồn mới có đúng tên/SKU của catalog, và đơn
   hàng của sản phẩm đó trừ được tồn.

Local: `cd apps/mobile && flutter test test/admin_inventory_picker_test.dart`.

Đồng bộ catalog → kho (cần NATS chạy):

```bash
make nats
# tạo/sửa một sản phẩm ở màn Sản phẩm, rồi:
curl -s http://127.0.0.1:8080/v1/stock/levels
# sản phẩm mới phải xuất hiện với on_hand = 0
docker compose -p <project> logs inventory-service | grep "catalog.product.updated"
```

`Local: go test ./services/inventory-service/ -run ProductUpdated`.

## Ghi chú / blocker

- **Dữ liệu cũ vẫn có thể lệch.** Picker chỉ chặn sai từ nay về sau; dòng
  `stock_items` tạo trước đây bằng id gõ tay vẫn có thể không khớp catalog. Đối
  chiếu một lần: `GET /v1/stock/levels` (product_id trong kho) với
  `GET /v1/products`. Dòng lệch phải nhập lại theo sản phẩm đúng.
- **Backfill chỉ phủ 7 ngày.** Stream có `MaxAge: 7 * 24h` (`pkg/natsx`), nên
  `DeliverAll` khi attach lần đầu chỉ thấy event còn trong stream. Sản phẩm tạo
  lâu hơn 7 ngày mà chưa từng sửa sẽ **không** tự có dòng tồn — sửa nhẹ sản phẩm
  đó (hoặc nhập kho tay) để phát lại event. Đó cũng là lý do vẫn giữ nút
  «Nhập mới» thay vì bỏ hẳn.
- **NATS chết thì không có đồng bộ.** Consumer attach ở nền qua
  `natsx.NewBackground`; broker chưa lên thì sản phẩm mới chưa có dòng tồn cho
  tới khi kết nối lại (event vẫn nằm trong stream, không mất).
- Consumer **không** tạo `stock_movements`: dòng tồn mới là định danh, không
  phải phiếu nhập. Báo cáo COGS không bị ảnh hưởng.
- Lỗi 502 vẫn hủy đơn (`status='CANCELLED'`) — đúng ý đồ, vì không có tồn nào bị
  trừ cho đơn đó.
