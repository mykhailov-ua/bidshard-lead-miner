#!/usr/bin/env bash
# Push Telegram MTProto + bot tokens from local .env to VPS .env (never logs values).
#
# Secrets only (API id/hash, phone, bot tokens). Chat ids stay on VPS via vps-apply-p0-env.
#
# Sources:
#   .env  (TELEGRAM_API_*, bot tokens)
#   config/env/.env.telegram-alert.local  (optional TELEGRAM_ALERT_CHANNEL)
#   config/env/.env.crm-telegram.local    (optional CRM_TELEGRAM_ALLOWED_CHAT_IDS)
#
# Usage:
#   make vps-sync-telegram-secrets
#   bash scripts/ops/vps-sync-telegram-secrets.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

env_get() {
	local key="$1"
	local file="$2"
	local line
	line="$(grep -E "^${key}=" "$file" 2>/dev/null | tail -1 || true)"
	if [[ -z "$line" ]]; then
		return 1
	fi
	printf '%s' "${line#*=}"
}

first_csv_token() {
	local raw="$1"
	raw="${raw// /}"
	printf '%s' "${raw%%,*}"
}

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

env_file="$ROOT/.env"
if [[ ! -f "$env_file" ]]; then
	printf 'vps-sync-telegram-secrets: missing %s\n' "$env_file" >&2
	exit 1
fi

set -a
# shellcheck disable=SC1091
source "$env_file"
alert_local="$ROOT/config/env/.env.telegram-alert.local"
crm_local="$ROOT/config/env/.env.crm-telegram.local"
if [[ -f "$alert_local" ]]; then
	# shellcheck disable=SC1091
	source "$alert_local"
fi
if [[ -f "$crm_local" ]]; then
	# shellcheck disable=SC1091
	source "$crm_local"
fi
set +a

api_id="${TELEGRAM_API_ID:-}"
api_hash="${TELEGRAM_API_HASH:-}"
phone="${TELEGRAM_PHONE:-}"
password="${TELEGRAM_PASSWORD:-}"

alert_token="${TELEGRAM_ALERT_BOT_TOKEN:-}"
alert_channel=""
alert_enabled="${TELEGRAM_ALERT_ENABLED:-}"
crm_bot="${CRM_TELEGRAM_BOT_TOKEN:-${CRM_BOT_TOKEN:-}}"
crm_chats=""
webhook_secret="${CRM_WEBHOOK_SECRET:-}"

if [[ -z "$alert_token" ]]; then
	alert_token="${CRM_TELEGRAM_BOT_TOKEN:-${CRM_BOT_TOKEN:-}}"
fi
# Chat ids only from explicit local overlay files (not root .env).
if [[ -f "$alert_local" ]]; then
	alert_channel="$(env_get TELEGRAM_ALERT_CHANNEL "$alert_local" 2>/dev/null || true)"
	alert_enabled="${alert_enabled:-$(env_get TELEGRAM_ALERT_ENABLED "$alert_local" 2>/dev/null || true)}"
fi
if [[ -f "$crm_local" ]]; then
	crm_chats="$(env_get CRM_TELEGRAM_ALLOWED_CHAT_IDS "$crm_local" 2>/dev/null || true)"
fi

fragment="$(mktemp)"
trap 'rm -f "$fragment"' EXIT

write_kv() {
	local k="$1"
	local v="$2"
	if [[ -z "$v" ]]; then
		return 0
	fi
	printf '%s=%s\n' "$k" "$v" >>"$fragment"
}

write_kv TELEGRAM_API_ID "$api_id"
write_kv TELEGRAM_API_HASH "$api_hash"
write_kv TELEGRAM_PHONE "$phone"
write_kv TELEGRAM_PASSWORD "$password"
write_kv TELEGRAM_ALERT_ENABLED "$alert_enabled"
write_kv TELEGRAM_ALERT_BOT_TOKEN "$alert_token"
write_kv TELEGRAM_ALERT_CHANNEL "$alert_channel"
write_kv CRM_TELEGRAM_BOT_TOKEN "$crm_bot"
write_kv CRM_TELEGRAM_ALLOWED_CHAT_IDS "$crm_chats"
write_kv CRM_WEBHOOK_SECRET "$webhook_secret"
write_kv PARSER_CRM_WEBHOOK_SECRET "$webhook_secret"

if [[ ! -s "$fragment" ]]; then
	printf 'vps-sync-telegram-secrets: nothing to sync (fill TELEGRAM_API_* in .env)\n' >&2
	exit 1
fi

remote_fragment="${VPS_REMOTE_DIR}/config/env/.env.telegram-sync.local"
vps_ssh "mkdir -p '${VPS_REMOTE_DIR}/config/env'"
# shellcheck disable=SC2086
rsync -az -e "ssh $(vps_ssh_opts)" "$fragment" "$(vps_ssh_target):${remote_fragment}"

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
touch .env
while IFS= read -r line || [[ -n \"\$line\" ]]; do
  [[ -z \"\$line\" || \"\$line\" =~ ^# ]] && continue
  k=\"\${line%%=*}\"
  v=\"\${line#*=}\"
  [[ -z \"\$k\" ]] && continue
  tmp=\"\$(mktemp)\"
  grep -v \"^\${k}=\" .env >\"\$tmp\" 2>/dev/null || true
  printf '%s=%s\n' \"\$k\" \"\$v\" >>\"\$tmp\"
  mv \"\$tmp\" .env
done <'${remote_fragment}'
rm -f '${remote_fragment}'
chmod 600 .env
"

printf 'vps-sync-telegram-secrets: merged keys (values not printed):\n'
cut -d= -f1 <"$fragment" | sed 's/^/  /'
printf '\nNext: make vps-telegram-login (session) then make vps-deploy-p0\n'
