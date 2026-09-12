#!/usr/bin/env bash
# Copy local Telethon SQLite session into VPS parser_runtime volume.
#
# Session must be authorized with the same TELEGRAM_API_ID / TELEGRAM_API_HASH as VPS .env.
#
# Usage:
#   make vps-sync-telethon-session
#   SESSION_FILE=my_session.session make vps-sync-telethon-session
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

SESSION_FILE="${SESSION_FILE:-}"
if [[ -z "$SESSION_FILE" ]]; then
	for candidate in \
		"$ROOT/my_session.session" \
		"$ROOT/data/runtime/telethon.session" \
		"$ROOT/data/runtime/telethon.session.session"; do
		if [[ -f "$candidate" ]]; then
			SESSION_FILE="$candidate"
			break
		fi
	done
fi

if [[ -z "$SESSION_FILE" || ! -f "$SESSION_FILE" ]]; then
	printf 'vps-sync-telethon-session: no session file (set SESSION_FILE=path/to/foo.session)\n' >&2
	exit 1
fi

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

remote_tmp="/tmp/telethon.session.upload"
printf 'vps-sync-telethon-session: %s -> %s:%s (parser_runtime volume)\n' \
	"$SESSION_FILE" "$(vps_ssh_target)" "$remote_tmp"

scp -P "${VPS_SSH_PORT}" -o BatchMode=yes -o ConnectTimeout=20 \
	"$SESSION_FILE" "$(vps_ssh_target):${remote_tmp}"

vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
	if ! grep -qE '^TELEGRAM_API_ID=[0-9]+' .env 2>/dev/null || ! grep -qE '^TELEGRAM_API_HASH=.+$' .env 2>/dev/null; then
		echo 'missing TELEGRAM_API_ID/HASH on VPS; run: make vps-sync-telegram-secrets' >&2
		exit 1
	fi
	docker compose build parser
	docker compose run --rm \
		-v '${remote_tmp}:/tmp/telethon.session.upload:ro' \
		--entrypoint sh parser -c '
			install -d -m 700 /app/data/runtime
			cp /tmp/telethon.session.upload /app/data/runtime/telethon.session
			chmod 600 /app/data/runtime/telethon.session
			ls -la /app/data/runtime/telethon.session
		'
	rm -f '${remote_tmp}'
	docker compose run --rm parser telegram --dry-run
	printf \"\nvps-sync-telethon-session: ok (restart parser if it was running)\n\"
"

printf 'Local hint: cp %s data/runtime/telethon.session\n' "$SESSION_FILE"
