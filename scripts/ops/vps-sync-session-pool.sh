#!/usr/bin/env bash
# Upload root session pool into VPS parser_runtime volume (telethon.session.N).
#
# Usage:
#   bash scripts/ops/vps-sync-session-pool.sh
#   SESSION_POOL_FILE=sessions.pool.json bash scripts/ops/vps-sync-session-pool.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

POOL="${SESSION_POOL_FILE:-$ROOT/sessions.pool.json}"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

if [[ ! -f "$POOL" ]]; then
	printf 'vps-sync-session-pool: missing %s\n' "$POOL" >&2
	exit 1
fi

bash "$ROOT/scripts/ops/session-pool-link.sh" --copy

mapfile -t rows < <(python3 - "$POOL" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    pool = json.load(fh)
for row in pool.get("sessions", []):
    print(f"{row['file']}|{row['runtime']}")
PY
)

if [[ "${#rows[@]}" -eq 0 ]]; then
	printf 'vps-sync-session-pool: empty pool\n' >&2
	exit 1
fi

vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
	if ! grep -qE '^TELEGRAM_API_ID=[0-9]+' .env 2>/dev/null || ! grep -qE '^TELEGRAM_API_HASH=.+$' .env 2>/dev/null; then
		echo 'missing TELEGRAM_API_ID/HASH on VPS; run: make vps-sync-telegram-secrets' >&2
		exit 1
	fi
	docker compose build parser
"

for row in "${rows[@]}"; do
	file="${row%%|*}"
	runtime="${row#*|}"
	local_path="$ROOT/$file"
	if [[ ! -f "$local_path" ]]; then
		printf 'vps-sync-session-pool: skip missing %s\n' "$file"
		continue
	fi
	remote_tmp="/tmp/$(basename "$file").upload"
	printf 'vps-sync-session-pool: %s -> %s\n' "$file" "$runtime"
	scp -P "${VPS_SSH_PORT}" -o BatchMode=yes -o ConnectTimeout=20 \
		"$local_path" "$(vps_ssh_target):${remote_tmp}"
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
		docker compose run --rm \
			-v '${remote_tmp}:/tmp/session.upload:ro' \
			--entrypoint sh parser -c '
				install -d -m 700 /app/data/runtime
				cp /tmp/session.upload /app/${runtime}
				chmod 600 /app/${runtime}
				ls -la /app/${runtime}
			'
		rm -f '${remote_tmp}'
	"
done

printf 'vps-sync-session-pool: ok\n'
