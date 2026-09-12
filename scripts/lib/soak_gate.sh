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

# evaluate_acceptance_supply_dominance_gate FILE [MAX_SUPPLY_PCT]
# Warn when supply (SSP ads.txt) rows dominate export - false GTM success per POST_MORTEM.
evaluate_acceptance_supply_dominance_gate() {
	local file="$1"
	local max_pct="${2:-40}"
	if [[ ! -s "$file" ]]; then
		return 0
	fi
	local total supply supply_pct
	total="$(jq -s 'length' "$file")"
	supply="$(jq -s '[.[] | select(.source == "supply" or (.source|startswith("supply:")))] | length' "$file")"
	if [[ "${total:-0}" -eq 0 ]]; then
		return 0
	fi
	supply_pct=$(( supply * 100 / total ))
	printf 'soak-gate: export_rows=%s supply_rows=%s supply_pct=%s max=%s\n' \
		"$total" "$supply" "$supply_pct" "$max_pct"
	if [[ "$supply_pct" -gt "$max_pct" ]]; then
		soak_gate_fail "supply_pct=${supply_pct} exceeds ${max_pct}% (SSP adops noise; drop supply from hot path)" || return 1
	fi
	return 0
}

# evaluate_acceptance_intent_source_gate FILE [MIN_INTENT_PCT]
# Require forum/telegram/github/reddit/webpain rows in export (BidShard ICP sources).
evaluate_acceptance_intent_source_gate() {
	local file="$1"
	local min_pct="${2:-25}"
	if [[ ! -s "$file" ]]; then
		soak_gate_fail "export empty: $file" || return 1
	fi
	local total intent intent_pct
	total="$(jq -s 'length' "$file")"
	intent="$(jq -s '
		[.[] | select(
		  .source == "forum" or .source == "reddit" or .source == "github"
		  or .source == "webpain" or .source == "serp" or .source == "reviews"
		  or (.source|startswith("forum:")) or (.source|startswith("telegram"))
		  or (.source|startswith("github:")) or (.source|startswith("webpain:"))
		  or (.source|startswith("reddit:")) or (.source|startswith("serp:"))
		)] | length
	' "$file")"
	if [[ "${total:-0}" -eq 0 ]]; then
		soak_gate_fail 'export has 0 rows' || return 1
	fi
	intent_pct=$(( intent * 100 / total ))
	printf 'soak-gate: intent_source_rows=%s intent_pct=%s min=%s\n' \
		"$intent" "$intent_pct" "$min_pct"
	if [[ "$intent_pct" -lt "$min_pct" ]]; then
		soak_gate_fail "intent_source_pct=${intent_pct} below ${min_pct}% (check PARSER_SOURCE, proxy, seeds)" || return 1
	fi
	return 0
}

