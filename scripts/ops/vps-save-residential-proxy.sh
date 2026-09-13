#!/usr/bin/env bash
# Save current VPS .env residential proxy lines for failover restore (run from dev laptop).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"
vps_load_config "$ROOT"

REMOTE_DIR="${VPS_REMOTE_DIR}/var/proxy-egress"
HOME_MARK="${PROXY_FAILOVER_HOME_MARK:-127.0.0.1:19888}"

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
test -f .env
mkdir -p var/proxy-egress
list_line=\$(grep -m1 '^PARSER_PROXY_LIST=' .env || true)
file_line=\$(grep -m1 '^PARSER_PROXY_LIST_FILE=' .env || true)
src_line=\$(grep -m1 '^PARSER_PROXY_SOURCES=' .env || true)
if [[ -n \"\$list_line\" && \"\$list_line\" == *'${HOME_MARK}'* ]]; then
  echo 'vps-save-residential-proxy: .env is home tunnel; set PARSER_PROXY_LIST_FILE in .env first or use RESIDENTIAL_LIST_FILE' >&2
  exit 1
fi
if [[ -z \"\$list_line\" && -z \"\$file_line\" ]]; then
  echo 'vps-save-residential-proxy: no PARSER_PROXY_LIST or PARSER_PROXY_LIST_FILE in .env' >&2
  exit 1
fi
{
  [[ -n \"\$list_line\" ]] && printf '%s\n' \"\$list_line\"
  [[ -n \"\$file_line\" ]] && printf '%s\n' \"\$file_line\"
  [[ -n \"\$src_line\" ]] && printf '%s\n' \"\$src_line\"
} > var/proxy-egress/residential.snapshot
chmod 600 var/proxy-egress/residential.snapshot
echo vps-save-residential-proxy: ok var/proxy-egress/residential.snapshot
"
