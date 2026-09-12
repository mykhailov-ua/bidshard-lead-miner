"""H1 geo + CIS grey market gate. Keep in sync with internal/geo/h1_block.go."""

from __future__ import annotations

import re

from .geo_heuristic import geo_heuristic_enabled, has_ua_affinity, is_ru_infrastructure_hard_stop

REJECT_CIS_GREY_MARKET = "cis_grey_market"
REJECT_GEO_BLOCK_RU_BY = "geo_block_ru_by"

_CIS_BOOKMAKER_RE = re.compile(
    r"(?i)(?:\b1win\b|\b1xbet\b|\b1хбет\b|\bmelbet\b|\bmostbet\b|\bмостбет\b|"
    r"pin-?up(?:\.ru)?|\bvavada\b|\bвавада\b)"
)
_CPA_RIP_RE = re.compile(r"(?i)cpa\.rip")
_RU_SETUP_CONTEXT_RE = re.compile(
    r"(?i)(?:\b(?:ru|rf|рф|рб)\b|россия|russia|рубл|сбер|тинькофф|\.ru\b|\+7)"
)
_RU_COUNTRY_MARKER_RE = re.compile(r"(?i)(?:\bрф\b|\bрб\b|\bRF\b|\bRU\b)")
_RU_RUB_CARD_RE = re.compile(r"(?i)(?:рублевые?\s+карт|карт[аы]\s+мир|оплата\s+руб)")
_RU_DROP_RE = re.compile(r"(?i)(?:дропы?\s+(?:рф|росси)|рф\s+дроп)")
_RU_FLAG_RE = re.compile("\U0001f1f7\U0001f1fa")  # RU regional indicator pair
_PAYMENT_INFRA_RE = re.compile(
    r"(?i)(?:сбер|тинькофф|т-банк|сбп|рубл|\+7|mail\.ru|yandex\.ru|reg\.ru)"
)
_LOCATION_ONLY_RE = re.compile(
    r"(?i)(?:\bmoscow\b|\bмосква\b|\bminsk\b|\bминск\b|\brussia\b|\bроссия\b|\bbelarus\b)"
)
_RU_HANDLE_RE = re.compile(r"(?i)(?:^|@)[a-z0-9_][a-z0-9_.]*\.ru\b")
_WW_BUYER_VOICE_RE = re.compile(
    r"(?i)(?:voluum|redtrack|binom|keitaro|self-?hosted|postback|usdt|trc20|erc20|"
    r"clickid|click\s+loss|alternative|migrat|502|nginx|mysql|clickhouse|tracker)"
)


def h1_geo_block_enabled() -> bool:
    return geo_heuristic_enabled()


def _has_ww_buyer_voice(lower: str) -> bool:
    return bool(_WW_BUYER_VOICE_RE.search(lower))


def _body(text: str, username: str = "", channel_about: str = "") -> str:
    parts = [text, username, channel_about]
    return "\n".join(p for p in parts if p).strip()


def reject_cis_grey_market(body: str) -> tuple[bool, str]:
    lower = body.lower()
    if _CIS_BOOKMAKER_RE.search(lower):
        if _has_ww_buyer_voice(lower):
            return False, ""
        return True, "cis bookmaker"
    if _CPA_RIP_RE.search(lower) and _RU_SETUP_CONTEXT_RE.search(lower):
        if _has_ww_buyer_voice(lower):
            return False, ""
        return True, "cpa.rip ru setup"
    return False, ""


def reject_h1_metadata(username: str, channel_about: str) -> str:
    meta = "\n".join(p for p in (username, channel_about) if p).strip()
    if not meta:
        return ""
    if _RU_FLAG_RE.search(meta):
        return "ru flag in profile"
    if _RU_HANDLE_RE.search(meta):
        return "ru handle"
    return ""


def reject_h1_geo_markers(body: str) -> str:
    lower = body.lower()
    if _RU_RUB_CARD_RE.search(lower):
        return "rub card"
    if _RU_DROP_RE.search(lower):
        return "ru drop"
    if _RU_COUNTRY_MARKER_RE.search(body) and not _has_ww_buyer_voice(lower):
        return "ru/by country marker"
    return ""


def _ua_location_only_hard_stop(body: str) -> bool:
    lower = body.lower()
    if not _LOCATION_ONLY_RE.search(lower):
        return False
    if _PAYMENT_INFRA_RE.search(lower) or _RU_RUB_CARD_RE.search(lower):
        return False
    return True


def h1_should_drop(
    text: str,
    username: str = "",
    *,
    channel_about: str = "",
) -> tuple[bool, str]:
    """Return (drop, reason_code). reason is cis_grey_market or geo_block_ru_by."""
    if not h1_geo_block_enabled():
        return False, ""
    body = _body(text, username, channel_about)
    if not body:
        return False, ""

    drop, detail = reject_cis_grey_market(body)
    if drop:
        return True, f"{REJECT_CIS_GREY_MARKET}: {detail}"

    if is_ru_infrastructure_hard_stop(body):
        if has_ua_affinity(body) and _ua_location_only_hard_stop(body):
            pass
        else:
            return True, f"{REJECT_GEO_BLOCK_RU_BY}: ru infrastructure"

    detail = reject_h1_metadata(username, channel_about)
    if detail:
        return True, f"{REJECT_GEO_BLOCK_RU_BY}: {detail}"

    detail = reject_h1_geo_markers(body)
    if detail:
        return True, f"{REJECT_GEO_BLOCK_RU_BY}: {detail}"

    return False, ""
