"""Pointer, wheel, and dwell patterns closer to interactive reading (HEADLESS.md section 6)."""

from __future__ import annotations

import random
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from playwright.sync_api import Page


def _rng() -> random.Random:
    seed_raw = __import__("os").environ.get("PARSER_HEADLESS_SEED", "").strip()
    if seed_raw:
        try:
            return random.Random(int(seed_raw))
        except ValueError:
            pass
    return random.Random()


def simulate_reading(page: Page, viewport_width: int, viewport_height: int) -> None:
    """Scroll and move pointer after navigation; uses Playwright input (not DOM click())."""
    rng = _rng()
    page.wait_for_timeout(rng.randint(500, 1400))

    try:
        scroll_height = int(page.evaluate("() => document.body?.scrollHeight || 0"))
    except Exception:
        scroll_height = viewport_height * 2

    cap = min(scroll_height, viewport_height * 5)
    y = 0
    while y < cap:
        step = rng.randint(90, 240)
        y += step
        page.mouse.wheel(0, step)
        page.wait_for_timeout(rng.randint(60, 220))

    moves = rng.randint(2, 5)
    margin = 80
    max_x = max(margin + 1, viewport_width - margin)
    max_y = max(margin + 1, viewport_height - margin)
    for _ in range(moves):
        x0, y0 = rng.randint(margin, max_x), rng.randint(margin, max_y)
        x1, y1 = rng.randint(margin, max_x), rng.randint(margin, max_y)
        steps = rng.randint(10, 24)
        for i in range(1, steps + 1):
            t = i / steps
            x = int(x0 + (x1 - x0) * t + rng.uniform(-2, 2))
            y = int(y0 + (y1 - y0) * t + rng.uniform(-2, 2))
            page.mouse.move(x, y)
            page.wait_for_timeout(rng.randint(8, 28))

    page.wait_for_timeout(rng.randint(300, 900))
    try:
        page.wait_for_load_state("networkidle", timeout=5000)
    except Exception:
        pass
