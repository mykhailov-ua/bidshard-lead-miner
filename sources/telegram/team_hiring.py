"""Team hiring detection and company hint extraction for cold outreach."""

from __future__ import annotations

import re

from .cpa_network_intel import is_media_buying_team_hiring

_COMPANY_PATTERNS = [
    re.compile(r"(?i)\b(?:at|@)\s+([A-Za-z0-9][A-Za-z0-9 &_\-.]{2,48})\b"),
    re.compile(r"(?i)\b([A-Za-z0-9][A-Za-z0-9 &_\-.]{2,40})\s+(?:is hiring|hiring|recruiting)\b"),
    re.compile(r"(?i)\b(?:join|welcome to)\s+([A-Za-z0-9][A-Za-z0-9 &_\-.]{2,40})\b"),
    re.compile(r"(?i)#([a-z][a-z0-9_]{2,32})\s+(?:hiring|team|buyer)"),
]


def is_telegram_cold_team_post(text: str) -> bool:
    return is_media_buying_team_hiring(text)


def extract_company_hint(text: str) -> str:
    body = (text or "").strip()
    if not body:
        return ""
    for pat in _COMPANY_PATTERNS:
        m = pat.search(body)
        if not m:
            continue
        name = (m.group(1) or "").strip(" .,-")
        if name.lower().endswith(" is"):
            name = name[:-3].strip()
        if len(name) < 3:
            continue
        lower = name.lower()
        if lower in ("we", "our", "the", "team", "media", "buyer"):
            continue
        return name
    return ""
