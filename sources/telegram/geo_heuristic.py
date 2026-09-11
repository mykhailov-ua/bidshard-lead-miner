from __future__ import annotations

import os
import re

# Ingress geo verdicts (channel / message). Language is not jurisdiction.
REJECT_RU = "REJECT_RU"
ACCEPT_UA = "ACCEPT_UA"
NEUTRAL_CIS = "NEUTRAL_CIS"

_BLOCKED_TLDS = frozenset({"ru", "by", "su", "рф", "бел"})

# Hard-stop: RU/BY infrastructure, payments, hosting, jurisdiction markers.
_RU_HARD_STOP_RE = re.compile(
    r"(?i)("
    r"\.(ru|by|su|рф|бел)\b|"
    r"reg\.ru|beget\.|timeweb\.|"
    r"сбер(?:банк)?|тинькофф|т-банк|"
    r"сбп|"
    r"юmoney|qiwi|юмани|"
    r"карта мир|"
    r"оплата руб|рубл|"
    r"europe/moscow|\bmoscow\b|\bмосква\b|\bпитер\b|\bспб\b|"
    r"санкт-петербург|\bminsk\b|\bминск\b|\brussia\b|\bроссия\b|\bbelarus\b|\bбеларусь\b"
    r")"
)

# Positive UA affinity: Russian-language teams with Ukrainian digital footprint.
_UA_AFFINITY_RE = re.compile(
    r"(?i)("
    r"киев|київ|днепр|дніпро|одесса|одеса|львов|львів|"
    r"подол|"
    r"\+380|"
    r"монобанк|monobank|privat24|приват|"
    r"remote ua|mac kyiv|sempro"
    r")"
)


def geo_heuristic_enabled() -> bool:
    return os.environ.get("TELEGRAM_GEO_HEURISTIC", "true").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def is_blocked_web_tld(host: str) -> bool:
    host = host.lower().strip()
    host = host.removeprefix("www.")
    parts = host.split(".")
    if len(parts) < 2:
        return False
    return parts[-1] in _BLOCKED_TLDS


def channel_geo_texts(chat_name: str, about: str, entity: object | None = None) -> list[str]:
    texts = [about, chat_name]
    if entity is not None:
        title = getattr(entity, "title", "")
        if title:
            texts.append(str(title))
    return texts


def evaluate_channel_geo(texts: list[str], links: list[str] | None = None) -> str:
    """Classify channel geo at ingress. Cyrillic alone is never a reject."""
    if not geo_heuristic_enabled():
        return NEUTRAL_CIS
    blob = "\n".join(t for t in texts if t).strip()
    if links:
        blob = f"{blob}\n{' '.join(links)}".strip()
    if not blob:
        return NEUTRAL_CIS
    if _RU_HARD_STOP_RE.search(blob):
        return REJECT_RU
    if _UA_AFFINITY_RE.search(blob):
        return ACCEPT_UA
    return NEUTRAL_CIS


def channel_geo_reject(texts: list[str], links: list[str] | None = None) -> bool:
    """True only for unconditional RU/BY infrastructure match."""
    return evaluate_channel_geo(texts, links) == REJECT_RU


def is_ru_infrastructure_hard_stop(text: str) -> bool:
    if not geo_heuristic_enabled():
        return False
    body = text.strip()
    if not body:
        return False
    return bool(_RU_HARD_STOP_RE.search(body))


def has_ua_affinity(text: str) -> bool:
    body = text.strip()
    if not body:
        return False
    return bool(_UA_AFFINITY_RE.search(body))
