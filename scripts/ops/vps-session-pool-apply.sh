#!/usr/bin/env bash
# Apply 3-session pool on VPS: stop stale TG jobs, sync sessions, health, cron.
#
# Usage:
#   bash scripts/ops/vps-session-pool-apply.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

log() { printf 'vps-session-pool-apply: %s\n' "$*"; }

log "rsync repo"
vps_rsync_push "$ROOT"
vps_post_sync_fix

log "stop stale telegram jobs (history-export / stuck cron)"
vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
pkill -f 'history_export' 2>/dev/null || true
pkill -f 'telegram-pain-cron' 2>/dev/null || true
pkill -f 'parser telegram' 2>/dev/null || true
sleep 2
docker ps -q --filter name=lead-intent-processor-parser-run | xargs -r docker stop 2>/dev/null || true
rm -f var/telegram-scrape-*.lock
docker compose up -d mongo parser crm-bot 2>&1 | tail -3
"

log "sync session files from local pool"
bash "$ROOT/scripts/ops/vps-sync-session-pool.sh"

log "session health"
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; bash scripts/ops/telegram-session-pool-health.sh" || true

log "repair only unauthorized hot slots (keeps distinct accounts when sync ok)"
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; bash scripts/ops/telegram-session-pool-repair.sh" || true

log "install staggered per-session cron"
bash "$ROOT/scripts/ops/vps-install-telegram-cron.sh"

log "cron preview"
vps_ssh "crontab -l | grep -E 'telegram-pain-cron|buyer-discover'"

log "ok"
