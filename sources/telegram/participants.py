"""Harvest chat member lists (Telethon iter_participants) into crawler.db."""

from __future__ import annotations

import json
import logging
import os
from pathlib import Path
from typing import Any

from .config import ChatConfig
from .cursor import CursorStore
from .telethon_retry import call_with_flood_wait, is_flood_wait

LOG = logging.getLogger("telegram.participants")

DEFAULT_EXPORT_PATH = "data/runtime/telegram_chat_members.json"


def participant_harvest_enabled() -> bool:
    return os.environ.get("TELEGRAM_PARTICIPANT_HARVEST", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def participant_limit_per_chat() -> int:
    raw = os.environ.get("TELEGRAM_PARTICIPANT_LIMIT", "500").strip()
    try:
        return max(1, int(raw))
    except ValueError:
        return 500


def participant_refresh_days() -> int:
    raw = os.environ.get("TELEGRAM_PARTICIPANT_REFRESH_DAYS", "7").strip()
    try:
        return max(1, int(raw))
    except ValueError:
        return 7


def participants_export_path() -> str:
    return os.environ.get("TELEGRAM_PARTICIPANTS_EXPORT", DEFAULT_EXPORT_PATH).strip()


def _member_row(user: Any) -> dict[str, Any] | None:
    from telethon.tl.types import User

    if not isinstance(user, User):
        return None
    user_id = int(user.id or 0)
    if user_id <= 0:
        return None
    username = (user.username or "").strip().lstrip("@")
    return {
        "user_id": user_id,
        "username": username,
        "first_name": (user.first_name or "").strip(),
        "last_name": (user.last_name or "").strip(),
        "is_bot": bool(user.bot),
    }


async def harvest_chat_participants(
    client: Any,
    entity: Any,
    chat: ChatConfig,
    store: CursorStore,
    chat_kind: str,
) -> int:
    """Fetch member roster for a group/supergroup; returns upserted row count."""
    if not participant_harvest_enabled():
        return 0
    if chat_kind not in ("group", "supergroup"):
        return 0

    chat_key = chat.channel_key()
    if not store.participants_harvest_due(chat_key, participant_refresh_days()):
        return 0

    limit = participant_limit_per_chat()
    members: list[dict[str, Any]] = []

    async def _iter() -> None:
        async for user in client.iter_participants(entity, limit=limit):
            row = _member_row(user)
            if row:
                members.append(row)

    try:
        await call_with_flood_wait(
            f"participants:{chat_key}",
            _iter,
            attempts=2,
        )
    except Exception as exc:
        if is_flood_wait(exc):
            LOG.warning(
                "participants flood wait chat=%s seconds=%s",
                chat_key,
                getattr(exc, "seconds", 0),
            )
        else:
            LOG.warning("participants harvest failed chat=%s error=%s", chat_key, exc)
        return 0

    if not members:
        store.touch_participants_harvest(chat_key)
        return 0

    chat_username = chat.username.lower().lstrip("@") if chat.username else ""
    upserted = store.upsert_chat_members(chat_key, chat_username, members)
    store.touch_participants_harvest(chat_key)
    LOG.info(
        "participants harvest chat=%s fetched=%d upserted=%d",
        chat_key,
        len(members),
        upserted,
    )
    await enrich_participant_profiles(client, members, store)
    return upserted


def participant_profile_limit() -> int:
    raw = os.environ.get("TELEGRAM_PARTICIPANT_PROFILE_LIMIT", "25").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 25


async def enrich_participant_profiles(
    client: Any,
    members: list[dict[str, Any]],
    store: CursorStore,
) -> int:
    from .user_enrich import UserProfileEnricher, user_enrich_limit

    limit = min(participant_profile_limit(), user_enrich_limit())
    if limit <= 0:
        return 0
    enricher = UserProfileEnricher(store, limit)
    done = 0
    for row in members:
        uid = int(row.get("user_id") or 0)
        if uid <= 0:
            continue
        try:
            user = await client.get_entity(uid)
            await enricher.enrich(client, user)
            done += 1
        except Exception as exc:
            LOG.debug("participant profile enrich skip user_id=%s error=%s", uid, exc)
    if done:
        LOG.info("participant profile enrich count=%d", done)
    return done


def export_participants_json(store: CursorStore, path: str | Path) -> int:
    """Write full member registry snapshot for ops/CRM scripts."""
    from .tg_graph_ids import (
        telegram_chat_ref,
        telegram_member_edge_key,
        telegram_person_key,
    )

    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    rows = store.list_all_chat_members()
    for row in rows:
        uid = int(row.get("user_id") or 0)
        person_key = telegram_person_key(uid)
        if person_key:
            row["person_key"] = person_key
        chat_key = str(row.get("chat_key") or "").strip()
        chat_ref = telegram_chat_ref(chat_key)
        if chat_ref:
            row["chat_ref"] = chat_ref
            edge = telegram_member_edge_key(chat_ref, person_key)
            if edge:
                row["member_edge"] = edge
    payload = {"members": rows, "count": len(rows)}
    path.write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return len(rows)
