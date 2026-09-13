"""Desktop Chrome profile aligned with internal/httpclient applyBrowserHeaders."""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Any

# Keep in sync with internal/httpclient/client.go applyBrowserHeaders.
DEFAULT_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
)
DEFAULT_LOCALE = "en-US"
DEFAULT_TIMEZONE = "UTC"
DEFAULT_VIEWPORT_WIDTH = 1920
DEFAULT_VIEWPORT_HEIGHT = 1080


@dataclass(frozen=True)
class BrowserProfile:
    user_agent: str
    locale: str
    timezone_id: str
    viewport: dict[str, int]
    screen: dict[str, int]
    document_headers: dict[str, str]
    headed: bool
    interactive: bool
    timeout_ms: int
    channel: str | None


def _env_bool(name: str, default: bool) -> bool:
    raw = os.environ.get(name, "").strip().lower()
    if raw == "":
        return default
    return raw in ("1", "true", "yes", "on")


def _env_int(name: str, default: int) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def load_profile() -> BrowserProfile:
    locale = os.environ.get("PARSER_HEADLESS_LOCALE", DEFAULT_LOCALE).strip() or DEFAULT_LOCALE
    tz = os.environ.get("PARSER_HEADLESS_TIMEZONE", DEFAULT_TIMEZONE).strip() or DEFAULT_TIMEZONE
    ua = os.environ.get("PARSER_HEADLESS_USER_AGENT", DEFAULT_USER_AGENT).strip() or DEFAULT_USER_AGENT
    lang_primary = locale.split("-")[0] if "-" in locale else locale
    accept_language = f"{locale},{lang_primary};q=0.9"

    w = _env_int("PARSER_HEADLESS_VIEWPORT_WIDTH", DEFAULT_VIEWPORT_WIDTH)
    h = _env_int("PARSER_HEADLESS_VIEWPORT_HEIGHT", DEFAULT_VIEWPORT_HEIGHT)
    if w < 320:
        w = DEFAULT_VIEWPORT_WIDTH
    if h < 240:
        h = DEFAULT_VIEWPORT_HEIGHT

    channel = os.environ.get("PARSER_HEADLESS_CHANNEL", "").strip() or None

    return BrowserProfile(
        user_agent=ua,
        locale=locale,
        timezone_id=tz,
        viewport={"width": w, "height": h},
        screen={"width": w, "height": h},
        document_headers={
            "Accept": (
                "text/html,application/xhtml+xml,application/xml;q=0.9,"
                "image/avif,image/webp,image/apng,*/*;q=0.8"
            ),
            "Accept-Language": accept_language,
            "Sec-Ch-Ua": '"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"',
            "Sec-Ch-Ua-Mobile": "?0",
            "Sec-Ch-Ua-Platform": '"Windows"',
        },
        headed=_env_bool("PARSER_HEADLESS_HEADED", False),
        interactive=_env_bool("PARSER_HEADLESS_INTERACTIVE", True),
        timeout_ms=_env_int("PARSER_HEADLESS_TIMEOUT_MS", 30_000),
        channel=channel,
    )


def chromium_launch_kwargs(profile: BrowserProfile) -> dict[str, Any]:
    """Playwright launch options: stock Chrome flags, no automation banner."""
    args = [
        "--disable-blink-features=AutomationControlled",
        "--disable-dev-shm-usage",
        "--no-sandbox",
    ]
    if not profile.headed:
        args.append("--headless=new")
    kwargs: dict[str, Any] = {
        "headless": not profile.headed,
        "args": args,
        "ignore_default_args": ["--enable-automation"],
    }
    if profile.channel:
        kwargs["channel"] = profile.channel
    return kwargs
