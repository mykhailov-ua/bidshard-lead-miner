"""Infer yaml/db channel role for H3-H5 discover segments."""

from __future__ import annotations

from .cpa_network_intel import is_cpa_network_channel_hint
from .hosting_incident import is_hosting_channel_hint
from .pwa_pain import is_pwa_channel_hint


def infer_channel_role(username: str = "", title: str = "", query: str = "") -> str:
    """Return buyer_supergroup, vendor_support, or supply for registry sync."""
    if is_pwa_channel_hint(username, title, query):
        return "vendor_support"
    if is_hosting_channel_hint(username, title, query):
        return "vendor_support"
    if is_cpa_network_channel_hint(username, title, query):
        return "supply"
    lower = f"{username} {title} {query}".lower()
    if any(
        token in lower
        for token in (
            "agency account",
            "agency support",
            "accounts shop",
            "hosting support",
            "client chat",
        )
    ):
        return "vendor_support"
    return "buyer_supergroup"
