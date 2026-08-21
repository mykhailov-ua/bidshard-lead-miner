#!/usr/bin/env bash
# BOX-9: supply + lander emit smoke on prod seeds (requires network + proxy on VPS).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

RUN_DIR="$ROOT/var/prod-source-smoke-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$RUN_DIR"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

export PARSER_SEED_PROFILE="${PARSER_SEED_PROFILE:-prod}"
export SUPPLY_SEED_PATH="${SUPPLY_SEED_PATH:-data/seeds/domains.prod.csv}"
export LANDER_SEED_PATH="${LANDER_SEED_PATH:-data/seeds/lander_urls.prod.csv}"

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" ]]; then
	printf 'prod-source-smoke: FAIL PARSER_PROXY_LIST required on datacenter VPS\n' >&2
	exit 1
fi

if [[ ! -x "$ROOT/bin/parser" ]]; then
	make -C "$ROOT" build
fi

bash "$ROOT/scripts/proxy/check-proxy.sh"
"$ROOT/bin/parser" config check

fail=0
for src in supply lander; do
	LOG="$RUN_DIR/${src}.log"
	"$ROOT/bin/parser" scan --source="$src" --output=quiet 2>&1 | tee "$LOG"
	# shellcheck source=scripts/lib/soak_gate.sh
	source "$ROOT/scripts/lib/soak_gate.sh"
	evaluate_scan_raw_gate "prod-source-smoke $src" "$LOG" || fail=1
done

if [[ "$fail" -ne 0 ]]; then
	printf 'prod-source-smoke: FAIL (logs under %s)\n' "$RUN_DIR" >&2
	exit 1
fi

printf 'prod-source-smoke: ok (logs under %s)\n' "$RUN_DIR"
