# Runbook — CD staging (GitHub Actions → GCP VM)

Áp dụng cho nhánh **`stag`**. Đích: VM GCP, thư mục `/opt/gas-tam-de`,
compose project `gas-tamde-stag`, domain https://tamde-stag.tinhgon.xyz/.

## Chuỗi CD

| Push chạm tới | Workflow build | Job deploy |
|---------------|----------------|------------|
| `apps/mobile/**`, `Dockerfile.web`, `nginx.web.conf` | `web-image.yml` | `deploy-gcp` (`needs: build-push`) |
| `services/**`, `pkg/**`, `go.*`, compose, `.env*` | `backend-ci.yml` | `deploy-gcp` (`needs: build-push`) |

Cả hai job đều gọi cùng một reusable workflow
`.github/workflows/deploy-stag.reusable.yml`. Nếu một push chạm cả hai nhóm
file, hai job cùng xếp hàng nhưng `concurrency: deploy-stag` bảo đảm chỉ **một**
`docker compose up` chạy tại một thời điểm.

> **Không** tách deploy ra workflow riêng dùng `workflow_run` /
> `workflow_dispatch`: GitHub chỉ kích hoạt hai event đó khi file workflow có
> mặt trên **nhánh mặc định** (`master`). Cấu hình deploy chỉ sống trên `stag`,
> nên workflow rời (`deploy-gcp-stag.yml` cũ) không bao giờ chạy.

## Chạy lại deploy không cần commit

Actions → **Backend CI — build & push images** (hoặc **Web image — build &
push**) → *Run workflow* → chọn nhánh `stag`. Job deploy có bật
`workflow_dispatch` nên sẽ build lại image `:stag` rồi deploy.

## Secrets

| Secret | Giá trị |
|--------|---------|
| `GCP_VM_HOST` | IP ngoài hoặc DNS của VM |
| `GCP_VM_USER` | user SSH — phải sở hữu `/opt/gas-tam-de` và thuộc group `docker` |
| `GCP_VM_SSH_KEY` | **private key** dạng PEM/OpenSSH, **không passphrase**, kết newline |

Bước *Preflight* in ra fingerprint của key (không lộ key) — dùng nó để đối chiếu
với key đã đăng ký trên VM.

## Lỗi `ssh: handshake failed ... [none publickey]`

Preflight đã pass ⇒ secret hợp lệ về mặt định dạng; **VM từ chối key**. Trên GCP
có ba nguyên nhân, theo thứ tự hay gặp:

### 1. Key chưa đăng ký qua metadata

Sửa tay `~/.ssh/authorized_keys` trên GCP **không bền**: guest agent ghi đè file
đó từ instance metadata. Đăng ký đúng cách:

```bash
# keys.txt: "<user>:<nội dung file .pub>"  (một dòng)
printf '%s:%s\n' "$VM_USER" "$(cat deploy_key.pub)" > keys.txt
gcloud compute instances add-metadata <vm-name> \
  --zone <zone> --metadata-from-file ssh-keys=keys.txt
```

`add-metadata` **ghi đè** toàn bộ `ssh-keys`; đọc key hiện có trước khi ghi:

```bash
gcloud compute instances describe <vm-name> --zone <zone> \
  --format="value(metadata.items.filter(key:ssh-keys).extract(value))"
```

### 2. Instance bật OS Login

Khi `enable-oslogin=TRUE` (metadata project hoặc instance), GCP **bỏ qua hoàn
toàn** `ssh-keys` metadata.

```bash
gcloud compute os-login ssh-keys add --key-file=deploy_key.pub
gcloud compute os-login describe-profile --format='value(posixAccounts.username)'
```

Đặt `GCP_VM_USER` = username OS Login vừa in ra (dạng `sa_...` hoặc
`<user>_<domain>`), **không** phải `ubuntu`.

### 3. Sai user hoặc sai host

Key đúng nhưng gắn cho user khác thì server vẫn trả `publickey`. Kiểm tra nhanh
từ máy có quyền:

```bash
ssh -i deploy_key -o IdentitiesOnly=yes -v "$VM_USER@$VM_HOST" true
ssh-keygen -lf deploy_key.pub    # so với fingerprint in trong log Actions
```

## Deploy chạy nhưng job fail ở health check

Job **cố ý fail** khi stack không xanh (trước đây chỉ in `NOT OK` rồi báo
success). Log đã kèm `docker compose ps -a` + 80 dòng log cuối. Trên VM:

```bash
cd /opt/gas-tam-de
export COMPOSE_PROJECT_NAME=gas-tamde-stag PROXY_NETWORK=tensorship-net
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.local.yml \
  --env-file deploy/.env -p "$COMPOSE_PROJECT_NAME" ps -a
curl -sv http://127.0.0.1:80/gateway-healthz
```

Các lỗi thường gặp và cách xử lý nằm ở README mục *«Khi deploy báo
`HEALTHCHECK FAILED cause=NotOnNet`»* và `make vps-api-diagnose`.

## Không có SSH

Xem `docs/workdocs_vps_deploy_khong_ssh_05082026/` — quy trình redeploy qua UI
platform khi VM không mở SSH.
