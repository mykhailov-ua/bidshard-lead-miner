#!/usr/bin/env bash
# Install systemd timer on VPS for proxy egress failover (run from dev laptop).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"
vps_load_config "$ROOT"

UNIT_SRC="$ROOT/scripts/ops/systemd"
EXAMPLE="$ROOT/config/env/.env.proxy-egress-failover.example"

chmod +x "$ROOT/scripts/ops/proxy-egress-failover.sh" "$ROOT/scripts/ops/vps-save-residential-proxy.sh"

# shellcheck disable=SC2086
rsync -az -e "ssh $(vps_ssh_opts)" \
	"$ROOT/scripts/ops/proxy-egress-failover.sh" \
	"$(vps_ssh_target):${VPS_REMOTE_DIR}/scripts/ops/proxy-egress-failover.sh"

# shellcheck disable=SC2086
rsync -az -e "ssh $(vps_ssh_opts)" \
	"$UNIT_SRC/bidshard-proxy-egress-failover.service" \
	"$UNIT_SRC/bidshard-proxy-egress-failover.timer" \
	"$(vps_ssh_target):/tmp/"

vps_ssh "set -euo pipefail
install -m 644 /tmp/bidshard-proxy-egress-failover.service /etc/systemd/system/
install -m 644 /tmp/bidshard-proxy-egress-failover.timer /etc/systemd/system/
rm -f /tmp/bidshard-proxy-egress-failover.service /tmp/bidshard-proxy-egress-failover.timer
mkdir -p '${VPS_REMOTE_DIR}/var/proxy-egress'
if [[ ! -f '${VPS_REMOTE_DIR}/var/proxy-egress/failover.env' ]]; then
  if [[ -f '${VPS_REMOTE_DIR}/config/env/.env.proxy-egress-failover.example' ]]; then
    cp '${VPS_REMOTE_DIR}/config/env/.env.proxy-egress-failover.example' '${VPS_REMOTE_DIR}/var/proxy-egress/failover.env'
  else
    echo 'PROXY_FAILOVER_MIN_OK_RATIO=0.55' > '${VPS_REMOTE_DIR}/var/proxy-egress/failover.env'
  fi
  chmod 600 '${VPS_REMOTE_DIR}/var/proxy-egress/failover.env'
fi
chmod +x '${VPS_REMOTE_DIR}/scripts/ops/proxy-egress-failover.sh'
systemctl daemon-reload
systemctl enable --now bidshard-proxy-egress-failover.timer
systemctl start bidshard-proxy-egress-failover.service || true
echo install-proxy-egress-failover: timer active
systemctl status bidshard-proxy-egress-failover.timer --no-pager || true
"

echo "install-proxy-egress-failover: ok"
echo "  ensure: ${VPS_REMOTE_DIR}/var/home-egress.credentials"
echo "  ensure: make vps-save-residential-proxy (commercial pool snapshot)"
echo "  logs:   ssh ... journalctl -u bidshard-proxy-egress-failover.service -f"
