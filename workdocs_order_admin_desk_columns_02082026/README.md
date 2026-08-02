# Cột Order Desk admin list (T5.1.2)

- **Thư mục:** `workdocs_order_admin_desk_columns_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-5.1 / Sprint 3 / T5.1.2

## Mục tiêu

Làm giàu `GET /v1/admin/orders` để Flutter Order Desk (T5.1.3) có đủ cột: **STT | Tên | SĐT | Địa chỉ | km | Thời gian đặt** trên danh sách FIFO.

## Phạm vi

- Trong scope:
  - Thêm `stt` (1-based) theo thứ tự FIFO trong response admin list
  - Đảm bảo / kiểm thử các field cột desk: `customer_name`, `phone_masked`, `address_text`, `distance_km`, `created_at`
  - Policy SĐT admin: vẫn `phone_masked` (orders không lưu `phone_e164`; architecture chưa có policy plaintext explicit)
- Ngoài scope:
  - Flutter Order Desk UI (T5.1.3)
  - Polling / SSE (T5.1.4)
  - `GET /v1/admin/orders/{id}` detail (vẫn stub; dùng chung field khi làm T5.2.x)
  - Full phone theo role (cần decrypt/store — chưa có)

## Quyết định chính

- `stt` = vị trí 1..n trong kết quả list đã sort `created_at ASC` (reset theo filter `status`).
- `stt` dùng `json:"stt,omitempty"` — API khách (`/orders/me`, create) không lộ field.
- Reuse `customerOrderView` + gán `Stt` trong `adminOrderViewsFromRows` (không nhân bản DTO).
- SĐT desk = `phone_masked` (vd. `090***4567`); PRD cho phép mask hoặc full theo role — hiện chỉ có mask trên bảng `orders`.

## Đã làm

- [x] `orderView.Stt` + `adminOrderViewsFromRows`
- [x] Admin list dùng builder có STT
- [x] Unit tests desk columns + STT FIFO + khách không có stt
- [x] Mark `[DONE] T5.1.2` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/create_order.go` | modified | field `stt` trên `orderView` |
| `services/order-service/list_orders.go` | modified | `adminOrderViewsFromRows` |
| `services/order-service/list_admin_orders_test.go` | modified | desk column tests |
| `docs/prd.md` | modified | `[DONE] T5.1.2` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

```powershell
go test ./services/order-service/ -count=1 -timeout 60s -run "ListAdmin|ParseAdmin"
```

Kỳ vọng JSON admin list:

```json
{
  "orders": [
    {
      "stt": 1,
      "customer_name": "...",
      "phone_masked": "090***4567",
      "address_text": "...",
      "distance_km": 3.25,
      "created_at": "..."
    }
  ]
}
```

## Ghi chú / blocker

- Next candidate: **T5.1.3** Flutter Order Desk UI.
- Detail admin order vẫn `notImplemented` — T5.2.1 sẽ lấy lat/lng.
