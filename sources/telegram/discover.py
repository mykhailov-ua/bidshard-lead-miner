from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

from .config import ChatConfig, DiscoverConfig
from .domains import append_domains
from .crossmention import discover_cross_mentions
from .invites import discover_invite_hashes

from .prefilter import channel_icp_relevant
from .geo_heuristic import channel_geo_reject

LOG = logging.getLogger("telegram.discover")


def load_serp_entries(path: str | Path) -> list[ChatConfig]:
    p = Path(path)
    if not p.exists():
        return []
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        LOG.warning("serp channel file unreadable path=%s error=%s", p, exc)
        return []

    out: list[ChatConfig] = []
    seen: set[str] = set()
    for entry in data.get("channels", []):
        username = str(entry.get("username", "")).strip().lstrip("@").lower()
        invite_hash = str(entry.get("invite_hash", "")).strip()
        title = str(entry.get("title", username or invite_hash or "channel"))
        if channel_geo_reject([title, str(entry.get("query", ""))]):
            continue
        chat = ChatConfig(
            name=title,
            username=username,
            invite_hash=invite_hash,
            geo="global",
            enabled=True,
        )
        key = chat.channel_key()
        if key in seen:
            continue
        if not username and not invite_hash:
            continue
        seen.add(key)
        out.append(chat)
    return out


async def discover_via_search(
    client: Any,
    queries: list[str],
    limit_per_query: int,
) -> list[ChatConfig]:
    from telethon.tl.functions.contacts import SearchRequest
    from telethon.tl.types import Channel, Chat

    seen: set[str] = set()
    out: list[ChatConfig] = []

    for query in queries:
        query = query.strip()
        if not query:
            continue
        try:
            result = await client(SearchRequest(q=query, limit=limit_per_query))
        except Exception as exc:
            LOG.warning("telegram search failed query=%s error=%s", query, exc)
            continue

        for chat in result.chats:
            if not isinstance(chat, (Channel, Chat)):
                continue
            username = getattr(chat, "username", None)
            if not username:
                continue
            username = username.lower()
            title = getattr(chat, "title", username) or username
            if not channel_icp_relevant([title, query]):
                continue
            if channel_geo_reject([title, query]):
                continue
            cfg = ChatConfig(
                name=title,
                username=username,
                geo="global",
                enabled=True,
                chat_id=getattr(chat, "id", None),
            )
            key = cfg.channel_key()
            if key in seen:
                continue
            seen.add(key)
            out.append(cfg)
        LOG.info(
            "telegram search query=%s hits=%d total=%d",
            query,
            len(result.chats),
            len(out),
        )

    return out


def merge_chat_lists(
    manual: list[ChatConfig], discovered: list[ChatConfig]
) -> list[ChatConfig]:
    by_key: dict[str, ChatConfig] = {}
    for chat in discovered:
        by_key[chat.channel_key()] = chat
    # Manual entries override discovered duplicates (curated channels win).
    for chat in manual:
        by_key[chat.channel_key()] = chat
    return list(by_key.values())


async def run_discover(
    client: Any,
    manual: list[ChatConfig],
    discover: DiscoverConfig,
    store: Any,
) -> int:
    discovered: list[ChatConfig] = []

    if discover.serp_channels_path:
        serp_entries = load_serp_entries(discover.serp_channels_path)
        invite_hashes = [c.invite_hash for c in serp_entries if c.invite_hash]
        discovered.extend([c for c in serp_entries if c.username])
        if invite_hashes:
            discovered.extend(await discover_invite_hashes(client, invite_hashes))

    if discover.enabled and discover.queries:
        discovered.extend(
            await discover_via_search(
                client, discover.queries, discover.limit_per_query
            )
        )

    for chat in discovered:
        store.upsert_channel(chat, "discover")
    for chat in manual:
        store.upsert_channel(chat, "manual")

    cross_new: list[ChatConfig] = []
    cross_forward: list[ChatConfig] = []
    domain_entries: list[tuple[str, str, str] | dict[str, str]] = []
    if discover.cross_mention.enabled:
        known_keys = {c.channel_key() for c in store.list_enabled_chats()}
        seeds = store.list_cross_mention_seeds(
            discover.cross_mention.max_seeds,
            discover.cross_mention.rescan_days,
        )
        cross_new, cross_forward, domain_entries = await discover_cross_mentions(
            client,
            seeds,
            discover.cross_mention,
            known_keys,
        )
        for seed in seeds:
            store.mark_cross_mention_scanned(seed.channel_key())
        for chat in cross_new:
            store.upsert_channel(chat, "cross_mention")
        for chat in cross_forward:
            # Forward-chased channels are a separate provenance line for ops tuning / rescan.
            store.upsert_channel(chat, "cross_mention_forward")

    if domain_entries and discover.domains_path:
        # Feed tgweb crawler registry; Go reads discovered_telegram_domains.json.
        append_domains(discover.domains_path, domain_entries)

    total = len(store.list_enabled_chats())
    LOG.info(
        "telegram discover finished manual=%d new=%d cross_mention=%d cross_forward=%d domains=%d registry=%d",
        len(manual),
        len(discovered),
        len(cross_new),
        len(cross_forward),
        len(domain_entries),
        total,
    )
    return total


def chats_for_scrape(manual: list[ChatConfig], store: Any) -> list[ChatConfig]:
    return merge_chat_lists(manual, store.list_enabled_chats())
