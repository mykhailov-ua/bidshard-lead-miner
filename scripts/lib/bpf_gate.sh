#!/usr/bin/env bash
# Evaluate BPF release gate on session summary.json (BOX-13).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
SESSION="${1:-}"

if [[ -z "$SESSION" ]]; then
	printf 'usage: bpf_gate.sh <session_dir>\n' >&2
	exit 2
fi

SUMMARY="$SESSION/bpf/maps/summary.json"
if [[ ! -f "$SUMMARY" ]]; then
	printf 'bpf-gate: FAIL missing %s\n' "$SUMMARY" >&2
	exit 1
fi

if ! go run ./cmd/bpf-report -gate "$SESSION"; then
	printf 'bpf-gate: FAIL (see above)\n' >&2
	exit 1
fi

printf 'bpf-gate: ok %s\n' "$SUMMARY"
