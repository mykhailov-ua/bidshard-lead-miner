#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

echo "== start Caddy edge on https://localhost:8443 =="
docker compose -f docker-compose.crm-edge.yaml up -d

echo "== waiting for Caddy =="
for _ in $(seq 1 30); do
	if curl -k -sS -o /dev/null -w '' https://localhost:8443/v1/admin/stats -u 'sales:change-me' 2>/dev/null; then
		echo "OK: https://localhost:8443 (sales / change-me)"
		echo "Use: CRM_API_URL=https://localhost:8443 CRM_API_USER=sales CRM_API_PASSWORD=change-me crm-bot api stats"
		exit 0
	fi
	sleep 0.5
done

echo "ERROR: Caddy not reachable on :8443" >&2
docker compose -f docker-compose.crm-edge.yaml logs --tail=20 caddy >&2
exit 1
