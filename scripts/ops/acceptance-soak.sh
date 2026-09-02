#!/usr/bin/env bash
# Epic J acceptance soak: jq gates on JSONL export after prod-like run.
# Requires: jq, non-empty leads export (default data/export/leads.jsonl).
#
# Usage:
#   make acceptance-soak
#   LEADS_JSONL=var/soak/leads.jsonl bash scripts/ops/acceptance-soak.sh
#
# Env overrides:
#   ACCEPTANCE_MAX_PENDING_PCT=20
#   ACCEPTANCE_MIN_TG_PAIN_PCT=30
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/.env" && -r "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

# shellcheck source=scripts/lib/host_paths.sh
source "$ROOT/scripts/lib/host_paths.sh"
apply_host_export_paths "$ROOT"

LEADS_JSONL="${LEADS_JSONL:-${PARSER_EXPORT_JSON_HOST:-$ROOT/data/export/leads.jsonl}}"

if ! command -v jq >/dev/null 2>&1; then
	printf 'acceptance-soak: FAIL jq required\n' >&2
	exit 1
fi

if [[ ! -s "$LEADS_JSONL" ]]; then
	printf 'acceptance-soak: FAIL missing or empty export: %s\n' "$LEADS_JSONL" >&2
	printf 'acceptance-soak: run 2h parser run on prod profile first (see docs/OPS.md#acceptance-soak)\n' >&2
	exit 1
fi

printf 'acceptance-soak: file=%s rows=%s\n' \
	"$LEADS_JSONL" "$(jq -s 'length' "$LEADS_JSONL")"

# shellcheck source=scripts/lib/soak_gate.sh
source "$ROOT/scripts/lib/soak_gate.sh"

fail=0
evaluate_acceptance_soak_gates "$LEADS_JSONL" || fail=1

printf '\nacceptance-soak: telegram High sample (first 20 snippets)\n'
jq -r 'select(.priority=="High" and (.source|startswith("telegram"))) | .snippet[0:120]' \
	"$LEADS_JSONL" 2>/dev/null | head -20 || true

if [[ "$fail" -ne 0 ]]; then
	printf 'acceptance-soak: FAIL (see soak-gate lines above)\n' >&2
	exit 1
fi

printf 'acceptance-soak: ok\n'
