#!/usr/bin/env bash
# Shared VPS SSH/rsync settings for lip CLI and GitHub Actions deploy.
#
# Config (first wins): process env, then config/env/.env.vps-deploy.local
#
#   VPS_SSH_HOST       SSH host or ~/.ssh/config alias (local: hostiq)
#   VPS_SSH_USER       default root
#   VPS_SSH_PORT       default 2222
#   VPS_REMOTE_DIR     default /opt/lead-intent-processor
#   VPS_RSYNC_DELETE   1 = rsync --delete (default 1)
#
set -euo pipefail

vps_load_config() {
	local root="${1:?repo root required}"

	if [[ -f "$root/config/env/.env.vps-deploy.local" ]]; then
		set -a
		# shellcheck disable=SC1091
		source "$root/config/env/.env.vps-deploy.local"
		set +a
	fi

	export VPS_SSH_HOST="${VPS_SSH_HOST:-hostiq}"
	export VPS_SSH_USER="${VPS_SSH_USER:-root}"
	export VPS_SSH_PORT="${VPS_SSH_PORT:-2222}"
	export VPS_REMOTE_DIR="${VPS_REMOTE_DIR:-/opt/lead-intent-processor}"
	export VPS_RSYNC_DELETE="${VPS_RSYNC_DELETE:-1}"
}

vps_ssh_target() {
	printf '%s@%s' "$VPS_SSH_USER" "$VPS_SSH_HOST"
}

vps_ssh_opts() {
	printf '%s' "-p ${VPS_SSH_PORT} -o BatchMode=yes -o ConnectTimeout=20"
}

vps_ssh() {
	# shellcheck disable=SC2086
	ssh $(vps_ssh_opts) "$(vps_ssh_target)" "$@"
}

# Exit 0 when SSH works; non-zero on timeout/auth/refused.
vps_ssh_probe() {
	# shellcheck disable=SC2086
	ssh $(vps_ssh_opts) "$(vps_ssh_target)" true
}

vps_ssh_print_help() {
	local root="${1:-}"
	printf 'SSH failed: %s (VPS_SSH_HOST=%s port=%s)\n' "$(vps_ssh_target)" "${VPS_SSH_HOST:-?}" "${VPS_SSH_PORT:-?}" >&2
	printf 'Try: ssh %s\n' "$(vps_ssh_target)" >&2
	if [[ -n "$root" ]]; then
		printf 'Config: %s/config/env/.env.vps-deploy.local\n' "$root" >&2
		printf 'Use ssh config alias (e.g. hostiq) instead of raw IP when ~/.ssh/config has Host entry.\n' >&2
	fi
}

vps_ssh_require() {
	local root="${1:?repo root}"
	vps_load_config "$root"
	if vps_ssh_probe; then
		return 0
	fi
	vps_ssh_print_help "$root"
	return 1
}

vps_rsync_ssh() {
	printf '%s' "ssh $(vps_ssh_opts)"
}

vps_rsync_push() {
	local root="${1:?repo root required}"
	local delete_flag=()
	if [[ "${VPS_RSYNC_DELETE:-1}" == "1" ]]; then
		delete_flag=(--delete)
	fi

	rsync -az "${delete_flag[@]}" \
		--no-perms --no-owner --no-group \
		--exclude '.git/' \
		--exclude 'var/' \
		--exclude '.env' \
		--exclude '.env.local' \
		--exclude 'config/env/.env.vps-deploy.local' \
		--exclude 'data/runtime/' \
		--exclude 'data/export/' \
		--exclude 'bin/' \
		--exclude '.venv/' \
		--exclude 'my_session*.session' \
		--exclude '*.session' \
		--exclude 'discord_token_*.txt' \
		--include 'sessions.pool.json' \
		--exclude '.cursor/' \
		--exclude 'backups/' \
		-e "$(vps_rsync_ssh)" \
		"$root/" "$(vps_ssh_target):${VPS_REMOTE_DIR}/"
}

vps_rsync_pull_export() {
	local root="${1:?repo root required}"
	local dest="${2:-$root/data/export/vps-leads.jsonl}"
	local remote="${VPS_REMOTE_DIR}/data/export/leads.jsonl"

	mkdir -p "$(dirname "$dest")"
	rsync -az -e "$(vps_rsync_ssh)" "$(vps_ssh_target):${remote}" "$dest"
}

vps_post_sync_fix() {
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; \
		if [[ -f config/env/proxy.list ]]; then chown 10001:10001 config/env/proxy.list; chmod 640 config/env/proxy.list; fi; \
		mkdir -p data/export data/runtime"
}

vps_remote_up() {
	local services="${1:-mongo parser crm-bot}"
	# shellcheck disable=SC2086
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; \
		docker compose build parser; \
		docker compose up -d --force-recreate ${services}; \
		docker compose run --rm parser config check; \
		docker compose ps"
}

vps_remote_telegram_realtime_stop() {
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
		docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime stop parser-telegram-realtime 2>/dev/null || true
		docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime rm -f parser-telegram-realtime 2>/dev/null || true
		printf 'vps: parser-telegram-realtime stopped (cron-only)\n'"
}

# Deprecated: production uses cron scrape (telegram-pain-cron.sh). Opt-in only.
vps_remote_telegram_realtime() {
	if [[ "${VPS_ENABLE_TELEGRAM_REALTIME:-}" != "1" ]]; then
		vps_remote_telegram_realtime_stop
		return 0
	fi
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'
		if ! grep -qE '^TELEGRAM_API_ID=[0-9]+' .env 2>/dev/null || ! grep -qE '^TELEGRAM_API_HASH=.+$' .env 2>/dev/null; then
			echo 'vps: skip parser-telegram-realtime (set TELEGRAM_API_ID + TELEGRAM_API_HASH in .env)'
			exit 0
		fi
		docker compose build parser
		docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime up -d --force-recreate
		docker compose -f docker-compose.telegram-realtime.yaml ps"
}
