#!/usr/bin/env bash
# Show which P0 secrets are missing locally or on VPS.
#
# Usage:
#   bash scripts/ops/p0-secrets-check.sh          # local .env + optional overlays
#   bash scripts/ops/p0-secrets-check.sh --vps    # remote VPS .env
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REMOTE=0
if [[ "${1:-}" == "--vps" ]]; then
	REMOTE=1
fi

check_key() {
	local file="$1"
	local key="$2"
	if grep -qE "^${key}=.+" "$file" 2>/dev/null; then
		printf '  ok  %s\n' "$key"
	else
		printf '  MISSING  %s\n' "$key"
	fi
}

check_file() {
	local label="$1"
	local file="$2"
	printf '\n[%s] %s\n' "$label" "$file"
	for key in \
		TELEGRAM_API_ID TELEGRAM_API_HASH \
		TELEGRAM_ALERT_BOT_TOKEN TELEGRAM_ALERT_CHANNEL \
		CRM_TELEGRAM_BOT_TOKEN CRM_TELEGRAM_ALLOWED_CHAT_IDS CRM_WEBHOOK_SECRET PARSER_CRM_WEBHOOK_SECRET \
		GEMINI_API_KEY GITHUB_TOKEN; do
		check_key "$file" "$key"
	done
}

if [[ "$REMOTE" == 1 ]]; then
	# shellcheck disable=SC1091
	source "$ROOT/scripts/lib/vps_ssh.sh"
	vps_load_config "$ROOT"
	if ! vps_ssh_require "$ROOT"; then
		exit 1
	fi
	tmp="$(mktemp)"
	vps_ssh "cat '${VPS_REMOTE_DIR}/.env'" >"$tmp"
	check_file "VPS" "$tmp"
	rm -f "$tmp"
	exit 0
fi

env_file="$ROOT/.env"
if [[ ! -f "$env_file" ]]; then
	printf 'local .env missing (cp .env.example .env)\n'
	exit 1
fi

check_file "local" "$env_file"

for overlay in config/env/.env.telegram-alert.local config/env/.env.crm-telegram.local; do
	path="$ROOT/$overlay"
	if [[ -f "$path" ]]; then
		check_file "overlay" "$path"
	else
		printf '\n[overlay] %s (not created)\n' "$overlay"
		printf '  hint: cp %s.example %s\n' "${overlay%.local}" "$overlay"
	fi
done

printf '\nAfter filling secrets: make vps-deploy-p0\n'
