#!/usr/bin/env bash
# Curl through home tunnel on VPS (parser host, network_mode: host).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"
vps_load_config "$ROOT"

PROXY_URL="${1:-}"
if [[ -z "$PROXY_URL" && -f "$ROOT/scripts/home-egress/.credentials" ]]; then
	PROXY_URL="$(grep -m1 '^PARSER_PROXY_LIST=' "$ROOT/scripts/home-egress/.credentials" | cut -d= -f2- | tr -d '\r"')"
fi
if [[ -z "$PROXY_URL" ]]; then
	echo "usage: $0 'http://user:pass@127.0.0.1:19888'" >&2
	echo "  or run setup on home and copy scripts/home-egress/.credentials" >&2
	exit 1
fi

printf 'vps-check-home-tunnel: probing via %s\n' "$(echo "$PROXY_URL" | sed 's/:[^:@]*@/:***@/')"
vps_ssh "curl -fsS --max-time 45 -x '$PROXY_URL' -o /dev/null -w 'http_code=%{http_code}\n' https://api.ipify.org || echo FAIL"
