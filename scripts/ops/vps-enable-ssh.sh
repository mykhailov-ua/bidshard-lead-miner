#!/usr/bin/env bash
# One-shot: normal SSH access (openssh + ufw + optional pubkey).
# Run once on the VPS as root (VNC console or after HOSTiQ enables SSH).
#
# Usage:
#   curl -fsSL .../vps-enable-ssh.sh | bash -s -- "$(cat ~/.ssh/id_ed25519.pub)"
#   bash scripts/ops/vps-enable-ssh.sh 'ssh-ed25519 AAAA... comment'
#
set -euo pipefail

PUBKEY="${1:-}"
if [[ -n "${SSH_PUBLIC_KEY:-}" ]]; then
	PUBKEY="$SSH_PUBLIC_KEY"
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq openssh-server ufw curl git ca-certificates

mkdir -p /root/.ssh
chmod 700 /root/.ssh
if [[ -n "$PUBKEY" ]]; then
	grep -qF "$PUBKEY" /root/.ssh/authorized_keys 2>/dev/null || echo "$PUBKEY" >> /root/.ssh/authorized_keys
	chmod 600 /root/.ssh/authorized_keys
fi

SSHD=/etc/ssh/sshd_config
cp -a "$SSHD" "${SSHD}.bak.$(date -u +%Y%m%dT%H%M%SZ)"
sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin prohibit-password/' "$SSHD"
sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication yes/' "$SSHD"
sed -i 's/^#\?PubkeyAuthentication.*/PubkeyAuthentication yes/' "$SSHD"
grep -q '^PermitRootLogin' "$SSHD" || echo 'PermitRootLogin prohibit-password' >> "$SSHD"
grep -q '^PubkeyAuthentication' "$SSHD" || echo 'PubkeyAuthentication yes' >> "$SSHD"

systemctl enable --now ssh
systemctl restart ssh

ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw --force enable

echo "ok ssh enabled on port 22"
ss -tlnp | grep ':22' || true
if [[ -n "$PUBKEY" ]]; then
	echo "ok root authorized_keys updated (key login; password login still possible until you harden)"
else
	echo "warn: no pubkey passed; set root password with: passwd root"
fi
echo "from laptop: ssh root@$(curl -fsSL -4 --max-time 5 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')"
