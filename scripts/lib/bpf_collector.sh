#!/usr/bin/env bash

bpf_collector_pid_alive() {
  local pid="${1:-}"
  [[ -n "$pid" ]] || return 1
  if kill -0 "$pid" 2> /dev/null; then
    return 0
  fi
  if command -v sudo > /dev/null 2>&1; then
    local pass="${PARSER_BPF_SUDO_PASS:-}"
    if [[ -n "$pass" ]]; then
      printf '%s\n' "$pass" | sudo -S kill -0 "$pid" 2> /dev/null
      return $?
    fi
    sudo -n kill -0 "$pid" 2> /dev/null
    return $?
  fi
  return 1
}

bpf_collector_log_failed() {
  local log_file="${1:-}"
  [[ -f "$log_file" ]] || return 1
  grep -qE 'ERROR (rlimit|probe start|open session)' "$log_file" 2> /dev/null
}

bpf_collector_log_ready() {
  local log_file="${1:-}"
  [[ -f "$log_file" ]] && grep -q 'bpf-collector running' "$log_file" 2> /dev/null
}

bpf_wait_collector_ready() {
  local pid="${1:-}"
  local log_file="${2:-}"
  local prefix="${3:-bpf-probe-session}"
  local i

  for i in $(seq 1 30); do
    if bpf_collector_log_ready "$log_file"; then
      printf '%s: collector ready pid=%s\n' "$prefix" "$pid"
      return 0
    fi
    if bpf_collector_log_failed "$log_file"; then
      printf '%s: ERROR: collector failed to start (see %s)\n' "$prefix" "$log_file" >&2
      tail -n 8 "$log_file" >&2 || true
      return 1
    fi
    if ! bpf_collector_pid_alive "$pid"; then
      printf '%s: ERROR: collector pid=%s exited before ready\n' "$prefix" "$pid" >&2
      if [[ -f "$log_file" ]]; then
        tail -n 12 "$log_file" >&2 || true
      fi
      return 1
    fi
    sleep 0.2
  done

  printf '%s: ERROR: collector not ready after 6s (pid=%s log=%s)\n' "$prefix" "$pid" "$log_file" >&2
  if [[ -f "$log_file" ]]; then
    tail -n 12 "$log_file" >&2 || true
  fi
  return 1
}

bpf_require_privileged_collector() {
  local prefix="${1:-bpf-probe-session}"
  if [[ "$(id -u)" == "0" ]]; then
    return 0
  fi
  local pass="${PARSER_BPF_SUDO_PASS:-}"
  if [[ -n "$pass" ]]; then
    return 0
  fi
  if sudo -n true 2> /dev/null; then
    return 0
  fi
  printf '%s: ERROR: bpf-collector needs root for memlock/BPF attach\n' "$prefix" >&2
  printf '%s: run: sudo make bpf-session-start\n' "$prefix" >&2
  return 1
}
