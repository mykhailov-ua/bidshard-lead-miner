"""Chunked GetHistory iteration with inter-batch delays (MTProto ban protection)."""

from __future__ import annotations

import asyncio
import logging
import os
import random
from datetime import datetime, timezone
from typing import Any, AsyncIterator

LOG = logging.getLogger("telegram.history_chunk")


def _env_float(name: str, default: float) -> float:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    try:
        return float(raw)
    except ValueError:
        return default


def _env_int(name: str, default: int) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    try:
        return max(1, int(raw))
    except ValueError:
        return default


def history_chunk_size() -> int:
    return _env_int("TELEGRAM_HISTORY_CHUNK_SIZE", 100)


def history_chunk_delay_min() -> float:
    return _env_float("TELEGRAM_HISTORY_CHUNK_DELAY_MIN", 1.5)


def history_chunk_delay_max() -> float:
    return max(
        history_chunk_delay_min(),
        _env_float("TELEGRAM_HISTORY_CHUNK_DELAY_MAX", 3.5),
    )


def realtime_backfill_limit() -> int:
    return _env_int("TELEGRAM_REALTIME_BACKFILL", 300)


def _message_dt_utc(message: Any) -> datetime | None:
    msg_date = getattr(message, "date", None)
    if msg_date is None:
        return None
    if msg_date.tzinfo is None:
        return msg_date.replace(tzinfo=timezone.utc)
    return msg_date.astimezone(timezone.utc)


async def iter_messages_chunked(
    client: Any,
    entity: Any,
    *,
    total_limit: int,
    stop_before_id: int = 0,
    stop_before_date: datetime | None = None,
    chunk_size: int | None = None,
    delay_min: float | None = None,
    delay_max: float | None = None,
    search: str | None = None,
) -> AsyncIterator[Any]:
    """Yield messages newest-first with pause after each chunk (default 100 msgs)."""
    if total_limit <= 0 and stop_before_date is None:
        return

    chunk = chunk_size if chunk_size is not None else history_chunk_size()
    dmin = delay_min if delay_min is not None else history_chunk_delay_min()
    dmax = delay_max if delay_max is not None else history_chunk_delay_max()

    yielded = 0
    in_batch = 0
    kwargs: dict[str, Any] = {}
    if total_limit > 0:
        kwargs["limit"] = total_limit
    if search:
        kwargs["search"] = search

    async for message in client.iter_messages(entity, **kwargs):
        if stop_before_id and message.id <= stop_before_id:
            break
        if stop_before_date is not None:
            msg_dt = _message_dt_utc(message)
            if msg_dt is not None and msg_dt < stop_before_date:
                break
        yield message
        yielded += 1
        in_batch += 1
        if total_limit > 0 and yielded >= total_limit:
            break
        if in_batch >= chunk:
            in_batch = 0
            wait = random.uniform(dmin, dmax)
            LOG.debug(
                "history chunk pause entity=%s waited=%.2fs batch=%d",
                entity,
                wait,
                chunk,
            )
            await asyncio.sleep(wait)
