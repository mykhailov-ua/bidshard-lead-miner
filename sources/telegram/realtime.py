"""Long-running Telethon NewMessage listener -> NDJSON stdout."""

from __future__ import annotations

import logging
import os
from typing import Any, TextIO

from .config import ChatConfig, ScraperConfig
from .connect import connect_telegram_client
from .cursor import CursorStore
from .discover import merge_chat_lists, sync_manual_curation
from .history_chunk import iter_messages_chunked, realtime_backfill_limit
from .geo_heuristic import channel_geo_reject, channel_geo_texts
from .scraper import (
    build_telegram_client,
    entity_chat_type,
    fetch_channel_about,
    process_scrape_message,
    resolve_chat_entity,
)
from .user_enrich import UserBioEnricher, user_enrich_limit
from .session_lock import session_exclusive_lock

LOG = logging.getLogger("telegram.realtime")


def realtime_env_enabled() -> bool:
    return os.environ.get("TELEGRAM_REALTIME", "0").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def realtime_chat_allowed(chat: ChatConfig) -> bool:
    """M4: optional role filter for realtime listener (not bg scrape)."""
    want = os.environ.get("TELEGRAM_REALTIME_ROLE_FILTER", "").strip().lower()
    if not want:
        return True
    role = (chat.role or "buyer_supergroup").strip().lower()
    if want == "buyer_supergroup":
        return role in ("buyer_supergroup", "buyer", "")
    return role == want


async def resolve_listen_targets(
    client: Any, chats: list[ChatConfig], store: CursorStore
) -> list[tuple[Any, ChatConfig, str, str]]:
    """Resolve enabled chats to Telethon entities; skip failures."""
    from telethon.utils import get_peer_id

    out: list[tuple[Any, ChatConfig, str, str]] = []
    for chat in chats:
        if not realtime_chat_allowed(chat):
            LOG.debug("realtime skip chat role filter chat=%s role=%s", chat.name, chat.role)
            continue
        chat_key = chat.channel_key()
        try:
            entity = await resolve_chat_entity(client, chat, store)
            full = await client.get_entity(entity)
            about = await fetch_channel_about(client, full)
            if channel_geo_reject(channel_geo_texts(chat.name, about, full)):
                LOG.info("realtime skip chat geo heuristic chat=%s", chat_key)
                continue
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


async def backfill_listen_targets(
    client: Any,
    targets: list[tuple[Any, ChatConfig, str, str]],
    store: CursorStore,
    out: TextIO,
    limit: int,
    bio_enricher: UserBioEnricher | None = None,
) -> int:
    """Chunked history read before listener mode (no GetHistory after startup)."""
    total = 0
    for entity, chat, about, kind in targets:
        chat_key = chat.channel_key()
        last_id = store.get_last_message_id(chat_key)
        max_seen = last_id
        emitted = 0
        try:
            async for message in iter_messages_chunked(
                client,
                entity,
                total_limit=limit,
                stop_before_id=last_id,
            ):
                if await process_scrape_message(
                    client,
                    entity,
                    message,
                    chat,
                    about,
                    kind,
                    out,
                    store,
                    chat_key,
                    bio_enricher,
                ):
                    emitted += 1
                    max_seen = max(max_seen, message.id)
        except Exception as exc:
            LOG.warning("realtime backfill error chat=%s: %s", chat_key, exc)
            continue
        if max_seen > last_id:
            store.set_last_message_id(chat_key, max_seen)
        total += emitted
        LOG.info(
            "realtime backfill done chat=%s limit=%d emitted=%d",
            chat_key,
            limit,
            emitted,
        )
    return total


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

    bio_enricher = UserBioEnricher(store, user_enrich_limit())
    backfill_limit = realtime_backfill_limit()
    LOG.info(
        "realtime backfill starting channels=%d limit=%d (then NewMessage only)",
        len(targets),
        backfill_limit,
    )
    await backfill_listen_targets(client, targets, store, out, backfill_limit, bio_enricher)

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
                bio_enricher,
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
        sync_manual_curation(cfg.chats, store)
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
