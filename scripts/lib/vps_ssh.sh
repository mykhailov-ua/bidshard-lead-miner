#!/usr/bin/env bash
# Shared VPS SSH/rsync settings for lip CLI and GitHub Actions deploy.
#
# Config (first wins): process env, then config/env/.env.vps-deploy.local
#
#   VPS_SSH_HOST       SSH host or ~/.ssh/config alias (local: hostiq)
#   VPS_SSH_USER       default root
#   VPS_SSH_PORT       default 22
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
	export VPS_SSH_PORT="${VPS_SSH_PORT:-22}"
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
		--exclude '.git/' \
		--exclude 'var/' \
		--exclude '.env' \
		--exclude '.env.local' \
		--exclude 'config/env/.env.vps-deploy.local' \
		--exclude 'data/runtime/' \
		--exclude 'data/export/' \
		--exclude 'bin/' \
		--exclude '.venv/' \
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

vps_remote_up() {
	local services="${1:-mongo parser}"
	# shellcheck disable=SC2086
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; docker compose build parser; docker compose up -d ${services}; docker compose run --rm parser config check; docker compose ps"
}
