#!/usr/bin/env bash
#
# Low-level BPF collector launcher (called by bpf_session.sh; prefer make bpf-session-start).
#
# Usage:
#   sudo bash scripts/perf/bpf_probe_session.sh start var/bpf-session/<ts>
#   sudo bash scripts/perf/bpf_probe_session.sh stop var/bpf-session/<ts> [pid]
#
# Requires: root, parser_probe.o, bin/bpf-collector. Env: PARSER_BPF_* (see docs/OPS.md).
#
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/go.sh"
source "$SCRIPTS/lib/bpf_collector.sh"
cd "$ROOT"

CMD="${1:?start|stop}"
OUT_DIR="${2:?output directory required}"
BPF_DIR="$OUT_DIR/bpf"
TARGETS_JSON="$BPF_DIR/targets.json"
PID_FILE="$BPF_DIR/collector.pid"
LOG_FILE="$BPF_DIR/collector.log"
READY_FILE="$BPF_DIR/collector.ready"

log() { printf 'bpf-probe-session: %s\n' "$*"; }

build_collector() {
  if [[ -x "$ROOT/bin/bpf-collector" && "${BPF_FORCE_REBUILD:-0}" != "1" ]]; then
    return 0
  fi
  mkdir -p "$ROOT/bin"
  if ! parser_go_build -o "$ROOT/bin/bpf-collector" ./cmd/bpf-collector; then
    log "ERROR: bpf-collector build failed"
    exit 1
  fi
}

sudo_collector() {
  local pass="${PARSER_BPF_SUDO_PASS:-}"
  local collector_quoted
  collector_quoted="$(printf '%q ' "${COLLECTOR_CMD[@]}")"
  # BPF needs memlock + root; optional PARSER_BPF_SUDO_PASS for non-interactive sudo.
  local launch_inner="ulimit -l unlimited 2>/dev/null || true; : > $(printf '%q' "$LOG_FILE"); nohup ${collector_quoted} >>$(printf '%q' "$LOG_FILE") 2>&1 & echo \$! > $(printf '%q' "$PID_FILE")"
  if [[ -n "$pass" ]]; then
    printf '%s\n' "$pass" | sudo -S env PATH="${PATH:-/usr/local/go/bin:/usr/bin:/bin}" PARSER_REPO_ROOT="$ROOT" \
      bash -c "$launch_inner" 2> /dev/null
  elif sudo -n true 2> /dev/null; then
    sudo -n env PATH="${PATH:-/usr/local/go/bin:/usr/bin:/bin}" PARSER_REPO_ROOT="$ROOT" \
      bash -c "$launch_inner"
  else
    return 1
  fi
}

kill_collector() {
  local pid=$1
  local pass="${PARSER_BPF_SUDO_PASS:-}"
  if [[ -z "$pid" ]]; then
    return 0
  fi
  log "stopping collector pid=$pid"
  if [[ -n "$pass" ]]; then
    printf '%s\n' "$pass" | sudo -S kill -TERM "$pid" 2> /dev/null || true
  else
    kill -TERM "$pid" 2> /dev/null || sudo -n kill -TERM "$pid" 2> /dev/null || true
  fi
  for _ in $(seq 1 50); do
    if bpf_collector_pid_alive "$pid"; then
      sleep 0.2
    else
      return 0
    fi
  done
  if [[ -n "$pass" ]]; then
    printf '%s\n' "$pass" | sudo -S kill -KILL "$pid" 2> /dev/null || true
  else
    kill -KILL "$pid" 2> /dev/null || sudo -n kill -KILL "$pid" 2> /dev/null || true
  fi
}

