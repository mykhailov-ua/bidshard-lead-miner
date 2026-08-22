#!/usr/bin/env bash
#
# Build parser_probe.o and bpf-collector for dev perf sessions.
#
# Usage:
#   make bpf-dev
#   bash scripts/dev/bpf_setup.sh --check
#
# Requires: Linux, clang (or docker for container build). Does not attach probes.
#
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

CHECK_ONLY=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=1
fi

log() { printf 'bpf-setup: %s\n' "$*"; }

log "preflight"
if ! bash "$SCRIPTS/perf/bpf_requirements.sh"; then
  log "WARN: BPF preflight reported failures (attach may still work as root)"
fi

if [[ "$CHECK_ONLY" == "1" ]]; then
  exit 0
fi

log "building parser_probe.o"
bash "$SCRIPTS/perf/bpf_build.sh"

log "building bpf-collector"
mkdir -p "$ROOT/bin"
if ! parser_go_build -o "$ROOT/bin/bpf-collector" ./cmd/bpf-collector; then
  log "ERROR: bpf-collector build failed - set PARSER_GO_BIN=/path/to/go"
  exit 1
fi

if ! parser_go_build -o "$ROOT/bin/bpf-report" ./cmd/bpf-report; then
  log "ERROR: bpf-report build failed - set PARSER_GO_BIN=/path/to/go"
  exit 1
fi

log "ready: deploy/dev/bpf/parser_probe.o bin/bpf-collector bin/bpf-report"
log "standalone session: sudo make bpf-session-start"
log "during tgweb crawl: sudo PARSER_BPF_NATIVE=1 make bpf-session-start && docker compose run ... telegram web"
