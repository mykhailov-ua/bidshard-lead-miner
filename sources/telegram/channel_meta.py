"""Channel/supergroup metadata: GetFullChannel + admin roster."""

from __future__ import annotations

import json
import logging
import os
from pathlib import Path
from typing import Any

from .config import ChatConfig
from .cursor import CursorStore
from .telethon_retry import call_with_flood_wait, is_flood_wait

LOG = logging.getLogger("telegram.channel_meta")

DEFAULT_EXPORT = "data/runtime/telegram_channel_meta.json"


def channel_meta_export_path() -> str:
    return os.environ.get("TELEGRAM_CHANNEL_META_EXPORT", DEFAULT_EXPORT).strip()


def channel_meta_refresh_days() -> int:
    raw = os.environ.get("TELEGRAM_CHANNEL_META_REFRESH_DAYS", "7").strip()
    try:
        return max(1, int(raw))
    except ValueError:
        return 7


async def harvest_channel_meta(
    client: Any,
    entity: Any,
    chat: ChatConfig,
    store: CursorStore,
    chat_kind: str,
) -> bool:
    if chat_kind not in ("channel", "supergroup", "group"):
        return False
    chat_key = chat.channel_key()
    if not store.channel_meta_due(chat_key, channel_meta_refresh_days()):
        return False

    from telethon.tl.functions.channels import GetFullChannelRequest
    from telethon.tl.types import Channel

    if not isinstance(entity, Channel):
        return False

    meta: dict[str, Any] = {
        "chat_key": chat_key,
        "chat_username": (chat.username or "").lower().lstrip("@"),
        "chat_type": chat_kind,
        "channel_role": chat.normalized_role(),
    }

    try:
        from .telethon_input import channel_input_peer

        full = await call_with_flood_wait(
            f"channel_meta:{chat_key}",
            lambda: client(GetFullChannelRequest(channel_input_peer(entity))),
            attempts=2,
        )
        if full is not None:
            fc = full.full_chat
            meta.update(
                {
                    "about": str(getattr(fc, "about", "") or ""),
                    "participants_count": int(getattr(fc, "participants_count", 0) or 0),
                    "admins_count": int(getattr(fc, "admins_count", 0) or 0),
                    "online_count": int(getattr(fc, "online_count", 0) or 0),
                    "slowmode_seconds": int(getattr(fc, "slowmode_seconds", 0) or 0),
                    "pinned_msg_id": int(getattr(fc, "pinned_msg_id", 0) or 0),
                    "linked_chat_id": int(getattr(fc, "linked_chat_id", 0) or 0),
                    "can_view_participants": bool(
                        getattr(fc, "can_view_participants", False)
                    ),
                }
            )
    except Exception as exc:
        if is_flood_wait(exc):
            LOG.warning("channel_meta flood wait chat=%s", chat_key)
        else:
            LOG.warning("channel_meta full failed chat=%s error=%s", chat_key, exc)
        return False

    admins = await _fetch_admins(client, entity, chat_key)
    if admins:
        meta["admins"] = admins

    store.upsert_channel_meta(chat_key, meta)
    LOG.info(
        "channel_meta harvest chat=%s participants=%s admins=%d",
        chat_key,
        meta.get("participants_count"),
        len(admins),
    )
    return True


async def _fetch_admins(client: Any, entity: Any, chat_key: str) -> list[dict[str, Any]]:
    from telethon.tl.types import ChannelParticipantsAdmins

    limit = 50
    raw = os.environ.get("TELEGRAM_CHANNEL_ADMIN_LIMIT", "50").strip()
    try:
        limit = max(1, int(raw))
    except ValueError:
        pass

    admins: list[dict[str, Any]] = []
    try:

        async def _iter() -> None:
            async for user in client.iter_participants(
                entity, filter=ChannelParticipantsAdmins, limit=limit
            ):
                row = _admin_row(user)
                if row:
                    admins.append(row)

        await call_with_flood_wait(f"channel_admins:{chat_key}", _iter, attempts=2)
    except Exception as exc:
        LOG.warning("channel_meta admins failed chat=%s error=%s", chat_key, exc)
    return admins


def _admin_row(user: Any) -> dict[str, Any] | None:
    from telethon.tl.types import User

    if not isinstance(user, User):
        return None
    uid = int(user.id or 0)
    if uid <= 0:
        return None
    return {
        "user_id": uid,
        "username": (user.username or "").strip().lstrip("@"),
        "first_name": (user.first_name or "").strip(),
        "is_bot": bool(user.bot),
    }


def export_channel_meta_json(store: CursorStore, path: str | Path) -> int:
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    rows = store.list_all_channel_meta()
    path.write_text(
        json.dumps({"channels": rows, "count": len(rows)}, indent=2, ensure_ascii=False)
        + "\n",
        encoding="utf-8",
    )
    return len(rows)


DEFAULT_SUPPLY_ADMINS_CSV = "data/export/telegram_supply_admins.csv"

_SUPPLY_CHANNEL_HINTS = (
    "cpa network",
    "affiliate network",
    "aff network",
    "affiliate program",
    "partner program",
)


def supply_admins_csv_path() -> str:
    return os.environ.get("TELEGRAM_SUPPLY_ADMINS_CSV", DEFAULT_SUPPLY_ADMINS_CSV).strip()


def _is_supply_channel_meta(meta: dict[str, Any]) -> bool:
    role = str(meta.get("channel_role") or "").strip().lower()
    if role == "supply":
        return True
    blob = " ".join(
        [
            str(meta.get("chat_username") or ""),
            str(meta.get("about") or ""),
        ]
    ).lower()
    return any(h in blob for h in _SUPPLY_CHANNEL_HINTS)


def export_supply_admins_csv(store: CursorStore, path: str | Path) -> int:
    """Cold outreach list for network AMs; not hot buyer leads."""
    import csv

    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    fieldnames = (
        "chat_key",
        "chat_username",
        "admin_user_id",
        "admin_username",
        "first_name",
        "person_key",
    )
    rows_out: list[dict[str, str]] = []
    from .tg_graph_ids import telegram_person_key

    for meta in store.list_all_channel_meta():
        if not _is_supply_channel_meta(meta):
            continue
        chat_key = str(meta.get("chat_key") or "")
        chat_username = str(meta.get("chat_username") or "")
        for admin in meta.get("admins") or []:
            if not isinstance(admin, dict):
                continue
            if admin.get("is_bot"):
                continue
            uid = int(admin.get("user_id") or 0)
            if uid <= 0:
                continue
            rows_out.append(
                {
                    "chat_key": chat_key,
                    "chat_username": chat_username,
                    "admin_user_id": str(uid),
                    "admin_username": str(admin.get("username") or ""),
                    "first_name": str(admin.get("first_name") or ""),
                    "person_key": telegram_person_key(uid),
                }
            )
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows_out)
    return len(rows_out)
