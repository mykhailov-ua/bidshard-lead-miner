#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

echo "== crm-bot unit smoke =="
go test -count=1 -run TestCRMBotWebhookSmoke ./internal/crm/app/

echo "== build bin/crm-bot =="
make build-crm-bot

echo "== crm-bot live smoke (HTTP + api CLI) =="

wait_mongo() {
	local i
	for i in $(seq 1 30); do
		if (echo >"/dev/tcp/127.0.0.1/27017") >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done
	return 1
}

if ! wait_mongo; then
	if command -v docker >/dev/null 2>&1; then
		echo "mongo not reachable; starting docker compose mongo"
		docker compose up -d mongo
		wait_mongo || {
			echo "ERROR: mongo still unreachable on 127.0.0.1:27017" >&2
			exit 1
		}
	else
		echo "ERROR: mongo not reachable on 127.0.0.1:27017 and docker unavailable" >&2
		exit 1
	fi
fi

BOT_PID=""

cleanup() {
	if [[ -n "$BOT_PID" ]] && kill -0 "$BOT_PID" 2>/dev/null; then
		kill "$BOT_PID" 2>/dev/null || true
		wait "$BOT_PID" 2>/dev/null || true
	fi
}
trap cleanup EXIT

WEBHOOK_PORT="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

export CRM_WEBHOOK_SECRET=smoke-secret
export CRM_WEBHOOK_ADDR="127.0.0.1:${WEBHOOK_PORT}"
export CRM_API_URL="http://127.0.0.1:${WEBHOOK_PORT}"
export MONGO_URI=mongodb://127.0.0.1:27017
export MONGO_DB=parser
export CRM_LOG_LEVEL=error

"$ROOT/bin/crm-bot" config check

"$ROOT/bin/crm-bot" run &
BOT_PID=$!

for _ in $(seq 1 50); do
	if (echo >"/dev/tcp/127.0.0.1/${WEBHOOK_PORT}") >/dev/null 2>&1; then
		break
	fi
	sleep 0.1
done

HTTP_CODE="$(curl -sS -o /tmp/crm_smoke_resp.txt -w '%{http_code}' \
	-X POST "http://127.0.0.1:${WEBHOOK_PORT}/v1/leads" \
	-H 'Authorization: Bearer smoke-secret' \
	-H 'Content-Type: application/json' \
	-d '{"hash_id":"abcdef1234567890abcdef1234567890","score":90,"source":"forum:smoke","status":"new","snippet":"live smoke lead"}')"
if [[ "$HTTP_CODE" != "202" ]]; then
	echo "ERROR: webhook status=$HTTP_CODE body=$(cat /tmp/crm_smoke_resp.txt)" >&2
	exit 1
fi

"$ROOT/bin/crm-bot" api stats >/tmp/crm_smoke_api.txt
if ! grep -q 'leads total:' /tmp/crm_smoke_api.txt; then
	echo "ERROR: crm-bot api stats failed:" >&2
	cat /tmp/crm_smoke_api.txt >&2
	exit 1
fi

echo "OK: crm-bot live smoke passed (webhook 202, api stats)"
