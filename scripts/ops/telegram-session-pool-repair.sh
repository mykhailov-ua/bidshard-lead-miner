#!/usr/bin/env bash
# Copy the authorized main Telethon session into hot pool slots when .0/.1 are missing auth.
# Safe stopgap until each pool slot has its own parser telegram login --qr.
#
# Usage (on VPS):
#   bash scripts/ops/telegram-session-pool-repair.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	docker compose run --rm --entrypoint python3 parser - <<'PY'
import asyncio
import os
import shutil
from pathlib import Path
from telethon import TelegramClient

MAIN = Path("/app/data/runtime/telethon.session")
SLOTS = [0, 1]

api_id = int(os.environ["TELEGRAM_API_ID"])
api_hash = os.environ["TELEGRAM_API_HASH"]


async def authorized(session_path: Path) -> bool:
    client = TelegramClient(str(session_path.with_suffix("")), api_id, api_hash)
    await client.connect()
    ok = await client.is_user_authorized()
    await client.disconnect()
    return ok


async def main() -> None:
    if not MAIN.exists():
        print("telegram-session-pool-repair: missing main session")
        return
    main_ok = await authorized(MAIN)
    if not main_ok:
        print("telegram-session-pool-repair: main session not authorized; skip")
        return
    for slot in SLOTS:
        target = Path(f"/app/data/runtime/telethon.session.{slot}.session")
        try:
            slot_ok = await authorized(target)
        except Exception:
            slot_ok = False
        if slot_ok:
            print(f"telegram-session-pool-repair: slot {slot} ok")
            continue
        shutil.copy2(MAIN, target)
        target.chmod(0o600)
        print(f"telegram-session-pool-repair: copied main -> slot {slot}")

asyncio.run(main())
PY
	exit 0
fi

RUNTIME="data/runtime"
MAIN="${RUNTIME}/telethon.session"
if [[ ! -f "$MAIN" ]]; then
	printf 'telegram-session-pool-repair: missing %s (no docker)\n' "$MAIN" >&2
	exit 1
fi
for slot in 0 1; do
	target="${RUNTIME}/telethon.session.${slot}.session"
	cp -a "$MAIN" "$target"
	chmod 600 "$target" 2>/dev/null || true
	printf 'telegram-session-pool-repair: synced %s -> %s\n' "$MAIN" "$target"
done

printf 'telegram-session-pool-repair: ok\n'
