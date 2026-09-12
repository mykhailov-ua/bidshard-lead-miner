"""H3 PWA / iOS rent client tracker pain signals."""

from __future__ import annotations

import re

_PWA_PAIN_RE = [
    re.compile(r"(?i)\bpostback\s+from\s+app\b"),
    re.compile(r"(?i)\bwebview\s+stuck\b"),
    re.compile(r"(?i)\bsub[_-]?id\b.{0,40}\b(?:keitaro|binom|voluum|tracker)\b"),
    re.compile(r"(?i)\b(?:keitaro|binom|voluum|tracker)\b.{0,40}\bsub[_-]?id\b"),
    re.compile(r"(?i)\bredirect\s+delay\b"),
    re.compile(r"(?i)\bs2s\s+from\s+pwa\b"),
    re.compile(r"(?i)\bpwa\b.{0,40}\b(?:postback|tracker|keitaro)\b"),
]

_PWA_CHANNEL_HINTS = (
    "pwa.group",
    "pwa.market",
    "irent",
    "td apps",
    "pwa rent",
    "ios rent",
    "webview",
)


def has_pwa_pain_signal(text: str) -> bool:
    body = (text or "").strip()
    if not body:
        return False
    if any(rx.search(body) for rx in _PWA_PAIN_RE):
        return True
    lower = body.lower()
    if "pwa" not in lower and "webview" not in lower:
        return False
    return any(
        hint in lower
        for hint in ("postback", "keitaro", "binom", "voluum", "tracker", "sub_id", "subid", "s2s")
    )


def is_pwa_channel_hint(username: str = "", title: str = "", query: str = "") -> bool:
    combined = f"{username} {title} {query}".lower()
    if not combined.strip():
        return False
    return any(h in combined for h in _PWA_CHANNEL_HINTS) or "pwa" in combined
