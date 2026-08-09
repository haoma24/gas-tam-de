# Fix kết nối trừ tồn kho khi đặt hàng

- **Thư mục:** `docs/workdocs_fix_ket_noi_tru_ton_kho_09082026`
- **Ngày:** 09/08/2026
- **Loại:** fix
- **Liên quan:** lỗi đặt hàng trên staging

## Mục tiêu

Khôi phục luồng đặt hàng để `order-service` gọi đúng `inventory-service` và trừ tồn kho.

## Phạm vi

- Trong scope: cấu hình kết nối nội bộ order → inventory trên Docker Compose; test hồi quy.
- Ngoài scope: thay đổi nghiệp vụ tính tồn kho và giao diện đặt hàng.

## Quyết định chính

- Truyền rõ `INVENTORY_SERVICE_URL=http://inventory-service:8085` cho `order-service`.
- Chờ healthcheck của inventory trước khi khởi động order để tránh lỗi kết nối lúc stack vừa lên.
- Khóa giá trị URL bằng test deploy, vì fallback `127.0.0.1:8085` chỉ đúng khi chạy service trực tiếp ngoài container.

## Đã làm

- [x] Xác định order container đang gọi localhost do thiếu biến môi trường.
- [x] Bổ sung URL nội bộ và dependency healthcheck.
- [x] Thêm test hồi quy cho cấu hình.

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `deploy/docker-compose.yml` | modified | Nối order-service tới inventory-service |
| `deploy/compose_env_test.go` | modified | Chặn thiếu/sai URL inventory |
| `CHANGESLOG.md` | modified | Ghi nhận bugfix |

## Cách verify

1. Chạy `go test ./deploy`.
2. Chạy `docker compose -f deploy/docker-compose.yml config`.
3. Sau deploy, đặt một đơn có tồn kho và xác nhận tồn giảm tương ứng.

## Ghi chú / blocker

- Cần deploy lại image/compose nhánh `stag` để cấu hình có hiệu lực trên VPS.
