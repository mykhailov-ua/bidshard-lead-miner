#!/usr/bin/env bash
# Reverse SSH: VPS 127.0.0.1:TUNNEL_PORT -> home 127.0.0.1:Squid. Run on HOME machine.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/../.." && pwd)"
ENV_FILE="${ENV_FILE:-$DIR/.env.local}"

if [[ -f "$ENV_FILE" ]]; then
	set -a
	# shellcheck disable=SC1090
	source "$ENV_FILE"
	set +a
fi

if [[ -f "$ROOT/config/env/.env.vps-deploy.local" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/config/env/.env.vps-deploy.local"
	set +a
fi

VPS_SSH_HOST="${VPS_SSH_HOST:-hostiq}"
VPS_SSH_USER="${VPS_SSH_USER:-root}"
VPS_SSH_PORT="${VPS_SSH_PORT:-2222}"
HOME_PROXY_PORT="${HOME_PROXY_PORT:-3128}"
VPS_TUNNEL_PORT="${VPS_TUNNEL_PORT:-19888}"
USE_AUTOSSH="${USE_AUTOSSH:-1}"

REMOTE_BIND="127.0.0.1:${VPS_TUNNEL_PORT}"
LOCAL_TARGET="127.0.0.1:${HOME_PROXY_PORT}"
SSH_TARGET="${VPS_SSH_USER}@${VPS_SSH_HOST}"

SSH_BASE=(
	-p "$VPS_SSH_PORT"
	-o ServerAliveInterval=30
	-o ServerAliveCountMax=3
	-o ExitOnForwardFailure=yes
	-N
	-R "${REMOTE_BIND}:${LOCAL_TARGET}"
)

echo "home-egress: tunnel ${SSH_TARGET} ${REMOTE_BIND} -> home ${LOCAL_TARGET}"
echo "home-egress: leave this running; restart parser on VPS after updating PARSER_PROXY_LIST"

if [[ "$USE_AUTOSSH" == "1" ]] && command -v autossh >/dev/null 2>&1; then
	exec autossh -M 0 "${SSH_BASE[@]}" "$SSH_TARGET"
fi

exec ssh "${SSH_BASE[@]}" "$SSH_TARGET"
