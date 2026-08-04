# Service không còn chờ NATS mới serve HTTP (+ `/readyz`)

- **Thư mục:** `docs/workdocs_nats_background_readyz_04082026`
- **Ngày:** 04/08/2026
- **Loại:** fix
- **Liên quan:** Deploy stag fail `container ts-tamde-stag-catalog-service-1 is unhealthy`

## Mục tiêu

`docker compose -p ts-tamde-stag up -d --no-build` fail:

```text
dependency failed to start: container ts-tamde-stag-catalog-service-1 is unhealthy
```

Làm cho container **luôn pass healthcheck ngay khi process serve HTTP**, kể cả
khi NATS chưa sẵn sàng — thay vì kéo cả `docker compose up` fail theo.

## Nguyên nhân

`main()` của catalog/order/inventory/billing/report gọi tuần tự:

1. `natsx.ConnectJS(natsURL)` — retry trong `NATS_STARTUP_TIMEOUT_SEC` (60s)
2. `natsx.EnsureStreams(js)` — retry thêm 60s nữa
3. *rồi mới* `httpx.ListenAndServe(...)` → mới có `/healthz`

⇒ Khi NATS chậm hoặc chưa xong JetStream restore, **HTTP server chưa từng chạy**,
`wget http://127.0.0.1:8082/healthz` refused → `unhealthy` trong `start_period`
90s; nếu quá budget thì `os.Exit(1)`. Bản chất: healthcheck (liveness) bị buộc
vào dependency bên ngoài (readiness).

## Quyết định chính

- Tách **liveness** (`/healthz`) khỏi **readiness** (`/readyz`).
  - `/healthz`: 200 ngay khi process serve; không phụ thuộc broker → dùng cho
    Docker healthcheck / `depends_on`.
  - `/readyz`: 200 khi mọi dependency OK, 503 + tên dependency lỗi.
- `natsx.Background`: kết nối JetStream ở goroutine, **retry vô hạn** (backoff
  1s → 30s). Không `os.Exit` vì NATS.
- Publisher nhận `natsx.JSProvider` (lazy) thay vì `nats.JetStreamContext`.
  Publish khi chưa ready → trả lỗi, và các call site vốn đã chỉ log (không fail
  request).
- Consumer (inventory `order.completed`, report `order.placed`/`order.completed`/
  `billing.debt.updated`) attach qua callback `Start(onReady)` khi JetStream lên;
  `onReady` lỗi → loop reconnect và thử lại.

## Đã làm

- [x] `pkg/natsx/background.go`: `JSProvider`, `Static`, `Background`, `ErrNotReady`
- [x] `pkg/httpx`: `MountReady` + `ReadyCheck`; comment rõ `/healthz` là liveness
- [x] 5 service main: bỏ chặn NATS, mount `/readyz`
- [x] Publisher catalog/order/billing dùng provider; test dùng `natsx.Static(js)`
- [x] Test mới: `TestBackgroundNotReadyBeforeBrokerArrives`,
      `TestBackgroundBecomesReadyWhenBrokerStartsLate`, `TestStaticProvider`,
      `TestMountHealthIgnoresDependencies`, `TestMountReadyOKWhenDependenciesPass`
- [x] `go test ./...` xanh

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `pkg/natsx/background.go` | added | lazy provider + retry vô hạn |
| `pkg/natsx/background_test.go` | added | broker lên trễ |
| `pkg/httpx/httpx.go` | modified | `MountReady`, `ReadyCheck` |
| `pkg/httpx/ready_test.go` | added | healthz vs readyz |
| `services/catalog-service/main.go` | modified | background NATS + `/readyz` |
| `services/order-service/main.go` | modified | background NATS + `/readyz` |
| `services/inventory-service/main.go` | modified | consumer attach async |
| `services/report-service/main.go` | modified | consumers attach async |
| `services/billing-service/main.go` | modified | background NATS + `/readyz` |
| `services/*/(product\|order\|billing)_events.go` | modified | dùng `JSProvider` |
| `services/*/(…)_events_test.go` | modified | `natsx.Static(js)` |

## Cách verify

1. `go test ./...`
2. NATS **không** tồn tại vẫn healthy:

```bash
docker run -d --name catalog-nonats -e NATS_URL=nats://127.0.0.1:4222 \
  --health-cmd='wget -qO- http://127.0.0.1:8082/healthz' \
  ts-gas-tam-de-catalog-service
# → health=healthy; /readyz → 503
```

3. NATS lên trễ → tự ready + consumer attach:
   log `jetstream consumer started …` rồi `nats ready … attempt=3`,
   `/readyz` chuyển 503 → `{"dependencies":{"nats":"ok"},"status":"ready"}`
4. Full stack: `docker compose -p ts-tamde-stag up -d` → 9/9 healthy,
   `/v1/products` trả `{"items":[]}`

## Ghi chú / blocker

- Deploy chạy `--no-build`, nên pipeline **phải build lại image** mới có fix này;
  image cũ vẫn chặn ở NATS.
- Healthcheck compose giữ `/healthz`. Nếu muốn gate traffic theo dependency thì
  dùng `/readyz` (ví dụ trong LB), đừng đưa vào healthcheck container.
