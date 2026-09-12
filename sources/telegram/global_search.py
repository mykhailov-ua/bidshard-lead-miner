"""Optional global pain search during scrape round (not discover).

TELEGRAM_GLOBAL_SEARCH=1 runs iter_messages(None, search=...) before channel loop.
Hourly query budget in telegram_runtime (TELEGRAM_GLOBAL_SEARCH_LIMIT, default 3/hour UTC).
Daily cap (TELEGRAM_GLOBAL_SEARCH_DAILY_LIMIT, default 3/day UTC).
Night window only (TELEGRAM_GLOBAL_SEARCH_UTC_HOURS, default 2-6 UTC).
"""

from __future__ import annotations

import json
import logging
import os
import re
from datetime import datetime, timezone
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
    raw = os.environ.get("TELEGRAM_GLOBAL_SEARCH_LIMIT", "3").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 3


def global_search_daily_limit() -> int:
    raw = os.environ.get("TELEGRAM_GLOBAL_SEARCH_DAILY_LIMIT", "3").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 3


def parse_utc_hours_window(raw: str) -> frozenset[int]:
    """Parse hour spec: '2-6' or '2,3,4' or '2-6,14'. Returns UTC hours 0-23."""
    hours: set[int] = set()
    for part in raw.split(","):
        part = part.strip()
        if not part:
            continue
        if "-" in part:
            start_s, end_s = part.split("-", 1)
            start = int(start_s.strip()) % 24
            end = int(end_s.strip()) % 24
            if start <= end:
                hours.update(range(start, end + 1))
            else:
                hours.update(range(start, 24))
                hours.update(range(0, end + 1))
        else:
            hours.add(int(part) % 24)
    return frozenset(hours)


def global_search_utc_hours_raw() -> str:
    return os.environ.get("TELEGRAM_GLOBAL_SEARCH_UTC_HOURS", "2-6").strip()


def in_global_search_window(now: datetime | None = None) -> bool:
    raw = global_search_utc_hours_raw()
    if not raw or raw.lower() in ("*", "all"):
        return True
    allowed = parse_utc_hours_window(raw)
    if not allowed:
        return False
    if now is None:
        now = datetime.now(timezone.utc)
    return now.hour in allowed


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
    if not in_global_search_window():
        LOG.info("global_search_skipped_window")
        return 0
    terms = cfg.global_search.terms
    if not terms:
        return 0
    hourly_limit = global_search_hourly_limit()
    daily_limit = global_search_daily_limit()
    if not store.can_global_search(hourly_limit):
        LOG.info("global search skipped hourly budget exhausted")
        return 0
    if not store.can_global_search_daily(daily_limit):
        LOG.info("global_search_skipped_daily_cap")
        return 0

    hourly_remaining = hourly_limit - store.global_search_count_this_hour()
    daily_remaining = daily_limit - store.global_search_count_today()
    remaining = min(hourly_remaining, daily_remaining)
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
                from .ipc import emit_payload

                emit_payload(out, payload)
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
