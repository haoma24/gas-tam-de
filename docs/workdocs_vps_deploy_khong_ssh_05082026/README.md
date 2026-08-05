# Deploy VPS / staging không có SSH

- **Thư mục:** `docs/workdocs_vps_deploy_khong_ssh_05082026`
- **Ngày:** 05/08/2026

## Bối cảnh

VPS (Cursor Cloud / Tensorship) **không mở SSH**. Chỉ redeploy qua UI + kiểm tra
bằng HTTPS từ bên ngoài.

## Kiểm tra nhanh (mọi người đều làm được)

```bash
curl -sS https://tamde-stag.tinhgon.xyz/gateway-healthz
curl -sS https://tamde-stag.tinhgon.xyz/v1/hello
curl -sS -X POST https://tamde-stag.tinhgon.xyz/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"0901234567"}'
```

| Kết quả | Ý nghĩa |
|---------|---------|
| `/gateway-healthz` trả **HTML** (Flutter) | Traefik/nginx phía trước **chỉ serve static**, chưa route `/v1` tới API |
| `/v1/hello` trả JSON `Gas Tam Đệ API Gateway` | API OK |
| POST OTP trả `ok` + `dev_code` (stag) | Đăng nhập OTP OK |

## Trên platform (không SSH)

1. **Deploy = Docker Compose** từ repo, nhánh **`stag`**, không chỉ “static website”.
2. **Environment**: copy key từ `deploy/.env.vps.example` (không dùng full `.env.example`).
   - **Không** set `SERVICE=`, `API_BASE_URL`, `GATEWAY_ADDR=127.0.0.1`, `NATS_URL=127.0.0.1`.
3. **Redeploy** sau mỗi merge vào `stag` (đợi CI push image `:stag`).
4. **Ports Exposes / labeledPort**: **8080** → service **`web`** (theo compose hiện tại).
5. Xem **log deploy** trên UI: phải pull/start **api-gateway**, **auth-service**, **nats**, không chỉ web.
6. Nếu vẫn lỗi: mở ticket / chạy lại **Health Check** / **Redeploy** trên dashboard Cursor Cloud.

## Fix repo (Traefik)

Label **api-gateway**: `PathPrefix(/v1)` + `/gateway-healthz` priority cao hơn static web,
để OTP hoạt động cả khi nginx phía trước không proxy `/v1`.

## Fix repo (embedded gateway trong image `web`)

Image **`web:stag`** chạy **api-gateway trên 127.0.0.1:8081** trong cùng container với nginx
(không cần container `api-gateway` riêng). Vẫn cần **`auth-service` + `nats`** trong compose
để OTP hoạt động.

Sau merge: đợi workflow **web-image.yml** push `:stag` mới → **Redeploy** trên dashboard.

## Domain staging

- https://tamde-stag.tinhgon.xyz/
