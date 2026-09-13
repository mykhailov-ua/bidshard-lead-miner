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
DIRECT_SOURCES="${BUYER_DISCOVER_DIRECT_SOURCES:-jobboard,serp,reviews,discord}"

mkdir -p var

# shellcheck disable=SC1091
source "$ROOT/scripts/lib/proxy_first_url.sh"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/proxy_baseline_ok.sh"

run_parser_discover() {
	local subcmd="$1"
	local dork_env=()
	if [[ -n "${PARSER_SERP_DORK_OFFSET:-}" ]]; then
		dork_env+=(-e "PARSER_SERP_DORK_OFFSET=${PARSER_SERP_DORK_OFFSET}")
	fi
	if [[ -n "${PARSER_SERP_DORK_BATCH:-}" ]]; then
		dork_env+=(-e "PARSER_SERP_DORK_BATCH=${PARSER_SERP_DORK_BATCH}")
	fi
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm "${dork_env[@]}" parser discover "$subcmd"
	elif [[ -x "$ROOT/bin/parser" ]]; then
		"$ROOT/bin/parser" discover "$subcmd"
	else
		go run ./cmd/parser discover "$subcmd"
	fi
}

printf 'buyer-discover: jobboard SERP + employer reverse (P1 employer->TG chain)\n'
run_parser_discover jobboard

printf 'buyer-discover: SERP harvest (jobboard, forum, tg catalog meta, t.me dorks)\n'
# Rotate which dorks run each day; cap per run via PARSER_SERP_TELEGRAM_DORK_MAX / PARSER_SERP_DORK_BATCH.
export PARSER_SERP_DORK_OFFSET="${PARSER_SERP_DORK_OFFSET:-$(( ($(date -u +%j) * 13) ))}"
run_parser_discover serp

printf 'buyer-discover: telethon discover (cold session, batched search queries)\n'
bash "$ROOT/scripts/ops/telegram-discover-cold.sh"

printf 'buyer-discover: triage telegram channel registry + pool sync\n'
bash "$ROOT/scripts/ops/triage-telegram-registry.sh"

printf 'buyer-discover: discord invite harvest (no residential proxy)\n'
bash "$ROOT/scripts/ops/discord-discover.sh"

if [[ "$(date -u +%u)" == "7" ]]; then
	printf 'buyer-discover: weekly discover feedback (dork prune)\n'
	bash "$ROOT/scripts/ops/discover-feedback-cron.sh" || printf 'buyer-discover: feedback skipped (non-fatal)\n'
fi

run_direct_scan() {
	if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
		docker compose run --rm parser scan --source="$DIRECT_SOURCES" --output=quiet
	elif [[ -x "$ROOT/bin/parser" ]]; then
		"$ROOT/bin/parser" scan --source="$DIRECT_SOURCES" --output=quiet
	else
		go run ./cmd/parser scan --source="$DIRECT_SOURCES" --output=quiet
	fi
}

if proxy_baseline_ok "$ROOT"; then
	printf 'buyer-discover: CF crawl sources=%s\n' "$SCAN_SOURCES"
	bash "$ROOT/scripts/ops/cf-crawl-cron.sh" "$SCAN_SOURCES"
else
	printf 'buyer-discover: proxy baseline failed; direct scan sources=%s (skip forum/tgweb)\n' \
		"$DIRECT_SOURCES"
	run_direct_scan
fi

printf 'buyer-discover: ok\n'
