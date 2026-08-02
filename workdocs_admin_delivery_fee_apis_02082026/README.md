# Admin APIs cấu hình phí giao (T4.1.2)

- **Thư mục:** `workdocs_admin_delivery_fee_apis_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** Sprint 3 / US-4.1 / T4.1.2

## Mục tiêu

Expose HTTP admin để đọc/cập nhật toggle phí giao và bậc khoảng cách trên order-service; path `/v1/admin/delivery-fee` khớp gateway RBAC `role=admin`. Không làm fee engine hay Flutter UI.

## Phạm vi

- Trong scope:
  - `GET /v1/admin/delivery-fee` — `enabled` + `rules[]`
  - `PUT /v1/admin/delivery-fee` — partial: `enabled` và/hoặc replace `rules`
  - Validate band (min/max/fee) + không overlap bậc `active`
  - Unit tests order-service + assert gateway `/v1/admin/delivery-fee` cần admin JWT
  - Mark `[DONE] T4.1.2` trên PRD
- Ngoài scope:
  - T4.1.3 Engine tính phí khi preview/place order
  - T4.1.4 Flutter admin màn phí giao
  - Gateway reverse-proxy thật (vẫn stub → order URL; RBAC đã cover prefix)

## Quyết định chính

- Response/body: `{ enabled, updated_at?, rules: [{ id, min_km, max_km|null, fee_vnd, sort_order, active }] }`.
- PUT partial giống geo store: chỉ `enabled` giữ nguyên rules; `rules` non-nil → delete-all + insert trong transaction.
- `id` rule trống → generate UUID; default `active=true`, `sort_order` = index nếu omit.
- Half-open `[min_km, max_km)`; `max_km` null = +inf; chỉ một open-ended active và phải là band cuối theo `min_km`.
- Authz tại gateway (`RequireJWT` + `RequireRole(admin)` trên `/v1/admin/*`); upstream không tự check JWT.

## Đã làm

- [x] `admin_delivery_fee.go`: load/save + GET/PUT handlers + validation
- [x] Wire `main.go` thay stub
- [x] Tests admin API + overlap + gateway path RBAC
- [x] Schema comment T4.1.2
- [x] Mark `[DONE] T4.1.2` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/order-service/admin_delivery_fee.go` | added | GET/PUT handlers |
| `services/order-service/admin_delivery_fee_test.go` | added | unit tests |
| `services/order-service/main.go` | modified | wire handlers |
| `services/order-service/schema.sql` | modified | comment T4.1.2 |
| `services/api-gateway/rbac_test.go` | modified | delivery-fee admin gate |
| `docs/prd.md` | modified | `[DONE] T4.1.2` |
| `CHANGESLOG.md` | modified | entry |
| `workdocs_admin_delivery_fee_apis_02082026/README.md` | added | Workdoc này |

## Cách verify

1. `go test ./services/order-service/ ./services/api-gateway/ -count=1`
2. Chạy order-service (sau seed), gọi:
   `curl http://127.0.0.1:8084/v1/admin/delivery-fee`
3. Cập nhật:
   `curl -X PUT http://127.0.0.1:8084/v1/admin/delivery-fee -H "Content-Type: application/json" -d "{\"enabled\":true}"`
4. Confirm PRD: `- [DONE] T4.1.2 Admin APIs cấu hình phí`

## Ghi chú / blocker

- Next unfinished: **T4.1.3** Engine tính phí khi preview/place order.
