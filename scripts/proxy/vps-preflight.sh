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

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" ]]; then
	printf 'vps-preflight: FAIL PARSER_PROXY_LIST required (set residential proxy in .env)\n' >&2
	exit 1
fi

bash "$ROOT/scripts/proxy/check-proxy.sh" || {
	printf 'vps-preflight: FAIL proxy-check (set real PARSER_PROXY_LIST)\n' >&2
	exit 1
}

bash "$ROOT/scripts/proxy/preflight-tgweb.sh"

"$ROOT/bin/parser" config check 2>&1 | tee "$ROOT/var/vps-preflight.log" || {
	printf 'vps-preflight: FAIL config check\n' >&2
	exit 1
}

printf 'vps-preflight: ok (log: var/vps-preflight.log)\n'
