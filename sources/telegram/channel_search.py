from __future__ import annotations

import logging
from typing import Any, Awaitable, Callable

from .telethon_retry import call_with_flood_wait, is_flood_wait

LOG = logging.getLogger("telegram.channel_search")


async def iter_search_hits(
    client: Any,
    entity: Any,
    terms: list[str],
    limit_per_term: int,
    max_terms: int,
    on_message: Callable[[Any], Awaitable[bool]],
) -> tuple[int, int]:
    """Run in-channel search for pain terms. Returns (searches_run, emitted)."""
    emitted = 0
    searches = 0
    for term in terms:
        if searches >= max_terms:
            break
        term = term.strip()
        if not term:
            continue
        searches += 1
        try:

            async def _search(term: str = term) -> int:
                count = 0
                async for message in client.iter_messages(
                    entity, search=term, limit=limit_per_term
                ):
                    if await on_message(message):
                        count += 1
                return count

            hit_emitted = await call_with_flood_wait(
                f"channel_search:{term}",
                _search,
                attempts=2,
            )
            if hit_emitted:
                emitted += int(hit_emitted)
        except Exception as exc:
            if is_flood_wait(exc):
                LOG.warning(
                    "channel search flood wait term=%s seconds=%s skip",
                    term,
                    getattr(exc, "seconds", 0),
                )
                continue
            LOG.warning("channel search failed term=%s error=%s", term, exc)
    return searches, emitted
