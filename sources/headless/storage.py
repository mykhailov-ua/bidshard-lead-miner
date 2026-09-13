"""Persistent Playwright storage_state per proxy persona."""

from __future__ import annotations

import os
from pathlib import Path

from sources.headless.proxy_list import profile_dir_name, resolve_proxy_index


def profile_root() -> Path:
    raw = os.environ.get("PARSER_HEADLESS_PROFILE_ROOT", "data/runtime/browser_profiles").strip()
    return Path(raw)


def storage_state_path(proxy_index: int | None = None) -> Path:
    if proxy_index is None:
        proxy_index = resolve_proxy_index()
    return profile_root() / profile_dir_name(proxy_index) / "storage_state.json"
