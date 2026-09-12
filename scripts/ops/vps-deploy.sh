#!/usr/bin/env bash
# Push repo to VPS and rebuild/restart parser stack.
#
# Usage:
#   make vps-deploy
#   bash scripts/ops/vps-deploy.sh
#   VPS_SYNC_ONLY=1 bash scripts/ops/vps-deploy.sh
#   VPS_SERVICES="mongo parser crm-bot" bash scripts/ops/vps-deploy.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"

log() { printf 'vps-deploy: %s\n' "$*"; }

log "sync -> $(vps_ssh_target):${VPS_REMOTE_DIR}"
vps_rsync_push "$ROOT"
vps_post_sync_fix

if [[ "${VPS_SYNC_ONLY:-}" == "1" ]]; then
	log "ok sync only (VPS_SYNC_ONLY=1)"
	exit 0
fi

services="${VPS_SERVICES:-mongo parser crm-bot}"
log "remote up (${services})"
vps_remote_up "$services"
if [[ "${VPS_SKIP_TELEGRAM_REALTIME:-}" != "1" ]]; then
	log "telegram realtime profile"
	vps_remote_telegram_realtime
fi
log "ok"
