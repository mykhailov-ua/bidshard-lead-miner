#!/usr/bin/env bash
# Shared lead tail stream (VPS JSONL over SSH or local file).

lead_tail_local() {
	local root="${1:?repo root}"
	local path="${2:-$root/data/export/leads.jsonl}"
	shift 2 || shift 1 || true

	local min_score=0
	local as_json=0
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--min-score)
			min_score="${2:?--min-score needs a number}"
			shift 2
			;;
		--json) as_json=1; shift ;;
		*) shift ;;
		esac
	done

	if [[ ! -f "$path" ]]; then
		printf 'lead-tail: local file missing: %s\n' "$path" >&2
		printf 'lead-tail: run: lip export   (when VPS is up)\n' >&2
		return 1
	fi

	printf 'lead-tail: following local %s (Ctrl+C to stop)\n' "$path"

	local fmt_args=(tail --jsonl "$path")
	if [[ "$min_score" -gt 0 ]]; then
		fmt_args+=(--min-score "$min_score")
	fi
	if [[ "$as_json" == 1 ]]; then
		fmt_args+=(--json)
	fi
	if [[ -x "$root/bin/crm-bot" ]]; then
		exec "$root/bin/crm-bot" "${fmt_args[@]}"
	fi
	if command -v go >/dev/null 2>&1; then
		( cd "$root" && exec go run ./cmd/crm-bot "${fmt_args[@]}" )
	fi
	exec tail -n 0 -f "$path"
}

lead_tail_from_vps() {
	local root="${1:?repo root}"
	shift || true

	local min_score=0
	local as_json=0
	local local_path=""
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--local)
			if [[ -n "${2:-}" && "$2" != --* ]]; then
				local_path="$2"
				shift 2
			else
				local_path="$root/data/export/leads.jsonl"
				shift
			fi
			;;
		--min-score)
			min_score="${2:?--min-score needs a number}"
			shift 2
			;;
		--json) as_json=1; shift ;;
		-h | --help)
			cat <<'EOF'
lead-tail - stream new accepted leads

Usage:
  lead-tail                    VPS JSONL over SSH (live)
  lead-tail --local            local data/export/leads.jsonl
  lead-tail --local /path.jsonl
  lead-tail --min-score 70
  lead-tail --json

When VPS SSH times out, use:
  lip export && lead-tail --local
EOF
			return 0
			;;
		*)
			printf 'lead-tail: unknown arg %s\n' "$1" >&2
			return 2
			;;
		esac
	done

	if [[ -n "$local_path" ]]; then
		local local_args=()
		if [[ "$min_score" -gt 0 ]]; then
			local_args+=(--min-score "$min_score")
		fi
		if [[ "$as_json" == 1 ]]; then
			local_args+=(--json)
		fi
		lead_tail_local "$root" "$local_path" "${local_args[@]}"
		return $?
	fi

	# shellcheck disable=SC1091
	source "$root/scripts/lib/vps_ssh.sh"
	if ! vps_ssh_require "$root"; then
		printf 'lead-tail: offline fallback: lip export && lead-tail --local\n' >&2
		return 1
	fi

	local remote_jsonl="${VPS_REMOTE_DIR}/data/export/leads.jsonl"
	if ! vps_ssh "test -f '${remote_jsonl}'"; then
		printf 'lead-tail: %s not found on VPS (SSH ok)\n' "$remote_jsonl" >&2
		printf 'lead-tail: on VPS .env add: PARSER_EXPORT_JSON=/app/data/export/leads.jsonl\n' >&2
		printf 'lead-tail: then: docker compose restart parser\n' >&2
		printf 'lead-tail: offline: lip export && lead-tail --local\n' >&2
		return 1
	fi

	printf 'lead-tail: streaming %s (Ctrl+C to stop)\n' "$remote_jsonl"

	local formatter=()
	local fmt_args=(tail --jsonl -)
	if [[ "$min_score" -gt 0 ]]; then
		fmt_args+=(--min-score "$min_score")
	fi
	if [[ "$as_json" == 1 ]]; then
		fmt_args+=(--json)
	fi
	if [[ -x "$root/bin/crm-bot" ]]; then
		formatter=("$root/bin/crm-bot" "${fmt_args[@]}")
	elif command -v go >/dev/null 2>&1; then
		formatter=(go run ./cmd/crm-bot "${fmt_args[@]}")
	fi

	if [[ ${#formatter[@]} -gt 0 ]]; then
		( cd "$root" && vps_ssh "tail -n 0 -f '${remote_jsonl}'" | "${formatter[@]}" )
	else
		vps_ssh "tail -n 0 -f '${remote_jsonl}'"
	fi
}
