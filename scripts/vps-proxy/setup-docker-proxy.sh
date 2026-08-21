#!/usr/bin/env bash
#
# Local Squid proxy in Docker (same squid.conf as VPS install).
#
# Usage:
#   cp scripts/vps-proxy/env.example scripts/vps-proxy/.env.local
#   make vps-proxy-docker
#   ./scripts/vps-proxy/print-env-snippet.sh   # paste into .env
#   make proxy-check
#
# Requires: docker, htpasswd or openssl. Not for Cloudflare igaming (use residential).
#
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

PROXY_PORT="${PROXY_PORT:-3128}"
PROXY_USER="${PROXY_USER:-parser}"
PROXY_PASS="${PROXY_PASS:-}"

if [[ -z "$PROXY_PASS" || "$PROXY_PASS" == "change-me-to-a-long-random-string" ]]; then
	PROXY_PASS="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
fi

if ! command -v docker >/dev/null 2>&1; then
	echo "docker not found. Install Docker or use install-on-vps.sh (native Squid)." >&2
	exit 1
fi

write_passwd() {
	local user="$1" pass="$2" dest="$3"
	if command -v htpasswd >/dev/null 2>&1; then
		htpasswd -bc "$dest" "$user" "$pass"
	else
		local hash
		hash="$(openssl passwd -apr1 "$pass")"
		printf '%s:%s\n' "$user" "$hash" >"$dest"
	fi
}

if ! command -v htpasswd >/dev/null 2>&1 && ! command -v openssl >/dev/null 2>&1; then
	echo "Need htpasswd (apache2-utils) or openssl." >&2
	exit 1
fi

write_passwd "$PROXY_USER" "$PROXY_PASS" "$DIR/passwd"

cd "$DIR"
docker compose -f docker-compose.proxy.yaml up -d --build

PUBLIC_IP="${VPS_PUBLIC_IP:-127.0.0.1}"
if [[ "$PUBLIC_IP" == "127.0.0.1" ]] && command -v curl >/dev/null 2>&1 && [[ -z "${VPS_PUBLIC_IP:-}" ]]; then
	echo "note: using 127.0.0.1 for local Docker proxy; set VPS_PUBLIC_IP on real VPS"
fi

SNIPPET_FILE="$DIR/.credentials"
cat >"$SNIPPET_FILE" <<EOF
PARSER_PROXY_LIST=http://${PROXY_USER}:${PROXY_PASS}@${PUBLIC_IP}:${PROXY_PORT}
EOF
chmod 600 "$SNIPPET_FILE"

echo ""
echo "=== Docker crawl proxy running on port ${PROXY_PORT} ==="
echo "Snippet: $SNIPPET_FILE"
cat "$SNIPPET_FILE"
echo ""
echo "Merge into $ROOT/.env then:"
echo "  make vps-proxy-check"
echo ""
