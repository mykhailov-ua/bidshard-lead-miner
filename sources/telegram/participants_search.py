"""ChannelParticipantsSearch in megagroups (P1.5 MVP)."""

from __future__ import annotations

import logging
import os
from typing import Any

from .config import ChatConfig
from .cursor import CursorStore

LOG = logging.getLogger("telegram.participants_search")


def participant_search_enabled() -> bool:
    return os.environ.get("TELEGRAM_PARTICIPANT_SEARCH", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def participant_search_queries() -> list[str]:
    raw = os.environ.get(
        "TELEGRAM_PARTICIPANT_SEARCH_QUERIES",
        "media buyer,mediabuyer,traffic manager",
    ).strip()
    if not raw:
        return []
    return [q.strip() for q in raw.split(",") if q.strip()]


def participant_search_limit() -> int:
    raw = os.environ.get("TELEGRAM_PARTICIPANT_SEARCH_LIMIT", "15").strip()
    try:
        return max(1, int(raw))
    except ValueError:
        return 15


async def search_megagroup_participants(
    client: Any,
    entity: Any,
    chat: ChatConfig,
    store: CursorStore,
    chat_kind: str,
) -> int:
    if not participant_search_enabled():
        return 0
    if chat_kind not in ("group", "supergroup"):
        return 0
    queries = participant_search_queries()
    if not queries:
        return 0

    from telethon.tl.functions.channels import GetParticipantsRequest
    from telethon.tl.types import ChannelParticipantsSearch, User

    from .participants import _member_row
    from .telethon_input import channel_input_peer

    chat_key = chat.channel_key()
    limit = participant_search_limit()
    upserted = 0
    peer = channel_input_peer(entity)

    for query in queries:
        try:
            resp = await client(
                GetParticipantsRequest(
                    channel=peer,
                    filter=ChannelParticipantsSearch(query),
                    offset=0,
                    limit=limit,
                    hash=0,
                )
            )
        except Exception as exc:
            LOG.debug(
                "participant search skip chat=%s query=%s error=%s",
                chat_key,
                query,
                exc,
            )
            continue
        users = getattr(resp, "users", None) or []
        members: list[dict[str, Any]] = []
        for user in users:
            if not isinstance(user, User):
                continue
            row = _member_row(user)
            if row:
                members.append(row)
        if members:
            upserted += store.upsert_chat_members(chat_key, chat.username or "", members)
        LOG.info(
            "participant search chat=%s query=%s hits=%d",
            chat_key,
            query,
            len(members),
        )
    return upserted
