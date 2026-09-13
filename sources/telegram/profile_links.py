"""Enqueue t.me / @ links found on Telegram user profiles for cold-path discovery."""

from __future__ import annotations

import logging
import os
from typing import Any

from .cursor import CursorStore
from .tglinks import extract_from_texts

LOG = logging.getLogger("telegram.profile_links")

LinkRow = tuple[str, str, str]  # kind, value, source


def profile_link_enqueue_enabled() -> bool:
    return os.environ.get("TELEGRAM_PROFILE_LINK_ENQUEUE", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def profile_link_enqueue_limit() -> int:
    raw = os.environ.get("TELEGRAM_PROFILE_LINK_ENQUEUE_LIMIT", "50").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 50


def profile_link_enqueue_min_fit() -> str:
    """yes | maybe | no — only enqueue when outreach_fit is at least this tier."""
    raw = os.environ.get("TELEGRAM_PROFILE_LINK_MIN_FIT", "maybe").strip().lower()
    if raw in ("yes", "maybe", "no"):
        return raw
    return "maybe"


def _fit_allows(enqueue_min: str, outreach_fit: str) -> bool:
    fit = (outreach_fit or "no").strip().lower()
    if enqueue_min == "no":
        return True
    if enqueue_min == "maybe":
        return fit in ("yes", "maybe")
    return fit == "yes"


def links_from_profile(profile: dict[str, Any]) -> list[LinkRow]:
    out: list[LinkRow] = []
    seen: set[tuple[str, str]] = set()

    def add(kind: str, value: str, source: str) -> None:
        value = value.strip()
        if not value:
            return
        key = (kind, value.lower() if kind == "username" else value)
        if key in seen:
            return
        seen.add(key)
        out.append((kind, value, source))

    about = str(profile.get("about") or "")
    if about:
        handles, invites = extract_from_texts([about])
        for h in handles:
            add("username", h, "about")
        for inv in invites:
            add("invite_hash", inv, "about")

    linked = str(profile.get("linked_chat_username") or "").strip().lstrip("@")
    if linked:
        add("username", linked.lower(), "personal_channel")

    return out


def enqueue_profile_links(
    store: CursorStore,
    profile: dict[str, Any],
    *,
    enqueued_so_far: int,
    limit: int,
) -> int:
    """Insert pending profile links; returns number newly queued this call."""
    if limit <= 0 or enqueued_so_far >= limit:
        return 0
    if not profile_link_enqueue_enabled():
        return 0

    user_id = int(profile.get("user_id") or 0)
    if user_id <= 0:
        return 0

    outreach_fit = str(profile.get("cold_outreach_fit") or "no")
    if not _fit_allows(profile_link_enqueue_min_fit(), outreach_fit):
        return 0

    user_username = str(profile.get("username") or "").strip().lstrip("@")
    added = 0
    for kind, value, source in links_from_profile(profile):
        if enqueued_so_far + added >= limit:
            break
        if store.channel_known_link(kind, value):
            continue
        if store.enqueue_profile_link(
            user_id,
            user_username,
            kind,
            value,
            source,
            outreach_fit,
        ):
            added += 1
    if added:
        LOG.info(
            "profile links enqueued user_id=%s added=%d outreach_fit=%s",
            user_id,
            added,
            outreach_fit,
        )
    return added
