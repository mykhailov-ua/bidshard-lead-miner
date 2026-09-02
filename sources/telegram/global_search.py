"""Optional global pain search during scrape round (not discover).

TELEGRAM_GLOBAL_SEARCH=1 runs iter_messages(None, search=...) before channel loop.
Hourly query budget in telegram_runtime (TELEGRAM_GLOBAL_SEARCH_LIMIT, default 5/hour UTC).
"""

from __future__ import annotations

import json
import logging
import os
import re
from typing import Any, TextIO

from .config import ScraperConfig
from .cursor import CursorStore
from .message_text import combined_message_text
from .pain import message_has_pain
from .prefilter import should_emit_message
from .telethon_retry import call_with_flood_wait, is_flood_wait

LOG = logging.getLogger("telegram.global_search")


def global_search_enabled() -> bool:
    return os.environ.get("TELEGRAM_GLOBAL_SEARCH", "").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def global_search_hourly_limit() -> int:
    raw = os.environ.get("TELEGRAM_GLOBAL_SEARCH_LIMIT", "5").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 5


def _slug_query(query: str) -> str:
    slug = re.sub(r"[^a-zA-Z0-9]+", "_", query.strip().lower()).strip("_")
    return slug[:48] or "query"


async def run_global_search(
    client: Any,
    cfg: ScraperConfig,
    store: CursorStore,
    out: TextIO,
) -> int:
    if not global_search_enabled():
        return 0
    terms = cfg.global_search.terms
    if not terms:
        return 0
    limit = global_search_hourly_limit()
    if not store.can_global_search(limit):
        LOG.info("global search skipped hourly budget exhausted")
        return 0

    remaining = limit - store.global_search_count_this_hour()
    if remaining <= 0:
        return 0

    emitted = 0
    searches = 0
    seen: set[tuple[int, int]] = set()

    for query in terms:
        if searches >= remaining:
            break
        query = query.strip()
        if not query:
            continue
        searches += 1
        source = f"telegram:global:{_slug_query(query)}"

        async def _search(q: str = query, src: str = source) -> int:
            count = 0
            async for message in client.iter_messages(
                None, search=q, limit=cfg.global_search.messages_per_query
            ):
                key = (int(getattr(message, "chat_id", 0) or 0), int(message.id))
                if key in seen:
                    continue
                seen.add(key)
                body = combined_message_text(message)
                if not body or not should_emit_message(body):
                    continue
                if not message_has_pain(body):
                    continue
                username = ""
                sender = await message.get_sender()
                if sender is not None and getattr(sender, "username", None):
                    username = str(sender.username)
                payload = {
                    "source": src,
                    "text": body,
                    "username": username,
                    "message_id": int(message.id),
                    "chat_type": "global_search",
                }
                if message.reply_to and getattr(message.reply_to, "reply_to_msg_id", None):
                    payload["reply_to_message_id"] = int(
                        message.reply_to.reply_to_msg_id
                    )
                out.write(json.dumps(payload, ensure_ascii=False) + "\n")
                out.flush()
                count += 1
            return count

        try:
            hit = await call_with_flood_wait(
                f"global_search:{query[:24]}",
                _search,
                attempts=2,
            )
            if hit:
                emitted += int(hit)
        except Exception as exc:
            if is_flood_wait(exc):
                LOG.warning(
                    "global search flood wait query=%s seconds=%s skip",
                    query,
                    getattr(exc, "seconds", 0),
                )
                continue
            LOG.warning("global search failed query=%s error=%s", query, exc)

    store.record_global_search(searches)
    if emitted:
        LOG.info("global search emitted=%d searches=%d", emitted, searches)
    return emitted
