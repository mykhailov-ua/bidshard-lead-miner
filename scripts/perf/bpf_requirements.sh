#!/usr/bin/env bash
set -euo pipefail

_SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$_SCRIPTS/lib/go.sh"

ok=0
warn=0

check() {
  local name=$1
  shift
  if "$@" > /dev/null 2>&1; then
    printf 'bpf-requirements: OK   %s\n' "$name"
  else
    printf 'bpf-requirements: FAIL %s\n' "$name" >&2
    ok=1
  fi
}

warn_check() {
  local name=$1
  shift
  if "$@" > /dev/null 2>&1; then
    printf 'bpf-requirements: OK   %s\n' "$name"
  else
    printf 'bpf-requirements: WARN %s (optional)\n' "$name" >&2
    warn=1
  fi
}

[[ -r /sys/kernel/btf/vmlinux ]] && btf=1 || btf=0
if [[ "$btf" == "1" ]]; then
  printf 'bpf-requirements: OK   BTF vmlinux\n'
else
  printf 'bpf-requirements: WARN BTF vmlinux missing (some kernels still load tracepoints)\n' >&2
  warn=1
fi

if [[ "$(id -u)" == "0" ]]; then
  printf 'bpf-requirements: OK   root privileges\n'
else
  if [[ -r /sys/kernel/debug/tracing ]]; then
    printf 'bpf-requirements: OK   tracing fs readable\n'
  else
    printf 'bpf-requirements: WARN not root; BPF attach may need sudo\n' >&2
    warn=1
  fi
fi

GO_BIN=""
if GO_BIN="$(parser_go_bin 2> /dev/null)"; then
  check go "$GO_BIN" version
else
  printf 'bpf-requirements: FAIL go (set PARSER_GO_BIN)\n' >&2
  ok=1
fi
warn_check clang clang --version
warn_check bpftool bpftool version
warn_check perf perf --version

if [[ "$ok" -ne 0 ]]; then
  exit 1
fi
exit 0
