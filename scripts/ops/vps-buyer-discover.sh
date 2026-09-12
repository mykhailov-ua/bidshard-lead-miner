#!/usr/bin/env bash
# Start buyer-discover crawl on VPS (detached).
#
# Usage:
#   bash scripts/ops/vps-buyer-discover.sh
#   bash scripts/ops/vps-buyer-discover.sh --detach
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

DETACH=1
SCAN_SOURCES="forum,jobboard,tgweb,serp"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--detach)
		DETACH=1
		shift
		;;
	--sources)
		SCAN_SOURCES="${2:-}"
		shift 2
		;;
	*)
		SCAN_SOURCES="$1"
		shift
		;;
	esac
done

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

if [[ "$DETACH" == "1" ]]; then
	vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
mkdir -p var
if [[ -f var/buyer-discover.pid ]] && kill -0 \"\$(cat var/buyer-discover.pid)\" 2>/dev/null; then
  printf 'buyer-discover already running pid=%s\n' \"\$(cat var/buyer-discover.pid)\"
  exit 0
fi
nohup bash scripts/ops/buyer-discover-inplace.sh '${SCAN_SOURCES}' \
  >> var/buyer-discover-nohup.log 2>&1 &
echo \$! > var/buyer-discover.pid
printf 'buyer-discover detached pid=%s log=var/buyer-discover-run.log\n' \"\$(cat var/buyer-discover.pid)\"
"
	printf 'vps-buyer-discover: detached (sources=%s)\n' "$SCAN_SOURCES"
	exit 0
fi

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
bash scripts/ops/buyer-discover-inplace.sh '${SCAN_SOURCES}'
"

printf 'vps-buyer-discover: ok (sources=%s)\n' "$SCAN_SOURCES"
