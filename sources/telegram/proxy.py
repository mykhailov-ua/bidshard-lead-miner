"""Parse TELEGRAM_PROXY_URL for Telethon MTProto (separate from PARSER_PROXY_LIST HTTP crawl)."""

from __future__ import annotations

import os
from typing import Any
from urllib.parse import urlparse


def telegram_proxy_from_env() -> tuple[Any, ...] | None:
    raw = os.environ.get("TELEGRAM_PROXY_URL", "").strip()
    if not raw:
        return None
    return parse_telegram_proxy(raw)


def parse_telegram_proxy(raw: str) -> tuple[Any, ...] | None:
    raw = (raw or "").strip()
    if not raw:
        return None
    parsed = urlparse(raw)
    scheme = (parsed.scheme or "").lower()
    if scheme not in ("socks5", "socks4", "http", "https"):
        raise ValueError(f"unsupported TELEGRAM_PROXY_URL scheme: {scheme}")

    try:
        import socks  # PySocks
    except ImportError as exc:
        raise RuntimeError(
            "PySocks required for TELEGRAM_PROXY_URL (pip install PySocks)"
        ) from exc

    scheme_map = {
        "socks5": socks.SOCKS5,
        "socks4": socks.SOCKS4,
        "http": socks.HTTP,
        "https": socks.HTTP,
    }
    host = parsed.hostname
    if not host:
        raise ValueError("TELEGRAM_PROXY_URL missing host")
    port = parsed.port
    if port is None:
        port = 1080 if scheme.startswith("socks") else 8080
    user = parsed.username or None
    password = parsed.password or None
    return (scheme_map[scheme], host, port, True, user, password)
