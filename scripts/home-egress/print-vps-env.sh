#!/usr/bin/env bash
# Print PARSER_PROXY_LIST for VPS after setup-home-proxy.sh
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
CRED="$DIR/.credentials"
if [[ ! -f "$CRED" ]]; then
	echo "missing $CRED - run make home-egress-proxy-up first" >&2
	exit 1
fi
cat "$CRED"
