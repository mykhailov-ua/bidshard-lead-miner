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

AGENCY_OUTREACH_PHRASES = (
    "i help ecommerce",
    "i help e-commerce",
    "grow through effective ads",
    "grow through effective ad",
    "verified ad accounts",
    "cloaking systems built",
    "open to connect",
    "dm me privately",
    "drop your main pain",
    "i help solve both",
    "smart tracking + automation",
    "gohighlevel",
)

AGENCY_SERVICE_OFFER_HINTS = (
    "i help",
    "i provide",
    "we offer",
    "open to connect",
    "dm me",
)

AGENCY_TOOL_PROMO_HINTS = (
    "cloaking",
    "redtrack",
    "gohighlevel",
)

BUYER_QUESTION_HINTS = (
    "resend postback",
    "updated payout",
    "duplicate postback",
    "postback failing",
    "does voluum",
    "support for tracking",
    "need alternative",
    "is there an option",
    "what needs to be done",
    "cannot edit the conversion",
    "anyone know how",
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

TRACKER_PAIN_HINTS = (
    "voluum",
    "keitaro",
    "binom",
    "redtrack",
    "postback",
    "tracker",
    "alternative",
    "clickid",
    "cloak",
    "s2s",
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

JOB_HINTS = (
    "hiring",
    "job offer",
    "open position",
    "recruiting",
    "вакансия",
    "ищем медиабайера",
)

TUTORIAL_HINTS = (
    "how to build",
    "step by step",
    "tutorial",
    "guide",
    "гайд",
    "мануал",
    "инструкция",
    "vibe cod",
    "вайбкод",
    "claude keitaro",
    "trafftok",
    "подготовили для вас",
    "мастхев",
    "скіли в claude",
)

PROGRAMMATIC_HINTS = (
    "programmatic",
    "openrtb",
    "header bidding",
    "prebid",
    "supply-side",
    "dooh",
    "pdooh",
    "brand awareness",
    "viewability",
    "programmatic guaranteed",
    "programmatic stack",
    "openrtb bidder",
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


def has_tracker_pain_signal(text: str) -> bool:
    body = _lower(text)
    return any(hint in body for hint in TRACKER_PAIN_HINTS)


def has_substance(text: str) -> bool:
    stripped = _EMAIL_RE.sub(" ", text)
    stripped = re.sub(r"(?:telegram:)?@[a-zA-Z][a-zA-Z0-9_]{3,}", " ", stripped)
    runes = sum(1 for ch in stripped if ch.isalnum())
    return runes >= MIN_MESSAGE_RUNES


def is_job_or_tutorial_noise(text: str) -> bool:
    body = _lower(text)
    if any(h in body for h in JOB_HINTS):
        return not has_tracker_pain_signal(text)
    if any(h in body for h in TUTORIAL_HINTS):
        return True
    return False


def is_programmatic_noise(text: str) -> bool:
    body = _lower(text)
    if not any(h in body for h in PROGRAMMATIC_HINTS):
        return False
    return not has_tracker_pain_signal(text)


def has_buyer_question_signal(text: str) -> bool:
    body = _lower(text)
    if any(hint in body for hint in BUYER_QUESTION_HINTS):
        return True
    if "?" not in body:
        return False
    # Question mark bait in agency broadcasts still counts as outreach noise.
    return not any(phrase in body for phrase in AGENCY_OUTREACH_PHRASES)


def is_agency_outreach_noise(text: str) -> bool:
    body = _lower(text)
    has_agency_phrase = any(phrase in body for phrase in AGENCY_OUTREACH_PHRASES)
    has_tool_promo = any(t in body for t in AGENCY_TOOL_PROMO_HINTS) and any(
        s in body for s in AGENCY_SERVICE_OFFER_HINTS
    )
    if not has_agency_phrase and not has_tool_promo:
        return False
    return not has_buyer_question_signal(text)


def should_emit_message(text: str) -> bool:
    if not prefilter_enabled():
        return True
    if is_spam_message(text):
        return False
    if is_job_or_tutorial_noise(text):
        return False
    if is_programmatic_noise(text):
        return False
    if is_agency_outreach_noise(text):
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
