#!/usr/bin/env bash
# MTProto channel discover on cold session: Contacts.Search + SERP registry + cross-mention.
# Run after Go SERP harvest (or alone on buyer-discover-fast nights).
#
# Usage:
#   bash scripts/ops/telegram-discover-cold.sh
#
# Env (defaults suit VPS cron with telethon.session.2):
#   TELEGRAM_DISCOVER_SESSION=data/runtime/telethon.session.2
#   TELEGRAM_DISCOVER_QUERY_BATCH=20
#   TELEGRAM_DISCOVER_QUERY_OFFSET=0   # buyer-discover sets from UTC day-of-year
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

SESSION="${TELEGRAM_DISCOVER_SESSION:-data/runtime/telethon.session.2}"
BATCH="${TELEGRAM_DISCOVER_QUERY_BATCH:-20}"
# Spread queries across the ICP list without running all each night.
OFFSET="${TELEGRAM_DISCOVER_QUERY_OFFSET:-$(( ($(date -u +%j) * 17) ))}"

mkdir -p var

export TELEGRAM_SESSION="$SESSION"
export TELEGRAM_SESSION_ROLE=cold
export TELEGRAM_INVITE_JOIN=1
export TELEGRAM_INVITE_JOIN_HOT=1
export TELEGRAM_DISCOVER_QUERY_BATCH="$BATCH"
export TELEGRAM_DISCOVER_QUERY_OFFSET="$OFFSET"
export TELETHON_IPC_SOCKET=
export TELETHON_IPC_FORMAT=ndjson

printf 'telegram-discover-cold: session=%s batch=%s offset=%s\n' \
	"$SESSION" "$BATCH" "$OFFSET"

run_discover() {
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm \
			-e "TELEGRAM_SESSION=${SESSION}" \
			-e "TELEGRAM_SESSION_ROLE=cold" \
			-e "TELEGRAM_INVITE_JOIN=1" \
			-e "TELEGRAM_INVITE_JOIN_HOT=1" \
			-e "TELEGRAM_DISCOVER_QUERY_BATCH=${BATCH}" \
			-e "TELEGRAM_DISCOVER_QUERY_OFFSET=${OFFSET}" \
			-e "TELETHON_IPC_SOCKET=" \
			-e "TELETHON_IPC_FORMAT=ndjson" \
			parser telegram discover
	elif [[ -x "$ROOT/bin/parser" ]]; then
		"$ROOT/bin/parser" telegram discover
	else
		go run ./cmd/parser telegram discover
	fi
}

run_discover

bash "$ROOT/scripts/ops/telegram-osint-metrics.sh" || true
printf 'telegram-discover-cold: ok\n'
