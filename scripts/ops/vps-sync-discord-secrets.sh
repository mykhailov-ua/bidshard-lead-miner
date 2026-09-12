#!/usr/bin/env bash
# Push Discord bot token pool + channel ids to VPS .env (never logs values).
#
# Sources:
#   config/env/.env.discord.local
#   discord_token_*.txt via discord-pool-link.sh --export
#
# Usage:
#   make vps-sync-discord-secrets
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

discord_local="$ROOT/config/env/.env.discord.local"
tokens=""
channels=""

if [[ -f "$discord_local" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$discord_local"
	set +a
	tokens="${DISCORD_BOT_TOKENS:-}"
	channels="${DISCORD_CHANNEL_IDS:-}"
	if [[ -z "$tokens" && -n "${DISCORD_BOT_TOKEN:-}" ]]; then
		tokens="${DISCORD_BOT_TOKEN}"
	fi
fi

if [[ -z "$tokens" && -f "$ROOT/discord.tokens.pool.json" ]]; then
	tokens="$(bash "$ROOT/scripts/ops/discord-pool-link.sh" --export 2>/dev/null | tail -1 || true)"
fi

if [[ -z "$tokens" ]]; then
	printf 'vps-sync-discord-secrets: no tokens (set config/env/.env.discord.local or discord_token_*.txt)\n' >&2
	exit 1
fi

if [[ -z "$channels" ]]; then
	printf 'vps-sync-discord-secrets: WARN DISCORD_CHANNEL_IDS unset; crawl will skip until set\n' >&2
fi

first_token="${tokens%%,*}"

remote_env="$(mktemp)"
{
	printf 'DISCORD_BOT_TOKENS=%s\n' "$tokens"
	printf 'DISCORD_BOT_TOKEN=%s\n' "$first_token"
	if [[ -n "$channels" ]]; then
		printf 'DISCORD_CHANNEL_IDS=%s\n' "$channels"
	fi
} >"$remote_env"

scp -P "${VPS_SSH_PORT}" -o BatchMode=yes -o ConnectTimeout=20 \
	"$remote_env" "$(vps_ssh_target):/tmp/discord.env.upload"
rm -f "$remote_env"

vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
set_env() {
  k=\"\$1\"; v=\"\$2\"
  if grep -q \"^\${k}=\" .env 2>/dev/null; then
    sed -i \"s|^\${k}=.*|\${k}=\${v}|\" .env
  else
    echo \"\${k}=\${v}\" >> .env
  fi
}
while IFS= read -r line || [[ -n \"\$line\" ]]; do
  [[ -z \"\$line\" || \"\$line\" =~ ^# ]] && continue
  k=\"\${line%%=*}\"; v=\"\${line#*=}\"
  [[ -z \"\$k\" ]] && continue
  set_env \"\$k\" \"\$v\"
done </tmp/discord.env.upload
rm -f /tmp/discord.env.upload
"

printf 'vps-sync-discord-secrets: merged DISCORD_BOT_TOKENS (%s bots)\n' "$(printf '%s' "$tokens" | awk -F, '{print NF}')"
if [[ -n "$channels" ]]; then
	printf 'vps-sync-discord-secrets: merged DISCORD_CHANNEL_IDS\n'
fi
printf 'Next: ensure discord in PARSER_SOURCE, then docker compose restart parser\n'
