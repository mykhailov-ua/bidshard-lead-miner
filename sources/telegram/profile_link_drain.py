"""Resolve profile_link_queue entries into telegram_channels (cold session)."""

from __future__ import annotations

import logging
import os
from typing import Any

from .channel_role import infer_channel_role
from .config import ChatConfig
from .cursor import CursorStore
from .invites import discover_invite_hashes
from .prefilter import channel_discover_reject, channel_icp_relevant
from .telethon_retry import call_with_flood_wait, is_flood_wait

LOG = logging.getLogger("telegram.profile_link_drain")


def profile_link_drain_enabled() -> bool:
    return os.environ.get("TELEGRAM_PROFILE_LINK_DRAIN", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def profile_link_drain_limit() -> int:
    raw = os.environ.get("TELEGRAM_PROFILE_LINK_DRAIN_LIMIT", "20").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 20


async def _resolve_username(client: Any, username: str) -> ChatConfig | None:
    username = username.strip().lstrip("@").lower()
    if not username:
        return None
    try:
        entity = await call_with_flood_wait(
            f"profile_link:resolve:{username}",
            lambda: client.get_entity(username),
            attempts=2,
        )
    except Exception as exc:
        if is_flood_wait(exc):
            raise
        LOG.debug("profile link resolve failed username=%s error=%s", username, exc)
        return None
    if entity is None:
        return None
    title = str(getattr(entity, "title", None) or username)
    uname = getattr(entity, "username", None) or username
    return ChatConfig(
        name=title,
        username=str(uname).lstrip("@").lower(),
        geo="global",
        enabled=True,
        chat_id=int(getattr(entity, "id", 0) or 0) or None,
        role=infer_channel_role(str(uname), title, "profile_link"),
    )


async def drain_profile_link_queue(client: Any, store: CursorStore) -> dict[str, int]:
    stats = {"pending": 0, "resolved": 0, "rejected": 0, "flood_wait": 0, "errors": 0}
    if not profile_link_drain_enabled():
        return stats

    limit = profile_link_drain_limit()
    rows = store.list_pending_profile_links(limit)
    stats["pending"] = len(rows)
    if not rows:
        return stats

    for row in rows:
        row_id = int(row["id"])
        kind = str(row["link_kind"])
        value = str(row["link_value"])
        source = str(row["source"])
        try:
            chat: ChatConfig | None = None
            if kind == "username":
                chat = await _resolve_username(client, value)
            elif kind == "invite_hash":
                found = await discover_invite_hashes(client, [value], rate_limit_qps=0.5)
                chat = found[0] if found else None
            else:
                store.mark_profile_link(row_id, "rejected", "unknown_kind")
                stats["rejected"] += 1
                continue

            if chat is None:
                store.mark_profile_link(row_id, "rejected", "resolve_failed")
                stats["rejected"] += 1
                continue

            texts = [chat.name, f"profile_link:{source}", value]
            reject, reason = channel_discover_reject(chat.username or "", texts)
            if reject:
                store.mark_profile_link(row_id, "rejected", reason)
                stats["rejected"] += 1
                continue
            if not channel_icp_relevant(texts):
                store.mark_profile_link(row_id, "rejected", "icp_irrelevant")
                stats["rejected"] += 1
                continue

            store.upsert_channel(chat, "profile_link")
            store.mark_profile_link(row_id, "resolved", "")
            stats["resolved"] += 1
        except Exception as exc:
            if is_flood_wait(exc):
                store.mark_profile_link(row_id, "flood_wait", str(getattr(exc, "seconds", 0)))
                stats["flood_wait"] += 1
                LOG.warning(
                    "profile link drain flood wait id=%s seconds=%s stop batch",
                    row_id,
                    getattr(exc, "seconds", 0),
                )
                break
            LOG.warning("profile link drain error id=%s error=%s", row_id, exc)
            store.mark_profile_link(row_id, "rejected", "error")
            stats["errors"] += 1

    LOG.info("profile link drain finished %s", stats)
    return stats
