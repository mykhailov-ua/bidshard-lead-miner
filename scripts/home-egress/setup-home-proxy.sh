#!/usr/bin/env bash
# Start Squid on 127.0.0.1 at home (ISP egress). Run on your home machine.
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

HOME_PROXY_PORT="${HOME_PROXY_PORT:-3128}"
PROXY_USER="${PROXY_USER:-parser}"
PROXY_PASS="${PROXY_PASS:-}"

if [[ -z "$PROXY_PASS" || "$PROXY_PASS" == "change-me-to-a-long-random-string" ]]; then
	PROXY_PASS="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
fi

if ! command -v docker >/dev/null 2>&1; then
	echo "home-egress: docker required" >&2
	exit 1
fi

write_passwd() {
	local user="$1" pass="$2" dest="$3"
	if command -v htpasswd >/dev/null 2>&1; then
		htpasswd -bc "$dest" "$user" "$pass"
	else
		printf '%s:%s\n' "$user" "$(openssl passwd -apr1 "$pass")" >"$dest"
	fi
}

write_passwd "$PROXY_USER" "$PROXY_PASS" "$DIR/passwd"

export HOME_PROXY_PORT
cd "$DIR"
docker compose -f docker-compose.home-proxy.yaml up -d --build

VPS_TUNNEL_PORT="${VPS_TUNNEL_PORT:-19888}"
SNIPPET="$DIR/.credentials"
cat >"$SNIPPET" <<EOF
# Paste on VPS /opt/lead-intent-processor/.env (parser uses network_mode: host)
PARSER_PROXY_LIST=http://${PROXY_USER}:${PROXY_PASS}@127.0.0.1:${VPS_TUNNEL_PORT}
PARSER_PROXY_SOURCES=forum,tgweb,lander,webpain,jobboard,serp
EOF
chmod 600 "$SNIPPET"

echo "home-egress: Squid on 127.0.0.1:${HOME_PROXY_PORT}"
echo "home-egress: next: make home-egress-tunnel (keep running)"
echo "home-egress: VPS env: $SNIPPET"
cat "$SNIPPET"
