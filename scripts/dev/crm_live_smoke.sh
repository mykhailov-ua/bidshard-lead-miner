#!/usr/bin/env bash
# Live smoke: real Mongo + crm-bot HTTP + api CLI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

set -a
# shellcheck disable=SC1091
. ./.env
set +a

echo "== ensure mongo =="
if ! (echo >"/dev/tcp/127.0.0.1/27017") >/dev/null 2>&1; then
	docker compose up -d mongo
	for _ in $(seq 1 30); do
		if (echo >"/dev/tcp/127.0.0.1/27017") >/dev/null 2>&1; then
			break
		fi
		sleep 1
	done
fi

echo "== build bin/crm-bot =="
make build-crm-bot

echo "== config check =="
"$ROOT/bin/crm-bot" config check

WEBHOOK_PORT="${CRM_WEBHOOK_ADDR##*:}"
if [[ "$WEBHOOK_PORT" == "$CRM_WEBHOOK_ADDR" || -z "$WEBHOOK_PORT" ]]; then
	WEBHOOK_PORT=8080
fi
WEBHOOK_HOST="${CRM_WEBHOOK_ADDR%%:*}"
if [[ "$WEBHOOK_HOST" == "$CRM_WEBHOOK_ADDR" || -z "$WEBHOOK_HOST" ]]; then
	WEBHOOK_HOST="127.0.0.1"
fi

export CRM_API_URL="http://${WEBHOOK_HOST}:${WEBHOOK_PORT}"

BOT_PID=""
cleanup() {
	if [[ -n "$BOT_PID" ]] && kill -0 "$BOT_PID" 2>/dev/null; then
		kill "$BOT_PID" 2>/dev/null || true
		wait "$BOT_PID" 2>/dev/null || true
	fi
}
trap cleanup EXIT

echo "== start crm-bot =="
"$ROOT/bin/crm-bot" run &
BOT_PID=$!

for _ in $(seq 1 50); do
	if (echo >"/dev/tcp/${WEBHOOK_HOST}/${WEBHOOK_PORT}") >/dev/null 2>&1; then
		break
	fi
	sleep 0.2
done

HASH_ID="$(python3 - <<'PY'
import secrets
print(secrets.token_hex(16))
PY
)"

echo "== POST webhook lead hash_id=${HASH_ID} =="
HTTP_CODE="$(curl -sS -o /tmp/crm_live_resp.txt -w '%{http_code}' \
	-X POST "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/v1/leads" \
	-H "Authorization: Bearer ${CRM_WEBHOOK_SECRET}" \
	-H 'Content-Type: application/json' \
	-d "{\"hash_id\":\"${HASH_ID}\",\"score\":91,\"source\":\"forum:live-smoke\",\"status\":\"new\",\"snippet\":\"live smoke lead\"}")"

if [[ "$HTTP_CODE" != "202" ]]; then
	echo "ERROR: webhook status=$HTTP_CODE body=$(cat /tmp/crm_live_resp.txt)" >&2
	exit 1
fi

"$ROOT/bin/crm-bot" api list --status new --limit 5

echo "OK: webhook accepted (202). hash_id=${HASH_ID}"
