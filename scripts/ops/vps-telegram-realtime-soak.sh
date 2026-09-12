#!/usr/bin/env bash
# M2 soak gate: parser-telegram-realtime listener, pain alerts, CRM telegram path, export quality.
#
# Usage:
#   bash scripts/ops/vps-telegram-realtime-soak.sh
#   SOAK_HOURS=48 M2_STRICT=1 bash scripts/ops/vps-telegram-realtime-soak.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"
# shellcheck source=scripts/lib/soak_gate.sh
source "$ROOT/scripts/lib/soak_gate.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

SOAK_HOURS="${SOAK_HOURS:-48}"
M2_STRICT="${M2_STRICT:-0}"
MIN_CONTACT_VALID_PCT="${SOAK_MIN_CONTACT_VALID_PCT:-95}"
MIN_ALERTS_PER_DAY="${M2_MIN_ALERTS_PER_DAY:-3}"
REMOTE_DIR="${VPS_REMOTE_DIR:-/opt/lead-intent-processor}"
STAMP="$(date -u +%Y%m%dT%H%MZ)"
OUT_LOCAL="$ROOT/var/telegram-realtime-soak-${STAMP}.txt"
mkdir -p "$ROOT/var"

log() { printf '%s\n' "$*"; }

fail=0
gate_fail() {
	soak_gate_fail "$1" || fail=1
}

vps_ssh "set -euo pipefail
cd '${REMOTE_DIR}'
SOAK_HOURS='${SOAK_HOURS}'
MIN_CONTACT_VALID_PCT='${MIN_CONTACT_VALID_PCT}'
MIN_ALERTS_PER_DAY='${MIN_ALERTS_PER_DAY}'
export SOAK_HOURS MIN_CONTACT_VALID_PCT MIN_ALERTS_PER_DAY

printf '=== M2 telegram realtime soak (last %sh) %s ===\n' \"\$SOAK_HOURS\" \"\$(date -Is)\"

printf '\n--- containers ---\n'
rt_up=0
if docker compose -f docker-compose.telegram-realtime.yaml ps --status running 2>/dev/null \
  | grep -q parser-telegram-realtime; then
  rt_up=1
fi
printf 'parser-telegram-realtime_up=%s\n' \"\$rt_up\"
docker compose ps 2>/dev/null | grep -E 'parser|crm-bot|mongo' || true
docker compose -f docker-compose.telegram-realtime.yaml ps 2>/dev/null || true

