# CD staging: gộp deploy dùng chung, preflight SSH, health check chặn

- **Thư mục:** `docs/workdocs_cd_stag_deploy_hardening_05082026`
- **Ngày:** 05/08/2026
- **Loại:** ci/cd
- **Liên quan:** nối tiếp PR #23 → #27 (chuỗi CD `stag`); run fail `31025224189`

## Mục tiêu

Push cuối vào `stag` (16:25) fail ở bước **SSH deploy** với đúng một dòng:

```
ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey]
```

Thông báo đó không phân biệt được *secret sai/thiếu* với *key chưa nằm trên VM*,
nên mỗi lần fail lại phải đoán. Ngoài ra khi rà workflow phát hiện thêm hai vấn
đề nghiêm trọng hơn:

1. `deploy-gcp-stag.yml` **không bao giờ chạy**. Sau PR #27 nó chỉ còn trigger
   `workflow_run` + `workflow_dispatch`, mà GitHub chỉ kích hoạt hai event này
   khi file workflow có trên **nhánh mặc định** (`master`). File chỉ có trên
   `stag` ⇒ thay đổi thuần Go merge vào `stag` **không được deploy**.
2. Health check sau deploy chạy `curl ... || echo " NOT OK"` nên **luôn exit 0**.
   Một lần deploy làm chết cả stack vẫn báo xanh.

## Phạm vi

- Trong scope: workflow CI/CD `stag`, runbook deploy, CHANGESLOG/workdocs.
- Ngoài scope: **không** đụng tới giá trị secret hay cấu hình VM (không có
  quyền); không đổi compose / nginx / code service.

## Quyết định chính

- **Một nguồn deploy duy nhất:** `.github/workflows/deploy-stag.reusable.yml`
  (`workflow_call`). `web-image.yml` và `backend-ci.yml` cùng gọi nó, hết cảnh
  hai bản script SSH giống hệt nhau trôi lệch.
- **Chain deploy vào chính workflow build** thay vì workflow rời — đây là cách
  duy nhất chạy được khi cấu hình chỉ sống trên `stag`. Xoá
  `deploy-gcp-stag.yml`.
- **`concurrency: deploy-stag`** (không cancel) để hai job build cùng push
  không chạy `docker compose up` chồng nhau.
- **Preflight tách lỗi client-side khỏi lỗi server-side**: rỗng / CRLF / dán
  nhầm public key / key có passphrase đều báo lỗi cụ thể ngay trên runner. Key
  hợp lệ thì in **fingerprint** (an toàn) để đối chiếu với VM.
- **Health check chặn thật**: poll tối đa 30 lần × 5s; fail thì in
  `docker compose ps -a` + 80 dòng log cuối rồi `exit 1`.
- **`workflow_dispatch`** cho cả hai workflow ⇒ chạy lại deploy không cần commit
  rỗng. Kéo theo: điều kiện push image đổi từ `event_name == 'push'` sang
  `!= 'pull_request'`, nếu không dispatch sẽ build mà không push rồi deploy image cũ.

## Đã làm

- [x] `.github/workflows/deploy-stag.reusable.yml` — preflight + deploy + health gate
- [x] `web-image.yml` — job `deploy-gcp` gọi reusable, thêm `workflow_dispatch`
- [x] `backend-ci.yml` — thêm job `deploy-gcp` + `workflow_dispatch`
- [x] Xoá `deploy-gcp-stag.yml` (trigger chết)
- [x] `docs/runbook-deploy-stag.md` — chẩn đoán `[none publickey]` trên GCP
- [x] CHANGESLOG entry
- [x] Verify: `actionlint` sạch; `bash -n` mọi script nhúng; chạy thử preflight 6 case

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `.github/workflows/deploy-stag.reusable.yml` | added | Deploy dùng chung |
| `.github/workflows/web-image.yml` | modified | Gọi reusable + dispatch |
| `.github/workflows/backend-ci.yml` | modified | Job deploy + dispatch + điều kiện push image |
| `.github/workflows/deploy-gcp-stag.yml` | deleted | Trigger không bao giờ fire |
| `docs/runbook-deploy-stag.md` | added | Runbook CD staging |
| `CHANGESLOG.md` | modified | Entry |

## Cách verify

1. `actionlint` → không lỗi.
2. Preflight tự kiểm (chạy được ngoài CI):

```bash
python3 - <<'PY'
import yaml
d = yaml.safe_load(open('.github/workflows/deploy-stag.reusable.yml'))
s = [x for x in d['jobs']['deploy']['steps'] if x['name'].startswith('Preflight')][0]['run']
open('/tmp/preflight.sh','w').write(s)
PY
ssh-keygen -q -t ed25519 -N '' -f /tmp/k
VM_HOST=1.2.3.4 VM_USER=ubuntu VM_KEY="$(cat /tmp/k)"     bash /tmp/preflight.sh  # exit 0 + fingerprint
VM_HOST=1.2.3.4 VM_USER=ubuntu VM_KEY="$(cat /tmp/k.pub)" bash /tmp/preflight.sh  # exit 1 "holds a public key"
```

3. Push vào `stag` → Actions: job **Deploy to GCP stag** phải in fingerprint ở
   bước Preflight trước khi thử SSH.

## Ghi chú / blocker

- **Không sửa được nguyên nhân gốc từ repo.** Key bị VM từ chối là chuyện
  secret + cấu hình GCP; agent không có quyền đọc/ghi secret của repo
  (`gh secret list` → 403) và không có đường vào VM. Việc cần người làm nằm ở
  `docs/runbook-deploy-stag.md` — nghi ngờ lớn nhất là **OS Login** đang bật
  (khi đó metadata `ssh-keys` bị bỏ qua hoàn toàn và `GCP_VM_USER` phải là
  username OS Login, không phải `ubuntu`).
- Sau khi key thông, lần deploy tới có thể fail ở **health check** — đó là hành
  vi mới, cố ý: trước đây lỗi này bị nuốt.
