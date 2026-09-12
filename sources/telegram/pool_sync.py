"""Auto-import discovered Telegram channels into crawler.db (minimal manual yaml)."""

from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

from .channel_role import infer_channel_role
from .config import ChatConfig
from .prefilter import channel_discover_reject

LOG = logging.getLogger("telegram.pool_sync")


def _denylist_set(handles: list[str] | None) -> set[str]:
    out: set[str] = set()
    for raw in handles or []:
        user = str(raw).strip().lstrip("@").lower()
        if user:
            out.add(user)
    return out


def sync_registry_to_store(
    registry_path: str | Path,
    store: Any,
    *,
    denylist: list[str] | None = None,
    max_active: int = 80,
) -> dict[str, int]:
    """Upsert triaged registry rows into crawler.db for cron scrape."""
    p = Path(registry_path)
    if not p.exists():
        return {"imported": 0, "skipped": 0, "disabled": 0, "total": 0}

    try:
        data: dict[str, Any] = json.loads(p.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        LOG.warning("pool sync unreadable path=%s error=%s", p, exc)
        return {"imported": 0, "skipped": 0, "disabled": 0, "total": 0}

    deny = _denylist_set(denylist)
    channels = data.get("channels", [])
    imported = 0
    skipped = 0

    for entry in channels:
        username = str(entry.get("username", "")).strip().lstrip("@").lower()
        invite_hash = str(entry.get("invite_hash", "")).strip()
        title = str(entry.get("title", username or invite_hash or ""))
        query = str(entry.get("query", ""))
        if username and username in deny:
            skipped += 1
            continue
        reject, reason = channel_discover_reject(username, [title, query])
        if reject:
            skipped += 1
            LOG.debug("pool sync skip username=%s reason=%s", username or invite_hash, reason)
            continue
        chat = ChatConfig(
            name=title or username or invite_hash,
            username=username,
            invite_hash=invite_hash,
            geo=str(entry.get("geo", "global") or "global"),
            enabled=True,
            role=infer_channel_role(username, title, query),
        )
        store.upsert_channel(chat, "registry_sync")
        imported += 1
        if max_active > 0 and imported >= max_active:
            break

    disabled = 0
    for row in store.list_enabled_chats():
        if row.username and row.username.lower() in deny:
            store.set_channel_enabled(row.channel_key(), False)
            disabled += 1

    stats = {
        "imported": imported,
        "skipped": skipped,
        "disabled": disabled,
        "total": len(channels),
    }
    LOG.info(
        "pool sync finished path=%s imported=%d skipped=%d disabled=%d",
        p,
        stats["imported"],
        stats["skipped"],
        stats["disabled"],
    )
    return stats


def sync_registry_pool(cfg: Any, store: Any) -> dict[str, int]:
    pool = getattr(cfg, "pool", None)
    if pool is None or not getattr(pool, "auto_from_registry", False):
        return {"imported": 0, "skipped": 0, "disabled": 0, "total": 0}
    path = cfg.discover.serp_channels_path
    return sync_registry_to_store(
        path,
        store,
        denylist=list(getattr(pool, "denylist", []) or []),
        max_active=int(getattr(pool, "max_active", 80) or 80),
    )
