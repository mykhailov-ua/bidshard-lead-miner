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

if [[ "$DRY_RUN" == "1" ]]; then
	docker compose -f docker-compose.headless.yaml --profile headless run --rm parser-headless headless drain --dry-run
	exit 0
fi

printf 'headless-crawl-cron: start\n'
docker compose -f docker-compose.headless.yaml --profile headless run --rm parser-headless headless drain 2>&1 | tee "var/headless-crawl-$(date -u +%Y%m%dT%H%M%SZ).log"
printf 'headless-crawl-cron: ok\n'
