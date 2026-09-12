"""H8 payment / vertical gate. Keep in sync with internal/filter/h8_payment.go."""

from __future__ import annotations

import re

from .pain import message_has_tracker_pain
from .prefilter import has_buyer_question_signal

_FOP_TOV_RE = re.compile(
    r"(?i)(?:\bфоп\b|\bтов\b|\bfop\b|\btov\b|гривн|grivna|\buah\b|privatbank|monobank)"
)
_INFOBIZ_RE = re.compile(
    r"(?i)(?:инфобиз|infobiz|white\s+goods|курсы?\s+по\s+арбитраж|course\s+seller|"
    r"how\s+to\s+buy\s+domain|купить\s+домен)"
)
_CRYPTO_PAYOUT_RE = re.compile(
    r"(?i)(?:\busdt\b|\btrc20\b|\berc20\b|crypto\s+payout|crypto\s+settlement|"
    r"weekly\s+usdt|daily\s+usdt)"
)
_BUYER_VOICE_RE = re.compile(
    r"(?i)(?:voluum|keitaro|binom|redtrack|postback|tracker|usdt|trc20|clickid|"
    r"self-hosted|502|nginx|mysql)"
)


def reject_h8_payment_vertical(text: str, title: str = "") -> tuple[bool, str]:
    combined = f"{title} {text}".strip().lower()
    if not combined:
        return False, ""
    if _FOP_TOV_RE.search(combined) and not _BUYER_VOICE_RE.search(combined):
        return True, "h8 fop/tov/grivna"
    if _INFOBIZ_RE.search(combined) and not message_has_tracker_pain(text):
        return True, "h8 infobiz/off-vertical"
    return False, ""


def reject_crypto_payout_only(text: str) -> tuple[bool, str]:
    body = (text or "").strip()
    if not body:
        return False, ""
    if not _CRYPTO_PAYOUT_RE.search(body):
        return False, ""
    if message_has_tracker_pain(body):
        return False, ""
    if has_buyer_question_signal(body):
        return False, ""
    return True, "h8 crypto payout without pain"
