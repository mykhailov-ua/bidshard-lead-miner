#!/usr/bin/env bash
#
# End-to-end tgweb crawl: seed registry, preflight, then docker or host go run.
#
# Usage:
#   make tgweb-crawl
#   make tgweb-crawl DOMAINS=blask.com,vietnam.vn
#   TGWEB_CRAWL_MODE=host make tgweb-crawl
#
# Requires: .env (GEMINI_API_KEY for sync ICP; PARSER_PROXY_LIST on datacenter egress).
# Rebuild after Go changes (docker mode): docker compose build parser
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

MODE="${TGWEB_CRAWL_MODE:-docker}"
DOMAINS="${DOMAINS:-}"

# Seed dev registry before preflight so first run does not fail on missing JSON.
bash "$ROOT/scripts/tgweb/seed-registry.sh"
bash "$ROOT/scripts/proxy/preflight-tgweb.sh"

# Optional BPF baseline: starts after preflight, captures summary.json for the crawl window.
# shellcheck source=scripts/tgweb/bpf_baseline.sh
source "$ROOT/scripts/tgweb/bpf_baseline.sh"
if tgweb_bpf_baseline_enabled; then
	trap tgweb_bpf_baseline_stop EXIT
	tgweb_bpf_baseline_start "$ROOT"
fi

extra_args=()
if [[ -n "$DOMAINS" ]]; then
	extra_args+=(--domains "$DOMAINS")
fi

case "$MODE" in
docker)
	echo "tgweb-crawl: docker compose (rebuild if code changed: docker compose build parser)"
	# Compose profile tgweb runs one-shot telegram web with host network + parser_runtime volume.
	# shellcheck disable=SC2086
	docker compose -f docker-compose.tgweb.yaml --profile tgweb run --rm tgweb-crawl "${extra_args[@]}"
	;;
host|go)
	# Use on laptop with direct egress; respects local .env without rebuilding Docker image.
	echo "tgweb-crawl: go run ./cmd/parser telegram web"
	# shellcheck disable=SC2086
	go run ./cmd/parser telegram web "${extra_args[@]}"
	;;
*)
	echo "tgweb-crawl: unknown TGWEB_CRAWL_MODE=$MODE (use docker or host)" >&2
	exit 1
	;;
esac
