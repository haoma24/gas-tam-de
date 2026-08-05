# Fix GCP stag SSH auth — `unable to authenticate, attempted methods [none publickey]`

- **Thư mục:** `docs/workdocs_fix_gcp_ssh_auth_05082026`
- **Ngày:** 05/08/2026
- **Loại:** fix
- **Liên quan:** Deploy → GCP stag / web-image `deploy-gcp`

## Mục tiêu

Sửa lỗi CD SSH vào GCP VM:

```text
ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain
```

Job đã tới được host (không phải timeout) nhưng **private key bị từ chối**.

## Phạm vi

- Trong scope: normalize/validate secret key trong CI; hướng dẫn operator sửa secrets + `authorized_keys`.
- Ngoài scope: không thể ghi GitHub Secrets / SSH vào VM từ agent (cần maintainer).

## Chẩn đoán (từ log GHA)

| Run | Thời điểm | Triệu chứng | Kết luận |
|-----|-----------|-------------|----------|
| `31019115236` | ~16:15 | `INPUT_HOST` / `USER` / `KEY` **trống** → `missing server host` | Secrets chưa set |
| `31025224189` | ~16:41 | `INPUT_*` đã mask `***` nhưng handshake publickey fail | Secret có giá trị nhưng **key không khớp** / format sai / RSA bị sshd từ chối |

## Quyết định chính

1. Thêm `scripts/ci-normalize-ssh-key.sh` — fail sớm nếu secret thiếu; hỗ trợ literal `\n`; cảnh báo RSA PEM.
2. Composite action `.github/actions/ssh-deploy-gcp` — dùng `key_path` sau khi normalize; hỗ trợ `GCP_VM_SSH_PASSPHRASE`.
3. Operator phải đảm bảo **ed25519** private key trong secret khớp public key trên VM.

## Việc maintainer phải làm (bắt buộc để job xanh)

### 1. Tạo key deploy (máy local, không passphrase hoặc nhớ passphrase)

```bash
ssh-keygen -t ed25519 -a 200 -f ./gha-gcp-stag -C "github-actions@gas-tam-de" -N ""
```

### 2. Cài public key lên GCP VM

User phải **trùng** secret `GCP_VM_USER` (thường `ubuntu` trên image Ubuntu GCP):

```bash
# Cách A — đã SSH được bằng key khác:
ssh -i <key-hien-co> ubuntu@<GCP_VM_HOST> \
  'mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys' \
  < ./gha-gcp-stag.pub

# Cách B — GCP metadata (project/instance SSH keys):
# Console → Compute Engine → VM → Edit → SSH Keys → thêm dòng trong .pub
```

Nếu VM bật **OS Login**, metadata `authorized_keys` có thể bị bỏ qua — thêm key qua OS Login hoặc tắt OS Login cho user deploy.

### 3. Ghi GitHub Secrets

Repo → **Settings → Secrets and variables → Actions**:

| Secret | Giá trị |
|--------|---------|
| `GCP_VM_HOST` | External IP / DNS VM |
| `GCP_VM_USER` | `ubuntu` (hoặc user thật trên VM) |
| `GCP_VM_SSH_KEY` | **Toàn bộ** file `gha-gcp-stag` (private), multi-line, gồm `BEGIN`/`END` |
| `GCP_VM_SSH_PASSPHRASE` | chỉ khi key có passphrase |

**Không** paste file `.pub`. **Không** bọc thêm dấu `"..."`. Prefer paste multi-line (GitHub UI hỗ trợ); nếu one-line thì dùng `\n` giữa các dòng — script CI sẽ normalize.

### 4. Verify local trước khi re-run workflow

```bash
ssh -i ./gha-gcp-stag -o IdentitiesOnly=yes "$GCP_VM_USER@$GCP_VM_HOST" 'echo ok && whoami && hostname'
```

Rồi **Actions → Deploy → GCP stag → Run workflow**, hoặc push lại `stag`.

## Đã làm (trong repo)

- [x] Script normalize/validate key
- [x] Composite action SSH deploy
- [x] Wire `deploy-gcp-stag.yml` + `web-image.yml`
- [x] Workdocs + CHANGESLOG

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `scripts/ci-normalize-ssh-key.sh` | added | validate + `\n` normalize |
| `.github/actions/ssh-deploy-gcp/action.yml` | added | composite |
| `.github/workflows/deploy-gcp-stag.yml` | modified | dùng composite |
| `.github/workflows/web-image.yml` | modified | dùng composite |

## Cách verify

1. Secret đúng + `ssh -i …` local OK.
2. Re-run deploy workflow → bước "Normalize & validate SSH key" in `type=OPENSSH`.
3. Bước SSH deploy chạy `git pull` / `compose up` trên VM.

## Ghi chú / blocker

- Agent **không** đọc/ghi được GitHub Actions secrets (API 403) — unblock phụ thuộc maintainer paste key đúng.
- drone-ssh / golang `x/crypto/ssh` hay từ chối RSA `ssh-rsa` trên Ubuntu 20.04+ → dùng **ed25519**.
