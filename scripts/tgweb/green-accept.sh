#!/usr/bin/env bash
# One-shot tgweb accept gate (BOX-1). Requires GEMINI_API_KEY + sync ICP.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

RUN_DIR="$ROOT/var/green-accept-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$RUN_DIR" "$ROOT/data/export"

log() { printf '[green-accept] %s\n' "$*"; }

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

# shellcheck source=scripts/lib/host_paths.sh
source "$ROOT/scripts/lib/host_paths.sh"
apply_host_export_paths "$ROOT"

export PARSER_ICP_CLASSIFY_TGWEB="${PARSER_ICP_CLASSIFY_TGWEB:-true}"

if [[ -z "${GEMINI_API_KEY:-}" ]]; then
	log "FAIL GEMINI_API_KEY unset (required for tgweb sync ICP)"
	exit 1
fi

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" ]]; then
	log "WARN PARSER_PROXY_LIST empty - OK on home IP; use residential on VPS"
fi

PROD_REGISTRY="$ROOT/data/runtime/discovered_telegram_domains.prod.json.example"
TARGET_REGISTRY="$ROOT/data/runtime/discovered_telegram_domains.json"
cp "$PROD_REGISTRY" "$TARGET_REGISTRY"
log "registry: $(basename "$PROD_REGISTRY") -> $(basename "$TARGET_REGISTRY")"

bash "$ROOT/scripts/proxy/preflight-tgweb.sh" 2>&1 | tee "$RUN_DIR/preflight.log"

DOMAINS="${GREEN_ACCEPT_DOMAINS:-buylink.pro,topxpartners.com}"
log "reset domains: $DOMAINS"
# shellcheck disable=SC2086
"$ROOT/bin/parser" telegram domains reset ${DOMAINS//,/ } 2>&1 | tee -a "$RUN_DIR/crawl.log"

log "telegram web crawl"
set +o pipefail
"$ROOT/bin/parser" telegram web 2>&1 | tee -a "$RUN_DIR/crawl.log"
RC=${PIPESTATUS[0]}
set -o pipefail

# shellcheck source=scripts/lib/run_summary.sh
source "$ROOT/scripts/lib/run_summary.sh"
summarize_collect_logs "tgweb" "$RUN_DIR/crawl.log" | tee "$RUN_DIR/summary.txt"
ACCEPTED="$(sum_log_field accepted "$RUN_DIR/crawl.log")"
LEADS="$(sum_log_field leads_written "$RUN_DIR/crawl.log")"

log "accepted=$ACCEPTED leads_written=$LEADS rc=$RC"
printf '%s\n' "$RUN_DIR" > "$ROOT/var/green-accept-latest.dir"

if [[ "${ACCEPTED:-0}" -lt 1 ]]; then
	log "FAIL accepted=0 (check GEMINI_API_KEY, proxy, domain fixture in crawl.log)"
	exit 1
fi

log "ok accepted>=1"
