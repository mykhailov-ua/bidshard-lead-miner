"""GetFullUser bio enrichment with crawler.db cache and per-run rate limit."""

from __future__ import annotations

import os
from typing import Any

from .cursor import CursorStore

BIO_CACHE_TTL_DAYS = 7


def user_enrich_limit() -> int:
    raw = os.environ.get("TELEGRAM_USER_ENRICH_LIMIT", "10").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 10


class UserBioEnricher:
    def __init__(self, store: CursorStore, limit: int) -> None:
        self._store = store
        self._limit = max(0, limit)
        self._fetched = 0

    async def enrich(self, client: Any, sender: Any) -> str:
        if self._limit <= 0:
            return ""
        user_id = _sender_id(sender)
        if user_id <= 0:
            return ""
        cached = self._store.get_user_bio(user_id, BIO_CACHE_TTL_DAYS)
        if cached is not None:
            return cached
        if self._fetched >= self._limit:
            return ""
        bio = await _fetch_full_user_bio(client, sender)
        self._fetched += 1
        self._store.set_user_bio(user_id, bio)
        return bio


def _sender_id(sender: Any) -> int:
    from telethon.tl.types import User

    if isinstance(sender, User) and sender.id:
        return int(sender.id)
    return 0


async def _fetch_full_user_bio(client: Any, sender: Any) -> str:
    from telethon.tl.functions.users import GetFullUserRequest
    from telethon.tl.types import User

    if not isinstance(sender, User):
        return ""
    try:
        full = await client(GetFullUserRequest(sender))
        about = getattr(full.full_user, "about", None)
        if about:
            return str(about).strip()
    except Exception:
        pass
    return ""
