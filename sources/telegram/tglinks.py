from __future__ import annotations

import logging
import re
import urllib.error
import urllib.request
from collections.abc import Callable
from contextlib import closing
from http.client import HTTPResponse
from typing import cast

LOG = logging.getLogger("telegram.tglinks")

_TME_PUBLIC = re.compile(
    r"(?i)(?:https?://)?(?:t\.me|telegram\.me)/([a-zA-Z][a-zA-Z0-9_]{4,})"
)
_TME_INVITE = re.compile(
    r"(?i)(?:https?://)?(?:t\.me|telegram\.me)/\+([A-Za-z0-9_-]{10,})"
)
_JOINCHAT = re.compile(
    r"(?i)(?:https?://)?(?:t\.me|telegram\.me)/joinchat/([A-Za-z0-9_-]+)"
)
_AT_HANDLE = re.compile(r"(?:^|[^\w])@([a-zA-Z][a-zA-Z0-9_]{4,})")

_BLOCKED_HANDLES = frozenset(
    {"telegram", "support", "share", "addstickers", "joinchat", "s", "c", "iv"}
)


def _blocked_handle(username: str) -> bool:
    u = username.lower()
    if u in _BLOCKED_HANDLES:
        return True
    return u.endswith("bot")


def telegram_handles(text: str) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for pattern in (_TME_PUBLIC, _AT_HANDLE):
        for match in pattern.finditer(text):
            u = match.group(1).lower()
            if _blocked_handle(u):
                continue
            if u not in seen:
                seen.add(u)
                out.append(u)
    return out


def telegram_invite_hashes(text: str) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for pattern in (_TME_INVITE, _JOINCHAT):
        for match in pattern.finditer(text):
            h = match.group(1)
            if h not in seen:
                seen.add(h)
                out.append(h)
    return out


def extract_from_texts(texts: list[str]) -> tuple[list[str], list[str]]:
    handles: list[str] = []
    invites: list[str] = []
    seen_h: set[str] = set()
    seen_i: set[str] = set()
    for text in texts:
        if not text:
            continue
        for u in telegram_handles(text):
            if u not in seen_h:
                seen_h.add(u)
                handles.append(u)
        for h in telegram_invite_hashes(text):
            if h not in seen_i:
                seen_i.add(h)
                invites.append(h)
    return handles, invites


_BLOCKED_WEB_HOSTS = frozenset(
    {
        "t.me",
        "telegram.me",
        "telegram.org",
        "instagram.com",
        "facebook.com",
        "fb.com",
        "twitter.com",
        "x.com",
        "youtube.com",
        "youtu.be",
        "google.com",
        "google.ru",
        "google.de",
        "google.fr",
        "google.es",
        "google.it",
        "google.ca",
        "google.com.au",
        "google.co.jp",
        "google.co.in",
        "google.com.br",
        "chatgpt.com",
        "openai.com",
        "linkedin.com",
        "bit.ly",
        "tiktok.com",
        "discord.gg",
        "discord.com",
        "wa.me",
        "whatsapp.com",
        "forms.gle",
        "goo.gl",
        "linktr.ee",
        "taplink.cc",
        "vk.com",
        "ok.ru",
        "medium.com",
        "github.com",
        "notion.site",
        "docs.google.com",
        "reuters.com",
        "gov.uk",
        "legislation.gov.uk",
        "gamblingcommission.gov.uk",
        "niesr.ac.uk",
        "icann.org",
        "cloudflare.com",
        "sentry.io",
        "google.co.uk",
        "googleapis.com",
        "gstatic.com",
        "googleusercontent.com",
        "site.com",
        "magiceden.io",
        "jup.ag",
        "raydium.io",
        "knowyourmeme.com",
        "pump.fun",
        "collab.land",
        "telegra.ph",
        "teletype.in",
        "bing.com",
        "yahoo.com",
        "wikipedia.org",
        "reddit.com",
        "old.reddit.com",
        "amazon.com",
        "apple.com",
        "microsoft.com",
        "binance.com",
    }
)

_SHORTENER_HOSTS = frozenset(
    {
        "bit.ly",
        "goo.gl",
        "forms.gle",
        "t.co",
        "tinyurl.com",
        "is.gd",
        "buff.ly",
    }
)

_URL = re.compile(r"https?://[^\s<>\"']+")
_BARE_HOST = re.compile(
    r"(?i)(?:^|(?<![\w-]))((?:[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,})(?:[^\w.-]|$)"
)


