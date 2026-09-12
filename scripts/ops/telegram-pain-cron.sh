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

printf 'telegram-pain-cron: role=%s lease=%s session=%s\n' \
	"$TELEGRAM_SESSION_ROLE" "$TELEGRAM_LEASE_ENABLED" "${TELEGRAM_SESSION:-data/runtime/telethon.session}"

bash "$ROOT/scripts/ops/telegram-scrape-lease.sh"

printf 'telegram-pain-cron: done (ingest NDJSON via parser telethon ingest if configured)\n'
