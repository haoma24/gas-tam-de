# APIs nhập/xuất/điều chỉnh tồn (T7.1.2)

- **Thư mục:** `workdocs_inventory_stock_apis_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-7.1 / T7.1.2 / PRD M7 / architecture §4.4

## Mục tiêu

Admin có API list tồn kho và tạo phiếu nhập (`IN`) / xuất (`OUT`) / điều chỉnh (`ADJUST`): cập nhật `on_hand` + `cost_price`, ghi `stock_movements`. Path dưới `/v1/admin/...` để gateway RBAC.

## Phạm vi

- Trong scope:
  - `GET /v1/admin/inventory` — list stock
  - `POST /v1/admin/inventory` — IN / OUT / ADJUST
  - Transaction: update `stock_items` + insert `stock_movements`
  - Unit tests + gateway RBAC coverage path inventory
  - Sync architecture §4.4; mark PRD `[DONE] T7.1.2`
- Ngoài scope:
  - T7.1.3 Consumer `order.completed` trừ tồn
  - T7.1.4 Flutter màn tồn kho
  - Publish `inventory.stock.adjusted` / `inventory.low_stock` (chưa bắt buộc ở task này)
  - Reverse-proxy thật gateway → inventory (vẫn stub 501 như các admin route khác)

## Quyết định chính

- Một `POST /v1/admin/inventory` với `movement_type` thay vì 3 endpoint tách (khớp stub sẵn có GET/POST).
- `IN` tạo `stock_items` nếu chưa có (cần `sku` + `name`); `cost_price` = `unit_cost` (giá nhập hiện tại).
- `OUT` snapshot `unit_cost` từ `cost_price`; MVP cho phép `on_hand` âm.
- `ADJUST` dùng `delta` signed; DB lưu `qty = |delta|`; response trả `delta` signed.
- `created_by` lấy từ header `X-User-Id` (gateway JWT).
- Không wire NATS ở task này (inventory vẫn chạy độc lập; events khi cần / consumer T7.1.3).

## Đã làm

- [x] `stock_api.go` — list + apply movement (IN/OUT/ADJUST)
- [x] Wire routes trong `main.go` (bỏ stub)
- [x] Unit tests happy-path + validation + negative on_hand + persist movements
- [x] Gateway RBAC test `/v1/admin/inventory`
- [x] Sync architecture §4.4 inventory endpoints
- [x] Mark `[DONE] T7.1.2` trên PRD; CHANGESLOG + workdocs

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/inventory-service/stock_api.go` | added | handlers + apply logic |
| `services/inventory-service/stock_api_test.go` | added | API/unit tests |
| `services/inventory-service/main.go` | modified | wire GET/POST; remove stub |
| `services/api-gateway/rbac_test.go` | modified | inventory RBAC coverage |
| `docs/architecture.md` | modified | §4.4 inventory API detail |
| `docs/prd.md` | modified | `[DONE] T7.1.2` |
| `CHANGESLOG.md` | modified | entry mới |
| `workdocs_inventory_stock_apis_02082026/README.md` | added | workdoc này |

## Cách verify

1. Unit:

```bash
go test ./services/inventory-service/ ./services/api-gateway/ -count=1
```

2. Manual (inventory `:8085`):

```bash
# List empty
curl -s http://127.0.0.1:8085/v1/admin/inventory

# Nhập kho
curl -s -X POST http://127.0.0.1:8085/v1/admin/inventory \
  -H "Content-Type: application/json" -H "X-User-Id: admin-1" \
  -d '{"movement_type":"IN","product_id":"p1","sku":"GAS12","name":"Gas 12kg","qty":10,"unit_cost":150000}'

# Xuất tay
curl -s -X POST http://127.0.0.1:8085/v1/admin/inventory \
  -H "Content-Type: application/json" -H "X-User-Id: admin-1" \
  -d '{"movement_type":"OUT","product_id":"p1","qty":2}'

# Điều chỉnh
curl -s -X POST http://127.0.0.1:8085/v1/admin/inventory \
  -H "Content-Type: application/json" -H "X-User-Id: admin-1" \
  -d '{"movement_type":"ADJUST","product_id":"p1","delta":-1,"note":"kiem ke"}'
```

3. Gateway RBAC (proxy vẫn 501):

```bash
# Customer token → 403; admin token → 501 stub
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/v1/admin/inventory \
  -H "Authorization: Bearer <token>"
```

## Ghi chú / blocker

- Next unfinished: **T7.1.3** Consumer `order.completed` trừ tồn.
- Gateway reverse-proxy tới inventory chưa wire (cùng pattern stub `/v1/admin/*`).
