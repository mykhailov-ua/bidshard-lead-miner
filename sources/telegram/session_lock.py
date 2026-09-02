"""Exclusive flock on telethon.session.lock (pairs with internal/telethon session lock)."""

from __future__ import annotations

import contextlib
import fcntl
import os
import sys
from pathlib import Path
from typing import Iterator


def session_lock_path(session_path: str | Path) -> Path:
    return Path(str(session_path) + ".lock")


def parent_holds_session_lock() -> bool:
    return os.environ.get("TELETHON_PARENT_HOLDS_LOCK", "").strip() in ("1", "true", "yes")


@contextlib.contextmanager
def session_exclusive_lock(session_path: str | Path) -> Iterator[None]:
    if parent_holds_session_lock():
        yield
        return
    if not sys.platform.startswith(("linux", "darwin")):
        yield
        return
    lock_path = session_lock_path(session_path)
    lock_path.parent.mkdir(parents=True, exist_ok=True)
    fh = lock_path.open("a+")
    try:
        fcntl.flock(fh.fileno(), fcntl.LOCK_EX)
        yield
    finally:
        try:
            fcntl.flock(fh.fileno(), fcntl.LOCK_UN)
        finally:
            fh.close()
