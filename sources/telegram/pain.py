from __future__ import annotations

from .prefilter import TRACKER_PAIN_HINTS


def message_has_pain(text: str) -> bool:
    lower = (text or "").lower()
    if not lower:
        return False
    return any(hint in lower for hint in TRACKER_PAIN_HINTS)
