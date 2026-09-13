#!/usr/bin/env bash
# True when the first configured HTTP proxy can reach https://example.com/.
#
# Usage:
#   source scripts/lib/proxy_first_url.sh
#   source scripts/lib/proxy_baseline_ok.sh
#   proxy_baseline_ok "$ROOT"
#
proxy_baseline_ok() {
	local root="${1:?repo root}"
	local url
	url="$(proxy_first_url_from_env "$root" || true)"
	if [[ -z "$url" ]]; then
		return 1
	fi
	curl -fsSL --max-time 20 -x "$url" -o /dev/null "https://example.com/"
}
