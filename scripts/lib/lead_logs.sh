#!/usr/bin/env bash
# VPS docker compose logs (parser / crm-bot / all).

lead_logs_from_vps() {
	local root="${1:?repo root}"
	shift || true

	local follow=0
	local tail_n=50
	local service=parser
	while [[ $# -gt 0 ]]; do
		case "$1" in
		-f | --follow) follow=1; shift ;;
		--no-follow) follow=0; shift ;;
		--tail)
			tail_n="${2:?--tail needs a number}"
			shift 2
			;;
		--service | -s)
			service="${2:?--service needs parser|crm-bot|mongo|all}"
			shift 2
			;;
		-h | --help)
			cat <<'EOF'
lead-logs - stream parser logs from VPS

Usage:
  lead-logs                      follow parser logs (last 100 lines)
  lead-logs --service crm-bot
  lead-logs --service all
  lead-logs --tail 200 --no-follow

Requires SSH: config/env/.env.vps-deploy.local
EOF
			return 0
			;;
		*)
			printf 'lead-logs: unknown arg %s\n' "$1" >&2
			return 2
			;;
		esac
	done

	# shellcheck disable=SC1091
	source "$root/scripts/lib/vps_ssh.sh"
	if ! vps_ssh_require "$root"; then
		return 1
	fi

	local follow_flag=()
	if [[ "$follow" == 1 ]]; then
		follow_flag=(-f)
	fi

	local services=()
	case "$service" in
	parser | crm-bot | mongo) services=("$service") ;;
	all) services=() ;;
	*)
		printf 'lead-logs: unknown service %q (parser|crm-bot|mongo|all)\n' "$service" >&2
		return 2
		;;
	esac

	printf 'lead-logs: %s (tail=%s follow=%s)\n' "$service" "$tail_n" "$follow"
	if [[ ${#services[@]} -gt 0 ]]; then
		vps_ssh "cd '${VPS_REMOTE_DIR}' && docker compose logs ${follow_flag[*]} --tail=${tail_n} ${services[*]}"
	else
		vps_ssh "cd '${VPS_REMOTE_DIR}' && docker compose logs ${follow_flag[*]} --tail=${tail_n}"
	fi
}
