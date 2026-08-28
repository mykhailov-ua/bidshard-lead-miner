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

if [[ "${VPS_SYNC_ONLY:-}" == "1" ]]; then
	log "ok sync only (VPS_SYNC_ONLY=1)"
	exit 0
fi

services="${VPS_SERVICES:-mongo parser}"
log "remote up (${services})"
vps_remote_up "$services"
log "ok"
