#!/usr/bin/env bash
# Heuristic triage on discovered_telegram_channels.json before discover/scrape.
#
# Usage:
#   bash scripts/ops/triage-telegram-registry.sh
#   bash scripts/ops/triage-telegram-registry.sh data/runtime/discovered_telegram_channels.json
#
# On VPS, registry lives in parser_runtime volume; use docker compose path when available.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

PATH_ARG="${1:-data/runtime/discovered_telegram_channels.json}"

run_triage_py() {
	local registry_path="$1"
	python3 - <<'PY' "$registry_path"
import sys
from sources.telegram.registry_triage import triage_channel_registry

stats = triage_channel_registry(sys.argv[1])
print(
    "triage-telegram-registry: kept={kept} dropped={dropped} total={total}".format(
        **stats
    )
)

from pathlib import Path
from sources.telegram.config import load_config
from sources.telegram.cursor import CursorStore
from sources.telegram.pool_sync import sync_registry_pool

cfg_path = Path("config/sources.telegram.yaml")
if cfg_path.exists():
    cfg = load_config(cfg_path)
    store = CursorStore(cfg.cursor_db)
    try:
        pool_stats = sync_registry_pool(cfg, store)
        print(
            "pool-sync: imported={imported} skipped={skipped} disabled={disabled}".format(
                **pool_stats
            )
        )
    finally:
        store.close()
PY
}

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]] && [[ ! -f "$PATH_ARG" ]]; then
	docker compose run --rm --entrypoint python3 parser -c "
from pathlib import Path
from sources.telegram.registry_triage import triage_channel_registry
from sources.telegram.config import load_config
from sources.telegram.cursor import CursorStore
from sources.telegram.pool_sync import sync_registry_pool

path = Path('${PATH_ARG}')
print('triage-telegram-registry:', triage_channel_registry(path))
cfg = load_config(Path('config/sources.telegram.yaml'))
store = CursorStore(cfg.cursor_db)
try:
    print('pool-sync:', sync_registry_pool(cfg, store))
finally:
    store.close()
"
	exit 0
fi

run_triage_py "$PATH_ARG"
