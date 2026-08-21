from __future__ import annotations

import logging
from typing import Any

from .config import ChatConfig

LOG = logging.getLogger("telegram.invites")


async def discover_invite_hashes(client: Any, hashes: list[str]) -> list[ChatConfig]:
    from telethon.tl.functions.messages import CheckChatInviteRequest

    out: list[ChatConfig] = []
    seen: set[str] = set()
    for raw_hash in hashes:
        invite_hash = raw_hash.strip()
        if not invite_hash:
            continue
        key = f"i:{invite_hash}"
        if key in seen:
            continue
        seen.add(key)
        title = invite_hash
        chat_id: int | None = None
        try:
            checked = await client(CheckChatInviteRequest(invite_hash))
            title = getattr(checked, "title", None) or title
            if getattr(checked, "chat", None) is not None:
                chat_id = getattr(checked.chat, "id", None)
                title = getattr(checked.chat, "title", title) or title
        except Exception as exc:
            LOG.warning("invite check failed hash=%s error=%s", invite_hash, exc)
            continue
        out.append(
            ChatConfig(
                name=str(title),
                invite_hash=invite_hash,
                geo="global",
                enabled=True,
                chat_id=chat_id,
            )
        )
    return out
