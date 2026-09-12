#!/usr/bin/env bash
# Zero-cost buyer discovery: SERP harvest (icp dorks) + CF crawl via residential proxy.
# Run from cron 1-2x/day; keeps PARSER_PROXY_LIST off the 24/7 poll loop.
#
# Usage:
#   bash scripts/ops/buyer-discover.sh
#   bash scripts/ops/buyer-discover.sh forum,tgweb   # skip serp scan emit
#
# Crontab (UTC example, twice daily):
#   0 6,18 * * * cd /path/lead-intent-processor && bash scripts/ops/buyer-discover.sh >> var/buyer-discover.log 2>&1
#
# Requires: docker compose OR go build, PARSER_PROXY_LIST in config/env/.env.proxy.local

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

# docker compose loads repo .env; do not source it here (REDDIT_QUERIES has spaces/semicolons).
PROXY_ENV="$ROOT/config/env/.env.proxy.local"
if [[ -f "$PROXY_ENV" ]]; then
	set -a
	# shellcheck disable=SC1090
	source "$PROXY_ENV"
	set +a
fi

SCAN_SOURCES="${1:-forum,jobboard,tgweb,serp}"

mkdir -p var

run_parser_discover() {
	local subcmd="$1"
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm parser discover "$subcmd"
	elif [[ -x "$ROOT/bin/parser" ]]; then
		"$ROOT/bin/parser" discover "$subcmd"
	else
		go run ./cmd/parser discover "$subcmd"
	fi
}

printf 'buyer-discover: jobboard SERP + employer reverse (P1 employer->TG chain)\n'
run_parser_discover jobboard

printf 'buyer-discover: SERP harvest (jobboard, forum, tg catalog meta, t.me dorks)\n'
run_parser_discover serp

printf 'buyer-discover: triage telegram channel registry + pool sync\n'
bash "$ROOT/scripts/ops/triage-telegram-registry.sh"

printf 'buyer-discover: CF crawl sources=%s\n' "$SCAN_SOURCES"
bash "$ROOT/scripts/ops/cf-crawl-cron.sh" "$SCAN_SOURCES"

printf 'buyer-discover: ok\n'
