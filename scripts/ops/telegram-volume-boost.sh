#!/usr/bin/env bash
# One-shot TG volume boost: triage registry, apply P0 env knobs, optional history backfill.
#
# Usage (local -> VPS):
#   bash scripts/ops/telegram-volume-boost.sh
#   bash scripts/ops/telegram-volume-boost.sh --since 2025-06-01
#   bash scripts/ops/telegram-volume-boost.sh --since 2025-06-01 --detach-export
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

SINCE="2025-06-01"
DETACH_EXPORT=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--since)
		SINCE="${2:-}"
		shift 2
		;;
	--detach-export)
		DETACH_EXPORT=1
		shift
		;;
	*)
		printf 'unknown arg: %s\n' "$1" >&2
		exit 1
		;;
	esac
done

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

printf 'telegram-volume-boost: sync code\n'
vps_rsync_push "$ROOT"
vps_post_sync_fix

printf 'telegram-volume-boost: apply P0 env (lease=40, tg score floor=30)\n'
bash "$ROOT/scripts/ops/vps-apply-p0-env.sh"

printf 'telegram-volume-boost: rebuild parser image\n'
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; docker compose build parser"

printf 'telegram-volume-boost: triage registry + pool sync on VPS\n'
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; docker compose run --rm --entrypoint python3 parser -c \"
from pathlib import Path
from sources.telegram.registry_triage import triage_channel_registry
from sources.telegram.config import load_config
from sources.telegram.cursor import CursorStore
from sources.telegram.pool_sync import sync_registry_pool
path = Path('data/runtime/discovered_telegram_channels.json')
print('triage', triage_channel_registry(path))
cfg = load_config(Path('config/sources.telegram.yaml'))
store = CursorStore(cfg.cursor_db)
try:
    print('pool_sync', sync_registry_pool(cfg, store))
finally:
    store.close()
\""

printf 'telegram-volume-boost: repair session pool sqlite paths\n'
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; bash scripts/ops/telegram-session-pool-repair.sh"

printf 'telegram-volume-boost: reinstall cron (single MTProto session + flock)\n'
bash "$ROOT/scripts/ops/vps-install-telegram-cron.sh"

printf 'telegram-volume-boost: restart parser (pick up env)\n'
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; docker compose up -d parser"

if [[ "$DETACH_EXPORT" == "1" ]]; then
	printf 'telegram-volume-boost: start detached history export since=%s\n' "$SINCE"
	bash "$ROOT/scripts/ops/vps-history-export.sh" --since "$SINCE" --relax --detach
else
	printf 'telegram-volume-boost: history export + ingest since=%s (stops parser briefly)\n' "$SINCE"
	bash "$ROOT/scripts/ops/vps-history-export.sh" --since "$SINCE" --relax
fi

printf 'telegram-volume-boost: done\n'
