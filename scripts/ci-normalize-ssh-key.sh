#!/usr/bin/env bash
# Normalize a GitHub Actions SSH private-key secret into a usable key file.
#
# Accepts either real newlines or literal "\n" sequences (common paste mistake
# when storing multi-line keys in GitHub Secrets).
#
# Usage:
#   GCP_VM_HOST=... GCP_VM_USER=... GCP_VM_SSH_KEY='...' \
#     ./scripts/ci-normalize-ssh-key.sh /path/to/out.key
set -euo pipefail

out="${1:?usage: $0 <output-key-path>}"

missing=()
[[ -n "${GCP_VM_HOST:-}" ]] || missing+=("GCP_VM_HOST")
[[ -n "${GCP_VM_USER:-}" ]] || missing+=("GCP_VM_USER")
[[ -n "${GCP_VM_SSH_KEY:-}" ]] || missing+=("GCP_VM_SSH_KEY")
if ((${#missing[@]})); then
  echo "::error::Missing GitHub Actions secrets: ${missing[*]}"
  echo "Set them under Settings → Secrets and variables → Actions."
  echo "See docs/workdocs_fix_gcp_ssh_auth_05082026/README.md"
  exit 1
fi

mkdir -p "$(dirname "${out}")"
umask 077

# Python: reliable handling of literal \n, CR, wrapping quotes, trailing newline.
python3 - "${out}" <<'PY'
import os, sys
from pathlib import Path

out = Path(sys.argv[1])
raw = os.environ["GCP_VM_SSH_KEY"]

# Strip a single pair of wrapping quotes.
if len(raw) >= 2 and raw[0] == raw[-1] and raw[0] in ("'", '"'):
    raw = raw[1:-1]

raw = raw.replace("\r\n", "\n").replace("\r", "\n")

# Expand literal \n when the secret was stored as a single line.
if "\\n" in raw and raw.count("\n") < 2:
    print("==> normalizing literal \\n sequences in GCP_VM_SSH_KEY")
    raw = raw.replace("\\n", "\n")

if not raw.endswith("\n"):
    raw += "\n"

out.write_text(raw)
out.chmod(0o600)
PY

if ! grep -qE 'BEGIN (OPENSSH |RSA |EC |DSA )?PRIVATE KEY' "${out}"; then
  if grep -qE '^ssh-(rsa|ed25519|ecdsa)|^-----BEGIN.*PUBLIC KEY' "${out}"; then
    echo "::error::GCP_VM_SSH_KEY looks like a *public* key (.pub). Store the private key instead."
  else
    echo "::error::GCP_VM_SSH_KEY does not look like a private key (missing BEGIN … PRIVATE KEY)."
    echo "Paste the private key (id_ed25519 / id_rsa), keep real newlines (multi-line secret)."
  fi
  echo "See docs/workdocs_fix_gcp_ssh_auth_05082026/README.md"
  exit 1
fi

lines="$(wc -l <"${out}" | tr -d ' ')"
if grep -q 'BEGIN OPENSSH PRIVATE KEY' "${out}"; then
  ktype="OPENSSH"
elif grep -q 'BEGIN RSA PRIVATE KEY' "${out}"; then
  ktype="PEM-RSA"
  echo "::warning::RSA PEM keys often fail with drone-ssh on modern OpenSSH servers."
  echo "Prefer: ssh-keygen -t ed25519 -a 200 -f gha-deploy -N ''"
else
  ktype="PEM"
fi

echo "==> SSH key ready: path=${out} lines=${lines} type=${ktype} user=${GCP_VM_USER} host=${GCP_VM_HOST}"
echo "==> Public half of this key must be in ${GCP_VM_USER}@${GCP_VM_HOST}:~/.ssh/authorized_keys"
echo "==> Prefer ed25519 keys (RSA ssh-rsa is often rejected by drone-ssh / modern sshd)."
