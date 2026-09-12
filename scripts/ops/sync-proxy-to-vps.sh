#!/usr/bin/env bash
# Copy residential proxy.list to VPS (never commit proxy.list).
#
# Usage:
#   cp proxy.txt config/env/proxy.list
#   bash scripts/ops/sync-proxy-to-vps.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

LOCAL_LIST="${1:-$ROOT/config/env/proxy.list}"
if [[ ! -f "$LOCAL_LIST" ]]; then
	if [[ -f "$ROOT/proxy.txt" ]]; then
		printf 'sync-proxy: copying proxy.txt -> config/env/proxy.list\n'
		cp "$ROOT/proxy.txt" "$ROOT/config/env/proxy.list"
		LOCAL_LIST="$ROOT/config/env/proxy.list"
	else
		printf 'sync-proxy: missing %s (and no proxy.txt)\n' "$LOCAL_LIST" >&2
		exit 1
	fi
fi

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

REMOTE_LIST="${VPS_REMOTE_DIR}/config/env/proxy.list"
vps_ssh "mkdir -p '${VPS_REMOTE_DIR}/config/env'"
# shellcheck disable=SC2086
rsync -az -e "ssh $(vps_ssh_opts)" "$LOCAL_LIST" "$(vps_ssh_target):${REMOTE_LIST}"
# Parser container runs as UID 10001; rsync uploads as root.
vps_ssh "chown 10001:10001 '${REMOTE_LIST}'; chmod 640 '${REMOTE_LIST}'"
lines="$(wc -l <"$LOCAL_LIST")"
printf 'sync-proxy: uploaded %s lines to %s:%s\n' "$lines" "$(vps_ssh_target)" "$REMOTE_LIST"
