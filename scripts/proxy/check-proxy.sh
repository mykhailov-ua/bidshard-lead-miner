#!/usr/bin/env bash
#
# Wrapper: verify PARSER_PROXY_LIST via scripts/vps-proxy/check-proxy.sh.
#
# Usage:
#   make proxy-check
#   ./scripts/proxy/check-proxy.sh http://user:pass@host:port
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
exec "$ROOT/scripts/vps-proxy/check-proxy.sh" "$@"
