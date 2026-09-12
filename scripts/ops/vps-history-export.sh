#!/usr/bin/env bash
# M5: VPS historical pain export (session-lock safe).
#
# Stops parser + telegram realtime (shared telethon.session), runs export,
# then restarts services.
#
# Usage:
#   bash scripts/ops/vps-history-export.sh --since 2025-03-01
#   bash scripts/ops/vps-history-export.sh --since 2025-03-01 --relax
#   bash scripts/ops/vps-history-export.sh --since 2025-03-01 --relax --detach
#   bash scripts/ops/vps-history-export.sh --since 2025-03-01 --role-filter buyer_supergroup
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

SINCE=""
RELAX=0
DETACH=0
ROLE_FILTER="${TELEGRAM_REALTIME_ROLE_FILTER:-buyer_supergroup}"
EXTRA_ARGS=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--since)
		SINCE="${2:-}"
		shift 2
		;;
	--relax)
		RELAX=1
		shift
		;;
	--detach)
		DETACH=1
		shift
		;;
	--role-filter)
		ROLE_FILTER="${2:-}"
		shift 2
		;;
	*)
		EXTRA_ARGS+=("$1")
		shift
		;;
	esac
done

if [[ -z "$SINCE" ]]; then
	printf 'usage: %s --since YYYY-MM-DD [--relax] [--role-filter ROLE]\n' "$0" >&2
	exit 1
fi

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

relax_flag=""
if [[ "$RELAX" == "1" ]]; then
	relax_flag="--relax"
fi

extra_joined=""
if ((${#EXTRA_ARGS[@]} > 0)); then
	extra_joined="${EXTRA_ARGS[*]}"
fi

inplace_args=(--since "$SINCE" --role-filter "$ROLE_FILTER")
if [[ "$RELAX" == "1" ]]; then
	inplace_args+=(--relax)
fi

if [[ "$DETACH" == "1" ]]; then
	vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
mkdir -p var
if [[ -f var/tg-history-export.pid ]] && kill -0 \"\$(cat var/tg-history-export.pid)\" 2>/dev/null; then
  printf 'history export already running pid=%s\n' \"\$(cat var/tg-history-export.pid)\"
  exit 0
fi
nohup bash scripts/ops/history-export-inplace.sh ${inplace_args[*]} ${extra_joined} \
  >> var/tg-history-export-nohup.log 2>&1 &
echo \$! > var/tg-history-export.pid
printf 'history export detached pid=%s log=var/tg-history-export-full.log\n' \"\$(cat var/tg-history-export.pid)\"
"
	printf 'vps-history-export: detached (since=%s)\n' "$SINCE"
	exit 0
fi

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
bash scripts/ops/history-export-inplace.sh ${inplace_args[*]} ${extra_joined}
"

printf 'vps-history-export: ok (since=%s)\n' "$SINCE"
