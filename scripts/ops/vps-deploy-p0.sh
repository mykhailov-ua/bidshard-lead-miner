#!/usr/bin/env bash
# P0 stack: sync, BidShard ICP env, mongo+parser+crm-bot, cron telegram scrape (no realtime).
#
# Usage:
#   make vps-deploy-p0
#
# Optional local secrets (gitignored):
#   config/env/.env.telegram-alert.local  -> appended on VPS if present
#   config/env/.env.crm-telegram.local    -> appended on VPS if present
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"

log() { printf 'vps-deploy-p0: %s\n' "$*"; }

log "sync -> $(vps_ssh_target):${VPS_REMOTE_DIR}"
vps_rsync_push "$ROOT"
vps_post_sync_fix

if [[ -f "$ROOT/.env" ]]; then
	log "sync telegram secrets (tokens only; chat ids stay on VPS)"
	bash "$ROOT/scripts/ops/vps-sync-telegram-secrets.sh"
fi

log "apply P0 env"
bash "$ROOT/scripts/ops/vps-apply-p0-env.sh"

# Optional local secret overlays (never rsynced); merge keys into .env (no append duplicates).
for overlay in .env.telegram-alert.local .env.crm-telegram.local; do
	local_path="$ROOT/config/env/$overlay"
	if [[ -f "$local_path" ]]; then
		log "overlay $overlay"
		# shellcheck disable=SC2086
		rsync -az -e "ssh $(vps_ssh_opts)" "$local_path" "$(vps_ssh_target):${VPS_REMOTE_DIR}/config/env/$overlay"
		vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
while IFS= read -r line || [[ -n \"\$line\" ]]; do
  [[ -z \"\$line\" || \"\$line\" =~ ^# ]] && continue
  k=\"\${line%%=*}\"; v=\"\${line#*=}\"
  [[ -z \"\$k\" ]] && continue
  tmp=\"\$(mktemp)\"
  grep -v \"^\${k}=\" .env >\"\$tmp\" 2>/dev/null || true
  printf '%s=%s\n' \"\$k\" \"\$v\" >>\"\$tmp\"
  mv \"\$tmp\" .env
done <'config/env/${overlay}'"
	fi
done

log "docker compose up (mongo parser crm-bot)"
vps_remote_up "mongo parser crm-bot"

log "telegram cron-only (stop realtime)"
vps_remote_telegram_realtime_stop

if [[ -f "$ROOT/sessions.pool.json" ]]; then
	log "sync session pool"
	bash "$ROOT/scripts/ops/vps-sync-session-pool.sh"
fi

log "install telegram pain cron"
bash "$ROOT/scripts/ops/vps-install-telegram-cron.sh"

log "status"
vps_ssh "cd '${VPS_REMOTE_DIR}' && docker compose ps && crontab -l 2>/dev/null | grep telegram-pain-cron || true"

log "ok P0 deploy complete (cron-only)"
log "If CRM/alert bots silent: set config/env/.env.crm-telegram.local and .env.telegram-alert.local locally, re-run make vps-deploy-p0"
