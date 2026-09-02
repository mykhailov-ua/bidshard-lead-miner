"""Linked discussion group scrape (channel comments).

Opt-in via TELEGRAM_DISCUSSION_SCRAPE=1 (default off): comments burn RPC and FloodWait.
Uses separate cursor key {channel_key}#discussion; NDJSON chat_type=discussion.
"""

from __future__ import annotations

import logging
import os
from typing import Any, Awaitable, Callable

LOG = logging.getLogger("telegram.discussion")


def discussion_scrape_enabled() -> bool:
    return os.environ.get("TELEGRAM_DISCUSSION_SCRAPE", "").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def discussion_cursor_key(channel_key: str) -> str:
    return f"{channel_key}#discussion"


async def linked_discussion_id(client: Any, channel_entity: Any) -> int | None:
    from telethon.tl.functions.channels import GetFullChannelRequest
    from telethon.tl.types import Channel

    if not isinstance(channel_entity, Channel) or channel_entity.megagroup:
        return None
    try:
        from .telethon_input import channel_input_peer

        full = await client(GetFullChannelRequest(channel_input_peer(channel_entity)))
        linked = getattr(full.full_chat, "linked_chat_id", None)
        if linked is None:
            return None
        return int(linked)
    except Exception as exc:
        LOG.debug("linked discussion lookup failed error=%s", exc)
        return None


async def scrape_discussion_messages(
    client: Any,
    discussion_entity: Any,
    *,
    cursor_key: str,
    last_id: int,
    message_limit: int,
    on_message: Callable[[Any], Awaitable[bool]],
) -> int:
    from .telethon_retry import call_with_flood_wait, is_flood_wait

    emitted = 0

    async def _iter() -> None:
        nonlocal emitted
        async for message in client.iter_messages(discussion_entity, limit=message_limit):
            if message.id <= last_id:
                break
            if await on_message(message):
                emitted += 1

    try:
        await call_with_flood_wait(f"discussion:{cursor_key}", _iter, attempts=2)
    except Exception as exc:
        if is_flood_wait(exc):
            LOG.warning(
                "discussion flood wait cursor=%s seconds=%s skip",
                cursor_key,
                getattr(exc, "seconds", 0),
            )
            return emitted
        LOG.warning("discussion scrape failed cursor=%s error=%s", cursor_key, exc)
        return emitted

    return emitted
