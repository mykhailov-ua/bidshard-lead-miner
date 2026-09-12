#!/usr/bin/env bash
# Back-compat wrapper for M2 realtime soak (see vps-telegram-realtime-soak.sh).
#
# Usage:
#   bash scripts/ops/vps-realtime-soak.sh
#   SOAK_HOURS=48 M2_STRICT=1 bash scripts/ops/vps-realtime-soak.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
exec bash "$ROOT/scripts/ops/vps-telegram-realtime-soak.sh" "$@"
