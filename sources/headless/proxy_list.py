"""Parse PARSER_PROXY_LIST for Playwright (same comma list as Go httpclient)."""

from __future__ import annotations

import os
from typing import cast
from urllib.parse import urlparse


def proxy_urls_from_env() -> list[str]:
    raw = os.environ.get("PARSER_PROXY_LIST", "").strip()
    if not raw:
        return []
    return [part.strip() for part in raw.split(",") if part.strip()]


def resolve_proxy_index() -> int:
    raw = os.environ.get("PARSER_HEADLESS_PROXY_INDEX", "-1").strip()
    try:
        return int(raw)
    except ValueError:
        return -1


def proxy_settings_at(index: int) -> dict[str, str] | None:
    urls = proxy_urls_from_env()
    if not urls:
        return None
    if index < 0:
        index = 0
    if index >= len(urls):
        index = index % len(urls)
    url = urls[index]
    parsed = urlparse(url)
    if not parsed.hostname:
        return None
    port = parsed.port or 8080
    server = f"{parsed.scheme}://{parsed.hostname}:{port}"
    proxy: dict[str, str] = {"server": server}
    if parsed.username:
        proxy["username"] = parsed.username
    if parsed.password:
        proxy["password"] = parsed.password
    return cast(dict[str, str], proxy)


def profile_dir_name(proxy_index: int) -> str:
    if proxy_index < 0:
        return "direct"
    return f"proxy_{proxy_index}"
