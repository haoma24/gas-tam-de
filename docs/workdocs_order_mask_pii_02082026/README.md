# Mask PII trong order response (T3.3.4)

- **Thư mục:** `docs/workdocs_order_mask_pii_02082026`
- **Ngày:** 02/08/2026
- **Loại:** security
- **Liên quan:** US-3.3 / Sprint 2 / T3.3.4

## Mục tiêu

Đảm bảo API khách của order-service không lộ SĐT đầy đủ: response create + list chỉ trả `phone_masked` kiểu auth (`090***4567`); remask defense-in-depth nếu header vô tình lộ số thật; không trả `phone_hash` / `phone_e164`.

## Phạm vi

- Trong scope:
  - Helper mask SĐT (cùng style auth-service)
  - `POST /v1/orders` response qua `customerOrderView` (remask trước persist + JSON)
  - `GET /v1/orders/me` list đơn của khách với cùng policy mask
  - Unit tests remask + list + không leak field secret
- Ngoài scope:
  - E4 fee engine / admin fee APIs
  - Admin order desk full phone theo role (T5.1.x)
  - phone_hash thật từ auth (vẫn placeholder `uid:<user_id>`)
  - Flutter UI thay đổi (đã hiện `phone_masked`)

## Quyết định chính

- Mask format khớp auth: `090***4567` (3 prefix + `***` + 4 cuối).
- Địa chỉ giao của chính khách vẫn trả full `address_text` / lat / lng (cần cho xác nhận đơn).
- `GET /v1/orders/me` scan hết order rows rồi mới load items (tránh deadlock SQLite `MaxOpenConns=1`).
- Event `order.placed` không chứa PII (`order_id` / `total` / `distance_km` / `created_at`).

## Đã làm

- [x] `pii.go`: `maskPhoneDisplay`, `ensurePhoneMasked`, `customerOrderView`
- [x] Create order remask `X-Phone-Masked` trước INSERT + response
- [x] Implement `GET /v1/orders/me` với PII mask + buffer rows trước nested query
- [x] Tests: mask table, remask full header, list masks, 401
- [x] Mark `[DONE] T3.3.4` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/pii.go` | added | mask helpers + customer view builder |
| `services/order-service/pii_test.go` | added | mask + create remask + list tests |
| `services/order-service/list_orders.go` | added | `GET /v1/orders/me` |
| `services/order-service/create_order.go` | modified | remask + `customerOrderView` |
| `services/order-service/create_order_test.go` | modified | mount list route in test router |
| `services/order-service/main.go` | modified | wire `GET /v1/orders/me` |
| `docs/prd.md` | modified | `[DONE] T3.3.4` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

```powershell
go test ./services/order-service/ -count=1 -timeout 60s -run "Mask|PII|CreateOrderRemasks|ListMyOrders"
```

Create với số đầy đủ trong header (không nên xảy ra trên prod) → kỳ vọng `phone_masked=090***4567`, body không có full digits / `phone_hash`. `GET /v1/orders/me` tương tự.

## Ghi chú / blocker

- Admin order endpoints vẫn stub — mask sẽ áp dụng khi implement (T5.x).
- Next unfinished PRD: **T4.1.1** Schema `delivery_fee_settings`, `delivery_fee_rules`.
