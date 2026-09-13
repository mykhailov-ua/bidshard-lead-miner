"""SQLite OSINT metrics for VPS ops (profile_link_queue, people counts)."""

from __future__ import annotations

import json
import logging
import os
import sys
from pathlib import Path

from .cursor import CursorStore

LOG = logging.getLogger("telegram.osint_metrics")


def collect_osint_metrics(store: CursorStore) -> dict[str, object]:
    link_stats = store.profile_link_queue_stats()
    return {
        "profile_link_queue": link_stats,
        "chat_members": store.count_chat_members(),
        "user_profiles": store.count_user_profiles(),
        "enabled_channels": len(store.list_enabled_chats()),
    }


def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(message)s", stream=sys.stderr)
    db = os.environ.get("TELEGRAM_CURSOR_DB_PATH", "data/runtime/crawler.db")
    path = Path(db)
    if not path.is_file():
        print(json.dumps({"error": "cursor_db_missing", "path": str(path)}))
        return 1
    store = CursorStore(path)
    try:
        payload = collect_osint_metrics(store)
    finally:
        store.close()
    print(json.dumps(payload, indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
