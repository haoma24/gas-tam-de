# API quote: distance + fee + total (T4.2.1)

- **Thư mục:** `workdocs_order_quote_api_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-4.2 / Sprint 2 / T4.2.1

## Mục tiêu

Cho khách preview phí giao + tổng trước khi place: `POST /v1/orders/quote` trả `distance_km`, `delivery_fee`, `subtotal`, `total`, `in_range` — tái sử dụng geo check, giá catalog, fee engine (T4.1.3). Không persist đơn.

## Phạm vi

- Trong scope:
  - `POST /v1/orders/quote` body `{ items[{product_id, qty}], lat, lng }`
  - AuthZ customer qua gateway headers (`X-User-Id` / `X-User-Role=customer` / `X-Phone-Masked`)
  - Geo check → luôn trả preview kèm `in_range` (kể cả ngoài bán kính; place vẫn 422)
  - Subtotal từ catalog active prices; fee qua `computeDeliveryFee`
  - Unit tests happy / out-of-range / fee off / auth / validation
- Ngoài scope:
  - Flutter review hiển thị phí (T4.2.2)
  - Thay đổi place-order / admin fee APIs

## Quyết định chính

- Quote **không** yêu cầu `customer_name` / `address_text` (architecture: `{ items, lat, lng }`).
- Ngoài bán kính: **200** + `in_range=false` (preview vẫn tính fee theo distance) — khác place order `422 OUT_OF_RANGE`.
- Response thêm `max_radius_km` từ geo (hữu ích cho UI chặn đặt).
- Không ghi `orders` / không publish event.

## Đã làm

- [x] `quote_order.go` — handler preview
- [x] Wire `POST /v1/orders/quote` trong `main.go` (thay stub)
- [x] `quote_order_test.go` + mount route trong test helper
- [x] Mark `[DONE] T4.2.1` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/quote_order.go` | added | handler quote |
| `services/order-service/quote_order_test.go` | added | unit tests |
| `services/order-service/main.go` | modified | wire handler |
| `services/order-service/create_order_test.go` | modified | mount quote route |
| `docs/prd.md` | modified | `[DONE] T4.2.1` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/order-service/ -count=1`
2. Chạy geo + catalog + order-service, rồi:

```bash
curl -s -X POST http://127.0.0.1:8084/v1/orders/quote \
  -H "Content-Type: application/json" \
  -H "X-User-Id: user-dev" \
  -H "X-User-Role: customer" \
  -H "X-Phone-Masked: 090***4567" \
  -d "{\"lat\":10.8039,\"lng\":106.7009,\"items\":[{\"product_id\":\"<active-id>\",\"qty\":2}]}"
```

3. Kỳ vọng `200` + `in_range`, `distance_km`, `subtotal`, `delivery_fee` (0 nếu fee tắt), `total = subtotal + delivery_fee`.
4. Confirm không có row mới trong `orders`.

## Ghi chú / blocker

- Next candidate: **T4.2.2** Hiển thị trên Flutter review step.
