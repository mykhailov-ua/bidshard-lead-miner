"""Cross-mention / forward-chase from scrape batch (not only discover)."""

from __future__ import annotations

import logging
import os
from typing import Any

from .config import ChatConfig, ScraperConfig
from .crossmention import extract_forward_channel_usernames, resolve_username_channels
from .cursor import CursorStore
from .domains import append_domains
from .tglinks import extract_from_texts, web_domains

LOG = logging.getLogger("telegram.scrape_crossmention")


def scrape_crossmention_enabled() -> bool:
    return os.environ.get("TELEGRAM_SCRAPE_CROSS_MENTION", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def scrape_crossmention_rescan_days() -> int:
    raw = os.environ.get("TELEGRAM_SCRAPE_CROSS_MENTION_DAYS", "3").strip()
    try:
        return max(1, int(raw))
    except ValueError:
        return 3


async def run_scrape_cross_mention(
    client: Any,
    chat: ChatConfig,
    messages: list[Any],
    about_text: str,
    store: CursorStore,
    cfg: ScraperConfig,
) -> int:
    """Mine handles, invites, forwards, domains from messages already fetched."""
    if not scrape_crossmention_enabled():
        return 0
    if not cfg.discover.cross_mention.enabled:
        return 0
    chat_key = chat.channel_key()
    if not store.scrape_crossmention_due(chat_key, scrape_crossmention_rescan_days()):
        return 0

    texts: list[str] = []
    if about_text:
        texts.append(about_text)
    from .message_text import combined_message_text

    for msg in messages:
        body = combined_message_text(msg)
        if body:
            texts.append(body)

    if not texts and not messages:
        store.touch_scrape_crossmention(chat_key)
        return 0

    known_keys = {c.channel_key() for c in store.list_enabled_chats()}
    seed_keys = {chat_key}
    channel_name = (chat.username or chat.name or "").strip().lstrip("@").lower()

    handles, invites = extract_from_texts(texts)
    pending_handles: list[str] = []
    pending_invites: list[str] = []
    seen_h: set[str] = set()
    seen_i: set[str] = set()
    for h in handles:
        key = f"u:{h}"
        if key in known_keys or key in seed_keys or h in seen_h:
            continue
        seen_h.add(h)
        pending_handles.append(h)
    for inv in invites:
        key = f"i:{inv}"
        if key in known_keys or key in seed_keys or inv in seen_i:
            continue
        seen_i.add(inv)
        pending_invites.append(inv)

    pending_forward: list[str] = []
    seen_f: set[str] = set()
    for fwd in await extract_forward_channel_usernames(client, messages):
        key = f"u:{fwd}"
        if key in known_keys or key in seed_keys or fwd in seen_f:
            continue
        seen_f.add(fwd)
        pending_forward.append(fwd)

    new_count = 0
    resolved = await resolve_username_channels(
        client, pending_handles, known_keys, seed_keys
    )
    forward_resolved = await resolve_username_channels(
        client, pending_forward, known_keys, seed_keys
    )
    for c in resolved:
        store.upsert_channel(c, "scrape_cross_mention")
        new_count += 1
    for c in forward_resolved:
        store.upsert_channel(c, "scrape_cross_mention_forward")
        new_count += 1
    if pending_invites:
        from .invites import discover_invite_hashes

        for c in await discover_invite_hashes(client, pending_invites):
            store.upsert_channel(c, "scrape_cross_mention_invite")
            new_count += 1

    domain_entries: list[dict[str, str]] = []
    if cfg.discover.domains_path:
        seen_domains: set[str] = set()
        for domain in web_domains("\n".join(texts)):
            if domain in seen_domains:
                continue
            seen_domains.add(domain)
            domain_entries.append(
                {
                    "domain": domain,
                    "channel": channel_name,
                    "source": "scrape_cross_mention",
                    "kind": "mentioned_in_message",
                    "discovered_via": channel_name or chat_key,
                }
            )
        if domain_entries:
            append_domains(cfg.discover.domains_path, domain_entries)

    store.touch_scrape_crossmention(chat_key)
    if new_count:
        store.bump_discover_boost(chat_key, new_count)
    if new_count or domain_entries:
        LOG.info(
            "scrape_cross_mention chat=%s new_channels=%d domains=%d",
            chat_key,
            new_count,
            len(domain_entries),
        )
    return new_count
