#!/usr/bin/env bash
# Authorize Telethon session on VPS (parser_runtime volume).
#
# Usage:
#   make vps-telegram-login              # phone step 1 (SMS code to Telegram app)
#   TELEGRAM_CODE=12345 make vps-telegram-login   # step 2
#   make vps-telegram-login-qr           # QR login (scan in Telegram app)
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

MODE="${1:-phone}"
vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

extra_env=""
if [[ -n "${TELEGRAM_CODE:-}" ]]; then
	extra_env="-e TELEGRAM_CODE=${TELEGRAM_CODE}"
fi

if [[ "$MODE" == "qr" ]]; then
	printf 'vps-telegram-login: QR mode (scan within TELEGRAM_QR_TIMEOUT_SEC, default 180s)\n'
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
		if ! grep -qE '^TELEGRAM_API_ID=[0-9]+' .env || ! grep -qE '^TELEGRAM_API_HASH=.+$' .env; then
			echo 'missing TELEGRAM_API_ID/HASH on VPS; run: make vps-sync-telegram-secrets' >&2
			exit 1
		fi
		docker compose build parser
		docker compose run --rm -e TELEGRAM_QR_TIMEOUT_SEC=300 parser telegram login --qr
		docker compose run --rm --entrypoint sh parser -c 'test -f /app/data/runtime/telethon.session && echo telethon.session ok'
	"
	exit 0
fi

printf 'vps-telegram-login: phone mode\n'
vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
	if ! grep -qE '^TELEGRAM_API_ID=[0-9]+' .env || ! grep -qE '^TELEGRAM_API_HASH=.+$' .env; then
		echo 'missing TELEGRAM_API_ID/HASH on VPS; run: make vps-sync-telegram-secrets' >&2
		exit 1
	fi
	if ! grep -qE '^TELEGRAM_PHONE=.+$' .env; then
		echo 'missing TELEGRAM_PHONE on VPS; run: make vps-sync-telegram-secrets' >&2
		exit 1
	fi
	docker compose build parser
	if docker compose run --rm ${extra_env} parser telegram login; then
		echo 'telethon.session authorized'
		exit 0
	fi
	rc=\$?
	if docker compose run --rm parser telegram login 2>&1 | grep -q 'already authorized'; then
		echo 'telethon.session authorized'
		exit 0
	fi
	exit \"\$rc\"
"
