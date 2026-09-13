"""Limit discussion scrape to top buyer supergroups (RPC budget)."""

from __future__ import annotations

import os

from .config import ChatConfig
from .cursor import CursorStore


def discussion_max_buyer_chats() -> int:
    raw = os.environ.get("TELEGRAM_DISCUSSION_MAX_CHATS", "8").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 8


def discussion_allowed_for_chat(chat: ChatConfig, store: CursorStore) -> bool:
    if chat.normalized_role() != "buyer_supergroup":
        return False
    limit = discussion_max_buyer_chats()
    if limit <= 0:
        return False
    allowed = {c.channel_key() for c in store.list_top_buyer_supergroups(limit)}
    return chat.channel_key() in allowed