printf '\n--- realtime listener ---\n'
rt_logs=\$(docker compose -f docker-compose.telegram-realtime.yaml logs parser-telegram-realtime --since \"\${SOAK_HOURS}h\" 2>&1 || true)
printf '%s\n' \"\$rt_logs\" | grep -E 'telegram realtime listening|realtime backfill done|telethon ipc connected' | tail -20 || true
listening=\$(printf '%s\n' \"\$rt_logs\" | grep -c 'telegram realtime listening' || true)
ipc_connected=\$(printf '%s\n' \"\$rt_logs\" | grep -c 'telethon ipc connected' || true)
ipc_format=\$(grep -E '^TELETHON_IPC_FORMAT=' .env 2>/dev/null | tail -1 | cut -d= -f2- || echo msgpack)
ipc_socket=\$(grep -E '^TELETHON_IPC_SOCKET=' .env 2>/dev/null | tail -1 | cut -d= -f2- || echo data/runtime/telethon.sock)
flood_wait_hits=\$(printf '%s\n' \"\$rt_logs\" | grep -cE 'wait of [0-9]+ seconds is required' || true)
printf 'listening_events=%s ipc_connected=%s ipc_format=%s ipc_socket=%s flood_wait_hits=%s\n' \
  \"\$listening\" \"\$ipc_connected\" \"\$ipc_format\" \"\$ipc_socket\" \"\$flood_wait_hits\"

printf '\n--- TELEGRAM_REALTIME e2e ingest ---\n'
# Realtime container runs Go parser telegram realtime + Python sidecar over UDS msgpack (or NDJSON pipe).
rt_accepted=\$(printf '%s\n' \"\$rt_logs\" | grep -cE 'lead accepted.*source=telegram:' || true)
rt_raw=\$(printf '%s\n' \"\$rt_logs\" | grep 'scan round finished' || true | sed -n 's/.*raw[= ][ ]*\\([0-9][0-9]*\\).*/\\1/p' | awk '{s+=\$1} END {print s+0}')
printf 'realtime_lead_accepted=%s realtime_raw_proxy=%s\n' \"\$rt_accepted\" \"\${rt_raw:-0}\"
if [[ \"\$ipc_connected\" -ge 1 ]]; then
  printf 'e2e_path=TELETHON_IPC_%s -> parser ingest (connected)\n' \"\$ipc_format\"
elif [[ -z \"\$ipc_socket\" ]]; then
  printf 'e2e_path=stdout NDJSON pipe (no TELETHON_IPC_SOCKET)\n'
else
  printf 'e2e_path=WARN ipc socket configured but no telethon ipc connected log in window\n'
fi

printf '\n--- pain alerts (tag=tracker_pain|crypto_gray) ---\n'
pain_lines=\$(printf '%s\n' \"\$rt_logs\" | grep 'pain alert sent' || true)
pain_alerts=\$(printf '%s\n' \"\$pain_lines\" | grep -c 'pain alert sent' || true)
tracker_pain=\$(printf '%s\n' \"\$pain_lines\" | grep -c 'tag=tracker_pain' || true)
crypto_gray=\$(printf '%s\n' \"\$pain_lines\" | grep -c 'tag=crypto_gray' || true)
printf '%s\n' \"\$pain_lines\" | tail -10 || true
printf 'pain_alert_sent=%s tracker_pain=%s crypto_gray=%s\n' \
  \"\$pain_alerts\" \"\$tracker_pain\" \"\$crypto_gray\"
hours=\"\$SOAK_HOURS\"
if [[ \"\${hours:-0}\" -le 0 ]]; then hours=24; fi
alert_per_day=\$(( pain_alerts * 24 / hours ))
printf 'alert_per_day_proxy=%s (min_target=%s)\n' \"\$alert_per_day\" \"\$MIN_ALERTS_PER_DAY\"

printf '\n--- CRM webhook / telegram accepts ---\n'
parser_logs=\$(docker compose logs parser parser-telegram-realtime crm-bot --since \"\${SOAK_HOURS}h\" 2>&1 || true)
tg_accepted=\$(printf '%s\n' \"\$parser_logs\" | grep -cE 'lead accepted.*source=telegram:' || true)
tg_username=\$(printf '%s\n' \"\$parser_logs\" | grep -E 'lead accepted.*source=telegram:' | grep -cE 'contact=telegram:@|contact=t\\*\\*\\*' || true)
crm_metrics=\$(curl -sf http://127.0.0.1:8080/metrics 2>/dev/null || true)
crm_webhook_total=\$(printf '%s\n' \"\$crm_metrics\" | awk '/^crm_webhook_accepted_total /{print \$2}' | tail -1)
printf 'telegram_lead_accepted_logs=%s crm_webhook_accepted_total=%s\n' \
  \"\$tg_accepted\" \"\${crm_webhook_total:-unavailable}\"
printf '%s\n' \"\$parser_logs\" | grep -E 'lead accepted.*source=telegram:' | tail -5 || true

printf '\n--- export JSONL (telegram family + contact_valid_rate) ---\n'
if [[ -f data/export/leads.jsonl ]]; then
  python3 - <<'PY'
import json, os
from pathlib import Path

path = Path('data/export/leads.jsonl')
hours = int(os.environ.get('SOAK_HOURS', '48'))
rows = []
for line in path.read_text(encoding='utf-8', errors='replace').splitlines():
    line = line.strip()
    if not line:
        continue
    try:
        row = json.loads(line)
    except json.JSONDecodeError:
        continue
    src = str(row.get('source', ''))
    if not src.startswith('telegram'):
        continue
    contacts = row.get('contacts') or []
    has_user = False
    for c in contacts:
        if isinstance(c, dict):
            if c.get('type') == 'telegram' and str(c.get('value', '')).startswith('@'):
                if 'serp' not in str(c.get('value', '')).lower():
                    has_user = True
        elif isinstance(c, str) and c.startswith('telegram:@') and '@serp:' not in c:
            has_user = True
    rows.append((src, has_user, row.get('score', ''), row.get('hash_id', '')))
print('telegram_export_rows', len(rows))
print('telegram_export_with_username', sum(1 for _r in rows if _r[1]))
for src, has_user, score, hid in rows[-8:]:
    print('sample', src, 'username=' + str(has_user), 'score=' + str(score), 'hash=' + str(hid)[:12])
PY
  python3 - <<'PY'
import json, re
from pathlib import Path

path = Path('data/export/leads.jsonl')
min_pct = int(__import__('os').environ.get('MIN_CONTACT_VALID_PCT', '95'))
rows = []
for line in path.read_text(encoding='utf-8', errors='replace').splitlines():
    line = line.strip()
    if not line:
        continue
    try:
        rows.append(json.loads(line))
    except json.JSONDecodeError:
        continue
tg_re = re.compile(r'^@[A-Za-z0-9_]{3,}$')
email_re = re.compile(r'^[^@]+@[^@]+\.')

def reachable(row):
    for c in row.get('contacts') or []:
        if isinstance(c, dict):
            t, v = c.get('type'), str(c.get('value', ''))
            if t == 'telegram' and tg_re.match(v):
                return True
            if t == 'forum_user' and v:
                return True
            if t == 'email' and email_re.match(v):
                return True
        elif isinstance(c, str):
            if c.startswith('telegram:@') and '@serp:' not in c:
                return True
            if c.startswith('forum:user/'):
                return True
            if email_re.match(c):
                return True
    return False

total = len(rows)
valid = sum(1 for r in rows if reachable(r))
pct = valid * 100 // total if total else 0
serp = sum(
    1 for r in rows
    for c in (r.get('contacts') or [])
    if (isinstance(c, str) and '@serp:' in c)
    or (isinstance(c, dict) and '@serp:' in str(c.get('value', '')))
)
print(f'contact_valid_rate={pct}% valid={valid} total={total} min={min_pct}%')
print(f'synthetic_serp_contacts_in_export={serp} (want 0)')
PY
else
  printf 'export_missing=data/export/leads.jsonl\n'
fi

printf '\n--- M2 gate summary ---\n'
printf 'PASS rt_up=1 if parser-telegram-realtime running\n'
printf 'PASS listening>=1 if telegram realtime listening in window\n'
printf 'PASS ipc_connected>=1 when TELETHON_IPC_SOCKET set (msgpack/ndjson ingest)\n'
printf 'PASS alert_per_day>=%s (scaled from %sh window)\n' \"\$MIN_ALERTS_PER_DAY\" \"\$SOAK_HOURS\"
printf 'PASS telegram export rows with telegram:@username (no @serp:)\n'
printf 'PASS contact_valid_rate>=%s%% on export\n' \"\$MIN_CONTACT_VALID_PCT\"
" | tee "$OUT_LOCAL"

# Local gate evaluation on captured output.
if ! grep -q 'parser-telegram-realtime_up=1' "$OUT_LOCAL"; then
	gate_fail 'parser-telegram-realtime not running'
fi
if ! grep -qE 'listening_events=[1-9]' "$OUT_LOCAL"; then
	if ! grep -qE 'ipc_connected=[1-9]' "$OUT_LOCAL"; then
		gate_fail 'no telegram realtime listening or ipc connected in soak window'
	fi
fi
if grep -qE 'ipc_socket=data/runtime/telethon.sock|ipc_socket=.+' "$OUT_LOCAL"; then
	if ! grep -qE 'ipc_connected=[1-9]' "$OUT_LOCAL"; then
		gate_fail 'TELETHON_IPC configured but no telethon ipc connected (ingest path broken?)'
	fi
fi
flood_wait="$(grep -E '^listening_events=.*flood_wait_hits=[1-9]' "$OUT_LOCAL" | tail -1 | sed -n 's/.*flood_wait_hits=\([0-9][0-9]*\).*/\1/p')"
if [[ -z "$flood_wait" ]]; then
	flood_wait="$(grep -c 'wait of [0-9][0-9]* seconds is required' "$OUT_LOCAL" 2>/dev/null || true)"
fi
alert_day="$(grep -E '^alert_per_day_proxy=' "$OUT_LOCAL" | tail -1 | cut -d= -f2 | cut -d' ' -f1 || true)"
if [[ -n "$alert_day" && "$alert_day" =~ ^[0-9]+$ ]]; then
	if [[ "$alert_day" -lt "$MIN_ALERTS_PER_DAY" ]]; then
		if [[ "${flood_wait:-0}" -gt 0 ]]; then
			printf 'soak-gate: WARN alert_per_day_proxy=%s below %s (FloodWait in window; known MTProto limit)\n' \
				"$alert_day" "$MIN_ALERTS_PER_DAY" >&2
		else
			gate_fail "alert_per_day_proxy=${alert_day} below ${MIN_ALERTS_PER_DAY}"
		fi
	fi
fi
if grep -q '^synthetic_serp_contacts_in_export=[1-9]' "$OUT_LOCAL"; then
	gate_fail 'export contains @serp: synthetic contacts'
fi
if grep -qE '^contact_valid_rate=' "$OUT_LOCAL"; then
	cvr="$(grep -E '^contact_valid_rate=' "$OUT_LOCAL" | tail -1 | sed -n 's/contact_valid_rate=\([0-9][0-9]*\)%.*/\1/p')"
	if [[ -n "$cvr" && "$cvr" -lt "$MIN_CONTACT_VALID_PCT" ]]; then
		gate_fail "contact_valid_rate=${cvr}% below ${MIN_CONTACT_VALID_PCT}%"
	fi
fi

log "wrote $OUT_LOCAL"

if [[ "$M2_STRICT" == "1" && "$fail" -ne 0 ]]; then
	log "M2 soak: FAIL (set M2_STRICT=0 for report-only)"
	exit 1
fi

if [[ "$fail" -ne 0 ]]; then
	log "M2 soak: WARN (gates failed; use M2_STRICT=1 to exit non-zero)"
else
	log "M2 soak: ok"
fi
