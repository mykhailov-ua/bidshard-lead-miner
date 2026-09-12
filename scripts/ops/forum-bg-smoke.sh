#!/usr/bin/env bash
# M6: smoke one forum thread fetch from discovered_forum_threads.json (bgworker path).
# Requires residential proxy on datacenter VPS (PARSER_PROXY_LIST + forum in PARSER_PROXY_SOURCES).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

RUN_DIR="$ROOT/var/forum-bg-smoke-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$RUN_DIR"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

REGISTRY="${FORUM_REGISTRY_PATH:-data/runtime/discovered_forum_threads.json}"
if [[ ! -f "$REGISTRY" ]]; then
	printf 'forum-bg-smoke: FAIL registry missing: %s\n' "$REGISTRY" >&2
	printf 'hint: run serp_forum_threads bgworker or copy VPS registry locally\n' >&2
	exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
	printf 'forum-bg-smoke: FAIL jq required\n' >&2
	exit 1
fi

THREAD_URL="$(jq -r '.threads[0].url // empty' "$REGISTRY")"
if [[ -z "$THREAD_URL" ]]; then
	printf 'forum-bg-smoke: FAIL no threads in %s\n' "$REGISTRY" >&2
	exit 1
fi

TMP_REG="$RUN_DIR/single_thread.json"
jq --arg url "$THREAD_URL" '{threads: [.threads[] | select(.url == $url)][0:1]}' "$REGISTRY" > "$TMP_REG"
COUNT="$(jq '.threads | length' "$TMP_REG")"
if [[ "$COUNT" != "1" ]]; then
	printf 'forum-bg-smoke: FAIL could not isolate thread url=%s\n' "$THREAD_URL" >&2
	exit 1
fi

export FORUM_REGISTRY_PATH="$TMP_REG"
export FORUM_SEED_PATH="${FORUM_SEED_PATH:-/dev/null}"
export WARRIOR_SEED_PATH="${WARRIOR_SEED_PATH:-/dev/null}"
export PARSER_SOURCE=forum
# Scope residential proxy to forum only for this smoke (other sources stay direct).
export PARSER_PROXY_SOURCES=forum

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" && -z "${PARSER_PROXY_LIST_FILE//[[:space:]]/}" ]]; then
	printf 'forum-bg-smoke: FAIL PARSER_PROXY_LIST or PARSER_PROXY_LIST_FILE required on datacenter VPS\n' >&2
	exit 1
fi

if [[ ! -x "$ROOT/bin/parser" ]]; then
	make -C "$ROOT" build
fi

if [[ -x "$ROOT/scripts/proxy/check-proxy.sh" ]]; then
	bash "$ROOT/scripts/proxy/check-proxy.sh"
fi

"$ROOT/bin/parser" config check

LOG="$RUN_DIR/scan.log"
printf 'forum-bg-smoke: fetching %s\n' "$THREAD_URL"
"$ROOT/bin/parser" scan --source=forum --output=quiet 2>&1 | tee "$LOG"

# shellcheck source=scripts/lib/soak_gate.sh
source "$ROOT/scripts/lib/soak_gate.sh"
evaluate_scan_raw_gate "forum-bg-smoke" "$LOG"

printf 'forum-bg-smoke: ok thread=%s log=%s\n' "$THREAD_URL" "$LOG"
