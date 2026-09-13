#!/usr/bin/env bash
# Weekly discover feedback: dork outcome report + disable weak SERP dorks (needs Mongo).
#
# Usage:
#   bash scripts/ops/discover-feedback-cron.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

mkdir -p var data/suggestions

printf 'discover-feedback-cron: start\n'

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	docker compose run --rm parser feedback run 2>&1 | tee "var/discover-feedback-$(date -u +%Y%m%dT%H%M%SZ).log"
elif [[ -x "$ROOT/bin/parser" ]]; then
	"$ROOT/bin/parser" feedback run
else
	go run ./cmd/parser feedback run
fi

printf 'discover-feedback-cron: ok\n'