def _normalize_host(host: str) -> str:
    host = host.lower().strip()
    host = host.removeprefix("www.")
    if ":" in host:
        host = host.split(":", 1)[0]
    return host


_FILE_EXT_HOST = re.compile(
    r"\.(jpg|jpeg|png|gif|webp|svg|pdf|txt|html|htm|php|css|js|json|xml|woff2?)$",
    re.IGNORECASE,
)
_INVALID_TLD = frozenset(
    {
        "gram",
        "ggate",
        "dxb",
        "txt",
        "html",
        "htm",
        "jpg",
        "jpeg",
        "png",
        "gif",
        "php",
        "pdf",
        "local",
        "bot",
    }
)
# RU/BY country-code TLDs (mirror internal/geo/filter.go).
_BLOCKED_GEO_TLD = frozenset({"ru", "by", "su", "рф", "бел"})


def is_valid_web_host(host: str) -> bool:
    host = _normalize_host(host)
    if not host or _blocked_host(host):
        return False
    if _FILE_EXT_HOST.search(host):
        return False
    labels = host.split(".")
    if len(labels) < 2:
        return False
    tld = labels[-1]
    if len(tld) < 2 or len(tld) > 24 or not tld.isalpha():
        return False
    if tld in _INVALID_TLD:
        return False
    if tld in _BLOCKED_GEO_TLD:
        return False
    return all(label for label in labels)


def _blocked_host(host: str) -> bool:
    host = _normalize_host(host)
    if not host or host.endswith((".bot", ".local")):
        return True
    if _is_gov_host(host):
        return True
    if _is_google_host(host):
        return True
    if host in _BLOCKED_WEB_HOSTS:
        return True
    return any(host.endswith("." + blocked) for blocked in _BLOCKED_WEB_HOSTS)


def _is_gov_host(host: str) -> bool:
    labels = host.split(".")
    if not labels:
        return False
    if labels[-1] == "gov":
        return True
    if len(labels) >= 2 and labels[-2] == "gov":
        return True
    return any(label.endswith(".gov") or label == "gov" for label in labels)


def _is_google_host(host: str) -> bool:
    labels = host.split(".")
    if labels and labels[0] == "google":
        return True
    return any(label == "google" for label in labels)


def resolve_redirect_url(
    url: str,
    timeout: float = 10.0,
    opener: Callable[..., object] | None = None,
) -> str | None:
    """Follow redirects for short links and return the final URL."""
    try:
        req = urllib.request.Request(
            url,
            headers={"User-Agent": "Mozilla/5.0 (compatible; BidShardParser/1.0)"},
        )
        with closing(
            cast(
                HTTPResponse,
                opener(req)
                if opener is not None
                else urllib.request.urlopen(req, timeout=timeout),
            )
        ) as resp:
            final = getattr(resp, "url", None) or url
            return str(final)
    except (urllib.error.URLError, OSError, ValueError) as exc:
        LOG.debug("short url resolve failed url=%s error=%s", url, exc)
        return None


def _host_from_url(raw: str) -> str:
    host = raw
    if "://" in raw:
        host = raw.split("://", 1)[1]
    host = host.split("/", 1)[0]
    return _normalize_host(host)


def web_domains(text: str, resolve_short: bool = True) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for match in _URL.finditer(text):
        raw = match.group(0)
        if resolve_short:
            host = _host_from_url(raw)
            # Resolve bit.ly/goo.gl to final host before blacklist check.
            if host in _SHORTENER_HOSTS:
                resolved = resolve_redirect_url(raw)
                if resolved:
                    raw = resolved
        host = _host_from_url(raw)
        if not host or not is_valid_web_host(host) or host in seen:
            continue
        seen.add(host)
        out.append(host)
    for match in _BARE_HOST.finditer(text):
        host = _normalize_host(match.group(1))
        if not host or not is_valid_web_host(host) or host in seen:
            continue
        seen.add(host)
        out.append(host)
    return _drop_hyphen_fragments(text, _drop_suffix_fragments(out))


def _drop_hyphen_fragments(text: str, hosts: list[str]) -> list[str]:
    lower = text.lower()
    return [h for h in hosts if f"-{h}" not in lower]


def _drop_suffix_fragments(hosts: list[str]) -> list[str]:
    """Drop foo.bar.com when bar.com is also present (avoid duplicate registry entries)."""
    if len(hosts) < 2:
        return hosts
    keep: list[str] = []
    for host in hosts:
        if any(host != other and other.endswith("." + host) for other in hosts):
            continue
        keep.append(host)
    return keep
