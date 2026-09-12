"""P1 static shard filter for cron scrape (TELEGRAM_SHARD / TELEGRAM_SHARD_COUNT)."""

from __future__ import annotations

import hashlib
import logging
import os

from .config import ChatConfig

LOG = logging.getLogger("telegram.shard")


def shard_settings() -> tuple[int, int]:
    """Return (shard_id, shard_count). shard_count<=1 disables filtering."""
    raw_count = os.environ.get("TELEGRAM_SHARD_COUNT", "").strip()
    if not raw_count:
        return 0, 1
    try:
        shard_count = max(1, int(raw_count))
    except ValueError:
        return 0, 1
    if shard_count <= 1:
        return 0, 1
    raw_shard = os.environ.get("TELEGRAM_SHARD", "0").strip()
    try:
        shard_id = int(raw_shard)
    except ValueError:
        shard_id = 0
    shard_id %= shard_count
    return shard_id, shard_count


def stable_shard(channel_key: str, shard_count: int) -> int:
    digest = hashlib.sha256(channel_key.encode("utf-8")).digest()
    return digest[0] % shard_count


def chat_shard(chat: ChatConfig, shard_count: int) -> int:
    if chat.shard is not None:
        return int(chat.shard) % shard_count
    return stable_shard(chat.channel_key(), shard_count)


def filter_chats_for_shard(chats: list[ChatConfig]) -> list[ChatConfig]:
    shard_id, shard_count = shard_settings()
    if shard_count <= 1:
        return chats
    out = [chat for chat in chats if chat_shard(chat, shard_count) == shard_id]
    LOG.info(
        "shard filter shard=%d count=%d due=%d of %d",
        shard_id,
        shard_count,
        len(out),
        len(chats),
    )
    return out
