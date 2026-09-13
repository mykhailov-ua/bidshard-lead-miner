"""Render a page with Playwright using a desktop Chrome-like profile."""

from __future__ import annotations

import sys
from typing import Any

from sources.headless.interact import simulate_reading
from sources.headless.profile import chromium_launch_kwargs, load_profile
from sources.headless.proxy_list import proxy_settings_at, resolve_proxy_index
from sources.headless.storage import storage_state_path


def fetch_html(url: str, timeout_ms: int | None = None) -> str:
    from playwright.sync_api import sync_playwright

    profile = load_profile()
    if timeout_ms is None:
        timeout_ms = profile.timeout_ms

    proxy_index = resolve_proxy_index()
    proxy_settings = proxy_settings_at(proxy_index)
    launch_kwargs = chromium_launch_kwargs(profile)
    state_path = storage_state_path(proxy_index)

    with sync_playwright() as pw:
        browser = pw.chromium.launch(**launch_kwargs)
        try:
            ctx_kwargs: dict[str, Any] = {
                "user_agent": profile.user_agent,
                "viewport": profile.viewport,
                "screen": profile.screen,
                "locale": profile.locale,
                "timezone_id": profile.timezone_id,
                "color_scheme": "light",
                "has_touch": False,
                "is_mobile": False,
                "device_scale_factor": 1,
                "extra_http_headers": profile.document_headers,
                "proxy": proxy_settings,
            }
            if state_path.is_file():
                ctx_kwargs["storage_state"] = str(state_path)

            context = browser.new_context(**ctx_kwargs)
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
            html = page.content()
            state_path.parent.mkdir(parents=True, exist_ok=True)
            context.storage_state(path=str(state_path))
            return html
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
