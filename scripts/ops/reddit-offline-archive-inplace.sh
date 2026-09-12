#!/usr/bin/env bash
# M7: run ON the VPS host (no nested SSH). Called by vps-reddit-offline-archive.sh.
#
# Usage (on VPS):
#   cd /opt/lead-intent-processor && bash scripts/ops/reddit-offline-archive-inplace.sh --since 2025-01-01
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

SINCE="2025-01-01"
UNTIL=""
OUT=""
NO_FILTER=0
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
	*)
		EXTRA_ARGS+=("$1")
		shift
		;;
	esac
done

if [[ -z "$OUT" ]]; then
	OUT="data/export/reddit_archive_${SINCE}.ndjson"
fi

mkdir -p data/export var
LOG="var/reddit-offline-archive-full.log"

ARGS=(reddit offline-archive --since "$SINCE" --out "$OUT")
if [[ -n "$UNTIL" ]]; then
	ARGS+=(--until "$UNTIL")
fi
if [[ "$NO_FILTER" -eq 1 ]]; then
	ARGS+=(--no-filter)
fi
ARGS+=("${EXTRA_ARGS[@]}")

printf '[%s] reddit offline archive start since=%s out=%s\n' "$(date -Is)" "$SINCE" "$OUT" | tee -a "$LOG"

set +e
docker compose run --rm parser "${ARGS[@]}" 2>&1 | tee -a "$LOG"
rc=$?
set -e

rows=0
if [[ -f "$OUT" ]]; then
	rows=$(wc -l < "$OUT" | tr -d ' ')
fi
printf '[%s] reddit archive file=%s rows=%s exit=%s\n' "$(date -Is)" "$OUT" "$rows" "$rc" | tee -a "$LOG"

exit "$rc"
