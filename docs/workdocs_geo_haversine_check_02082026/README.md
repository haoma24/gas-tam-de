# Haversine distance + in_range (POST /geo/check)

- **Thư mục:** `docs/workdocs_geo_haversine_check_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.2 / Sprint 2 / T3.2.2

## Mục tiêu

Cho khách (qua gateway JWT customer) kiểm tra điểm giao có nằm trong bán kính cửa hàng: tính Haversine từ `store_settings` tới `{lat,lng}` và trả `distance_km` + `in_range`.

## Phạm vi

- Trong scope:
  - Haversine km + làm tròn 2 chữ số thập phân
  - `POST /v1/geo/check` `{ lat, lng }` → `{ distance_km, in_range, max_radius_km }`
  - Unit tests math + handler
- Ngoài scope:
  - Flutter UI ngoài phạm vi (T3.2.3)
  - Gateway reverse-proxy thật (route RBAC `POST /geo/check` đã có; upstream vẫn stub)
  - Order place / fee quote dùng distance

## Quyết định chính

- Earth radius = 6371 km (convention phổ biến với Haversine).
- `in_range` inclusive: `distance_km <= max_radius_km` (khớp PRD từ chối khi `>`).
- Response luôn HTTP 200 khi coords hợp lệ (kể cả ngoài bán kính) — client/UI T3.2.3 quyết định hiển thị.
- Auth/RBAC ở gateway; geo-service không tự check JWT.

## Đã làm

- [x] `check.go`: Haversine, round 2dp, `handleCheck`
- [x] Wire `POST /v1/geo/check` trong `main.go` (bỏ stub)
- [x] Unit tests math + API
- [x] Mark `[DONE] T3.2.2` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/geo-service/check.go` | added | Haversine + handler |
| `services/geo-service/check_test.go` | added | unit tests |
| `services/geo-service/main.go` | modified | wire handler, remove stub |
| `docs/prd.md` | modified | `[DONE] T3.2.2` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/geo-service/ -count=1`
2. Chạy geo-service (đã seed store), gọi:
   `curl -X POST http://127.0.0.1:8083/v1/geo/check -H "Content-Type: application/json" -d "{\"lat\":10.8039,\"lng\":106.7009}"`
   → `in_range: true`, `distance_km` ≈ 3
3. Điểm xa hơn radius:
   `curl -X POST http://127.0.0.1:8083/v1/geo/check -H "Content-Type: application/json" -d "{\"lat\":10.885,\"lng\":106.7009}"`
   → `in_range: false`, `distance_km` ≈ 12

## Ghi chú / blocker

- Next candidate: **T3.2.3** UI thông báo ngoài phạm vi.
