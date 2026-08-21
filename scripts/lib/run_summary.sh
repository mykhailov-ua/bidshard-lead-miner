#!/usr/bin/env bash
# Parse parser logs for collect-once / scan-round totals.

sum_log_field() {
	local field="$1"
	shift
	local total=0 val
	while IFS= read -r line; do
		val="$(printf '%s' "$line" | sed -n "s/.*${field}=\([0-9][0-9]*\).*/\1/p" | tail -1)"
		if [[ -n "$val" ]]; then
			total=$(( total + val ))
		fi
	done < <(cat "$@" 2>/dev/null || true)
	printf '%s' "$total"
}

summarize_collect_logs() {
	local label="$1"
	shift
	local accepted leads_written raw_total
	accepted="$(sum_log_field accepted "$@")"
	leads_written="$(sum_log_field leads_written "$@")"
	raw_total="$(sum_log_field raw_total "$@")"
	printf '%s accepted=%s leads_written=%s raw_total=%s\n' "$label" "$accepted" "$leads_written" "$raw_total"
}

summarize_scan_logs() {
	local label="$1"
	shift
	local accepted raw
	accepted="$(sum_log_field accepted "$@")"
	raw="$(sum_log_field 'raw[^_]' "$@")"
	if [[ "$raw" == "0" ]]; then
		raw="$(sum_log_field '"raw"' "$@")"
	fi
	# scan round finished uses "raw": N in JSON and "raw N" in pretty output
	if [[ "$raw" == "0" ]]; then
		raw="$(grep -E 'scan round finished|"raw":|"raw [0-9]' "$@" 2>/dev/null | sed -n 's/.*"raw"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p; s/.*raw \([0-9][0-9]*\).*/\1/p' | awk '{s+=$1} END {print s+0}')"
	fi
	printf '%s accepted=%s raw=%s\n' "$label" "$accepted" "${raw:-0}"
}

mongo_lead_count() {
	local root="${1:?root}"
	local uri db coll
	uri="${MONGO_URI:-mongodb://localhost:27017}"
	db="${MONGO_DB:-parser}"
	coll="${PARSER_MONGO_COLLECTION:-leads}"
	if ! command -v mongosh >/dev/null 2>&1; then
		printf 'mongo_leads=unknown (mongosh missing)\n'
		return 0
	fi
	local count
	count="$(mongosh "$uri" --quiet --eval "db.getSiblingDB('${db}').getCollection('${coll}').countDocuments({})" 2>/dev/null || true)"
	if [[ "$count" =~ ^[0-9]+$ ]]; then
		printf 'mongo_leads=%s\n' "$count"
	else
		printf 'mongo_leads=unknown\n'
	fi
}
