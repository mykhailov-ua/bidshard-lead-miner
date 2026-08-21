#!/usr/bin/env bash
#
# Print PARSER_PROXY_LIST line from Squid install credentials file.
#
# Usage:
#   ./scripts/vps-proxy/print-env-snippet.sh
#   ./scripts/vps-proxy/print-env-snippet.sh /root/bidshard-proxy.credentials
#
# Requires: prior install-on-vps.sh or setup-docker-proxy.sh (writes .credentials).
#
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/../.." && pwd)"

CRED="${1:-}"
if [[ -z "$CRED" ]]; then
	if [[ -f "$DIR/.credentials" ]]; then
		CRED="$DIR/.credentials"
	elif [[ -f /root/bidshard-proxy.credentials ]]; then
		CRED=/root/bidshard-proxy.credentials
	else
		echo "No credentials file. Run install-on-vps.sh or setup-docker-proxy.sh first." >&2
		exit 1
	fi
fi

grep '^PARSER_PROXY_LIST=' "$CRED" || {
	echo "No PARSER_PROXY_LIST in $CRED" >&2
	exit 1
}

echo ""
echo "Paste into: $ROOT/.env"
