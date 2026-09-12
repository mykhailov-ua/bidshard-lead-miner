#!/usr/bin/env bash
# H13 TG-first collect: forum + SERP seeds + tgweb; skip reddit/webpain 24/7 churn.
#
# Usage:
#   bash scripts/ops/tg-first-collect.sh
#   bash scripts/ops/tg-first-collect.sh once

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

MODE="${1:-once}"
export PARSER_SOURCE="${PARSER_SOURCE:-forum,serp,jobboard,tgweb}"

printf 'tg-first-collect: PARSER_SOURCE=%s mode=%s\n' "$PARSER_SOURCE" "$MODE"

run_parser() {
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm parser "$@"
	elif [[ -x "$ROOT/bin/parser" ]]; then
		"$ROOT/bin/parser" "$@"
	else
		go run ./cmd/parser "$@"
	fi
}

if [[ "$MODE" == "once" ]]; then
	run_parser run --once
else
	run_parser run
fi
