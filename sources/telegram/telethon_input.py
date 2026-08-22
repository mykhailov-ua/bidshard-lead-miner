"""Telethon TL input helpers for strict type stubs."""

from __future__ import annotations

from telethon.tl.types import Channel, InputChannel, TypeInputChannel
from telethon.utils import get_input_channel


def channel_input_peer(channel: Channel) -> TypeInputChannel:
    """Map a resolved Channel to InputChannel for channels.* requests."""
    peer = get_input_channel(channel)
    if not isinstance(peer, InputChannel):
        raise TypeError(f"expected InputChannel, got {type(peer).__name__}")
    return peer
