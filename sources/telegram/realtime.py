"""Long-running Telethon NewMessage listener -> NDJSON stdout."""

from __future__ import annotations

import logging
import os
from typing import Any, TextIO

from .config import ChatConfig, ScraperConfig
from .connect import connect_telegram_client
from .cursor import CursorStore
from .discover import merge_chat_lists
from .scraper import (
    build_telegram_client,
    entity_chat_type,
    fetch_channel_about,
    process_scrape_message,
    resolve_chat_entity,
)
from .session_lock import session_exclusive_lock

LOG = logging.getLogger("telegram.realtime")


def realtime_env_enabled() -> bool:
    return os.environ.get("TELEGRAM_REALTIME", "0").strip().lower() in (
        "1",
        "true",
        "yes",
    )


async def resolve_listen_targets(
    client: Any, chats: list[ChatConfig], store: CursorStore
) -> list[tuple[Any, ChatConfig, str, str]]:
    """Resolve enabled chats to Telethon entities; skip failures."""
    from telethon.utils import get_peer_id

    out: list[tuple[Any, ChatConfig, str, str]] = []
    for chat in chats:
        chat_key = chat.channel_key()
        try:
            entity = await resolve_chat_entity(client, chat, store)
            full = await client.get_entity(entity)
            about = await fetch_channel_about(client, full)
            kind = entity_chat_type(full)
            out.append((full, chat, about, kind))
            LOG.debug(
                "realtime channel ready chat=%s peer_id=%s",
                chat_key,
                get_peer_id(full),
            )
        except Exception as exc:
            LOG.warning("realtime skip chat=%s: %s", chat_key, exc)
    return out


async def run_realtime_listener(
    client: Any,
    cfg: ScraperConfig,
    store: CursorStore,
    out: TextIO,
) -> int:
    from telethon import events
    from telethon.utils import get_peer_id

    chats = merge_chat_lists(cfg.chats, store.list_enabled_chats())
    if not chats:
        LOG.error("no enabled telegram channels for realtime listener")
        return 1

    targets = await resolve_listen_targets(client, chats, store)
    if not targets:
        LOG.error("no resolvable channels for realtime listener")
        return 1

    entities = [row[0] for row in targets]
    meta_by_peer: dict[int, tuple[ChatConfig, str, str]] = {}
    for entity, chat, about, kind in targets:
        meta_by_peer[get_peer_id(entity)] = (chat, about, kind)

    @client.on(events.NewMessage(chats=entities))
    async def on_message(event: events.NewMessage.Event) -> None:
        meta = meta_by_peer.get(event.chat_id)
        if meta is None:
            return
        chat, about, kind = meta
        chat_key = chat.channel_key()
        try:
            entity = await event.get_chat()
            await process_scrape_message(
                client,
                entity,
                event.message,
                chat,
                about,
                kind,
                out,
                store,
                chat_key,
            )
        except Exception as exc:
            LOG.warning("realtime message error chat=%s: %s", chat_key, exc)

    LOG.info("telegram realtime listening channels=%d", len(entities))
    await client.run_until_disconnected()
    return 0


async def realtime_listen(cfg: ScraperConfig, out: TextIO) -> int:
    if not realtime_env_enabled():
        LOG.error("TELEGRAM_REALTIME not enabled; set TELEGRAM_REALTIME=1")
        return 1

    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    from pathlib import Path

    session_path = Path(cfg.session)
    with session_exclusive_lock(session_path):
        store = CursorStore(cfg.cursor_db)
        client = build_telegram_client(cfg, api_id, api_hash)
        await connect_telegram_client(client)
        if not await client.is_user_authorized():
            LOG.error("telethon session not authorized; run: parser telegram login")
            await client.disconnect()
            store.close()
            return 1

        try:
            return await run_realtime_listener(client, cfg, store, out)
        finally:
            await client.disconnect()
            store.close()
