from __future__ import annotations

import re

from .hosting_incident import has_hosting_incident_pain
from .pain_taxonomy import classify_bidshard_pain
from .pwa_pain import has_pwa_pain_signal
from .prefilter import (
    TRACKER_PAIN_HINTS,
    has_buyer_question_signal,
    has_crypto_gray_icp_signal,
    has_tracker_pain_signal,
)

# Operational pain beyond bare tracker name (M3 AND-gate second leg).
OPERATIONAL_PAIN_HINTS = (
    "ошибка",
    "упал",
    "упала",
    "oom",
    "502",
    "503",
    "504",
    "база",
    "database",
    "тормозит",
    "failing",
    "failed",
    "fail",
    "crash",
    "crashed",
    "timeout",
    "timed out",
    "migration",
    "migrate",
    "broken",
    "not working",
    "doesn't work",
    "does not work",
    "не работает",
    "сломал",
    "сломался",
    "падает",
    "discrepancy",
    "mismatch",
    "upstream",
    "nginx",
    "отвалился",
    "expensive",
    "overprice",
    "overpriced",
    "too expensive",
    "shaves",
    "shaving",
    "scrubbing",
    "admin",
    "кликов",
)

_COMMERCIAL_PAIN_RE = [
    re.compile(r"(?i)\b(?:recommend|suggest|advise)\w*\s+(?:me\s+)?(?:a\s+)?tracker\b"),
    re.compile(r"(?i)\b(?:which|what)\s+tracker\b"),
    re.compile(r"(?i)\b(?:need|looking for)\s+(?:a\s+)?tracker\b"),
    re.compile(
        r"(?i)\balternative\s+to\s+(?:keitaro|binom|voluum|redtrack|clickflare|bemob)"
    ),
    re.compile(r"(?i)\b(?:keitaro|binom|voluum|redtrack)\s+alternative\b"),
    re.compile(r"(?i)postback\s+(?:fail(?:ed|ing|s)?|broken|not\s+work(?:ing)?|down)"),
    re.compile(
        r"(?i)\bmigrat(?:e|ing|ion)\s+(?:from|to)\s+(?:keitaro|binom|voluum|redtrack)"
    ),
    re.compile(r"(?i)\bswitching\s+from\s+(?:keitaro|binom|voluum|redtrack)"),
    re.compile(r"(?i)посоветуйте\s+трекер"),
    re.compile(r"(?i)какой\s+трекер\s+(?:взять|выбрать|лучше)"),
    re.compile(r"(?i)альтернатива\s+(?:keitaro|binom|voluum|redtrack)"),
    re.compile(r"(?i)не\s+трекает\s+клик"),
    re.compile(r"(?i)отвалился\s+постбек"),
    re.compile(r"(?i)\b(?:pp|network)\s+shav(?:e|es|ing)\s+leads\b"),
    re.compile(r"(?i)\bhow\s+to\s+prove\b.{0,40}\b(?:shav|scrub|discrep)"),
    re.compile(r"(?i)\bв\s+трекере\s+\d+.{0,40}\bпартнерк"),
]

_VOLUME_SCALE_RE = [
    re.compile(r"(?i)\b\d{2,}\s*k\s*(?:click|clicks)\b"),
    re.compile(r"(?i)\b\d+\s*(?:к|k)\s*клик"),
]


def has_operational_pain_signal(text: str) -> bool:
    lower = (text or "").lower()
    if not lower:
        return False
    return any(hint in lower for hint in OPERATIONAL_PAIN_HINTS)


def has_commercial_pain_intent(text: str) -> bool:
    body = (text or "").strip()
    if not body:
        return False
    return any(rx.search(body) for rx in _COMMERCIAL_PAIN_RE)


def has_volume_scale_signal(text: str) -> bool:
    body = (text or "").strip()
    if not body:
        return False
    return any(rx.search(body) for rx in _VOLUME_SCALE_RE)


def has_tracker_or_scale_leg(text: str) -> bool:
    if has_tracker_pain_signal(text):
        return True
    bucket = classify_bidshard_pain(text)
    if bucket is not None:
        return True
    return has_volume_scale_signal(text)


def message_has_tracker_pain(text: str) -> bool:
    """Tracker/cloak + operational pain, or crypto-gray ICP AND-gate (M3/M11)."""
    if not (text or "").strip():
        return False
    if has_crypto_gray_icp_signal(text):
        return True
    if has_pwa_pain_signal(text) or has_hosting_incident_pain(text):
        return True
    bucket = classify_bidshard_pain(text)
    if bucket is not None and bucket.passes_m3_bucket_gate():
        return True
    if not has_tracker_or_scale_leg(text):
        return False
    return (
        has_operational_pain_signal(text)
        or has_commercial_pain_intent(text)
        or has_volume_scale_signal(text)
    )


def message_has_pain(text: str) -> bool:
    return message_has_tracker_pain(text)


def is_channel_broadcast_alert_skip(
    chat_type: str, reply_to_message_id: int, text: str
) -> bool:
    """Mirror Go TelegramChannelBroadcastReject for pain alerts."""
    kind = (chat_type or "").strip().lower()
    if kind in ("", "group", "supergroup"):
        return False
    if kind != "channel":
        return False
    if reply_to_message_id > 0:
        return False
    if has_commercial_pain_intent(text) or has_buyer_question_signal(text):
        return False
    if message_has_tracker_pain(text):
        return False
    return True
