# Admin order lat/lng + GET by id (T5.2.1)

- **Thư mục:** `docs/workdocs_order_admin_lat_lng_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.2 / Sprint 3 / T5.2.1

## Mục tiêu

CCH cần `lat`/`lng` điểm giao của đơn để mở chỉ đường (T5.2.2/T5.2.3). Đơn đã lưu toạ độ lúc place; task này đảm bảo API admin list + detail expose rõ `lat`/`lng`, và bổ sung `GET /v1/admin/orders/{id}` (trước đó stub 501).

## Phạm vi

- Trong scope:
  - `GET /v1/admin/orders/{id}` trả `orderView` gồm `lat`/`lng` (WGS84 destination)
  - Khẳng định `GET /v1/admin/orders` cũng trả `lat`/`lng` trên mỗi phần tử
  - Unit tests get-by-id, 404, list coords
- Ngoài scope:
  - Deep-link Google Maps / geo intent (T5.2.2)
  - Nút “Dẫn đường” Flutter (T5.2.3)
  - Flutter gọi GET by id (desk vẫn dùng payload list)

## Quyết định chính

- Response detail = cùng shape `orderView` như create/list (không wrap `{order:...}`), khớp catalog admin get-by-id.
- Không gán `stt` trên detail — STT chỉ có nghĩa trong list FIFO hiện tại.
- Trust gateway RBAC `/v1/admin/*`; order-service không parse JWT.
- Keys JSON giữ tên rõ `lat` / `lng` (không đổi tên field).

## Đã làm

- [x] `handleGetAdminOrder` + `loadOrderByID`
- [x] Wire `GET /v1/admin/orders/{id}` (thay stub)
- [x] Comment `orderView.Lat/Lng` = destination cho navigation
- [x] Tests: get lat/lng, not found, list exposes coords
- [x] Mark `[DONE] T5.2.1` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/list_orders.go` | modified | get-by-id + loadOrderByID |
| `services/order-service/main.go` | modified | wire handler |
| `services/order-service/create_order.go` | modified | comment Lat/Lng |
| `services/order-service/create_order_test.go` | modified | mount get route |
| `services/order-service/list_admin_orders_test.go` | modified | insert helper lat/lng; desk assert |
| `services/order-service/get_admin_order_test.go` | added | T5.2.1 tests |
| `docs/prd.md` | modified | `[DONE] T5.2.1` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/order-service/ -count=1`
2. Với order-service chạy + DB có đơn:

```bash
curl -s "http://127.0.0.1:8084/v1/admin/orders"
curl -s "http://127.0.0.1:8084/v1/admin/orders/<id>"
```

3. Kỳ vọng: cả list item và detail có `"lat"` / `"lng"` số thực khớp điểm giao.

## Ghi chú / blocker

- Next candidate: **T5.2.2** Deep-link Google Maps / geo intent.
- Gateway proxy upstream vẫn có thể stub — gọi thẳng order-service `:8084` khi dev local.
