# Engine tính phí giao khi preview/place order (T4.1.3)

- **Thư mục:** `workdocs_delivery_fee_engine_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.3

## Mục tiêu

Thay stub `delivery_fee = 0` trên place order bằng engine đọc `delivery_fee_settings` + `delivery_fee_rules`: tắt → phí 0; bật → chọn bậc khớp `distance_km`. Engine pure/reusable cho quote sau này (T4.2.1).

## Phạm vi

- Trong scope:
  - `matchDeliveryFee` (pure) + `computeDeliveryFee` (load DB)
  - Wire vào `POST /v1/orders`
  - Unit tests band matching / disabled / inactive / gap
  - Place-order tests enabled vs disabled
  - Mark PRD `[DONE] T4.1.3`
- Ngoài scope:
  - T4.1.4 Flutter admin màn phí giao hàng
  - T4.2.1 API quote preview
  - T4.2.2 Flutter hiển thị phí trên review

## Quyết định chính

- Bậc half-open `[min_km, max_km)` với `max_km` null = +∞ (khớp schema / admin validate).
- Settings chưa seed (`ErrNoRows`) → fee 0 (giống disabled).
- Không khớp band active nào → fee 0 (không fail place order).
- Quote endpoint chưa làm; caller sau chỉ cần gọi lại `matchDeliveryFee` / `computeDeliveryFee`.

## Đã làm

- [x] `matchDeliveryFee` + `computeDeliveryFee` trong `delivery_fee.go`
- [x] Thay stub trong `create_order.go`
- [x] Unit + place-order tests
- [x] Mark `[DONE] T4.1.3` trên PRD
- [x] CHANGESLOG entry

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/delivery_fee.go` | modified | engine match + compute |
| `services/order-service/delivery_fee_test.go` | modified | unit tests engine |
| `services/order-service/create_order.go` | modified | wire engine, bỏ stub |
| `services/order-service/create_order_test.go` | modified | place-order fee tests |
| `services/order-service/schema.sql` | modified | comment T4.1.3 |
| `docs/prd.md` | modified | `[DONE] T4.1.3` |
| `CHANGESLOG.md` | modified | Entry mới |
| `workdocs_delivery_fee_engine_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/order-service/ -count=1 -run "MatchDeliveryFee|ComputeDeliveryFee|CreateOrderApplies|CreateOrderDeliveryFee|CreateOrderHappy"`
2. Confirm PRD: `- [DONE] T4.1.3 Engine tính phí khi preview/place order`
3. Seed fee enabled → place order `distance_km` trong [0,5) → `delivery_fee=10000`

## Ghi chú / blocker

- Next unfinished trong US-4.1: **T4.1.4** Flutter admin màn phí giao hàng.
- Next epic task sau E4 story này cũng gồm **T4.2.1** API quote (dùng lại engine).
