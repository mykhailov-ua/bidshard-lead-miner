#!/usr/bin/env bash
# Resolve lead-intent-processor repo root for global CLI wrappers.

lead_cli_script_path() {
	if command -v readlink >/dev/null 2>&1; then
		readlink -f "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"
	else
		printf '%s\n' "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"
	fi
}

lead_cli_resolve_root() {
	if [[ -n "${LEAD_INTENT_PROCESSOR_ROOT:-}" ]]; then
		printf '%s\n' "$LEAD_INTENT_PROCESSOR_ROOT"
		return 0
	fi
	local marker="${HOME}/.config/lead-intent-processor/root"
	if [[ -f "$marker" ]]; then
		local root
		root="$(tr -d '[:space:]' <"$marker")"
		if [[ -d "$root/scripts/ops/lip" ]]; then
			printf '%s\n' "$root"
			return 0
		fi
	fi
	local caller
	caller="$(lead_cli_script_path)"
	local here
	here="$(cd "$(dirname "$caller")" && pwd)"
	printf '%s\n' "$(cd "$here/../.." && pwd)"
}

lead_cli_require_root() {
	local root
	root="$(lead_cli_resolve_root)"
	if [[ ! -f "$root/scripts/ops/lip" ]]; then
		printf '%s: repo not found at %s\n' "${1:-lead-cli}" "$root" >&2
		printf '%s: run: make install-lead-tail\n' "${1:-lead-cli}" >&2
		return 1
	fi
	printf '%s\n' "$root"
}
