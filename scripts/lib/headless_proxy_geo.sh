#!/usr/bin/env bash
# Export PARSER_HEADLESS_LOCALE/TIMEZONE from proxy username when unset.
headless_export_proxy_geo_env() {
	local root="${1:-}"
	if [[ -z "$root" ]]; then
		root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
	fi
	if [[ -n "${PARSER_HEADLESS_LOCALE:-}" && -n "${PARSER_HEADLESS_TIMEZONE:-}" ]]; then
		return 0
	fi
	if [[ -z "${PARSER_PROXY_LIST:-}" && -z "${PARSER_PROXY_LIST_FILE:-}" ]]; then
		return 0
	fi
	local line
	line="$(PYTHONPATH="$root" python3 -m sources.headless.geo_env 2>/dev/null || true)"
	if [[ -n "$line" ]]; then
		# shellcheck disable=SC1090
		eval "$line"
	fi
}
