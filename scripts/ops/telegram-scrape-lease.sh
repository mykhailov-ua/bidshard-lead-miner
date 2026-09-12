#!/usr/bin/env bash
# P2 lease pool scrape: CLAIM -> scrape -> HEARTBEAT -> release on FloodWait.
#
# Usage:
#   TELEGRAM_LEASE_ENABLED=1 TELEGRAM_WORKER_ID=vps-hot-0 \
#     bash scripts/ops/telegram-scrape-lease.sh
#
# Combine with P1 shard + separate session files:
#   TELEGRAM_LEASE_ENABLED=1 TELEGRAM_WORKER_ID=vps-0 \
#   TELEGRAM_SHARD=0 TELEGRAM_SHARD_COUNT=2 TELEGRAM_SESSION=data/runtime/telethon.session.0 \
#     bash scripts/ops/telegram-scrape-lease.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

export TELEGRAM_LEASE_ENABLED="${TELEGRAM_LEASE_ENABLED:-1}"
export TELEGRAM_WORKER_ID="${TELEGRAM_WORKER_ID:-$(hostname -s)-$$}"
export TELEGRAM_LEASE_TTL_SEC="${TELEGRAM_LEASE_TTL_SEC:-300}"
export TELEGRAM_LEASE_MAX_CHATS="${TELEGRAM_LEASE_MAX_CHATS:-15}"
export TELEGRAM_LEASE_STALE_SEC="${TELEGRAM_LEASE_STALE_SEC:-120}"

printf 'telegram-scrape-lease: worker=%s ttl=%ss max=%s stale=%ss\n' \
	"$TELEGRAM_WORKER_ID" "$TELEGRAM_LEASE_TTL_SEC" "$TELEGRAM_LEASE_MAX_CHATS" "$TELEGRAM_LEASE_STALE_SEC"

exec bash "$ROOT/scripts/ops/telegram-scrape-shard.sh"
