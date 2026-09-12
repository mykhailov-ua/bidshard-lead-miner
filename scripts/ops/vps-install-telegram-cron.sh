#!/usr/bin/env bash
# Install cron-only telegram pain scrape on VPS (no parser-telegram-realtime).
#
# Usage:
#   bash scripts/ops/vps-install-telegram-cron.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

POOL="${SESSION_POOL_FILE:-$ROOT/sessions.pool.json}"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

if [[ ! -f "$POOL" ]]; then
	printf 'vps-install-telegram-cron: missing %s\n' "$POOL" >&2
	exit 1
fi

CRON_BLOCK="$(python3 - "$POOL" "$VPS_REMOTE_DIR" <<'PY'
import json
import sys

pool_path, remote_dir = sys.argv[1], sys.argv[2]
with open(pool_path, encoding="utf-8") as fh:
    pool = json.load(fh)

shard_count = int(pool.get("shard_count") or 1)
slots = [(5, 35), (20, 50), (10, 40)]
lines = [
    "# lead-intent-processor telegram cron (H9, no realtime)",
    f"TELEGRAM_SHARD_COUNT={shard_count}",
    "TELEGRAM_LEASE_ENABLED=1",
    "TELEGRAM_SESSION_ROLE=hot",
]
idx = 0
for row in pool.get("sessions", []):
    if row.get("role") != "hot":
        continue
    shard = row.get("shard")
    if shard is None:
        continue
    m1, m2 = slots[idx % len(slots)]
    idx += 1
    runtime = row["runtime"]
    worker = f"vps-hot-{row['id']}"
    for minute in (m1, m2):
        lines.append(
            f"{minute} * * * * cd {remote_dir} && "
            f"TELEGRAM_SHARD={shard} TELEGRAM_SHARD_COUNT={shard_count} "
            f"TELEGRAM_SESSION={runtime} TELEGRAM_WORKER_ID={worker} "
            f"TELEGRAM_SESSION_ROLE=hot TELEGRAM_LEASE_ENABLED=1 "
            f"bash scripts/ops/telegram-pain-cron.sh >> var/telegram-pain-cron.log 2>&1"
        )

for row in pool.get("sessions", []):
    if row.get("role") != "cold":
        continue
    runtime = row["runtime"]
    lines.append(
        f"15 3 * * * cd {remote_dir} && "
        f"TELEGRAM_SESSION={runtime} TELEGRAM_SESSION_ROLE=cold "
        f"bash scripts/ops/buyer-discover.sh >> var/buyer-discover.log 2>&1"
    )

print("\n".join(lines))
PY
)"

vps_ssh "set -euo pipefail; cd '${VPS_REMOTE_DIR}'; mkdir -p var"
printf '%s\n' "$CRON_BLOCK" | vps_ssh "set -euo pipefail
tmp=\$(mktemp)
crontab -l 2>/dev/null | grep -v 'lead-intent-processor telegram cron' | grep -v 'scripts/ops/telegram-pain-cron.sh' | grep -v 'scripts/ops/buyer-discover.sh' >\"\$tmp\" || true
cat >>\"\$tmp\"
crontab \"\$tmp\"
rm -f \"\$tmp\"
crontab -l | tail -20
"

printf 'vps-install-telegram-cron: ok\n'
