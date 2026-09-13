#!/usr/bin/env bash
# Install cron-only telegram pain scrape on VPS (no parser-telegram-realtime).
#
# Usage:
#   bash scripts/ops/vps-install-telegram-cron.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

POOL="${SESSION_POOL_FILE:-$ROOT/sessions.pool.json}"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

if [[ ! -f "$POOL" ]]; then
	printf 'vps-install-telegram-cron: missing %s\n' "$POOL" >&2
	exit 1
fi

CRON_BLOCK="$(python3 "$ROOT/scripts/ops/telegram-cron-schedule.py" "$POOL" "$VPS_REMOTE_DIR")"

vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; mkdir -p var"
printf '%s\n' "$CRON_BLOCK" | vps_ssh "set -euo pipefail
tmp=\$(mktemp)
crontab -l 2>/dev/null | grep -v 'lead-intent-processor telegram cron' | grep -v 'scripts/ops/telegram-pain-cron.sh' | grep -v 'scripts/ops/buyer-discover.sh' >\"\$tmp\" || true
cat >>\"\$tmp\"
crontab \"\$tmp\"
rm -f \"\$tmp\"
crontab -l | tail -20
"

printf 'vps-install-telegram-cron: ok\n'
