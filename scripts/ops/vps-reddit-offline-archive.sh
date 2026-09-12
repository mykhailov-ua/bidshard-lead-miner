#!/usr/bin/env bash
# M7: VPS Reddit offline archive (PullPush / Arctic Shift, detached long run).
#
# Usage:
#   bash scripts/ops/vps-reddit-offline-archive.sh --since 2025-01-01 --detach
#   bash scripts/ops/vps-reddit-offline-archive.sh --since 2025-01-01 --out data/export/reddit_pain.ndjson
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

SINCE="2025-01-01"
UNTIL=""
OUT=""
NO_FILTER=0
DETACH=0
EXTRA_ARGS=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--since)
		SINCE="${2:-}"
		shift 2
		;;
	--until)
		UNTIL="${2:-}"
		shift 2
		;;
	--out)
		OUT="${2:-}"
		shift 2
		;;
	--no-filter)
		NO_FILTER=1
		shift
		;;
	--detach)
		DETACH=1
		shift
		;;
	*)
		EXTRA_ARGS+=("$1")
		shift
		;;
	esac
done

if [[ -z "$OUT" ]]; then
	OUT="data/export/reddit_archive_${SINCE}.ndjson"
fi

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

inplace_args=(--since "$SINCE" --out "$OUT")
if [[ -n "$UNTIL" ]]; then
	inplace_args+=(--until "$UNTIL")
fi
if [[ "$NO_FILTER" -eq 1 ]]; then
	inplace_args+=(--no-filter)
fi

extra_joined=""
if ((${#EXTRA_ARGS[@]} > 0)); then
	extra_joined="${EXTRA_ARGS[*]}"
fi

if [[ "$DETACH" == "1" ]]; then
	vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
mkdir -p var
if [[ -f var/reddit-offline-archive.pid ]] && kill -0 \"\$(cat var/reddit-offline-archive.pid)\" 2>/dev/null; then
  printf 'reddit archive already running pid=%s\n' \"\$(cat var/reddit-offline-archive.pid)\"
  exit 0
fi
nohup bash scripts/ops/reddit-offline-archive-inplace.sh ${inplace_args[*]} ${extra_joined} \
  >> var/reddit-offline-archive-nohup.log 2>&1 &
echo \$! > var/reddit-offline-archive.pid
printf 'reddit archive detached pid=%s log=var/reddit-offline-archive-full.log out=${OUT}\n' \"\$(cat var/reddit-offline-archive.pid)\"
"
	printf 'vps-reddit-offline-archive: detached (since=%s out=%s)\n' "$SINCE" "$OUT"
	exit 0
fi

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
bash scripts/ops/reddit-offline-archive-inplace.sh ${inplace_args[*]} ${extra_joined}
"

printf 'vps-reddit-offline-archive: ok (since=%s out=%s)\n' "$SINCE" "$OUT"
