# Store settings: lat/lng, max_radius_km

- **Thư mục:** `docs/workdocs_geo_store_settings_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.2 / Sprint 2 / T3.2.1

## Mục tiêu

Lưu cấu hình cửa hàng (vị trí + bán kính giao tối đa) trên SQLite geo-service, seed local từ env, và expose API đọc để T3.2.2 (Haversine `in_range`) dùng sau.

## Phạm vi

- Trong scope:
  - Persist singleton `store_settings` (schema đã có)
  - Seed từ `STORE_*` env (idempotent)
  - `GET /v1/geo/store` (public)
  - `PUT /v1/admin/geo/store` (cập nhật settings; không publish event)
  - Unit tests + `.env.example`
- Ngoài scope:
  - Haversine / `POST /v1/geo/check` (T3.2.2)
  - UI ngoài phạm vi (T3.2.3)
  - Event `geo.store_config.updated`
  - Gateway reverse-proxy thật (path public `/geo/store` đã align; upstream vẫn stub)

## Quyết định chính

- Singleton id = `default` (architecture §6.3).
- Default local ≈ Bến Thành HCMC (`10.7769`, `106.7009`), `max_radius_km=10` — đổi qua env / admin PUT.
- Seed không ghi đè row đã có (giống admin seed).
- Public GET chỉ trả field cần cho client (không `updated_by`).

## Đã làm

- [x] `store.go`: seed, get, GET/PUT handlers + validation
- [x] Wire main: seed on start, replace stubs
- [x] `config.GetFloat`
- [x] Unit tests
- [x] `.env.example` `STORE_*`
- [x] Mark `[DONE] T3.2.1` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/geo-service/store.go` | added | seed + GET/PUT |
| `services/geo-service/store_test.go` | added | unit tests |
| `services/geo-service/main.go` | modified | seed + wire handlers |
| `pkg/config/config.go` | modified | `GetFloat` |
| `deploy/.env.example` | modified | `STORE_*` |
| `docs/prd.md` | modified | `[DONE] T3.2.1` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/geo-service/ -count=1`
2. Chạy geo-service, gọi:
   `curl http://127.0.0.1:8083/v1/geo/store`
3. Cập nhật:
   `curl -X PUT http://127.0.0.1:8083/v1/admin/geo/store -H "Content-Type: application/json" -d "{\"max_radius_km\":8}"`
4. Restart service → seed không đổi row đã có

## Ghi chú / blocker

- Next candidate: **T3.2.2** API tính Haversine distance + `in_range`.
