#!/usr/bin/env bash
# CI entry for make deploy-preflight: minimal env, no residential proxy, host tgweb crawl.
#
# Usage (GitHub Actions or local):
#   bash scripts/ci/deploy-preflight.sh
#
# Overrides for CI:
#   - skip PARSER_PROXY_LIST requirement (direct egress smoke)
#   - PARSER_ICP_CLASSIFY_TGWEB=false (no GEMINI_API_KEY in CI)
#   - TGWEB_CRAWL_MODE=host (no docker image build)
#   - single domain crawl for BPF window
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

export DEPLOY_PREFLIGHT_CI=1
export PARSER_SEED_PROFILE=dev
export PARSER_BG_TELEGRAM=false
export PARSER_ICP_CLASSIFY_TGWEB=false
export PARSER_GEMINI_DEFER=true
export TGWEB_CRAWL_MODE=host
export DOMAINS="${DOMAINS:-blask.com}"
export MONGO_URI="${MONGO_URI:-mongodb://127.0.0.1:27017}"
export PARSER_BPF_PREFLIGHT="${PARSER_BPF_PREFLIGHT:-1}"

log() { printf 'ci-deploy-preflight: %s\n' "$*"; }

if [[ "$(uname -s)" == "Linux" && "${PARSER_BPF_PREFLIGHT}" == "1" ]]; then
	if ! command -v clang >/dev/null 2>&1; then
		log "WARN clang missing; bpf_build may use Docker fallback"
	fi
fi

log "mongo=$MONGO_URI crawl_mode=$TGWEB_CRAWL_MODE domains=$DOMAINS bpf=$PARSER_BPF_PREFLIGHT"
make tgweb-seed
make deploy-preflight
