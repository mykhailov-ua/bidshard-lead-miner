"""Render a page with Playwright using a desktop Chrome-like profile."""

from __future__ import annotations

import sys
from typing import cast
from urllib.parse import urlparse

from playwright.sync_api import ProxySettings

from sources.headless.interact import simulate_reading
from sources.headless.profile import chromium_launch_kwargs, load_profile


def first_http_proxy() -> ProxySettings | None:
    raw = __import__("os").environ.get("PARSER_PROXY_LIST", "").strip()
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
    return cast(ProxySettings, proxy)


def fetch_html(url: str, timeout_ms: int | None = None) -> str:
    from playwright.sync_api import sync_playwright

    profile = load_profile()
    if timeout_ms is None:
        timeout_ms = profile.timeout_ms

    proxy_settings = first_http_proxy()
    launch_kwargs = chromium_launch_kwargs(profile)

    with sync_playwright() as pw:
        browser = pw.chromium.launch(**launch_kwargs)
        try:
            context = browser.new_context(
                user_agent=profile.user_agent,
                viewport=profile.viewport,
                screen=profile.screen,
                locale=profile.locale,
                timezone_id=profile.timezone_id,
                color_scheme="light",
                has_touch=False,
                is_mobile=False,
                device_scale_factor=1,
                extra_http_headers=profile.document_headers,
                proxy=proxy_settings,
            )
            page = context.new_page()
            page.set_default_navigation_timeout(timeout_ms)
            page.set_default_timeout(timeout_ms)
            page.goto(url, wait_until="domcontentloaded", timeout=timeout_ms)
            if profile.interactive:
                simulate_reading(
                    page,
                    profile.viewport["width"],
                    profile.viewport["height"],
                )
            else:
                try:
                    page.wait_for_load_state("load", timeout=min(timeout_ms, 15_000))
                except Exception:
                    pass
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
