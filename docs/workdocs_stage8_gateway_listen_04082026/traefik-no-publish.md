# Stage 8 Unreachable — bỏ `ports:` publish trên api-gateway

## Lỗi (08:13)

```text
FAILED: Health check failed: Unreachable — App KHÔNG lắng nghe
(không nối được) trên cổng Traefik (8080).
```

(Đã qua NotOnNet; lỗi quay lại Unreachable.)

## Root cause

Traefik forum / Docker provider: khi service **publish host port**
(`ports: "8080:8080"`), container có thêm endpoint trên mạng publish/ingress.
Traefik (kể cả khi đã set `traefik.docker.network`) vẫn có thể chọn IP không
route được → `Unreachable`, dù process đang `Listen 0.0.0.0:8080`.

Cách họ fix: **bỏ `ports`**, chỉ để Traefik nối qua Docker network + label
`loadbalancer.server.port`.

## Fix

| Thay đổi | Lý do |
|----------|--------|
| Xóa `ports: 8080:8080` trên api-gateway | Không còn endpoint publish gây lệch IP |
| `expose: ["8080"]` | Ghi nhận cổng container cho tooling |
| `traefik.docker.network=tensorship-net` literal | Tránh `${VAR}` không expand trên platform |
| Bỏ toàn bộ `depends_on` của gateway | Listen ngay, không chờ 8 backend |
| DB lỗi → vẫn serve `/healthz` | TCP :8080 không chết vì SQLite |
| `docker-compose.local.yml` | Local DX vẫn map `:8080`/`:8090`/NATS |

## Verify local

```bash
docker compose -f deploy/docker-compose.yml config | grep -A2 'api-gateway' 
# ports published của gateway phải trống; labels có tensorship-net

make compose-up   # merges docker-compose.local.yml → curl :8080/healthz OK
```

## VPS

Redeploy **không** kèm file `.local.yml`. Sau deploy:

```bash
./scripts/vps-net-check.sh --fix
docker inspect <api-gateway> --format '{{json .NetworkSettings.Networks}}'
# phải có tensorship-net; Traefik dùng IP đó + port 8080
```
