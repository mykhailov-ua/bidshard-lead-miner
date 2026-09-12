#!/usr/bin/env bash
# P1 cron scrape wrapper: run one static shard (disjoint chat subset).
#
# Usage:
#   TELEGRAM_SHARD=0 TELEGRAM_SHARD_COUNT=2 TELEGRAM_SESSION=data/runtime/telethon.session.0 \
#     bash scripts/ops/telegram-scrape-shard.sh
#
# Crontab example (two hot sessions):
#   5 * * * * cd /path/lead-intent-processor && TELEGRAM_SHARD=0 TELEGRAM_SHARD_COUNT=2 TELEGRAM_SESSION=data/runtime/telethon.session.0 bash scripts/ops/telegram-scrape-shard.sh
#   35 * * * * cd /path/lead-intent-processor && TELEGRAM_SHARD=1 TELEGRAM_SHARD_COUNT=2 TELEGRAM_SESSION=data/runtime/telethon.session.1 bash scripts/ops/telegram-scrape-shard.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

SHARD="${TELEGRAM_SHARD:-0}"
COUNT="${TELEGRAM_SHARD_COUNT:-1}"
SESSION="${TELEGRAM_SESSION:-data/runtime/telethon.session}"

export TELEGRAM_SHARD="$SHARD"
export TELEGRAM_SHARD_COUNT="$COUNT"
export TELEGRAM_SESSION="$SESSION"

printf 'telegram-scrape-shard: shard=%s count=%s session=%s\n' "$SHARD" "$COUNT" "$SESSION"

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	docker compose run --rm parser telegram scrape
elif [[ -x "$ROOT/bin/parser" ]]; then
	"$ROOT/bin/parser" telegram scrape
else
	go run ./cmd/parser telegram scrape
fi
