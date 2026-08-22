from __future__ import annotations

import os
import re

SPAM_PHRASES = (
    "join our channel",
    "subscribe to",
    "subscribе to",
    "via @",
    "forwarded from",
    "dm me for course",
    "free course",
    "affiliate course",
    "mentorship program",
    "signal group",
    "vip signals",
    "paid group",
    "click the link below",
    "limited slots",
)

PAIN_HINTS = (
    "voluum",
    "keitaro",
    "binom",
    "redtrack",
    "postback",
    "tracker",
    "alternative",
    "self-hosted",
    "self hosted",
    "migration",
    "media buy",
    "igaming",
    "affiliate",
    "s2s",
    "clickid",
    "ftd",
    "arbitrage",
)

CHANNEL_SPAM_HINTS = (
    "signal",
    "signals",
    "course",
    "mentorship",
    "vip group",
    "paid tips",
    "casino tips",
)

CHANNEL_POSITIVE_HINTS = (
    "affiliate",
    "igaming",
    "media buy",
    "arbitrage",
    "tracker",
    "acquisition",
    "performance marketing",
    "cpa",
)

MIN_MESSAGE_RUNES = 40
_EMAIL_RE = re.compile(r"[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}")


def prefilter_enabled() -> bool:
    return os.environ.get("TELEGRAM_PREFILTER", "true").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def _lower(text: str) -> str:
    return text.strip().lower()


def is_spam_message(text: str) -> bool:
    body = _lower(text)
    if not body:
        return True
    return any(phrase in body for phrase in SPAM_PHRASES)


def has_pain_signal(text: str) -> bool:
    body = _lower(text)
    return any(hint in body for hint in PAIN_HINTS)


def has_substance(text: str) -> bool:
    stripped = _EMAIL_RE.sub(" ", text)
    stripped = re.sub(r"(?:telegram:)?@[a-zA-Z][a-zA-Z0-9_]{3,}", " ", stripped)
    runes = sum(1 for ch in stripped if ch.isalnum())
    return runes >= MIN_MESSAGE_RUNES


def should_emit_message(text: str) -> bool:
    if not prefilter_enabled():
        return True
    if is_spam_message(text):
        return False
    # Emit on pain keywords even when message is short; otherwise require MIN_MESSAGE_RUNES substance.
    if has_pain_signal(text):
        return True
    return has_substance(text)


def channel_icp_relevant(texts: list[str]) -> bool:
    if not prefilter_enabled():
        return True
    blob = "\n".join(t for t in texts if t).lower()
    if not blob.strip():
        return False
    spam_hits = sum(1 for h in CHANNEL_SPAM_HINTS if h in blob)
    pos_hits = sum(1 for h in CHANNEL_POSITIVE_HINTS if h in blob)
    # Reject signal-spam channels unless at least one affiliate/tracker hint is present.
    if spam_hits >= 2 and pos_hits == 0:
        return False
    return pos_hits > 0 or has_pain_signal(blob)
