#!/usr/bin/env bash
# Default CI gate: Go tests, slop check, Python lint/tests, BPF fixture leak-gate.
#
# Usage:
#   bash scripts/ci/run.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

log() { printf 'ci: %s\n' "$*"; }

ensure_python_venv() {
	if [[ -x "$ROOT/.venv/bin/python" ]]; then
		return 0
	fi
	log "python venv setup"
	python3 -m venv "$ROOT/.venv" --without-pip 2>/dev/null || python3 -m venv "$ROOT/.venv"
	pip3 install --target "$("$ROOT/.venv/bin/python" -c 'import site; print(site.getsitepackages()[0])')" \
		-r requirements.txt -r requirements-headless.txt -r requirements-dev.txt
}

ensure_python_venv
export PARSER_PYTHON_BIN="$ROOT/.venv/bin/python"

log "go build"
go build ./...

log "go test -race"
go test -race -count=1 ./...

log "parser slop check"
bash scripts/ci/check_parser_slop.sh

log "python lint + tests"
make lint
make test-py

if [[ "$(uname -s)" == "Linux" ]]; then
	log "bpf-report fixture leak-gate"
	make bpf-dev
	FIXTURE="$ROOT/internal/perf/testdata/bpf_disk_gate"
	if ! "$ROOT/bin/bpf-report" -leak-gate "$FIXTURE"; then
		log "FAIL bpf fixture leak-gate"
		exit 1
	fi
	log "ok bpf fixture leak-gate"
else
	log "skip bpf fixture leak-gate (Linux only)"
fi

log "ok ci gates passed"
