#!/usr/bin/env bash
# Nightly headless drain for deferred lander/tgweb URLs (Playwright + Chromium).
# Run from cron 1x/night; keeps PARSER_LANDER_HEADLESS off the 24/7 poll loop.
#
# Usage:
#   bash scripts/ops/headless-crawl-cron.sh
#   bash scripts/ops/headless-crawl-cron.sh --dry-run
#
# Crontab (UTC example, once nightly):
#   30 2 * * * cd /path/lead-intent-processor && set -a && source .env && set +a && \
#     bash scripts/ops/headless-crawl-cron.sh >> var/headless-crawl-cron.log 2>&1
#
# Requires: docker compose, docker-compose.headless.yaml profile, Playwright image built.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

DRY_RUN=0
if [[ "${1:-}" == "--dry-run" ]]; then
	DRY_RUN=1
fi

mkdir -p var

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

# shellcheck source=scripts/lib/headless_proxy_geo.sh
source "$ROOT/scripts/lib/headless_proxy_geo.sh"
headless_export_proxy_geo_env "$ROOT"

# Prefer system Chrome in headless image when installed (better WebGL than bundled Chromium).
: "${PARSER_HEADLESS_CHANNEL:=chrome}"

compose_run() {
	docker compose -f docker-compose.headless.yaml --profile headless run --rm \
		-e "PARSER_HEADLESS_LOCALE=${PARSER_HEADLESS_LOCALE:-}" \
		-e "PARSER_HEADLESS_TIMEZONE=${PARSER_HEADLESS_TIMEZONE:-}" \
		-e "PARSER_HEADLESS_CHANNEL=${PARSER_HEADLESS_CHANNEL:-}" \
		-e "PARSER_HEADLESS_HEADED=${PARSER_HEADLESS_HEADED:-}" \
		-e "PARSER_HEADLESS_XVFB=${PARSER_HEADLESS_XVFB:-}" \
		parser-headless "$@"
}

if [[ "$DRY_RUN" == "1" ]]; then
	compose_run headless drain --dry-run
	exit 0
fi

printf 'headless-crawl-cron: start channel=%s locale=%s tz=%s\n' \
	"${PARSER_HEADLESS_CHANNEL:-}" "${PARSER_HEADLESS_LOCALE:-}" "${PARSER_HEADLESS_TIMEZONE:-}"
compose_run headless drain 2>&1 | tee "var/headless-crawl-$(date -u +%Y%m%dT%H%M%SZ).log"
printf 'headless-crawl-cron: ok\n'
