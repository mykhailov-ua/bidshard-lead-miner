#!/usr/bin/env bash
# BOX-8: forum live smoke via proxy (XenForo seeds + parse path).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

RUN_DIR="$ROOT/var/forum-live-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$RUN_DIR"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

export PARSER_SEED_PROFILE="${PARSER_SEED_PROFILE:-prod}"
export PARSER_SOURCE=forum
export FORUM_SEED_PATH="${FORUM_SEED_PATH:-data/seeds/forum_threads.live.csv}"

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" ]]; then
	printf 'forum-live-check: FAIL PARSER_PROXY_LIST required on datacenter VPS\n' >&2
	exit 1
fi

if [[ ! -x "$ROOT/bin/parser" ]]; then
	make -C "$ROOT" build
fi

bash "$ROOT/scripts/proxy/check-proxy.sh"

"$ROOT/bin/parser" config check

LOG="$RUN_DIR/scan.log"
"$ROOT/bin/parser" scan --source=forum --output=quiet 2>&1 | tee "$LOG"

# shellcheck source=scripts/lib/soak_gate.sh
source "$ROOT/scripts/lib/soak_gate.sh"
evaluate_scan_raw_gate "forum-live-check" "$LOG"

printf 'forum-live-check: ok (log: %s)\n' "$LOG"
