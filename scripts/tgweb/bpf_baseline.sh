#!/usr/bin/env bash
#
# Optional BPF baseline wrapper for tgweb crawl (Linux + root/CAP_BPF).
# Started after proxy preflight; stopped on crawl exit with summary.json + bpf-report.md.
#
# Enable:
#   PARSER_BPF_BASELINE=1 make tgweb-crawl
#   make tgweb-crawl-bpf
#
set -euo pipefail

TGWEB_BPF_SESSION_DIR=""
TGWEB_BPF_ROOT=""

tgweb_bpf_baseline_enabled() {
	[[ "${PARSER_BPF_BASELINE:-${TGWEB_BPF_BASELINE:-0}}" == "1" ]]
}

tgweb_bpf_baseline_log() {
	printf 'tgweb-bpf-baseline: %s\n' "$*"
}

tgweb_bpf_baseline_start() {
	local root="${1:?repo root required}"
	TGWEB_BPF_ROOT="$root"
	if ! tgweb_bpf_baseline_enabled; then
		return 0
	fi
	if [[ "$(uname -s)" != "Linux" ]]; then
		tgweb_bpf_baseline_log "WARN: BPF baseline requires Linux; skipping"
		return 0
	fi

	# shellcheck source=scripts/lib/bpf_collector.sh
	source "$root/scripts/lib/bpf_collector.sh"
	if ! bpf_require_privileged_collector "tgweb-bpf-baseline"; then
		tgweb_bpf_baseline_log "WARN: no root/sudo for BPF; skipping baseline (set PARSER_BPF_BASELINE=0 to silence)"
		return 0
	fi

	bash "$root/scripts/dev/bpf_setup.sh" >/dev/null

	local session_root="${PARSER_BPF_SESSION_ROOT:-$root/var/bpf-session}"
	TGWEB_BPF_SESSION_DIR="$session_root/tgweb-$(date -u +%Y%m%dT%H%M%SZ)"
	export PARSER_BPF_NATIVE="${PARSER_BPF_NATIVE:-1}"
	export PARSER_BPF_DUMP_INTERVAL="${PARSER_BPF_DUMP_INTERVAL:-30}"
	export PARSER_BPF_REFRESH_TARGETS="${PARSER_BPF_REFRESH_TARGETS:-30}"
	export PARSER_BPF_METRICS_ADDR="${PARSER_BPF_METRICS_ADDR:-:9464}"

	tgweb_bpf_baseline_log "starting session $TGWEB_BPF_SESSION_DIR"
	export TGWEB_BPF_SESSION_DIR
	bash "$root/scripts/dev/bpf_session.sh" start "$TGWEB_BPF_SESSION_DIR"
	tgweb_bpf_baseline_log "collector ready; crawl will be profiled"
}

tgweb_bpf_baseline_stop() {
	if [[ -z "$TGWEB_BPF_SESSION_DIR" ]]; then
		return 0
	fi
	local session_dir="$TGWEB_BPF_SESSION_DIR"
	TGWEB_BPF_SESSION_DIR=""
	export TGWEB_BPF_SESSION_DIR=""

	tgweb_bpf_baseline_log "stopping session $session_dir"
	bash "${TGWEB_BPF_ROOT:-${ROOT:-}}/scripts/dev/bpf_session.sh" stop "$session_dir" || true

	local summary="$session_dir/bpf/maps/summary.json"
	if [[ -f "$summary" ]]; then
		tgweb_bpf_baseline_log "summary: $summary"
		if [[ "${PARSER_BPF_LEAK_GATE:-0}" == "1" ]]; then
			bash "${TGWEB_BPF_ROOT:-${ROOT:-}}/scripts/lib/bpf_leak_gate.sh" "$session_dir" || tgweb_bpf_baseline_log "WARN: BPF leak gate failed"
		fi
		if [[ "${PARSER_BPF_GATE:-0}" == "1" ]]; then
			bash "${TGWEB_BPF_ROOT:-${ROOT:-}}/scripts/lib/bpf_gate.sh" "$session_dir" || tgweb_bpf_baseline_log "WARN: BPF release gate failed"
		fi
	else
		tgweb_bpf_baseline_log "WARN: no summary.json (see $session_dir/bpf/collector.log)"
	fi
	if [[ -f "$session_dir/bpf-report.md" ]]; then
		tgweb_bpf_baseline_log "report: $session_dir/bpf-report.md"
	fi
}
