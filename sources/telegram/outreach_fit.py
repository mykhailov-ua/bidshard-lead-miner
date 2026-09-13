"""Heuristic cold-outreach fit from Telegram user profile (no Gemini)."""

from __future__ import annotations

import re
from typing import Any

_BUYER_RE = re.compile(
    r"(?i)(media\s+buyer|mediabuyer|traffic\s+manager|affiliate\s+manager|"
    r"team\s+lead|head\s+of\s+acquisition|ppc|fb\s+buyer|google\s+ads)"
)
_VENDOR_RE = re.compile(
    r"(?i)(affiliate\s+network|cpa\s+network|account\s+manager|am\s+@|"
    r"support\s+team|official\s+channel)"
)


def cold_outreach_fit(profile: dict[str, Any], bio: str = "") -> str:
    """Return yes | maybe | no."""
    text = " ".join(
        [
            bio,
            str(profile.get("about") or ""),
            str(profile.get("first_name") or ""),
            str(profile.get("last_name") or ""),
        ]
    ).strip()
    if not text and not profile.get("username"):
        return "no"
    if profile.get("is_bot"):
        return "no"
    if _VENDOR_RE.search(text):
        return "no"
    if _BUYER_RE.search(text):
        return "yes"
    if profile.get("premium") or profile.get("has_photo"):
        return "maybe"
    if bio.strip():
        return "maybe"
    return "no"
