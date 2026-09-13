#!/usr/bin/env bash
# User systemd units for home Squid + VPS tunnel (survives logout with linger).
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/../.." && pwd)"
UNIT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
REPO_HOME="${HOME}/dev/bidshard/lead-intent-processor"

if [[ ! -d "$REPO_HOME/scripts/home-egress" ]]; then
	echo "install-user-systemd: expected repo at $REPO_HOME" >&2
	exit 1
fi

mkdir -p "$UNIT_DIR"
for u in bidshard-home-crawl-proxy bidshard-home-tunnel; do
	sed "s|%h/dev/bidshard/lead-intent-processor|${REPO_HOME}|g" \
		"$DIR/systemd/${u}.service" >"$UNIT_DIR/${u}.service"
done

systemctl --user daemon-reload
systemctl --user enable bidshard-home-crawl-proxy.service bidshard-home-tunnel.service

if command -v loginctl >/dev/null 2>&1; then
	loginctl enable-linger "$USER" 2>/dev/null || true
fi

# Stop one-off tunnel if running; user units take over.
pkill -f 'ssh -p.*-R 127.0.0.1:19888' 2>/dev/null || true
sleep 1
systemctl --user start bidshard-home-crawl-proxy.service
systemctl --user start bidshard-home-tunnel.service

echo "install-user-systemd: ok"
systemctl --user status bidshard-home-crawl-proxy.service --no-pager | head -8
systemctl --user status bidshard-home-tunnel.service --no-pager | head -8
