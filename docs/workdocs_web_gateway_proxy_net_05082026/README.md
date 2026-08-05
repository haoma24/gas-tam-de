# Fix OTP api_unavailable — web ↔ api-gateway trên tensorship-net

- **Thư mục:** `docs/workdocs_web_gateway_proxy_net_05082026`
- **Ngày:** 05/08/2026
- **Loại:** fix deploy

## Triệu chứng

Deploy Stage 5/8 OK, UI web mở được, nhưng **Gửi OTP** → «API gateway không sẵn sàng»
(`api_unavailable` từ nginx).

## Nguyên nhân

nginx trong container `web` proxy `/v1/*` tới `http://api-gateway:8080` qua DNS
Docker. Traefik/Coolify thường chỉ gắn **`web`** lên `tensorship-net`; nếu
**api-gateway** chỉ ở network default của compose, container `web` (chủ yếu trên
proxy net) **không resolve / không connect** được `api-gateway` → `@api_unavailable`.

## Quyết định

- Khai báo `networks.tensorship-net` (external) và gắn **mọi service** vào
  `default` + `tensorship-net`.
- Healthcheck `web` → `/gateway-healthz` (proxy thật tới gateway).
- `scripts/vps-api-diagnose.sh`, `make vps-api-diagnose`.

## Verify

```bash
docker network create tensorship-net  # local
docker compose -f deploy/docker-compose.yml -p vps-test up -d --no-build
COMPOSE_PROJECT_NAME=vps-test ./scripts/vps-api-diagnose.sh
```

## File

| Path | Ghi chú |
|------|---------|
| `deploy/docker-compose.yml` | networks + web healthcheck |
| `Makefile` | ensure-proxy-net, vps-api-diagnose |
| `scripts/vps-api-diagnose.sh` | mới |
| `scripts/vps-compose-up.sh` | tạo net + net-check |
