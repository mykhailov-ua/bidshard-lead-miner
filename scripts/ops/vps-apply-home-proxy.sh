#!/usr/bin/env bash
# Push PARSER_PROXY_LIST from scripts/home-egress/.credentials to VPS .env and restart parser.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CRED="$ROOT/scripts/home-egress/.credentials"

if [[ ! -f "$CRED" ]]; then
	echo "vps-apply-home-proxy: missing $CRED (run make home-egress-proxy-up on home)" >&2
	exit 1
fi

# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"
vps_load_config "$ROOT"

REMOTE_UPLOAD="${VPS_REMOTE_DIR}/var/home-egress.credentials.upload"
REMOTE_STORE="${VPS_REMOTE_DIR}/var/home-egress.credentials"
# shellcheck disable=SC2086
rsync -az -e "ssh $(vps_ssh_opts)" "$CRED" "$(vps_ssh_target):${REMOTE_UPLOAD}"

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
test -f .env
set -a
# shellcheck disable=SC1090
source '${REMOTE_UPLOAD}'
set +a
grep -v '^PARSER_PROXY_LIST=' .env | grep -v '^PARSER_PROXY_LIST_FILE=' > .env.tmp
printf 'PARSER_PROXY_LIST=%s\n' \"\${PARSER_PROXY_LIST}\" >> .env.tmp
if [[ -n \"\${PARSER_PROXY_SOURCES:-}\" ]]; then
  grep -v '^PARSER_PROXY_SOURCES=' .env.tmp > .env.tmp2
  mv .env.tmp2 .env.tmp
  printf 'PARSER_PROXY_SOURCES=%s\n' \"\${PARSER_PROXY_SOURCES}\" >> .env.tmp
fi
mv .env.tmp .env
chmod 600 .env
mv '${REMOTE_UPLOAD}' '${REMOTE_STORE}'
chmod 600 '${REMOTE_STORE}'
docker compose restart parser
"

echo "vps-apply-home-proxy: ok (keep make home-egress-tunnel running on your PC)"
