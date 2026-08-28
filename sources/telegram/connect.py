"""Telethon connect with sqlite session lock retries."""

from __future__ import annotations

import asyncio
import sqlite3
from typing import Any


async def connect_telegram_client(
    client: Any,
    retries: int = 6,
    base_delay_sec: float = 2.0,
) -> None:
    last: sqlite3.OperationalError | None = None
    for attempt in range(retries):
        try:
            await client.connect()
            return
        except sqlite3.OperationalError as exc:
            if "locked" not in str(exc).lower():
                raise
            last = exc
            if attempt >= retries - 1:
                break
            await asyncio.sleep(base_delay_sec * (attempt + 1))
    if last is not None:
        raise last
