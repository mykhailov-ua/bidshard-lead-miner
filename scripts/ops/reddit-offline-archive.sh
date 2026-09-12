#!/usr/bin/env bash
# M7: Reddit offline archive (PullPush / Arctic Shift, not hot poll).
#
# Usage:
#   bash scripts/ops/reddit-offline-archive.sh
#   bash scripts/ops/reddit-offline-archive.sh --since 2025-01-01 --out data/export/reddit_pain.ndjson
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

SINCE="2025-01-01"
UNTIL=""
OUT="data/export/reddit_archive.ndjson"
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

if [[ -f .env ]]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

ARGS=(reddit offline-archive --since "$SINCE" --out "$OUT")
if [[ -n "$UNTIL" ]]; then
	ARGS+=(--until "$UNTIL")
fi
if [[ "$NO_FILTER" -eq 1 ]]; then
	ARGS+=(--no-filter)
fi
ARGS+=("${EXTRA_ARGS[@]}")

mkdir -p "$(dirname "$OUT")"

if command -v docker >/dev/null 2>&1 && docker compose ps parser >/dev/null 2>&1; then
	docker compose run --rm parser "${ARGS[@]}"
else
	go run ./cmd/parser "${ARGS[@]}"
fi

printf 'reddit-offline-archive: ok (since=%s out=%s)\n' "$SINCE" "$OUT"
