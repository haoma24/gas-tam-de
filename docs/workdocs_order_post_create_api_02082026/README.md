# API POST /orders — validate JWT, items, geo, fee (T3.3.1)

- **Thư mục:** `docs/workdocs_order_post_create_api_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.3 / Sprint 2 / T3.3.1

## Mục tiêu

Cho khách (JWT customer qua gateway) tạo đơn: validate identity headers, items (giá từ catalog), geo in-range, phí (stub 0 đến E4), thin persist `orders` + `order_items`.

## Phạm vi

- Trong scope:
  - `POST /v1/orders` với body `{ customer_name, address_text, lat, lng, items[{product_id, qty}] }`
  - AuthZ trust gateway: `X-User-Id` + `X-User-Role=customer` + `X-Phone-Masked`
  - Geo: gọi geo-service `POST /v1/geo/check` → `422 OUT_OF_RANGE` nếu ngoài bán kính
  - Items: load active products từ catalog-service; giá server-side; merge qty trùng `product_id`
  - Fee stub = 0 (`TODO E4/T4.1.3`)
  - Thin persist PENDING order + items; **không** publish `order.placed`
- Ngoài scope:
  - Persist polish + `order.placed` (T3.3.2)
  - Flutter review UI (T3.3.3)
  - PII mask đầy đủ / phone_hash thật từ auth (T3.3.4)
  - Fee engine + admin fee APIs (E4)
  - Gateway reverse-proxy thật (route RBAC đã có; upstream vẫn stub)

## Quyết định chính

- Identity: order-service không parse JWT; tin headers gateway (cùng pattern catalog).
- `phone_hash` tạm: `uid:<user_id>` vì JWT chỉ có `phone_masked`.
- Response trả `phone_masked` (đã mask); không lộ phone đầy đủ.
- Fee = 0 rõ ràng cho đến engine E4; `total = subtotal + delivery_fee`.

## Đã làm

- [x] `create_order.go` + `clients.go` (geo/catalog HTTP)
- [x] Migrate `schema.sql` khi boot; wire `POST /v1/orders`
- [x] Env `GEO_SERVICE_URL` / `CATALOG_SERVICE_URL` (`.env.example` + compose)
- [x] Unit tests (happy, 401/403, OUT_OF_RANGE, product missing, geo down, merge lines)
- [x] Mark `[DONE] T3.3.1` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/create_order.go` | added | handler validate + thin persist |
| `services/order-service/clients.go` | added | geo + catalog clients |
| `services/order-service/create_order_test.go` | added | unit tests |
| `services/order-service/main.go` | modified | migrate + wire handler |
| `deploy/.env.example` | modified | upstream URLs |
| `deploy/docker-compose.yml` | modified | order-service env + depends |
| `docs/prd.md` | modified | `[DONE] T3.3.1` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/order-service/ -count=1`
2. Chạy catalog + geo (đã seed store + có product active) + order-service, rồi:

```bash
curl -s -X POST http://127.0.0.1:8084/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-User-Id: user-dev" \
  -H "X-User-Role: customer" \
  -H "X-Phone-Masked: 090***4567" \
  -d "{\"customer_name\":\"Nguyen Van A\",\"address_text\":\"123 Le Loi\",\"lat\":10.8039,\"lng\":106.7009,\"items\":[{\"product_id\":\"<active-id>\",\"qty\":1}]}"
```

3. Kỳ vọng `201` + `status=PENDING`, `delivery_fee=0`, `distance_km` từ geo.
4. Điểm ngoài bán kính → `422 OUT_OF_RANGE`.

## Ghi chú / blocker

- Next candidate: **T3.3.2** Persist order + items polish; publish `order.placed`.
- Fee vẫn 0 cho đến E4; không block place-order trong bán kính.
