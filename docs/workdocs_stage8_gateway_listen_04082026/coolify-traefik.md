# Stage 8 Unreachable labeledPort=8080 — Coolify / Traefik

## Lỗi thật (07:31)

```text
[HEALTHCHECK FAILED] cause=Unreachable labeledPort=8080
→ App KHÔNG lắng nghe (không nối được) trên cổng Traefik (8080).
── Log container (cuối) ──
(empty)
```

## Root cause

Coolify gắn Traefik vào **network do Coolify tạo**. Compose cũ định nghĩa
`networks.gastamde` và gắn mọi service vào đó → container nằm trên **hai**
network. Traefik chỉ có mặt trên network Coolify nhưng đôi khi resolve IP
của `gastamde` → **Unreachable** dù process đang listen `0.0.0.0:8080`.

Đây là anti-pattern được Coolify docs cảnh báo rõ
(“Do Not Define Custom Networks”).

## Fix

1. Xóa toàn bộ `networks:` (per-service + top-level `gastamde`).
2. Label api-gateway:
   `traefik.http.services.api-gateway.loadbalancer.server.port=8080`
3. Giữ `PORT=8080` / `GATEWAY_ADDR=0.0.0.0:8080`.

## Coolify UI

- Service public / Ports Exposes = **8080** (api-gateway), không phải web:80.
- Sau deploy: Traefik phải reach được container IP trên network Coolify.

## CI

`secrets.GHCR_WRITE_TOKEN != ''` trong `if:` làm workflow **không parse được**
→ CI fail 0s, image `:stag` không được push. Đã chuyển check token vào
bên trong `run:` script.
