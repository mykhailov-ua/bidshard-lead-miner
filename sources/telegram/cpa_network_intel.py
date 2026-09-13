"""CPA network channel policy: scrape for team OSINT, not network-as-buyer."""

from __future__ import annotations

import re

from .pain import message_has_tracker_pain
from .prefilter import has_buyer_question_signal, is_instant_drop_message

_TEAM_HIRING_RE = re.compile(
    r"(?i)("
    r"media\s+buyer|mediabuyer|медиабайер|"
    r"team\s+lead\s+media|head\s+of\s+acquisition|"
    r"traffic\s+manager|buyer\s+fb|buyer\s+google"
    r").*("
    r"hiring|recruiting|вакансия|ищем|open\s+position|job\s+offer|"
    r"набор\s+в\s+команду|#buyer|#head"
    r")|("
    r"hiring|recruiting|вакансия|ищем|open\s+position|job\s+offer"
    r").*("
    r"media\s+buyer|mediabuyer|медиабайер|team\s+lead\s+media"
    r")"
)

_NETWORK_SUPPLY_RE = re.compile(
    r"(?i)("
    r"cpa\s+network|affiliate\s+network|affiliate\s+program|"
    r"join\s+our\s+network|register\s+as\s+affiliate|"
    r"high\s+converting\s+offers|we\s+have\s+offers|"
    r"looking\s+for\s+affiliates|looking\s+for\s+publishers|"
    r"affiliate\s+manager|aff\s+manager|partnership\s+manager"
    r")"
)


def is_media_buying_team_hiring(text: str) -> bool:
    body = text.strip()
    if not body:
        return False
    return bool(_TEAM_HIRING_RE.search(body))


def is_cpa_network_supply_promo(text: str) -> bool:
    body = text.strip()
    if not body:
        return False
    if not _NETWORK_SUPPLY_RE.search(body):
        return False
    if is_media_buying_team_hiring(body):
        return False
    if message_has_tracker_pain(body) or has_buyer_question_signal(body):
        return False
    return True


def is_cpa_network_channel_hint(username: str = "", title: str = "", query: str = "") -> bool:
    blob = f"{username} {title} {query}".lower()
    if not blob.strip():
        return False
    hints = (
        "cpa network",
        "affiliate network",
        "affiliate partners",
        "affiliate program",
        "partner channel",
        "drcash",
        "maxbounty",
        "cpamatica",
        "affise",
        "partners africa",
        "affiliate duniya",
    )
    return any(h in blob for h in hints)


def supply_emit_allowed(text: str, *, channel_role: str = "") -> bool:
    """Allow buyer/team signals in CPA network channels; drop AM supply promos."""
    role = (channel_role or "").strip().lower()
    if role != "supply":
        return True
    if is_instant_drop_message(text):
        return False
    if is_media_buying_team_hiring(text):
        return True
    if message_has_tracker_pain(text) or has_buyer_question_signal(text):
        return True
    if is_cpa_network_supply_promo(text):
        return False
    return False
