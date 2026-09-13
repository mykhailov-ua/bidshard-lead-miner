#!/usr/bin/env bash
# Merge Telethon people JSON into Mongo (telegram_people) for CRM /export people.
#
# Usage:
#   bash scripts/ops/telegram-osint-sync.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

run_sync() {
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm crm-bot sync-osint
	elif [[ -x "$ROOT/bin/crm-bot" ]]; then
		"$ROOT/bin/crm-bot" sync-osint
	else
		go run ./cmd/crm-bot sync-osint
	fi
}

run_sync
