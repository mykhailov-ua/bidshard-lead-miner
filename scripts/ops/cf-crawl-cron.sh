#!/usr/bin/env bash
# Cloudflare-heavy crawl via residential proxy (forum/tgweb/lander).
# Run from cron 1-2x/day; keeps PARSER_PROXY_LIST off the 24/7 poll loop.
#
# Usage:
#   export PARSER_PROXY_LIST=http://user:pass@gw.dataimpulse.com:823
#   bash scripts/ops/cf-crawl-cron.sh
#   bash scripts/ops/cf-crawl-cron.sh forum,tgweb,serp
#
# Crontab (UTC example, twice daily):
#   0 6,18 * * * cd /path/lead-intent-processor && set -a && source .env && set +a && \
#     bash scripts/ops/cf-crawl-cron.sh >> var/cf-crawl-cron.log 2>&1
#
# Requires: docker compose, PARSER_PROXY_LIST, GEMINI_API_KEY for tgweb ICP.

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

SOURCES="${1:-forum,tgweb,serp}"

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" && -z "${PARSER_PROXY_LIST_FILE//[[:space:]]/}" ]]; then
	printf 'cf-crawl-cron: FAIL PARSER_PROXY_LIST or PARSER_PROXY_LIST_FILE required\n' >&2
	exit 1
fi

mkdir -p var

printf 'cf-crawl-cron: start sources=%s\n' "$SOURCES"
bash "$ROOT/scripts/proxy/check-proxy.sh"

proxy_args=()
if [[ -n "${PARSER_PROXY_LIST//[[:space:]]/}" ]]; then
	proxy_args+=(-e "PARSER_PROXY_LIST=${PARSER_PROXY_LIST}")
fi
if [[ -n "${PARSER_PROXY_LIST_FILE//[[:space:]]/}" ]]; then
	proxy_args+=(-e "PARSER_PROXY_LIST_FILE=${PARSER_PROXY_LIST_FILE}")
fi

docker compose run --rm "${proxy_args[@]}" \
	parser scan --source="$SOURCES" 2>&1 | tee "var/cf-crawl-$(date -u +%Y%m%dT%H%M%SZ).log"

printf 'cf-crawl-cron: ok\n'
