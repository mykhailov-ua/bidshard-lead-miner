#!/usr/bin/env bash
# Run go benchmarks under a short BPF session (Linux + root).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/bpf_collector.sh"
cd "$ROOT"

SESSION_DIR="${PARSER_BPF_SESSION_ROOT:-$ROOT/var/bpf-session}/bench-refactor-$(date -u +%Y%m%dT%H%M%SZ)"
BENCH_PKGS="${1:-./internal/coldpath/... ./internal/discover/... ./internal/warmpath/... ./internal/pretty/...}"
BENCH_FILTER="${2:-Benchmark(TryCapture|MergeSuggestions|WritePending|Truncate)}"

log() { printf 'bpf-bench: %s\n' "$*"; }

if [[ "$(uname -s)" != "Linux" ]]; then
  log "WARN: BPF requires Linux; skipping"
  exit 0
fi

if ! bpf_require_privileged_collector "bpf-bench"; then
  log "WARN: no root/sudo; skipping BPF bench"
  exit 0
fi

bash "$SCRIPTS/dev/bpf_setup.sh" >/dev/null

export PARSER_BPF_NATIVE=0
export PARSER_BPF_TRACK_LOADGEN=1
export PARSER_BPF_LOADGEN_COMM="coldpath.test,discover.test,pretty.test,warmpath.test,gemini.test"
export PARSER_BPF_DUMP_INTERVAL=5
export PARSER_BPF_REFRESH_TARGETS=5
export PARSER_BPF_SAMPLE_RATE=1

log "starting session $SESSION_DIR"
bash "$SCRIPTS/dev/bpf_session.sh" start "$SESSION_DIR"

log "running benchmarks: $BENCH_PKGS filter=$BENCH_FILTER"
go test -bench="$BENCH_FILTER" -benchmem -benchtime=500ms -count=2 $BENCH_PKGS

log "stopping session"
bash "$SCRIPTS/dev/bpf_session.sh" stop "$SESSION_DIR"
bash "$SCRIPTS/dev/bpf_session.sh" report "$SESSION_DIR"

log "session dir: $SESSION_DIR"
log "report: $SESSION_DIR/bpf-report.md"
