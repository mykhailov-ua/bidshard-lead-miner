#!/usr/bin/env bash
#
# Install Squid HTTP proxy on Ubuntu/Debian VPS (datacenter egress only).
#
# Usage (on VPS as root):
#   scp -r scripts/vps-proxy user@VPS:/tmp/bidshard-proxy
#   ssh user@VPS 'cd /tmp/bidshard-proxy && cp env.example .env.local && sudo ENV_FILE=.env.local ./install-on-vps.sh'
#
# Requires: root, ufw recommended (allow 3128 from parser host IP only).
# Not for Cloudflare igaming - see docs/ops.md#vps-squid-optional
#
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
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

if [[ "$(id -u)" -ne 0 ]]; then
	echo "Run as root: sudo $0" >&2
	exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
	echo "This installer targets Debian/Ubuntu (apt). Use setup-docker-proxy.sh elsewhere." >&2
	exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y squid apache2-utils

install -d -m 0750 /etc/squid
install -m 0644 "$DIR/squid.conf" /etc/squid/squid.conf
htpasswd -bc /etc/squid/passwd "$PROXY_USER" "$PROXY_PASS"
chmod 640 /etc/squid/passwd
chown root:proxy /etc/squid/passwd 2>/dev/null || chown root:squid /etc/squid/passwd 2>/dev/null || true

# Ubuntu package may use squid or proxy group
if getent group proxy >/dev/null 2>&1; then
	chown root:proxy /etc/squid/passwd
elif getent group squid >/dev/null 2>&1; then
	chown root:squid /etc/squid/passwd
fi

systemctl enable squid
systemctl restart squid

PUBLIC_IP="${VPS_PUBLIC_IP:-}"
if [[ -z "$PUBLIC_IP" ]]; then
	PUBLIC_IP="$(curl -fsSL --max-time 5 https://api.ipify.org 2>/dev/null || true)"
fi
if [[ -z "$PUBLIC_IP" ]]; then
	PUBLIC_IP="<VPS_PUBLIC_IP>"
fi

CRED_FILE="/root/bidshard-proxy.credentials"
cat >"$CRED_FILE" <<EOF
# BidShard crawl proxy - generated $(date -Iseconds)
PROXY_USER=$PROXY_USER
PROXY_PASS=$PROXY_PASS
PROXY_PORT=$PROXY_PORT
PARSER_PROXY_LIST=http://${PROXY_USER}:${PROXY_PASS}@${PUBLIC_IP}:${PROXY_PORT}
EOF
chmod 600 "$CRED_FILE"

echo ""
echo "=== Squid proxy installed ==="
echo "Credentials saved: $CRED_FILE"
echo ""
echo "Add to parser .env on your dev machine:"
echo "PARSER_PROXY_LIST=http://${PROXY_USER}:${PROXY_PASS}@${PUBLIC_IP}:${PROXY_PORT}"
echo ""
echo "Firewall (recommended - only your parser host IP):"
echo "  ufw allow from <YOUR_HOME_IP> to any port ${PROXY_PORT} proto tcp"
echo "  ufw enable"
echo ""
echo "Test from laptop:"
echo "  export PARSER_PROXY_LIST=http://${PROXY_USER}:${PROXY_PASS}@${PUBLIC_IP}:${PROXY_PORT}"
	echo "  $DIR/check-proxy.sh"
echo ""
