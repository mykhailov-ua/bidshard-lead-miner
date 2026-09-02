#!/usr/bin/env bash
# VPS preflight: proxy smoke + tgweb preflight + prod seed config check.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

export PARSER_SEED_PROFILE="${PARSER_SEED_PROFILE:-prod}"

mkdir -p "$ROOT/var"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

if [[ "${DEPLOY_PREFLIGHT_CI:-}" == "1" || "${GITHUB_ACTIONS:-}" == "true" ]]; then
	export PARSER_SEED_PROFILE=dev
	export PARSER_ICP_CLASSIFY_TGWEB=false
	export PARSER_BG_TELEGRAM=false
fi

proxy="${PARSER_PROXY_LIST:-}"
if [[ -z "${proxy//[[:space:]]/}" ]]; then
	if [[ "${DEPLOY_PREFLIGHT_CI:-}" == "1" || "${GITHUB_ACTIONS:-}" == "true" ]]; then
		printf 'vps-preflight: WARN CI mode - skipping proxy-check (direct egress)\n'
	else
		printf 'vps-preflight: FAIL PARSER_PROXY_LIST required (set residential proxy in .env)\n' >&2
		exit 1
	fi
else
	bash "$ROOT/scripts/proxy/check-proxy.sh" || {
		printf 'vps-preflight: FAIL proxy-check (set real PARSER_PROXY_LIST)\n' >&2
		exit 1
	}
fi

bash "$ROOT/scripts/proxy/preflight-tgweb.sh"

# Fail early on deprecated Gemini models that return NOT_FOUND for new API keys.
# parser config check also enforces this; this gate keeps the preflight loud and explicit.
if [[ -n "${GEMINI_API_KEY:-}" ]]; then
	model="${GEMINI_MODEL:-}"
	if [[ "$model" == *"2.5-flash"* || "$model" == *"2.0-flash"* ]]; then
		printf 'vps-preflight: FAIL GEMINI_MODEL=%s is deprecated for new API keys; use gemini-3.6-flash (see docs/OPS.md)\n' "$model" >&2
		exit 1
	fi
fi

# Propagate parser config check failures as non-zero so deploy preflight stops.
"$ROOT/bin/parser" config check 2>&1 | tee "$ROOT/var/vps-preflight.log" || {
	printf 'vps-preflight: FAIL config check\n' >&2
	exit 1
}

printf 'vps-preflight: ok (log: var/vps-preflight.log)\n'
