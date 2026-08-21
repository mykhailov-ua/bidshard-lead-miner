#!/usr/bin/env bash
# Evaluate BPF leak gate (FD/thread drift) on session summary.json.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
SESSION="${1:-}"

if [[ -z "$SESSION" ]]; then
	printf 'usage: bpf_leak_gate.sh <session_dir>\n' >&2
	exit 2
fi

SUMMARY="$SESSION/bpf/maps/summary.json"
if [[ ! -f "$SUMMARY" ]]; then
	printf 'bpf-leak-gate: FAIL missing %s\n' "$SUMMARY" >&2
	exit 1
fi

if ! go run ./cmd/bpf-report -leak-gate "$SESSION"; then
	printf 'bpf-leak-gate: FAIL (see above)\n' >&2
	exit 1
fi

printf 'bpf-leak-gate: ok %s\n' "$SUMMARY"
