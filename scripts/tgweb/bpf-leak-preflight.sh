#!/usr/bin/env bash
#
# Pre-deploy BPF leak gate: tgweb crawl under eBPF probe, then strict leak-gate.
#
# Usage:
#   make tgweb-bpf-leak-gate
#   DOMAINS=blask.com,vietnam.vn make tgweb-bpf-leak-gate
#   TGWEB_CRAWL_MODE=host make tgweb-bpf-leak-gate
#   PARSER_BPF_SUDO_PASS=secret make tgweb-bpf-leak-gate   # non-interactive sudo
#
# Requires: Linux, sudo (or root), .env, mongo, tgweb domain registry (make tgweb-seed).
# Artifacts: var/bpf-session/tgweb-<utc>/{bpf/maps/summary.json,bpf-report.md}
# Exit 1 when leak-gate fails (fd_delta, net_fd_estimate, thread drift).
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

log() { printf 'tgweb-bpf-leak: %s\n' "$*"; }
fail() { printf 'tgweb-bpf-leak: FAIL %s\n' "$*" >&2; exit 1; }

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

if [[ "${DEPLOY_PREFLIGHT_CI:-}" == "1" || "${GITHUB_ACTIONS:-}" == "true" ]]; then
	export PARSER_SEED_PROFILE=dev
	export PARSER_ICP_CLASSIFY_TGWEB=false
	export PARSER_BG_TELEGRAM=false
fi
source "$ROOT/scripts/lib/host_paths.sh"
apply_host_export_paths "$ROOT"

if [[ "$(uname -s)" != "Linux" ]]; then
	fail "BPF leak gate requires Linux"
fi

# shellcheck source=scripts/lib/bpf_collector.sh
source "$ROOT/scripts/lib/bpf_collector.sh"
if ! bpf_require_privileged_collector "tgweb-bpf-leak"; then
	fail "run with sudo or set PARSER_BPF_SUDO_PASS for bpf-collector attach"
fi

OBJ="$ROOT/deploy/dev/bpf/parser_probe.o"
if [[ ! -s "$OBJ" ]]; then
	log "rebuilding empty or missing parser_probe.o (no host clang: uses Docker)"
	BPF_FORCE_REBUILD=1 bash "$ROOT/scripts/perf/bpf_build.sh"
fi
if [[ ! -x "$ROOT/bin/bpf-collector" ]]; then
	log "building bpf-collector"
	bash "$ROOT/scripts/dev/bpf_setup.sh"
fi

if [[ -z "${GEMINI_API_KEY:-}" ]]; then
	log "WARN GEMINI_API_KEY unset - tgweb sync ICP gate may be skipped"
fi

proxy="${PARSER_PROXY_LIST:-}"
if [[ -z "${proxy//[[:space:]]/}" ]]; then
	log "WARN PARSER_PROXY_LIST empty - OK on home IP; use residential proxy on VPS"
fi

export PARSER_BPF_BASELINE=1
export PARSER_BPF_LEAK_GATE=1
export PARSER_BPF_LEAK_GATE_STRICT=1
export TGWEB_CRAWL_MODE="${TGWEB_CRAWL_MODE:-docker}"
export DOMAINS="${DOMAINS:-${TGWEB_BPF_LEAK_DOMAINS:-blask.com}}"

log "crawl mode=$TGWEB_CRAWL_MODE domains=$DOMAINS"
log "live metrics: ${PARSER_BPF_METRICS_ADDR:-:9464}/metrics (parser_bpf_fd_delta)"

bash "$ROOT/scripts/tgweb/crawl.sh"

SESSION="$(ls -td "$ROOT"/var/bpf-session/tgweb-* 2>/dev/null | head -1 || true)"
if [[ -z "$SESSION" || ! -f "$SESSION/bpf/maps/summary.json" ]]; then
	fail "no BPF session summary (see var/bpf-session/*/bpf/collector.log)"
fi

log "session: $SESSION"
if [[ -f "$SESSION/bpf-report.md" ]]; then
	log "report: $SESSION/bpf-report.md"
fi

if ! bash "$ROOT/scripts/lib/bpf_leak_gate.sh" "$SESSION"; then
	fail "leak gate failed - inspect $SESSION/bpf-report.md"
fi

log "ok pre-deploy BPF leak gate passed"
