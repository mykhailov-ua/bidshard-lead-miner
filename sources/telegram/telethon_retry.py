from __future__ import annotations

import asyncio
import logging
import random
from collections.abc import Awaitable, Callable
from typing import Any, TypeVar

LOG = logging.getLogger("telegram.telethon_retry")

MAX_FLOOD_WAIT_SEC = 300

T = TypeVar("T")

try:
    from telethon.errors import FloodWaitError
except ImportError:  # pragma: no cover - telethon optional in CI unit tests
    FloodWaitError = None  # type: ignore[misc, assignment]


def capped_flood_wait_seconds(seconds: int) -> int:
    return min(max(int(seconds), 0), MAX_FLOOD_WAIT_SEC)


async def sleep_flood_wait(seconds: int, *, label: str = "") -> int:
    """Sleep capped FloodWait with small jitter. Returns capped seconds."""
    wait = capped_flood_wait_seconds(seconds)
    jitter = random.uniform(0, min(5.0, wait * 0.05))
    LOG.warning(
        "FloodWait label=%s requested=%s capped=%s",
        label or "telethon",
        seconds,
        wait,
    )
    await asyncio.sleep(wait + jitter)
    return wait


def is_flood_wait(exc: BaseException) -> bool:
    return FloodWaitError is not None and isinstance(exc, FloodWaitError)


async def call_with_flood_wait(
    label: str,
    fn: Callable[[], Awaitable[T]],
    *,
    attempts: int = 2,
) -> T | None:
    """Run async callable; sleep capped and retry on FloodWait."""
    last_exc: BaseException | None = None
    for attempt in range(attempts):
        try:
            return await fn()
        except BaseException as exc:
            if not is_flood_wait(exc) or attempt >= attempts - 1:
                raise
            last_exc = exc
            seconds = getattr(exc, "seconds", 0)
            await sleep_flood_wait(seconds, label=label)
    if last_exc is not None:
        raise last_exc
    return None
