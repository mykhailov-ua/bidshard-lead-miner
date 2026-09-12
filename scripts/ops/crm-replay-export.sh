#!/usr/bin/env bash
# POST leads from JSONL export to CRM webhook (Telegram notify via crm-bot).
#
# Usage:
#   bash scripts/ops/crm-replay-export.sh
#   CRM_REPLAY_MIN_SCORE=50 bash scripts/ops/crm-replay-export.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

export CRM_REPLAY_MIN_SCORE="${CRM_REPLAY_MIN_SCORE:-50}"
export EXPORT_PATH="${EXPORT_PATH:-data/export/leads.jsonl}"

local_export="$ROOT/$EXPORT_PATH"
if [[ -f "$local_export" ]]; then
	EXPORT_PATH="$local_export" python3 "$ROOT/scripts/ops/crm-replay-export-remote.py"
	exit 0
fi

if ! vps_load_config "$ROOT" 2>/dev/null || ! vps_ssh_require "$ROOT" 2>/dev/null; then
	printf 'crm-replay-export: missing %s (no VPS)\n' "$EXPORT_PATH" >&2
	exit 1
fi

vps_ssh "cd '${VPS_REMOTE_DIR}' && EXPORT_PATH='${EXPORT_PATH}' CRM_REPLAY_MIN_SCORE='${CRM_REPLAY_MIN_SCORE}' python3 scripts/ops/crm-replay-export-remote.py"
