# VPS / Cursor Cloud project `.env` — checklist

## File `.env` hiện tại bị lỗi gì?

Từ bản user paste trên VPS:

| Key | Vấn đề |
|-----|--------|
| `GEOCODE_USER_AGENT=... contact: local` | Còn `: ` → Stage 5 parse override YAML fail |
| `API_BASE_URL="\\\"\\\""` | Giá trị rác; xóa hẳn |
| `NATS_URL=nats://127.0.0.1:4222` | Nếu inject vào container → không nối được NATS (`nats` service) |
| `*_SERVICE_URL=http://127.0.0.1:...` | Override DNS Docker (`geo-service`, …) |
| `IMAGE_PREFIX` / `IMAGE_TAG` | Không cần — image đã hardcode trong compose |
| `SERVICE=api-gateway` | Build-arg Dockerfile; nguy hiểm nếu apply mọi service |
| `*_DB=data/...` | Compose dùng `/data/...` trong container |
| `STORE_NAME="..."` / quote quanh UA | Nên bỏ quote; platform strip quote rồi ghi YAML trần |

## Cách sửa trên VPS

1. Mở Environment của project Cursor Cloud / file `.env` trên VPS.
2. **Thay toàn bộ** bằng nội dung từ `deploy/.env.vps.example` (điền secret thật).
3. Đặc biệt sửa:
   ```env
   GEOCODE_USER_AGENT=GasTamDe/1.0 (local-dev; geo-service; contact=local)
   ```
4. **Xóa** các key: `API_BASE_URL`, `IMAGE_PREFIX`, `IMAGE_TAG`, `SERVICE`,
   `NATS_URL`, `GEO_SERVICE_URL`, `CATALOG_SERVICE_URL`, `BILLING_SERVICE_URL`,
   và mọi `*_ADDR` / `*_DB` (để compose giữ giá trị đúng).
5. Redeploy Stage 5.

## Bảo mật

Nếu đã paste `SMS_API_*` vào chat/log công khai: rotate key trên Stringee
dashboard và cập nhật lại env.
