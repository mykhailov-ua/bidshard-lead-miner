"""P2 dynamic pool lease (shared crawler.db)."""

from __future__ import annotations

import logging
import os
import socket

LOG = logging.getLogger("telegram.lease")


def lease_enabled() -> bool:
    return os.environ.get("TELEGRAM_LEASE_ENABLED", "").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def worker_id() -> str:
    explicit = os.environ.get("TELEGRAM_WORKER_ID", "").strip()
    if explicit:
        return explicit
    return f"{socket.gethostname()}-{os.getpid()}"


def _env_int(name: str, default: int) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    try:
        return max(0, int(raw))
    except ValueError:
        return default


def lease_settings() -> tuple[str, int, int, int]:
    """Return worker_id, ttl_sec, max_claim, stale_sec."""
    return (
        worker_id(),
        _env_int("TELEGRAM_LEASE_TTL_SEC", 300),
        max(1, _env_int("TELEGRAM_LEASE_MAX_CHATS", 15)),
        max(30, _env_int("TELEGRAM_LEASE_STALE_SEC", 120)),
    )
