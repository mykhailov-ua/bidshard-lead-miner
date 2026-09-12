"""H2 WW/UA chat pool curation. Keep aligned with internal/filter/h2_pool.go."""

from __future__ import annotations

H2_CIS_ARBITRAGE_HANDLES = frozenset(
    {
        "maximaffiliate",
        "zuevaff",
        "zuevchatcpa",
        "traffic_trickster",
        "bakalov_info",
        "seodroplab",
        "po_ushi_v_gambling",
        "frbs_team",
        "g_gate_media",
        "affwriter",
        "m2ensenchannel",
        "cpalenta",
        "cpa_lenta",
    }
)

H2_CIS_ARBITRAGE_SUBSTRINGS = (
    "zuevaff",
    "maximaff",
    "cpalent",
    "cparip",
    "arbitrage_ru",
    "_ru_arb",
    "ru_arbitrage",
)

_WW_BUYER_VOICE = (
    "voluum",
    "redtrack",
    "binom",
    "keitaro",
    "self-hosted",
    "postback",
    "usdt",
    "trc20",
    "tracker",
    "alternative",
    "clickhouse",
)


def _cis_ru_setup_context(lower: str) -> bool:
    return any(
        x in lower
        for x in (
            "россия",
            "russia",
            " ru ",
            " рф ",
            ".ru",
            "+7",
            "сбер",
            "рубл",
        )
    )


def _has_ww_buyer_voice(lower: str) -> bool:
    return any(h in lower for h in _WW_BUYER_VOICE)


def reject_h2_cis_pool(username: str, texts: list[str] | None = None) -> tuple[bool, str]:
    user = username.strip().lstrip("@").lower()
    if user:
        if user in H2_CIS_ARBITRAGE_HANDLES:
            return True, "h2_cis_arbitrage"
        for sub in H2_CIS_ARBITRAGE_SUBSTRINGS:
            if sub in user:
                return True, "h2_cis_username"
    parts = [str(t).strip() for t in (texts or []) if str(t).strip()]
    blob = " ".join(parts).lower()
    if "cpa.rip" in blob and _cis_ru_setup_context(blob) and not _has_ww_buyer_voice(blob):
        return True, "h2_cpa_rip_ru"
    return False, ""
