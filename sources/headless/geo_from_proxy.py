"""Infer locale/timezone from residential proxy URL username tokens (country-*, geo-*, etc.)."""

from __future__ import annotations

import re
from urllib.parse import urlparse

# ISO 3166-1 alpha-2 -> (BCP47 locale, IANA timezone). Approximate capital/timezone per country.
_COUNTRY_PROFILE: dict[str, tuple[str, str]] = {
    "us": ("en-US", "America/New_York"),
    "gb": ("en-GB", "Europe/London"),
    "uk": ("en-GB", "Europe/London"),
    "de": ("de-DE", "Europe/Berlin"),
    "fr": ("fr-FR", "Europe/Paris"),
    "es": ("es-ES", "Europe/Madrid"),
    "it": ("it-IT", "Europe/Rome"),
    "nl": ("nl-NL", "Europe/Amsterdam"),
    "pl": ("pl-PL", "Europe/Warsaw"),
    "ua": ("uk-UA", "Europe/Kyiv"),
    "ru": ("ru-RU", "Europe/Moscow"),
    "by": ("be-BY", "Europe/Minsk"),
    "kz": ("kk-KZ", "Asia/Almaty"),
    "tr": ("tr-TR", "Europe/Istanbul"),
    "br": ("pt-BR", "America/Sao_Paulo"),
    "mx": ("es-MX", "America/Mexico_City"),
    "ca": ("en-CA", "America/Toronto"),
    "au": ("en-AU", "Australia/Sydney"),
    "in": ("en-IN", "Asia/Kolkata"),
    "sg": ("en-SG", "Asia/Singapore"),
    "jp": ("ja-JP", "Asia/Tokyo"),
    "vn": ("vi-VN", "Asia/Ho_Chi_Minh"),
    "id": ("id-ID", "Asia/Jakarta"),
    "pt": ("pt-PT", "Europe/Lisbon"),
    "cz": ("cs-CZ", "Europe/Prague"),
    "ro": ("ro-RO", "Europe/Bucharest"),
    "se": ("sv-SE", "Europe/Stockholm"),
    "no": ("nb-NO", "Europe/Oslo"),
    "fi": ("fi-FI", "Europe/Helsinki"),
    "at": ("de-AT", "Europe/Vienna"),
    "ch": ("de-CH", "Europe/Zurich"),
    "ie": ("en-IE", "Europe/Dublin"),
    "gr": ("el-GR", "Europe/Athens"),
    "hu": ("hu-HU", "Europe/Budapest"),
    "bg": ("bg-BG", "Europe/Sofia"),
    "lt": ("lt-LT", "Europe/Vilnius"),
    "lv": ("lv-LV", "Europe/Riga"),
    "ee": ("et-EE", "Europe/Tallinn"),
}

_COUNTRY_TOKEN = re.compile(
    r"(?:country|geo|region)[-_]?([a-z]{2})\b|"
    r"[-_]([a-z]{2})[-_](?:session|sess|sid|pool)\b|"
    r"[-_]([a-z]{2})$",
    re.IGNORECASE,
)


def _username_from_proxy_url(proxy_url: str) -> str:
    parsed = urlparse(proxy_url.strip())
    if parsed.username:
        return parsed.username
    # user:pass@host without scheme
    if "@" in proxy_url and "://" not in proxy_url:
        return urlparse("http://" + proxy_url).username or ""
    return ""


def country_code_from_proxy_username(username: str) -> str | None:
    if not username:
        return None
    lower = username.lower()
    for match in _COUNTRY_TOKEN.finditer(lower):
        for group in match.groups():
            if group and len(group) == 2:
                return group.lower()
    # DataImpulse-style: ...-country-pl-...
    m = re.search(r"country[-_]?([a-z]{2})", lower)
    if m:
        return m.group(1).lower()
    return None


def infer_locale_timezone(proxy_url: str) -> tuple[str, str] | None:
    code = country_code_from_proxy_username(_username_from_proxy_url(proxy_url))
    if not code:
        return None
    profile = _COUNTRY_PROFILE.get(code)
    if not profile:
        return None
    return profile


def active_proxy_url() -> str | None:
    from sources.headless.proxy_list import proxy_urls_from_env, resolve_proxy_index

    urls = proxy_urls_from_env()
    if not urls:
        return None
    idx = resolve_proxy_index()
    if idx < 0:
        idx = 0
    if idx >= len(urls):
        idx = idx % len(urls)
    return urls[idx]
