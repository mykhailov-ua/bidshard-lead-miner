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
	# Cron one-shot uses stdout NDJSON pipe; TELETHON_IPC_SOCKET is for realtime only.
	docker compose run --rm \
		-e "TELEGRAM_SESSION=${SESSION}" \
		-e "TELETHON_IPC_SOCKET=" \
		-e "TELETHON_IPC_FORMAT=ndjson" \
		parser telegram
elif [[ -x "$ROOT/bin/parser" ]]; then
	"$ROOT/bin/parser" telegram
else
	go run ./cmd/parser telegram
fi
