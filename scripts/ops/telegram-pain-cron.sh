#!/usr/bin/env bash
# H9 cron pain path: Telethon scrape (P0 cache + optional P1/P2) without realtime.
#
# Usage:
#   bash scripts/ops/telegram-pain-cron.sh
#
# Recommended crontab (UTC):
#   5,35 * * * * cd /path/lead-intent-processor && bash scripts/ops/telegram-pain-cron.sh >> var/telegram-pain-cron.log 2>&1

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

mkdir -p var

export TELEGRAM_SESSION_ROLE="${TELEGRAM_SESSION_ROLE:-hot}"
export TELEGRAM_LEASE_ENABLED="${TELEGRAM_LEASE_ENABLED:-1}"

SESSION="${TELEGRAM_SESSION:-data/runtime/telethon.session}"
LOCK="$ROOT/var/telegram-scrape-$(basename "$SESSION").lock"
LAST_RUN_FILE="$ROOT/var/telegram-scrape-last-$(basename "$SESSION").ts"
MIN_GAP_SEC="${TELEGRAM_CRON_MIN_GAP_SEC:-2400}"
JITTER_SEC="${TELEGRAM_CRON_JITTER_SEC:-120}"

if [[ "$MIN_GAP_SEC" -gt 0 && -f "$LAST_RUN_FILE" ]]; then
	last_run="$(cat "$LAST_RUN_FILE" 2>/dev/null || echo 0)"
	now="$(date +%s)"
	elapsed=$((now - last_run))
	if [[ "$elapsed" -lt "$MIN_GAP_SEC" ]]; then
		printf 'telegram-pain-cron: skip min_gap elapsed=%ss need=%ss session=%s\n' \
			"$elapsed" "$MIN_GAP_SEC" "$SESSION" >&2
		exit 0
	fi
fi

if [[ "$JITTER_SEC" -gt 0 ]]; then
	sleep_sec=$((RANDOM % JITTER_SEC))
	if [[ "$sleep_sec" -gt 0 ]]; then
		printf 'telegram-pain-cron: jitter sleep=%ss session=%s\n' "$sleep_sec" "$SESSION"
		sleep "$sleep_sec"
	fi
fi

printf 'telegram-pain-cron: role=%s lease=%s session=%s\n' \
	"$TELEGRAM_SESSION_ROLE" "$TELEGRAM_LEASE_ENABLED" "$SESSION"

if ! command -v flock >/dev/null 2>&1; then
	printf 'telegram-pain-cron: flock missing; running without overlap guard\n' >&2
	bash "$ROOT/scripts/ops/telegram-scrape-lease.sh"
	date +%s >"$LAST_RUN_FILE"
else
	if flock -n "$LOCK" bash "$ROOT/scripts/ops/telegram-scrape-lease.sh"; then
		date +%s >"$LAST_RUN_FILE"
	else
		printf 'telegram-pain-cron: skip (another scrape holds %s)\n' "$LOCK" >&2
		exit 0
	fi
fi

if [[ "${TELEGRAM_OSINT_SYNC:-0}" == "1" ]]; then
	bash "$ROOT/scripts/ops/telegram-osint-sync.sh" || true
fi
bash "$ROOT/scripts/ops/telegram-osint-metrics.sh" || true
printf 'telegram-pain-cron: done\n'
