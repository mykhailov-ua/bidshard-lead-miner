"""H4 bulletproof hosting incident + tracker stack signals."""

from __future__ import annotations

import re

_INCIDENT_RE = [
    re.compile(r"(?i)\b502\b"),
    re.compile(r"(?i)\b503\b"),
    re.compile(r"(?i)\b504\b"),
    re.compile(r"(?i)\bupstream\b"),
    re.compile(r"(?i)\babuse\s+suspend"),
    re.compile(r"(?i)\bhost(?:er|ing)\b.{0,40}\b(?:block|suspend|down)\b"),
    re.compile(r"(?i)\bredirect\b.{0,30}\bdown\b"),
    re.compile(r"(?i)\bnginx\b.{0,40}\b(?:timeout|error|down)\b"),
]

_TRACKER_HINTS = (
    "keitaro",
    "binom",
    "voluum",
    "redtrack",
    "tracker",
    "postback",
    "clickid",
)

_HOSTING_CHANNEL_HINTS = (
    "alexhost",
    "pq.hosting",
    "pq hosting",
    "zomro",
    "friendhosting",
    "bulletproof",
    "offshore hosting",
)


def has_hosting_incident_pain(text: str) -> bool:
    body = (text or "").strip()
    if not body:
        return False
    if not any(rx.search(body) for rx in _INCIDENT_RE):
        return False
    lower = body.lower()
    return any(h in lower for h in _TRACKER_HINTS)


def is_hosting_channel_hint(username: str = "", title: str = "", query: str = "") -> bool:
    combined = f"{username} {title} {query}".lower()
    return any(h in combined for h in _HOSTING_CHANNEL_HINTS)