case "$CMD" in
  start)
    mkdir -p "$BPF_DIR"
    rm -f "$READY_FILE"
    if ! bash "$SCRIPTS/perf/bpf_requirements.sh"; then
      log "preflight failed - set PARSER_BPF_PROBE=0 to skip"
      exit 1
    fi
    if ! bpf_require_privileged_collector; then
      exit 1
    fi
    bash "$SCRIPTS/perf/bpf_build.sh"
    bash "$SCRIPTS/perf/bpf_resolve_targets.sh" "$TARGETS_JSON" "${PARSER_BPF_TARGETS:-parser,mongo,telegram}"

    build_collector
    if [[ ! -x "$ROOT/bin/bpf-collector" ]]; then
      log "ERROR: missing $ROOT/bin/bpf-collector (run: make bpf-dev)"
      exit 1
    fi
    if [[ ! -f "${PARSER_BPF_OBJECT:-$ROOT/deploy/dev/bpf/parser_probe.o}" ]]; then
      log "ERROR: missing BPF object (run: make bpf-dev)"
      exit 1
    fi

    SAMPLE="${PARSER_BPF_SAMPLE_RATE:-1}"
    SLOW_US="${PARSER_BPF_SLOW_US:-10000}"
    BPF_OBJ="${PARSER_BPF_OBJECT:-$ROOT/deploy/dev/bpf/parser_probe.o}"
    DISCOVER_LOADGEN=0
    if [[ "${PARSER_BPF_TRACK_LOADGEN:-0}" == "1" ]]; then
      DISCOVER_LOADGEN=1
    fi
    DUMP_INTERVAL="${PARSER_BPF_DUMP_INTERVAL:-0}"
    REFRESH_TARGETS="${PARSER_BPF_REFRESH_TARGETS:-0}"
    METRICS_ADDR="${PARSER_BPF_METRICS_ADDR:-}"
    LOADGEN_COMM="${PARSER_BPF_LOADGEN_COMM:-loadgen}"

    ulimit -l unlimited 2> /dev/null || true

    COLLECTOR_CMD=(
      "$ROOT/bin/bpf-collector"
      -session-dir "$BPF_DIR"
      -bpf-object "$BPF_OBJ"
      -sample-rate "$SAMPLE"
      -slow-us "$SLOW_US"
      -discover-loadgen="$DISCOVER_LOADGEN"
      -loadgen-comms "$LOADGEN_COMM"
    )
    if [[ "$DUMP_INTERVAL" != "0" ]]; then
      COLLECTOR_CMD+=(-dump-interval "${DUMP_INTERVAL}s")
    fi
    if [[ "$REFRESH_TARGETS" != "0" ]]; then
      COLLECTOR_CMD+=(-refresh-targets "${REFRESH_TARGETS}s")
    fi
    if [[ -n "$METRICS_ADDR" ]]; then
      COLLECTOR_CMD+=(-metrics-addr "$METRICS_ADDR")
    fi
    if [[ -n "${PARSER_BPF_PARSER_BINARY:-}" ]]; then
      COLLECTOR_CMD+=(-parser-binary "$PARSER_BPF_PARSER_BINARY")
    fi

    launch_bg() {
      : > "$LOG_FILE"
      nohup "$@" >> "$LOG_FILE" 2>&1 &
      echo $! > "$PID_FILE"
    }

    if [[ "$(id -u)" == "0" ]]; then
      launch_bg env PARSER_REPO_ROOT="$ROOT" "${COLLECTOR_CMD[@]}"
    elif sudo_collector; then
      :
    else
      log "ERROR: could not launch collector as root"
      exit 1
    fi

    COL_PID="$(cat "$PID_FILE")"
    if ! bpf_wait_collector_ready "$COL_PID" "$LOG_FILE"; then
      rm -f "$PID_FILE" "$READY_FILE"
      kill_collector "$COL_PID" || true
      exit 1
    fi
    date -u +%Y-%m-%dT%H:%M:%SZ > "$READY_FILE"
    log "started collector pid=$COL_PID log=$LOG_FILE"
    ;;
  stop)
    COL_PID="${3:-}"
    if [[ -z "$COL_PID" && -f "$PID_FILE" ]]; then
      COL_PID="$(cat "$PID_FILE")"
    fi
    kill_collector "$COL_PID"
    rm -f "$READY_FILE"
    if [[ -f "$LOG_FILE" ]] && grep -q "memlock rlimit" "$LOG_FILE" 2> /dev/null; then
      log "WARN: collector failed memlock - run: sudo make bpf-session-start"
    fi
    if [[ -f "$BPF_DIR/maps/summary.json" ]]; then
      log "maps dumped"
    elif [[ -f "$LOG_FILE" ]] && grep -q 'bpf-collector running' "$LOG_FILE" 2> /dev/null; then
      log "WARN: no maps/summary.json - session may have ended before dump-interval"
    else
      log "WARN: no maps/summary.json (collector may have failed - see $LOG_FILE)"
    fi
    if parser_go_run ./cmd/bpf-report "$OUT_DIR"; then
      :
    else
      log "WARN: bpf-report skipped (go unavailable or no summary)"
    fi
    ;;
  *)
    printf 'usage: %s start|stop <out_dir> [collector_pid]\n' "$0" >&2
    exit 1
    ;;
esac
