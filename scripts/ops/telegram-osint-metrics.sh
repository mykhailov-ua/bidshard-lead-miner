#!/usr/bin/env bash
# Print OSINT metrics JSON (profile_link_queue, user_profiles, chat_members).
#
# Usage:
#   bash scripts/ops/telegram-osint-metrics.sh
#   bash scripts/ops/telegram-osint-metrics.sh | jq .profile_link_queue
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

export TELEGRAM_CURSOR_DB_PATH="${TELEGRAM_CURSOR_DB_PATH:-data/runtime/crawler.db}"

run_metrics() {
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm \
			-e "TELEGRAM_CURSOR_DB_PATH=${TELEGRAM_CURSOR_DB_PATH}" \
			parser python -m sources.telegram.osint_metrics
	elif [[ -d "$ROOT/.venv" ]]; then
		"$ROOT/.venv/bin/python" -m sources.telegram.osint_metrics
	else
		python3 -m sources.telegram.osint_metrics
	fi
}

run_metrics
