#!/usr/bin/env bash
# LEADS.md Phase 0: deploy config, discover+triage, baseline snapshot.
#
# Usage:
#   bash scripts/ops/vps-phase0.sh              # deploy + discover + collect
#   bash scripts/ops/vps-phase0.sh --deploy-only
#   bash scripts/ops/vps-phase0.sh --discover-only
#   bash scripts/ops/vps-phase0.sh --collect-only
#
# Phase 0 baseline without long jobboard discover:
#   bash scripts/ops/vps-phase0.sh --deploy-only && bash scripts/ops/vps-phase0.sh --collect-only
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

DO_DEPLOY=1
DO_DISCOVER=1
DO_COLLECT=1

while [[ $# -gt 0 ]]; do
	case "$1" in
	--deploy-only)
		DO_DEPLOY=1
		DO_DISCOVER=0
		DO_COLLECT=0
		shift
		;;
	--discover-only)
		DO_DEPLOY=0
		DO_DISCOVER=1
		DO_COLLECT=0
		shift
		;;
	--collect-only)
		DO_DEPLOY=0
		DO_DISCOVER=0
		DO_COLLECT=1
		shift
		;;
	*)
		printf 'vps-phase0: unknown arg %s\n' "$1" >&2
		exit 1
		;;
	esac
done

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
LOCAL_VAR="$ROOT/var"
mkdir -p "$LOCAL_VAR"

log() { printf 'vps-phase0: %s\n' "$*"; }

if [[ "$DO_DEPLOY" == "1" ]]; then
	log "0.1 rsync config to $(vps_ssh_target)"
	vps_rsync_push "$ROOT"
	vps_post_sync_fix

	log "0.1 apply P0 env (KEYWORDS_LOCALE=es,pt,id,vi)"
	bash "$ROOT/scripts/ops/vps-apply-p0-env.sh"

	if [[ -f "$ROOT/sessions.pool.json" ]]; then
		log "0.1 sync session pool"
		bash "$ROOT/scripts/ops/vps-sync-session-pool.sh" || true
	fi

	log "0.1 install telegram cron (staggered schedule)"
	bash "$ROOT/scripts/ops/vps-install-telegram-cron.sh"

	log "0.1 rebuild parser image if compose present"
	vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; docker compose build parser 2>&1 | tail -5; docker compose up -d mongo parser crm-bot 2>&1 | tail -5"
fi

if [[ "$DO_DISCOVER" == "1" ]]; then
	log "0.2 discover jobboard + serp + triage (no CF crawl)"
	vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
mkdir -p var data/runtime
export STAMP='${STAMP}'
run_discover() {
  if docker compose run --rm parser discover \"\$1\"; then
    return 0
  fi
  return 1
}
run_discover jobboard
run_discover serp
bash scripts/ops/triage-telegram-registry.sh | tee var/phase0-triage-\${STAMP}.log
if [[ -f data/runtime/discovered_telegram_channels.json ]]; then
  cp -f data/runtime/discovered_telegram_channels.json var/phase0-registry-\${STAMP}.json
  printf 'snapshot: var/phase0-registry-%s.json\n' \"\${STAMP}\"
fi
"
fi

if [[ "$DO_COLLECT" == "1" ]]; then
	log "0.3 collect baseline metrics"
	vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
bash scripts/ops/phase0-baseline-collect.sh --json var/phase0-baseline-${STAMP}.json \
  | tee var/phase0-baseline-${STAMP}.txt
"
	log "0.3 pull baseline artifacts"
	rsync -az -e "$(vps_rsync_ssh)" \
		"$(vps_ssh_target):${VPS_REMOTE_DIR}/var/phase0-baseline-${STAMP}."* \
		"$(vps_ssh_target):${VPS_REMOTE_DIR}/var/phase0-registry-${STAMP}.json" \
		"$LOCAL_VAR/" 2>/dev/null || true
	if [[ -f "$LOCAL_VAR/phase0-baseline-${STAMP}.json" ]]; then
		cp -f "$LOCAL_VAR/phase0-baseline-${STAMP}.json" "$LOCAL_VAR/phase0-baseline-latest.json"
		cp -f "$LOCAL_VAR/phase0-baseline-${STAMP}.txt" "$LOCAL_VAR/phase0-baseline-latest.txt"
		log "local copy: var/phase0-baseline-latest.json"
	fi
fi

log "ok phase0 stamp=${STAMP}"
