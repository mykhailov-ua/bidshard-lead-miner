"""H5 vendor support chat policy: buyer voice in seller-branded channels."""

from __future__ import annotations

from .pain import message_has_tracker_pain
from .prefilter import has_buyer_question_signal, is_instant_drop_message


def vendor_support_emit_allowed(text: str, *, channel_role: str = "") -> bool:
    """Allow buyer pain in vendor_support; still drop seller promos."""
    role = (channel_role or "").strip().lower()
    if role != "vendor_support":
        return True
    if is_instant_drop_message(text):
        return False
    return message_has_tracker_pain(text) or has_buyer_question_signal(text)
