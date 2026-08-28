#!/usr/bin/env bash
# Source from ~/.zshrc for short VPS commands.
#
#   ./scripts/ops/lip install-shell
#   source scripts/ops/lip-aliases.sh
#
if [[ -z "${LIP_ROOT:-}" ]]; then
	if [[ -n "${BASH_VERSION:-}" ]]; then
		_LIP_ALIASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
	else
		_LIP_ALIASES_DIR="$(cd "$(dirname "${(%):-%x}")" && pwd)"
	fi
	LIP_ROOT="$(cd "$_LIP_ALIASES_DIR/../.." && pwd)"
	export LIP_ROOT
fi

_lip_root() {
	if [[ -f "${LIP_ROOT}/scripts/ops/lip" ]]; then
		printf '%s' "$LIP_ROOT"
		return 0
	fi
	printf 'lip-aliases: missing %s/scripts/ops/lip\n' "$LIP_ROOT" >&2
	return 1
}

lip() {
	local root
	root="$(_lip_root)" || return 1
	"$root/scripts/ops/lip" "$@"
}

alias lip-deploy='lip deploy'
alias lip-logs='lip logs -f'
alias lip-stats='lip stats'
alias lip-export='lip export'
alias lip-ps='lip ps'
alias lip-check='lip check'
alias lip-backup='lip backup'
alias lip-ssh='lip ssh'
