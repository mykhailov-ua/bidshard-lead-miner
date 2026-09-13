"""Canonical prefixed ids for person-chat OSINT graph (matches Go crm/store/telegram_keys.go)."""

from __future__ import annotations


def telegram_person_key(user_id: int) -> str:
    uid = int(user_id or 0)
    if uid <= 0:
        return ""
    return f"tg_user:{uid}"


def telegram_chat_ref(chat_key: str) -> str:
    key = (chat_key or "").strip().lower()
    if not key:
        return ""
    if key.startswith("tg_chat:"):
        return key
    return f"tg_chat:{key}"


def telegram_chat_ref_from_username(username: str) -> str:
    name = (username or "").strip().lstrip("@").lower()
    if not name:
        return ""
    return telegram_chat_ref(f"u:{name}")


def telegram_member_edge_key(chat_ref: str, person_key: str) -> str:
    chat_ref = (chat_ref or "").strip()
    person_key = (person_key or "").strip()
    if not chat_ref or not person_key:
        return ""
    return f"tg_member:{chat_ref}|{person_key}"
