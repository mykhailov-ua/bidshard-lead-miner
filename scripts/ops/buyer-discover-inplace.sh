#!/usr/bin/env bash
# Discovery crawl ON the VPS (SERP harvest + CF crawl). Detached via vps-buyer-discover.sh.
#
# Usage (on VPS):
#   cd /opt/lead-intent-processor && bash scripts/ops/buyer-discover-inplace.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

LOG="var/buyer-discover-run.log"
SCAN_SOURCES="${1:-forum,jobboard,tgweb,serp}"

mkdir -p var
printf '[%s] buyer-discover inplace start sources=%s\n' "$(date -Is)" "$SCAN_SOURCES" | tee -a "$LOG"

set +e
bash "$ROOT/scripts/ops/buyer-discover.sh" "$SCAN_SOURCES" 2>&1 | tee -a "$LOG"
rc=$?
set -e

printf '[%s] buyer-discover inplace exit=%s\n' "$(date -Is)" "$rc" | tee -a "$LOG"
exit "$rc"
