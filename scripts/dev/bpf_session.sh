#!/usr/bin/env bash
#
# Start/stop BPF collector session; symlink var/bpf-session/current to active out dir.
#
# Usage:
#   sudo make bpf-session-start
#   sudo make bpf-session-stop
#   sudo bash scripts/dev/bpf_session.sh status
#   sudo bash scripts/dev/bpf_session.sh report
#
# Requires: root/CAP_BPF, prior make bpf-dev. Report: var/bpf-session/<ts>/bpf-report.md
#
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/bpf_collector.sh"
cd "$ROOT"

CMD="${1:-status}"
SESSION_ROOT="${PARSER_BPF_SESSION_ROOT:-$ROOT/var/bpf-session}"
CURRENT_LINK="$SESSION_ROOT/current"
CURRENT_PATH_FILE="$SESSION_ROOT/current_path"

log() { printf 'bpf-session: %s\n' "$*"; }

resolve_out_dir() {
  local explicit="${1:-}"
  if [[ -n "$explicit" ]]; then
    printf '%s' "$explicit"
    return
  fi
  if [[ -f "$CURRENT_PATH_FILE" ]]; then
    cat "$CURRENT_PATH_FILE"
    return
  fi
  printf '%s/%s' "$SESSION_ROOT" "$(date -u +%Y%m%dT%H%M%SZ)"
}

case "$CMD" in
  start)
    OUT="$(resolve_out_dir "${2:-}")"
    if [[ "$OUT" != */* ]]; then
      OUT="$SESSION_ROOT/$OUT"
    fi
    mkdir -p "$OUT"
    export PARSER_BPF_NATIVE="${PARSER_BPF_NATIVE:-1}"
    export PARSER_BPF_TRACK_LOADGEN="${PARSER_BPF_TRACK_LOADGEN:-0}"
    export PARSER_BPF_DUMP_INTERVAL="${PARSER_BPF_DUMP_INTERVAL:-30}"
    export PARSER_BPF_REFRESH_TARGETS="${PARSER_BPF_REFRESH_TARGETS:-30}"
    export PARSER_BPF_METRICS_ADDR="${PARSER_BPF_METRICS_ADDR:-:9464}"
    bash "$SCRIPTS/perf/bpf_probe_session.sh" start "$OUT"
    if [[ ! -f "$OUT/bpf/collector.ready" ]]; then
      log "ERROR: bpf collector did not start - see $OUT/bpf/collector.log"
      exit 1
    fi
    mkdir -p "$SESSION_ROOT"
    # current -> basename of OUT; current_path stores absolute path for scripts that avoid symlink traversal.
    ln -sfn "$(basename "$OUT")" "$CURRENT_LINK"
    printf '%s\n' "$OUT" > "$CURRENT_PATH_FILE"
    log "session started: $OUT"
    log "metrics: ${PARSER_BPF_METRICS_ADDR} (/metrics)"
    log "stop: make bpf-session-stop"
    ;;
  stop)
    OUT="$(resolve_out_dir "${2:-}")"
    PID=""
    if [[ -f "$OUT/bpf/collector.pid" ]]; then
      PID="$(cat "$OUT/bpf/collector.pid")"
    fi
    bash "$SCRIPTS/perf/bpf_probe_session.sh" stop "$OUT" "$PID"
    log "session stopped: $OUT"
    ;;
  status)
    if [[ -f "$CURRENT_PATH_FILE" ]]; then
      OUT="$(cat "$CURRENT_PATH_FILE")"
      log "current session: $OUT"
      if [[ -f "$OUT/bpf/collector.pid" ]]; then
        PID="$(cat "$OUT/bpf/collector.pid")"
        if bpf_collector_pid_alive "$PID" && [[ -f "$OUT/bpf/collector.ready" ]]; then
          log "collector: running pid=$PID"
        elif bpf_collector_pid_alive "$PID"; then
          log "collector: pid=$PID (not ready - see $OUT/bpf/collector.log)"
        else
          log "collector: dead (stale pid=$PID)"
        fi
      else
        log "collector: not running"
      fi
    else
      log "no active session (run: sudo make bpf-session-start)"
    fi
    ;;
  report)
    OUT="$(resolve_out_dir "${2:-}")"
    source "$SCRIPTS/lib/go.sh"
    if ! parser_go_run ./cmd/bpf-report "$OUT"; then
      log "ERROR: report failed"
      exit 1
    fi
    log "report: $OUT/bpf-report.md"
    ;;
  *)
    printf 'usage: %s start|stop|status|report [session_dir]\n' "$0" >&2
    exit 2
    ;;
esac
