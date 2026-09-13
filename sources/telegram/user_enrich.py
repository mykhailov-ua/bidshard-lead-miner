"""GetFullUser profile enrichment with crawler.db cache and per-run rate limit."""

from __future__ import annotations

import json
import logging
import os
from typing import Any

from .cursor import CursorStore
from .outreach_fit import cold_outreach_fit
from .profile_links import enqueue_profile_links, profile_link_enqueue_limit

LOG = logging.getLogger("telegram.user_enrich")

PROFILE_CACHE_TTL_DAYS = 7


def user_enrich_limit() -> int:
    raw = os.environ.get("TELEGRAM_USER_ENRICH_LIMIT", "40").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 40


def user_enrich_prefilter_only() -> bool:
    return os.environ.get("TELEGRAM_USER_ENRICH_PREFILTER_ONLY", "0").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def user_profiles_export_path() -> str:
    return os.environ.get(
        "TELEGRAM_USER_PROFILES_EXPORT",
        "data/runtime/telegram_user_profiles.json",
    ).strip()


class UserProfileEnricher:
    def __init__(self, store: CursorStore, limit: int) -> None:
        self._store = store
        self._limit = max(0, limit)
        self._fetched = 0
        self._links_enqueued = 0

    async def enrich(
        self,
        client: Any,
        sender: Any,
        *,
        force: bool = False,
    ) -> tuple[str, dict[str, Any]]:
        """Return (about text, profile dict)."""
        user_id = _sender_id(sender)
        if user_id <= 0:
            return "", {}
        cached = self._store.get_user_profile(user_id, PROFILE_CACHE_TTL_DAYS)
        if cached is not None and not force:
            about = str(cached.get("about") or "")
            return about, cached
        if self._fetched >= self._limit:
            if cached is not None:
                about = str(cached.get("about") or "")
                return about, cached
            return "", {}
        profile = await _fetch_full_user_profile(client, sender)
        if not profile:
            return "", {}
        self._fetched += 1
        profile["cold_outreach_fit"] = cold_outreach_fit(profile, str(profile.get("about") or ""))
        self._store.set_user_profile(user_id, profile)
        self._links_enqueued += enqueue_profile_links(
            self._store,
            profile,
            enqueued_so_far=self._links_enqueued,
            limit=profile_link_enqueue_limit(),
        )
        about = str(profile.get("about") or "")
        return about, profile


# Back-compat alias for imports
UserBioEnricher = UserProfileEnricher


def profile_summary_for_pipeline(profile: dict[str, Any]) -> str:
    """Compact line for Go processor user_bio append."""
    if not profile:
        return ""
    parts: list[str] = []
    about = str(profile.get("about") or "").strip()
    if about:
        parts.append(about)
    flags: list[str] = []
    if profile.get("premium"):
        flags.append("premium")
    if profile.get("verified"):
        flags.append("verified")
    if profile.get("has_photo"):
        flags.append("has_photo")
    if profile.get("business"):
        flags.append("business")
    linked = str(profile.get("linked_chat_username") or "").strip()
    if linked:
        flags.append(f"linked=@{linked}")
    fit = str(profile.get("cold_outreach_fit") or "").strip()
    if fit:
        flags.append(f"outreach_fit={fit}")
    if flags:
        parts.append("profile_flags: " + ",".join(flags))
    return " | ".join(parts)


def _sender_id(sender: Any) -> int:
    from telethon.tl.types import User

    if isinstance(sender, User) and sender.id:
        return int(sender.id)
    return 0


async def _fetch_full_user_profile(client: Any, sender: Any) -> dict[str, Any]:
    from telethon.tl.functions.users import GetFullUserRequest
    from telethon.tl.types import User

    if not isinstance(sender, User):
        return {}
    try:
        full = await client(GetFullUserRequest(sender))
    except Exception as exc:
        LOG.debug("GetFullUser failed user_id=%s error=%s", sender.id, exc)
        return {}

    fu = full.full_user
    about = str(getattr(fu, "about", "") or "").strip()
    row: dict[str, Any] = {
        "user_id": int(sender.id),
        "username": (sender.username or "").strip().lstrip("@"),
        "first_name": (sender.first_name or "").strip(),
        "last_name": (sender.last_name or "").strip(),
        "about": about,
        "is_bot": bool(sender.bot),
        "premium": bool(getattr(sender, "premium", False)),
        "verified": bool(getattr(sender, "verified", False)),
        "scam": bool(getattr(sender, "scam", False)),
        "fake": bool(getattr(sender, "fake", False)),
        "has_photo": bool(getattr(sender, "photo", None)),
        "business": bool(getattr(fu, "business_work_hours", None) is not None),
        "common_chats_count": int(getattr(fu, "common_chats_count", 0) or 0),
        "linked_chat_id": int(getattr(fu, "personal_channel_id", 0) or 0),
        "linked_chat_username": "",
    }
    status = getattr(sender, "status", None)
    if status is not None:
        row["status"] = type(status).__name__
    if row["linked_chat_id"]:
        try:
            linked = await client.get_entity(row["linked_chat_id"])
            uname = getattr(linked, "username", None)
            if uname:
                row["linked_chat_username"] = str(uname).lstrip("@")
        except Exception:
            pass
    return row


def export_user_profiles_json(store: CursorStore, path: str) -> int:
    from pathlib import Path

    from .tg_graph_ids import telegram_chat_ref_from_username, telegram_person_key

    rows = store.list_all_user_profiles()
    for row in rows:
        uid = int(row.get("user_id") or 0)
        if uid <= 0:
            continue
        row["person_key"] = telegram_person_key(uid)
        via = store.profile_link_discovered_via(uid)
        if via:
            row["discovered_via"] = via
        src = str(row.get("source_chat") or "").strip()
        if src:
            row["source_chat_ref"] = telegram_chat_ref_from_username(src)
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(
        json.dumps({"profiles": rows, "count": len(rows)}, indent=2, ensure_ascii=False)
        + "\n",
        encoding="utf-8",
    )
    return len(rows)
