#!/usr/bin/env bash
# Resolve one HTTP proxy URL for smoke tests (no full .env source).
#
# Usage:
#   source scripts/lib/proxy_first_url.sh
#   proxy_first_url_from_env "$ROOT"
#
set -euo pipefail

proxy_first_url_from_env() {
	local root="${1:?repo root}"
	local url="${PARSER_PROXY_LIST%%,*}"
	url="${url//[[:space:]]/}"
	if [[ -n "$url" ]]; then
		printf '%s' "$url"
		return 0
	fi

	local list_file="${PARSER_PROXY_LIST_FILE:-}"
	if [[ -z "$list_file" && -f "$root/.env" ]]; then
		list_file="$(grep -m1 '^PARSER_PROXY_LIST_FILE=' "$root/.env" | cut -d= -f2- | tr -d '\r"')"
	fi
	if [[ -z "$list_file" ]]; then
		return 1
	fi
	if [[ "$list_file" != /* ]]; then
		list_file="$root/$list_file"
	fi
	if [[ ! -f "$list_file" ]]; then
		return 1
	fi

	local line
	line="$(grep -v '^[[:space:]]*#' "$list_file" | grep -v '^[[:space:]]*$' | head -1 | tr -d '\r')"
	if [[ -z "$line" ]]; then
		return 1
	fi
	if [[ "$line" == http://* || "$line" == https://* ]]; then
		printf '%s' "$line"
	else
		printf 'http://%s' "$line"
	fi
}
