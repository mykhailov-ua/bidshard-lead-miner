from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

LOG = logging.getLogger("telegram.registry_export")


def export_channels_json(store: Any, path: str | Path) -> int:
    """Write discovered_telegram_channels.json from SQLite registry (ops read-only)."""
    rows = store.list_channel_export_rows()
    payload = {"channels": rows}
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    LOG.info("telegram registry export path=%s channels=%d", p, len(rows))
    return len(rows)
