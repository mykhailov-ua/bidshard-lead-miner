"""Invite join policy for scrape path.

Discover uses CheckChatInvite only (invites.py). Scrape calls resolve_invite_entity:
already-joined invites return chat without ImportChatInvite; preview-only hashes need
TELEGRAM_INVITE_JOIN=1 and stay under TELEGRAM_INVITE_JOIN_LIMIT per day (crawler.db).
"""
import os
from typing import Any

from .config import ChatConfig
from .cursor import CursorStore
from .telethon_retry import call_with_flood_wait, is_flood_wait

LOG = logging.getLogger("telegram.join_policy")


def invite_join_enabled() -> bool:
    return os.environ.get("TELEGRAM_INVITE_JOIN", "").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def invite_join_daily_limit() -> int:
    raw = os.environ.get("TELEGRAM_INVITE_JOIN_LIMIT", "3").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 3


async def resolve_invite_entity(
    client: Any, chat: ChatConfig, store: CursorStore
) -> Any:
    """Check invite metadata first; join only when explicitly enabled and under daily cap."""
    from telethon.tl.functions.messages import (
        CheckChatInviteRequest,
        ImportChatInviteRequest,
    )

    invite_hash = (chat.invite_hash or "").strip()
    if not invite_hash:
        raise ValueError("missing invite hash")

    try:
        checked = await call_with_flood_wait(
            f"invite_check:{invite_hash[:8]}",
            lambda: client(CheckChatInviteRequest(invite_hash)),
            attempts=2,
        )
    except Exception as exc:
        if is_flood_wait(exc):
            raise ValueError("invite check flood wait") from exc
        raise ValueError(f"invite check failed: {exc}") from exc

    if checked is None:
        raise ValueError("invite check returned empty")

    existing = getattr(checked, "chat", None)
    if existing is not None:
        chat_id = int(getattr(existing, "id", 0) or 0)
        if chat_id:
            store.set_chat_id(chat.channel_key(), chat_id)
        return existing

    if not invite_join_enabled():
        raise ValueError(
            "invite preview only; discover should persist chat_id or set TELEGRAM_INVITE_JOIN=1"
        )
    if not store.can_invite_join(invite_join_daily_limit()):
        raise ValueError("invite join daily limit reached")

    updates = await client(ImportChatInviteRequest(invite_hash))
    chats = getattr(updates, "chats", None) or []
    if not chats:
        raise ValueError("invite import returned no chat")
    entity = chats[0]
    store.set_chat_id(chat.channel_key(), int(entity.id))
    store.record_invite_join()
    LOG.info("invite join ok channel_key=%s chat_id=%s", chat.channel_key(), entity.id)
    return entity
