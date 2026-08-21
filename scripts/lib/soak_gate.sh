#!/usr/bin/env bash
# SLA gate helpers for soak / smoke logs (BOX-7).

soak_gate_fail() {
	local reason="$1"
	printf 'soak-gate: FAIL %s\n' "$reason" >&2
	return 1
}

# evaluate_soak_gates LOG...
# Exit 0 when accepted>=1 and leads_written>=1; warn-only on raw_total=0.
evaluate_soak_gates() {
	local accepted=0 leads_written=0 raw_total=0
	# shellcheck disable=SC1091
	source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/run_summary.sh"

	accepted="$(sum_log_field accepted "$@")"
	leads_written="$(sum_log_field leads_written "$@")"
	raw_total="$(sum_log_field raw_total "$@")"

	printf 'soak-gate: accepted=%s leads_written=%s raw_total=%s\n' \
		"$accepted" "$leads_written" "$raw_total"

	local fail=0
	if [[ "$accepted" == "0" ]]; then
		soak_gate_fail 'accepted=0 (check Gemini, proxy, seeds)' || fail=1
	fi
	if [[ "$leads_written" == "0" ]]; then
		soak_gate_fail 'leads_written=0 (check MONGO_URI / PARSER_EXPORT_JSON)' || fail=1
	fi
	local dropped
	dropped="$(sum_log_field dropped "$@")"
	if [[ "${dropped:-0}" != "0" ]]; then
		printf 'soak-gate: WARN dropped=%s (raise PARSER_TASK_BUFFER or PARSER_WORKERS)\n' "$dropped" >&2
	fi
	if [[ "$raw_total" == "0" ]]; then
		printf 'soak-gate: WARN raw_total=0 (crawl may be blocked; check PARSER_PROXY_LIST)\n' >&2
	fi
	return "$fail"
}

# evaluate_scan_raw_gate LABEL LOG...
# Exit 0 when raw>0 in scan-round logs.
evaluate_scan_raw_gate() {
	local label="$1"
	shift
	local raw=0
	# shellcheck disable=SC1091
	source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/run_summary.sh"
	raw="$(sum_log_field 'raw[^_]' "$@")"
	if [[ "$raw" == "0" ]]; then
		raw="$(grep -E 'scan round finished|"raw"' "$@" 2>/dev/null | sed -n 's/.*"raw"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p; s/.*raw \([0-9][0-9]*\).*/\1/p' | awk '{s+=$1} END {print s+0}')"
	fi
	printf '%s raw=%s\n' "$label" "${raw:-0}"
	if [[ "${raw:-0}" == "0" ]]; then
		soak_gate_fail "$label raw=0"
		return 1
	fi
	return 0
}
