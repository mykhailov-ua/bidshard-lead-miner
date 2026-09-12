#!/usr/bin/env bash
# Quick VPS stack health (crawler + telegram). No secrets printed.
#
# Usage:
#   bash scripts/ops/vps-status.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
echo '=== docker ==='
docker compose ps
docker compose -f docker-compose.telegram-realtime.yaml ps 2>/dev/null || true
echo
echo '=== env (non-secret keys) ==='
grep -E '^(PARSER_SOURCE|TELEGRAM_REALTIME|PARSER_BG_|TELEGRAM_ALERT_ENABLED|TELEGRAM_ALERT_CHANNEL|CRM_TELEGRAM_ALLOWED_CHAT_IDS|PARSER_CRM_WEBHOOK)=' .env 2>/dev/null || true
echo
echo '=== parser health ==='
docker compose exec -T parser parser config check 2>&1 | tail -3
echo
echo '=== recent parser log ==='
docker compose logs parser --tail=3 2>&1
echo
echo '=== recent telegram realtime ==='
docker compose -f docker-compose.telegram-realtime.yaml logs parser-telegram-realtime --tail=3 2>&1 || true
"

printf 'vps-status: ok\n'
