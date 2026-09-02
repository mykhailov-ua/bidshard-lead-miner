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

# count_jsonl_field FILE JQ_EXPR
# Prints integer count of jq filter matches (0 on empty/missing file).
count_jsonl_field() {
	local file="$1"
	local expr="$2"
	if [[ ! -s "$file" ]]; then
		printf '0'
		return 0
	fi
	if ! command -v jq >/dev/null 2>&1; then
		soak_gate_fail 'jq required for acceptance soak gates' >&2 || return 1
		printf '0'
		return 1
	fi
	jq -s "[.[] | select(${expr})] | length" "$file" 2>/dev/null || printf '0'
}

# evaluate_acceptance_pending_gate FILE [MAX_PENDING_PCT]
# Exit 0 when pending share is below threshold (default 20%).
evaluate_acceptance_pending_gate() {
	local file="$1"
	local max_pct="${2:-20}"
	if [[ ! -s "$file" ]]; then
		soak_gate_fail "export empty: $file" || return 1
	fi
	if ! command -v jq >/dev/null 2>&1; then
		soak_gate_fail 'jq required' || return 1
	fi
	local total pending done failed pending_pct
	total="$(jq -s 'length' "$file")"
	pending="$(jq -s '[.[] | select(.analysis_status == "pending")] | length' "$file")"
	done="$(jq -s '[.[] | select(.analysis_status == "done")] | length' "$file")"
	failed="$(jq -s '[.[] | select(.analysis_status == "failed" or .analysis_status == "geo_rejected")] | length' "$file")"
	if [[ "${total:-0}" -eq 0 ]]; then
		soak_gate_fail 'export has 0 rows' || return 1
	fi
	pending_pct=$(( pending * 100 / total ))
	printf 'soak-gate: analysis_status pending=%s done=%s failed=%s pending_pct=%s max=%s\n' \
		"$pending" "$done" "$failed" "$pending_pct" "$max_pct"
	if [[ "$pending_pct" -gt "$max_pct" ]]; then
		soak_gate_fail "pending_pct=${pending_pct} exceeds ${max_pct}% (check GEMINI_MODEL, warm path, WARM_ANALYSIS_PENDING_SCAN_INTERVAL)" || return 1
	fi
	return 0
}

# evaluate_acceptance_lander_junk_gate FILE
# Exit 0 when no voluum/keitaro lander rows in export.
evaluate_acceptance_lander_junk_gate() {
	local file="$1"
	local junk
	junk="$(count_jsonl_field "$file" '(.source|startswith("lander:voluum") or startswith("lander:keitaro"))')"
	printf 'soak-gate: lander_competitor_junk=%s\n' "${junk:-0}"
	if [[ "${junk:-0}" != "0" ]]; then
		soak_gate_fail 'competitor lander rows in export (voluum/keitaro)' || return 1
	fi
	return 0
}

# evaluate_acceptance_css_contacts_gate FILE
# Exit 0 when no CSS artifact handles (@media, @keyframes, @supports).
evaluate_acceptance_css_contacts_gate() {
	local file="$1"
	local css
	css="$(count_jsonl_field "$file" '(.contacts[]?.value|test("^@(media|keyframes|supports)$"))')"
	printf 'soak-gate: css_contact_junk=%s\n' "${css:-0}"
	if [[ "${css:-0}" != "0" ]]; then
		soak_gate_fail 'CSS artifact contacts in export' || return 1
	fi
	return 0
}

# evaluate_acceptance_telegram_high_gate FILE [MIN_PAIN_PCT]
# Exit 0 when telegram High rows with tracker pain >= min pct (default 30).
# Warn-only when no telegram High rows (coverage not proven).
evaluate_acceptance_telegram_high_gate() {
	local file="$1"
	local min_pct="${2:-30}"
	if [[ ! -s "$file" ]]; then
		return 0
	fi
	if ! command -v jq >/dev/null 2>&1; then
		soak_gate_fail 'jq required' || return 1
	fi
	local total pain
	total="$(jq -s '[.[] | select(.priority=="High" and (.source|startswith("telegram")))] | length' "$file")"
	if [[ "${total:-0}" -eq 0 ]]; then
		printf 'soak-gate: WARN telegram_high=0 (no High telegram sample; review manually)\n' >&2
		return 0
	fi
	pain="$(jq -s '
		[.[] | select(.priority=="High" and (.source|startswith("telegram"))) |
		 select(
		   (.snippet // "" | test("voluum|keitaro|postback|tracker|adjust|affiliate|media buyer|click id|s2s"; "i"))
		   or (.matched[]? | test("voluum|keitaro|postback|tracker|adjust"; "i"))
		 )
		] | length
	' "$file")"
	local pain_pct=$(( pain * 100 / total ))
	printf 'soak-gate: telegram_high=%s pain_hits=%s pain_pct=%s min=%s\n' \
		"$total" "$pain" "$pain_pct" "$min_pct"
	if [[ "$pain_pct" -lt "$min_pct" ]]; then
		soak_gate_fail "telegram High pain_pct=${pain_pct} below ${min_pct}% (job-board noise?)" || return 1
	fi
	return 0
}

# evaluate_acceptance_soak_gates FILE...
# Run all jq export gates; exit non-zero on first failure.
evaluate_acceptance_soak_gates() {
	local file="$1"
	shift
	local fail=0
	evaluate_acceptance_pending_gate "$file" "${ACCEPTANCE_MAX_PENDING_PCT:-20}" || fail=1
	evaluate_acceptance_lander_junk_gate "$file" || fail=1
	evaluate_acceptance_css_contacts_gate "$file" || fail=1
	evaluate_acceptance_telegram_high_gate "$file" "${ACCEPTANCE_MIN_TG_PAIN_PCT:-30}" || fail=1
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
