"""Render a page with Playwright and print HTML to stdout."""

from __future__ import annotations

import os
import sys
from urllib.parse import urlparse


def first_http_proxy() -> dict[str, str] | None:
    raw = os.environ.get("PARSER_PROXY_LIST", "").strip()
    if not raw:
        return None
    url = raw.split(",")[0].strip()
    if not url:
        return None
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
    return proxy


def fetch_html(url: str, timeout_ms: int = 30000) -> str:
    from playwright.sync_api import sync_playwright

    proxy = first_http_proxy()
    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=True)
        try:
            context = (
                browser.new_context(proxy=proxy) if proxy else browser.new_context()
            )
            page = context.new_page()
            page.goto(url, wait_until="domcontentloaded", timeout=timeout_ms)
            return page.content()
        finally:
            browser.close()


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: python -m sources.headless.fetch <url>", file=sys.stderr)
        return 2
    url = sys.argv[1].strip()
    if not url:
        return 2
    try:
        html = fetch_html(url)
    except Exception as exc:
        print(f"headless fetch failed: {exc}", file=sys.stderr)
        return 1
    sys.stdout.write(html)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
