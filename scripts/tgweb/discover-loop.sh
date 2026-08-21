#!/usr/bin/env bash
# Telethon discover -> prune -> tgweb crawl (BOX-4).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

MODE="${TGWEB_DISCOVER_MODE:-docker}"

log() { printf '[discover-loop] %s\n' "$*"; }

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

# shellcheck source=scripts/lib/telethon_preflight.sh
source "$ROOT/scripts/lib/telethon_preflight.sh"

if [[ -z "${TELEGRAM_API_ID:-}" || -z "${TELEGRAM_API_HASH:-}" ]]; then
	log "FAIL set TELEGRAM_API_ID and TELEGRAM_API_HASH in .env"
	exit 1
fi

case "$MODE" in
host)
	if ! require_telethon_ready "$ROOT"; then
		log "FAIL telethon preflight - run: make venv && go run ./cmd/parser telegram login --qr"
		exit 1
	fi
	log "host discover (venv + telethon)"
	go run ./cmd/parser telegram discover
	;;
docker)
	if [[ ! -f "$ROOT/data/runtime/telethon.session" ]]; then
		log "FAIL missing data/runtime/telethon.session"
		log "run: docker compose run --rm -it parser telegram login --qr"
		exit 1
	fi
	log "docker discover"
	make tgweb-discover
	;;
*)
	log "unknown TGWEB_DISCOVER_MODE=$MODE (docker|host)"
	exit 1
	;;
esac

make tgweb-prune

if [[ -z "${PARSER_PROXY_LIST//[[:space:]]/}" ]]; then
	log "WARN crawling without proxy - use make tgweb-crawl-residential on VPS"
	make tgweb-crawl
else
	make tgweb-crawl-residential
fi

log "done"
