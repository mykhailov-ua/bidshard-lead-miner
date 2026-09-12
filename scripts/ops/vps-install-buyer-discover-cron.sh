#!/usr/bin/env bash
# Install twice-daily buyer-discover cron on VPS (SERP harvest + CF crawl).
#
# Usage:
#   make vps-install-buyer-discover-cron
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

CRON_LINE="0 6,18 * * * cd ${VPS_REMOTE_DIR} && bash scripts/ops/buyer-discover.sh >> var/buyer-discover.log 2>&1"
MARKER="# lead-intent-processor buyer-discover"

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
mkdir -p var
( crontab -l 2>/dev/null | grep -v '${MARKER}' || true
  echo '0 6,18 * * * cd ${VPS_REMOTE_DIR} && bash scripts/ops/buyer-discover.sh >> var/buyer-discover.log 2>&1 ${MARKER}'
) | crontab -
crontab -l | grep buyer-discover || true
"

printf 'vps-install-buyer-discover-cron: ok (06:00 and 18:00 UTC)\n'
