#!/usr/bin/env bash
# Fast discover path: triage existing registry + pool sync only.
# Full discover serp runs 150+ DuckDuckGo dorks (~1-2 min each on 202/EOF) plus employer reverse.
#
# Usage:
#   bash scripts/ops/buyer-discover-fast.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

printf 'buyer-discover-fast: telethon discover (cold) + triage (skip Go SERP harvest)\n'
bash "$ROOT/scripts/ops/telegram-discover-cold.sh"
bash "$ROOT/scripts/ops/triage-telegram-registry.sh"
if [[ "$(date -u +%u)" == "7" ]]; then
	bash "$ROOT/scripts/ops/discover-feedback-cron.sh" || true
fi
bash "$ROOT/scripts/ops/telegram-osint-metrics.sh" || true
printf 'buyer-discover-fast: ok\n'
