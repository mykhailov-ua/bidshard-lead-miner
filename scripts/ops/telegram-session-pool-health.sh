#!/usr/bin/env bash
# Print Telethon authorization status for pool runtime paths (run on VPS repo root).
#
# Usage:
#   bash scripts/ops/telegram-session-pool-health.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

POOL="${SESSION_POOL_FILE:-$ROOT/sessions.pool.json}"

if [[ ! -f "$POOL" ]]; then
	printf 'telegram-session-pool-health: missing %s\n' "$POOL" >&2
	exit 1
fi

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	docker compose run --rm \
		-v "${POOL}:/tmp/sessions.pool.json:ro" \
		--entrypoint python3 parser - <<'PY'
import asyncio
import json
import os
from pathlib import Path

from telethon import TelegramClient

pool = json.loads(Path("/tmp/sessions.pool.json").read_text(encoding="utf-8"))
paths = ["data/runtime/telethon.session"]
for row in pool.get("sessions", []):
    p = str(row.get("cron_session") or row.get("runtime") or "").strip()
    if p:
        paths.append(p)
seen = set()
ordered = []
for p in paths:
    if p not in seen:
        seen.add(p)
        ordered.append(p)

api_id = int(os.environ["TELEGRAM_API_ID"])
api_hash = os.environ["TELEGRAM_API_HASH"]


async def check(session_path: str) -> None:
    full = Path("/app") / session_path
    client = TelegramClient(str(full.with_suffix("")), api_id, api_hash)
    await client.connect()
    ok = await client.is_user_authorized()
    user = await client.get_me() if ok else None
    await client.disconnect()
    who = (user.username or str(user.id)) if user else "-"
    size = full.stat().st_size if full.exists() else 0
    print(f"{session_path}\tauthorized={ok}\tuser={who}\tbytes={size}")


async def main() -> None:
    for p in ordered:
        try:
            await check(p)
        except Exception as exc:
            print(f"{p}\terror={exc}")

asyncio.run(main())
PY
	exit 0
fi

printf 'telegram-session-pool-health: docker compose required on VPS\n' >&2
exit 1
