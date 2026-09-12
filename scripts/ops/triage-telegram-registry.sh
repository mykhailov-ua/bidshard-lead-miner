#!/usr/bin/env bash
# Heuristic triage on discovered_telegram_channels.json before discover/scrape.
#
# Usage:
#   bash scripts/ops/triage-telegram-registry.sh
#   bash scripts/ops/triage-telegram-registry.sh data/runtime/discovered_telegram_channels.json

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

PATH_ARG="${1:-data/runtime/discovered_telegram_channels.json}"

python3 - <<'PY' "$PATH_ARG"
import sys
from sources.telegram.registry_triage import triage_channel_registry

stats = triage_channel_registry(sys.argv[1])
print(
    "triage-telegram-registry: kept={kept} dropped={dropped} total={total}".format(
        **stats
    )
)
PY
