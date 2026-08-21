#!/usr/bin/env bash
#
# Curl smoke tests through PARSER_PROXY_LIST (baseline + blask.com/about).
#
# Usage:
#   make proxy-check
#   ./scripts/vps-proxy/check-proxy.sh http://user:pass@host:port
#
# Requires: PARSER_PROXY_LIST in .env or URL as first argument.
# Note: betfans.nl may 403 even when credentials are valid (CF diagnostic only).
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../" && pwd)"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

PROXY_URL="${1:-}"
if [[ -z "$PROXY_URL" ]]; then
	PROXY_URL="${PARSER_PROXY_LIST%*,*}"
	PROXY_URL="${PROXY_URL//[[:space:]]/}"
fi

if [[ -z "$PROXY_URL" ]]; then
	echo "usage: $0 [http://user:pass@host:port]" >&2
	echo "  or set PARSER_PROXY_LIST in .env" >&2
	exit 1
fi

check() {
	local url="$1"
	local label="$2"
	echo "-> $label ($url)"
	if curl -fsSL --max-time 25 -x "$PROXY_URL" -o /dev/null -w "  HTTP %{http_code} in %{time_total}s\n" "$url"; then
		return 0
	fi
	echo "  FAILED" >&2
	return 1
}

echo "Proxy: $PROXY_URL"
fail=0
check "https://example.com/" "baseline" || fail=1
check "https://blask.com/about" "blask about" || fail=1
# Diagnostic only: CF igaming may 403 even when proxy credentials are valid.
check "https://www.betfans.nl/" "betfans (may still 403)" || true

if [[ "$fail" -ne 0 ]]; then
	echo "Proxy check failed - fix credentials, firewall, or VPS reachability." >&2
	exit 1
fi

echo "ok proxy reachable"
