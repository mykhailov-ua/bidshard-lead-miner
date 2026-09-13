#!/usr/bin/env bash
# Collect Phase 0 baseline metrics (run on VPS repo root or via vps-phase0.sh).
#
# Usage:
#   bash scripts/ops/phase0-baseline-collect.sh
#   bash scripts/ops/phase0-baseline-collect.sh --json var/phase0-baseline.json
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

OUT_JSON=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--json)
		OUT_JSON="${2:-}"
		shift 2
		;;
	*)
		printf 'phase0-baseline-collect: unknown arg %s\n' "$1" >&2
		exit 1
		;;
	esac
done

TS="$(date -Is)"
REGISTRY="data/runtime/discovered_telegram_channels.json"
CURSOR_DB="data/runtime/crawler.db"
EXPORT="data/export/leads.jsonl"

registry_total=0
if [[ -f "$REGISTRY" ]]; then
	registry_total="$(python3 - <<'PY' "$REGISTRY"
import json, sys
with open(sys.argv[1], encoding="utf-8") as f:
    data = json.load(f)
print(len(data.get("channels") or []))
PY
)"
elif command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	registry_total="$(docker compose exec -T parser python3 - <<'PY' 2>/dev/null || true
import json
from pathlib import Path
p = Path("/app/data/runtime/discovered_telegram_channels.json")
if p.exists():
    data = json.loads(p.read_text(encoding="utf-8"))
    print(len(data.get("channels") or []))
PY
)"
	registry_total="${registry_total:-0}"
fi

pool_enabled=0
pool_disabled=0
read_pool_counts() {
	python3 - <<'PY' "$1"
import sqlite3, sys
conn = sqlite3.connect(sys.argv[1])
try:
    en = conn.execute("SELECT COUNT(*) FROM telegram_channels WHERE enabled = 1").fetchone()[0]
    dis = conn.execute("SELECT COUNT(*) FROM telegram_channels WHERE enabled = 0").fetchone()[0]
    print(en, dis)
finally:
    conn.close()
PY
}
if [[ -f "$CURSOR_DB" ]]; then
	read -r pool_enabled pool_disabled <<<"$(read_pool_counts "$CURSOR_DB")"
elif command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	pool_out="$(docker compose exec -T parser python3 - <<'PY' 2>/dev/null || true
import sqlite3
conn = sqlite3.connect("/app/data/runtime/crawler.db")
try:
    en = conn.execute("SELECT COUNT(*) FROM telegram_channels WHERE enabled = 1").fetchone()[0]
    dis = conn.execute("SELECT COUNT(*) FROM telegram_channels WHERE enabled = 0").fetchone()[0]
    print(en, dis)
finally:
    conn.close()
PY
)"
	if [[ -n "$pool_out" ]]; then
		read -r pool_enabled pool_disabled <<<"$pool_out"
	fi
fi

export_lines=0
export_tg=0
if [[ -f "$EXPORT" ]]; then
	export_lines="$(wc -l <"$EXPORT" | tr -d ' ')"
	export_tg="$(grep -c '"source":"telegram' "$EXPORT" 2>/dev/null || grep -c 'telegram:' "$EXPORT" 2>/dev/null || echo 0)"
fi

mongo_leads=0
mongo_tg_today=0
if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	mongo_uri="$(grep -E '^MONGO_URI=' .env 2>/dev/null | tail -1 | cut -d= -f2- || echo 'mongodb://127.0.0.1:27017')"
	mongo_out="$(docker compose exec -T mongo mongosh "$mongo_uri/lead_intent" --quiet --eval "
const total = db.leads.countDocuments({});
const today = db.leads.countDocuments({ source: /telegram/i, created_at: { \$gte: new Date(new Date().setUTCHours(0,0,0,0)) } });
print(total + ' ' + today);
" 2>/dev/null || true)"
	if [[ -n "$mongo_out" ]]; then
		read -r mongo_leads mongo_tg_today <<<"$mongo_out"
	fi
fi

metrics_snippet=""
if curl -sf http://127.0.0.1:9465/metrics >/dev/null 2>&1; then
	metrics_snippet="$(curl -sf http://127.0.0.1:9465/metrics | grep -E '^parser_(leads_accepted_total|processor_reject_total|junk_total)' | head -40 || true)"
fi

cron_telegram="$(crontab -l 2>/dev/null | grep -E 'telegram-pain-cron|buyer-discover' || true)"

keywords_locale="$(grep -E '^KEYWORDS_LOCALE=' .env 2>/dev/null | tail -1 | cut -d= -f2- || echo '')"

printf '=== phase0 baseline %s ===\n' "$TS"
printf 'registry_channels=%s\n' "$registry_total"
printf 'pool_enabled=%s pool_disabled=%s\n' "$pool_enabled" "$pool_disabled"
printf 'export_lines=%s export_telegram_lines=%s\n' "$export_lines" "$export_tg"
printf 'mongo_leads_total=%s mongo_telegram_today=%s\n' "$mongo_leads" "$mongo_tg_today"
printf 'keywords_locale=%s\n' "$keywords_locale"
printf '\n--- cron (telegram / discover) ---\n%s\n' "$cron_telegram"
if [[ -n "$metrics_snippet" ]]; then
	printf '\n--- prometheus (top rejects/accepts) ---\n%s\n' "$metrics_snippet"
else
	printf '\n--- prometheus: unavailable ---\n'
fi

if [[ -n "$OUT_JSON" ]]; then
	mkdir -p "$(dirname "$OUT_JSON")"
	python3 - <<'PY' "$OUT_JSON" "$TS" "$registry_total" "$pool_enabled" "$pool_disabled" "$export_lines" "$export_tg" "$mongo_leads" "$mongo_tg_today" "$keywords_locale"
import json, sys
path, ts, reg, pen, pdis, exp, exp_tg, mongo, mongo_tg, loc = sys.argv[1:11]
doc = {
    "collected_at": ts,
    "registry_channels": int(reg or 0),
    "pool_enabled": int(pen or 0),
    "pool_disabled": int(pdis or 0),
    "export_lines": int(exp or 0),
    "export_telegram_lines": int(exp_tg or 0),
    "mongo_leads_total": int(mongo or 0),
    "mongo_telegram_today": int(mongo_tg or 0),
    "keywords_locale": loc,
}
with open(path, "w", encoding="utf-8") as f:
    json.dump(doc, f, indent=2)
    f.write("\n")
print(f"phase0-baseline-collect: wrote {path}")
PY
fi
