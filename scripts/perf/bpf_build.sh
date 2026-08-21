#!/usr/bin/env bash
#
# Compile deploy/dev/bpf/parser_probe.o (host clang or fedora container fallback).
#
# Usage:
#   bash scripts/perf/bpf_build.sh
#   BPF_FORCE_REBUILD=1 bash scripts/perf/bpf_build.sh
#
# Output: deploy/dev/bpf/parser_probe.o (gitignored)
#
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

BPF_DIR="$ROOT/deploy/dev/bpf"
OBJ="$BPF_DIR/parser_probe.o"

if [[ -f "$OBJ" && "${BPF_FORCE_REBUILD:-0}" != "1" ]]; then
  printf 'bpf-build: object exists: %s\n' "$OBJ"
  exit 0
fi

if command -v clang > /dev/null 2>&1; then
  make -C "$BPF_DIR"
  printf 'bpf-build: built with host clang -> %s\n' "$OBJ"
  exit 0
fi

if ! command -v docker > /dev/null 2>&1; then
  printf 'bpf-build: ERROR: clang not found and docker unavailable\n' >&2
  exit 1
fi

printf 'bpf-build: using Docker (clang not on host)\n'
docker run --rm \
  -v "$BPF_DIR:/work" \
  -w /work \
  quay.io/fedora/fedora:40 \
  bash -lc 'dnf install -y clang llvm make libbpf-devel >/dev/null && make clean && make'
printf 'bpf-build: built in container -> %s\n' "$OBJ"
