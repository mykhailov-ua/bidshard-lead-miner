from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

from .prefilter import channel_discover_reject

LOG = logging.getLogger("telegram.registry_triage")


def triage_channel_registry(path: str | Path) -> dict[str, int]:
    """Drop noise channels from discovered_telegram_channels.json before discover/scrape."""
    p = Path(path)
    if not p.exists():
        return {"kept": 0, "dropped": 0, "total": 0}

    try:
        data: dict[str, Any] = json.loads(p.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        LOG.warning("registry triage unreadable path=%s error=%s", p, exc)
        return {"kept": 0, "dropped": 0, "total": 0}

    channels = data.get("channels", [])
    kept: list[dict[str, Any]] = []
    dropped = 0
    for entry in channels:
        username = str(entry.get("username", "")).strip().lstrip("@").lower()
        title = str(entry.get("title", username or ""))
        query = str(entry.get("query", ""))
        reject, reason = channel_discover_reject(username, [title, query])
        if reject:
            dropped += 1
            LOG.info(
                "registry triage drop username=%s reason=%s",
                username or entry.get("invite_hash", ""),
                reason,
            )
            continue
        kept.append(entry)

    if dropped > 0:
        data["channels"] = kept
        p.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

    stats = {"kept": len(kept), "dropped": dropped, "total": len(channels)}
    LOG.info(
        "registry triage finished path=%s kept=%d dropped=%d",
        p,
        stats["kept"],
        stats["dropped"],
    )
    return stats