# evaluate_acceptance_jobboard_intel_gate FILE
# Jobboard rows are company OSINT only; warn when they dominate without intent sources.
evaluate_acceptance_jobboard_intel_gate() {
	local file="$1"
	if [[ ! -s "$file" ]]; then
		return 0
	fi
	local jobboard intent
	jobboard="$(jq -s '[.[] | select(.source|startswith("jobboard"))] | length' "$file")"
	intent="$(jq -s '
		[.[] | select(
		  (.source|startswith("forum")) or (.source|startswith("telegram"))
		  or .source == "reddit" or (.source|startswith("github:"))
		)] | length
	' "$file")"
	printf 'soak-gate: jobboard_rows=%s intent_pain_rows=%s\n' "${jobboard:-0}" "${intent:-0}"
	if [[ "${jobboard:-0}" -gt 0 ]]; then
		printf 'soak-gate: WARN jobboard_accepted=%s (expect 0 after no_reachable_contact gate)\n' "${jobboard:-0}" >&2
	fi
	if [[ "${jobboard:-0}" -gt 0 && "${intent:-0}" -eq 0 ]]; then
		printf 'soak-gate: WARN jobboard-only export (employer OSINT; not outreach-ready)\n' >&2
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
	evaluate_acceptance_supply_dominance_gate "$file" "${ACCEPTANCE_MAX_SUPPLY_PCT:-40}" || fail=1
	evaluate_acceptance_intent_source_gate "$file" "${ACCEPTANCE_MIN_INTENT_PCT:-25}" || fail=1
	evaluate_acceptance_jobboard_intel_gate "$file" || true
	evaluate_acceptance_lander_junk_gate "$file" || fail=1
	evaluate_acceptance_css_contacts_gate "$file" || fail=1
	evaluate_acceptance_telegram_high_gate "$file" "${ACCEPTANCE_MIN_TG_PAIN_PCT:-30}" || fail=1
	print_soak_contact_valid_rate "$file" "${SOAK_MIN_CONTACT_VALID_PCT:-95}" || fail=1
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

# soak_sum_prometheus_counters METRICS PATTERN
# Sum the last field on lines matching extended grep PATTERN (0 when empty).
soak_sum_prometheus_counters() {
	local metrics="${1:-}"
	local pattern="${2:?pattern required}"
	if [[ -z "$metrics" ]]; then
		printf '0'
		return 0
	fi
	printf '%s\n' "$metrics" | grep -E "$pattern" | awk '{s+=$2} END {print s+0}'
}

# soak_prometheus_counter METRICS METRIC_NAME [REASON_LABEL]
# Read a single counter value (0 when missing).
soak_prometheus_counter() {
	local metrics="${1:-}"
	local name="${2:?metric name required}"
	local reason="${3:-}"
	if [[ -z "$metrics" ]]; then
		printf '0'
		return 0
	fi
	if [[ -n "$reason" ]]; then
		printf '%s\n' "$metrics" | grep -E "^${name}\{reason=\"${reason}\"\}" | awk '{print $2}' | tail -1
	else
		printf '%s\n' "$metrics" | grep -E "^${name} " | awk '{print $2}' | tail -1
	fi
}

# print_soak_source_breakdown FILE [TOP_N]
print_soak_source_breakdown() {
	local file="$1"
	local top_n="${2:-15}"
	if [[ ! -s "$file" ]]; then
		printf 'soak-gate: source_breakdown unavailable (empty export)\n'
		return 0
	fi
	if ! command -v jq >/dev/null 2>&1; then
		soak_gate_fail 'jq required for source_breakdown' || return 1
	fi
	printf 'soak-gate: source_breakdown (top %s)\n' "$top_n"
	jq -s --argjson n "$top_n" '
	  def family:
	    if .source == "" or .source == null then "unknown"
	    elif .source | contains(":") then (.source | split(":")[0])
	    else .source end;
	  [group_by(family)[] | {family: (.[0] | family), n: length}]
	  | sort_by(-.n) | .[:$n][] | "\(.family)\t\(.n)"
	' -r "$file" 2>/dev/null || true
}

# print_soak_contact_valid_rate FILE [MIN_PCT]
# Reachable = telegram:@user, forum:user, or email (not @serp: synthetic).
print_soak_contact_valid_rate() {
	local file="$1"
	local min_pct="${2:-95}"
	if [[ ! -s "$file" ]]; then
		soak_gate_fail "export empty: $file" || return 1
	fi
	if ! command -v jq >/dev/null 2>&1; then
		soak_gate_fail 'jq required' || return 1
	fi
	local total valid pct
	total="$(jq -s 'length' "$file")"
	valid="$(jq -s '
	  def reachable:
	    any(.contacts[]?;
	      if type == "object" then
	        (.type == "telegram" and (.value | test("^@[A-Za-z0-9_]{3,}$")))
	        or (.type == "forum_user" and (.value | length > 0))
	        or (.type == "email" and (.value | test("^[^@]+@[^@]+\\.")))
	      elif type == "string" then
	        (startswith("telegram:@") and (contains("@serp:") | not))
	        or startswith("forum:user/")
	        or test("^[^@]+@[^@]+\\.")
	      else false end
	    );
	  [.[] | select(reachable)] | length
	' "$file")"
	if [[ "${total:-0}" -eq 0 ]]; then
		soak_gate_fail 'export has 0 rows' || return 1
	fi
	pct=$(( valid * 100 / total ))
	printf 'soak-gate: contact_valid_rate=%s%% valid=%s total=%s min=%s%%\n' \
		"$pct" "$valid" "$total" "$min_pct"
	if [[ "$pct" -lt "$min_pct" ]]; then
		soak_gate_fail "contact_valid_rate=${pct}% below ${min_pct}%" || return 1
	fi
	return 0
}

# print_soak_synthetic_reject_rate METRICS
# Approximate raw = accepted + all processor rejects (Prometheus counters).
print_soak_synthetic_reject_rate() {
	local metrics="${1:-}"
	if [[ -z "$metrics" ]]; then
		printf 'soak-gate: synthetic_reject_rate unavailable (no metrics)\n'
		return 0
	fi
	local synthetic raw accepted rejects pct sc sf
	sc="$(soak_prometheus_counter "$metrics" parser_processor_reject_total synthetic_contact)"
	sf="$(soak_prometheus_counter "$metrics" parser_processor_reject_total serp_forum_snippet)"
	sc="${sc:-0}"
	sf="${sf:-0}"
	synthetic=$(( sc + sf ))
	accepted="$(soak_sum_prometheus_counters "$metrics" '^parser_leads_accepted_total\{')"
	rejects="$(soak_sum_prometheus_counters "$metrics" '^parser_processor_reject_total\{')"
	accepted="${accepted:-0}"
	rejects="${rejects:-0}"
	raw=$(( accepted + rejects ))
	if [[ "$raw" -le 0 ]]; then
		printf 'soak-gate: synthetic_reject_rate unavailable (raw_proxy=0)\n'
		return 0
	fi
	pct=$(( synthetic * 100 / raw ))
	printf 'soak-gate: synthetic_reject_rate=%s%% synthetic=%s raw_proxy=%s (accepted=%s rejects=%s)\n' \
		"$pct" "$synthetic" "$raw" "$accepted" "$rejects"
	return 0
}

# print_soak_alert_per_day LOG_TEXT SOAK_HOURS
# LOG_TEXT is newline-separated docker log lines containing pain alert sends.
print_soak_alert_per_day() {
	local log_text="${1:-}"
	local hours="${2:-24}"
	if [[ -z "$log_text" ]]; then
		printf 'soak-gate: alert_per_day unavailable (no log text)\n'
		return 0
	fi
	local count per_day tracker crypto
	count="$(printf '%s\n' "$log_text" | grep -c 'pain alert sent' || true)"
	tracker="$(printf '%s\n' "$log_text" | grep -c 'tag=tracker_pain' || true)"
	crypto="$(printf '%s\n' "$log_text" | grep -c 'tag=crypto_gray' || true)"
	if [[ "${hours:-0}" -le 0 ]]; then
		hours=24
	fi
	# Scale to 24h window (integer math).
	per_day=$(( count * 24 / hours ))
	printf 'soak-gate: alert_per_day=%s (window_h=%s alerts=%s tracker_pain=%s crypto_gray=%s)\n' \
		"$per_day" "$hours" "$count" "$tracker" "$crypto"
	return 0
}
