# Proxy search geocode (Photon/Nominatim)

- **Thư mục:** `docs/workdocs_geo_search_proxy_02082026`
- **Ngày:** 02/08/2026
- **Loại:** feature
- **Liên quan:** US-3.1 / Sprint 2 / T3.1.2

## Mục tiêu

Khách search địa chỉ qua backend — geo-service proxy Photon hoặc Nominatim — không gọi OSM API từ Flutter. Tôn trọng User-Agent + rate; cache kết quả.

## Phạm vi

- Trong scope:
  - `GET /v1/geo/search?q=&limit=` trên geo-service
  - Provider Photon (default) / Nominatim
  - Cache `geocode_cache`, rate limit theo IP, User-Agent bắt buộc
  - Unit tests (mock upstream)
  - Ghi chú client `ApiConfig` `:8083`
- Ngoài scope:
  - Flutter map / autocomplete UI (T3.1.3)
  - `GET /geo/store`, `POST /geo/check`, admin store (US-3.2)
  - Gateway reverse-proxy thật (vẫn stub 501; path public đã align)

## Quyết định chính

- Default **Photon** (autocomplete-friendly); `GEOCODE_PROVIDER=nominatim` khi cần.
- Response chuẩn hóa `{ items: [{ label, lat, lng, source }], cached }` để Flutter T3.1.3 dùng.
- Nominatim: User-Agent identifying app + serialize ≥1s giữa các request upstream.
- Abuse: `GEOCODE_MAX_PER_IP_MINUTE` (default 30) + cache TTL giờ.

## Đã làm

- [x] Implement search handler + Photon/Nominatim clients
- [x] SQLite cache + migrate schema embed
- [x] IP rate limit + validation `q`/`limit`
- [x] Tests mock HTTP upstream
- [x] `.env.example` + ApiConfig note
- [x] Mark `[DONE] T3.1.2` trên PRD + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `services/geo-service/main.go` | modified | migrate, wire search |
| `services/geo-service/search.go` | added | handler + cache |
| `services/geo-service/geocode.go` | added | Photon + Nominatim |
| `services/geo-service/search_ratelimit.go` | added | IP limiter |
| `services/geo-service/search_test.go` | added | unit tests |
| `services/geo-service/schema.sql` | modified | migrate note |
| `deploy/.env.example` | modified | GEOCODE_* |
| `apps/mobile/lib/core/api_config.dart` | modified | note `:8083` |
| `docs/prd.md` | modified | `[DONE] T3.1.2` |
| `CHANGESLOG.md` | modified | entry |

## Cách verify

1. `go test ./services/geo-service/ -count=1`
2. Chạy geo-service, gọi:
   `curl "http://127.0.0.1:8083/v1/geo/search?q=ben%20thanh"`
3. Lần 2 cùng `q` → `"cached":true`
4. Gateway path public: `GET /v1/geo/search` (RBAC public; upstream proxy vẫn stub cho đến E9)

## Ghi chú / blocker

- Next candidate: **T3.1.3** Flutter map/picker + autocomplete (gọi geo-service, không gọi OSM trực tiếp).
