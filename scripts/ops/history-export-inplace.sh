#!/usr/bin/env bash
# M5: run ON the VPS host (no nested SSH). Called by vps-history-export.sh via ssh.
#
# Usage (on VPS):
#   cd /opt/lead-intent-processor && bash scripts/ops/history-export-inplace.sh --since 2025-03-01 --relax
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

restart_services() {
	printf '=== restarting services ===\n' | tee -a "${LOG:-var/tg-history-export-full.log}"
	docker compose up -d parser
	docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime up -d --force-recreate parser-telegram-realtime
}

trap restart_services EXIT

SINCE=""
RELAX=0
ROLE_FILTER="${TELEGRAM_REALTIME_ROLE_FILTER:-buyer_supergroup}"

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
	--role-filter)
		ROLE_FILTER="${2:-}"
		shift 2
		;;
	*)
		printf 'unknown arg: %s\n' "$1" >&2
		exit 1
		;;
	esac
done

if [[ -z "$SINCE" ]]; then
	printf 'usage: %s --since YYYY-MM-DD [--relax] [--role-filter ROLE]\n' "$0" >&2
	exit 1
fi

if ! grep -qE '^TELEGRAM_API_ID=[0-9]+' .env 2>/dev/null || ! grep -qE '^TELEGRAM_API_HASH=.+$' .env 2>/dev/null; then
	printf 'missing TELEGRAM_API_ID/HASH in .env\n' >&2
	exit 1
fi

mkdir -p data/export var
OUT="data/export/tg_history_pain_${SINCE}.ndjson"
LOG="var/tg-history-export-full.log"

relax_flag=()
if [[ "$RELAX" == "1" ]]; then
	relax_flag=(--relax)
fi

printf '[%s] history export inplace start since=%s\n' "$(date -Is)" "$SINCE" | tee -a "$LOG"

printf '=== stopping session holders ===\n' | tee -a "$LOG"
docker compose stop parser
docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime stop parser-telegram-realtime 2>/dev/null || true
sleep 3

printf '=== history export ===\n' | tee -a "$LOG"
set +e
docker compose run --rm parser telegram history-export \
	--since "$SINCE" \
	--out "$OUT" \
	--role-filter "$ROLE_FILTER" \
	"${relax_flag[@]}" 2>&1 | tee -a "$LOG"
rc=$?
set -e

rows=0
if [[ -f "$OUT" ]]; then
	rows=$(wc -l < "$OUT" | tr -d ' ')
fi
printf '[%s] export file=%s rows=%s exit=%s\n' "$(date -Is)" "$OUT" "$rows" "$rc" | tee -a "$LOG"
tail -10 "$LOG" || true

trap - EXIT
restart_services

exit "$rc"
