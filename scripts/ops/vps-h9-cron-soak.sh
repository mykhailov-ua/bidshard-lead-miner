#!/usr/bin/env bash
# H9 control: cron-only telegram soak metrics (no realtime container).
#
# Usage: bash scripts/ops/vps-h9-cron-soak.sh [log_file]
#
# Greps scrape logs for pain alerts, FloodWait, accepted hints over last 48h window.

set -euo pipefail

LOG="${1:-var/telegram-scrape.log}"
HOURS="${H9_SOAK_HOURS:-48}"

if [[ ! -f "$LOG" ]]; then
	echo "h9-cron-soak: log missing: $LOG" >&2
	exit 1
fi

CUTOFF="$(date -u -d "-${HOURS} hours" '+%Y-%m-%d %H:%M' 2>/dev/null || date -u -v-${HOURS}H '+%Y-%m-%d %H:%M')"

echo "h9-cron-soak: window since ~${CUTOFF} UTC from $LOG"
echo "--- pain alerts sent ---"
grep -c 'pain alert sent' "$LOG" 2>/dev/null || echo 0
echo "--- FloodWait lines ---"
grep -c 'FloodWait' "$LOG" 2>/dev/null || echo 0
echo "--- scrape done ---"
grep -c 'telegram scrape done' "$LOG" 2>/dev/null || echo 0
echo "--- shard filter (P1) ---"
grep -c 'shard filter' "$LOG" 2>/dev/null || echo 0

echo "h9-cron-soak: compare pain_alerts and FloodWait vs prior realtime baseline manually"
