#!/usr/bin/env bash
# Host-side path overrides when .env uses Docker /app/... paths.

apply_host_export_paths() {
	local root="${1:?root required}"
	export PARSER_EXPORT_JSON="${PARSER_EXPORT_JSON_HOST:-$root/data/export/leads.jsonl}"
	mkdir -p "$(dirname "$PARSER_EXPORT_JSON")"
}
