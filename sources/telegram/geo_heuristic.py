from __future__ import annotations

import os
import re

# Mirror internal/geo/filter.go blocked TLDs and bio signals.
_BLOCKED_TLDS = frozenset({"ru", "by", "su", "рф", "бел"})

_BIO_REJECT_RE = re.compile(
    r"(?i)(europe/moscow|\bmoscow\b|\bminsk\b|\brussia\b|\bbelarus\b|\bроссия\b|\bбеларусь\b)"
)

_CYRILLIC_RUN = re.compile(r"[а-яё]{8,}")


def geo_heuristic_enabled() -> bool:
    return os.environ.get("TELEGRAM_GEO_HEURISTIC", "true").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def is_blocked_web_tld(host: str) -> bool:
    host = host.lower().strip()
    host = host.removeprefix("www.")
    parts = host.split(".")
    if len(parts) < 2:
        return False
    return parts[-1] in _BLOCKED_TLDS


def channel_geo_reject(texts: list[str]) -> bool:
    """Return True when channel title/about looks RU/BY (skip discover/scrape)."""
    if not geo_heuristic_enabled():
        return False
    blob = "\n".join(t for t in texts if t).strip()
    if not blob:
        return False
    if _BIO_REJECT_RE.search(blob):
        return True
    cyr = len(_CYRILLIC_RUN.findall(blob))
    latin = sum(1 for ch in blob if "a" <= ch.lower() <= "z")
    return bool(cyr >= 2 and latin < 12)
