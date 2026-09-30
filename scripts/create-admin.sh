#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd -- "$script_dir/.." && pwd)"
if [[ -f /etc/tkdhub.env ]]; then
    set -a
    source /etc/tkdhub.env
    set +a
fi
if [[ ! -x "$repo_dir/bin/create-admin" ]]; then
    echo 'Build bin/create-admin first; see docs/admin-provisioning.md.' >&2
    exit 1
fi
read -r -p 'Admin email: ' admin_email
read -r -s -p 'Password (10-72 UTF-8 bytes): ' admin_password
printf '\n'
read -r -s -p 'Confirm password: ' admin_confirm
printf '\n'
trap 'unset admin_password admin_confirm' EXIT
if [[ "$admin_password" != "$admin_confirm" ]]; then
    echo 'Passwords do not match.' >&2
    exit 1
fi
printf '%s' "$admin_password" | "$repo_dir/bin/create-admin" -email "$admin_email" "$@"
