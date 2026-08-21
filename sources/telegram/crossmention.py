from __future__ import annotations

import logging
from typing import Any

from .config import ChatConfig, CrossMentionConfig
from .domains import RegistryEntry
from .invites import discover_invite_hashes
from .tglinks import extract_from_texts, web_domains

from .prefilter import channel_icp_relevant

LOG = logging.getLogger("telegram.crossmention")


async def resolve_seed_entity(client: Any, chat: ChatConfig) -> Any | None:
    from telethon.tl.functions.messages import CheckChatInviteRequest

    try:
        if chat.username:
            return await client.get_entity(chat.username)
        if chat.chat_id is not None:
            return await client.get_entity(chat.chat_id)
        if chat.invite_hash:
            checked = await client(CheckChatInviteRequest(chat.invite_hash))
            if getattr(checked, "chat", None) is not None:
                return checked.chat
    except Exception as exc:
        LOG.warning(
            "cross-mention seed resolve failed key=%s error=%s",
            chat.channel_key(),
            exc,
        )
    return None


async def collect_seed_content(
    client: Any,
    entity: Any,
    messages_per_channel: int,
) -> tuple[list[str], list[Any], str]:
    """Gather about text, message bodies, and raw messages for cross-mention extraction."""
    from telethon.tl.functions.channels import GetFullChannelRequest
    from telethon.tl.types import Channel

    texts: list[str] = []
    messages: list[Any] = []
    about_text = ""

    if isinstance(entity, Channel):
        try:
            full = await client(GetFullChannelRequest(entity))
            about = getattr(full.full_chat, "about", None)
            if about:
                about_text = str(about)
                texts.append(about_text)
        except Exception as exc:
            LOG.warning(
                "cross-mention about fetch failed title=%s error=%s", entity.title, exc
            )

    try:
        async for msg in client.iter_messages(entity, limit=messages_per_channel):
            messages.append(msg)
            body = msg.text or msg.message
            if body:
                texts.append(str(body))
    except Exception as exc:
        LOG.warning("cross-mention messages fetch failed error=%s", exc)

    return texts, messages, about_text


async def extract_forward_channel_usernames(
    client: Any,
    messages: list[Any],
) -> list[str]:
    """Resolve public usernames from forwarded channel posts (forward-chasing).

    Skips private channels (no public username). Telethon may set from_id or saved_from_peer.
    """
    from telethon.tl.types import PeerChannel

    out: list[str] = []
    seen: set[str] = set()

    for msg in messages:
        fwd = getattr(msg, "fwd_from", None)
        if fwd is None:
            continue
        peer = getattr(fwd, "from_id", None)
        if peer is None:
            # Saved forwards sometimes only expose saved_from_peer.
            peer = getattr(fwd, "saved_from_peer", None)
        if not isinstance(peer, PeerChannel):
            continue
        try:
            ent = await client.get_entity(peer)
        except Exception as exc:
            LOG.debug(
                "forward-chase entity miss channel_id=%s error=%s", peer.channel_id, exc
            )
            continue
        username = getattr(ent, "username", None)
        if not username:
            continue
        handle = username.lower().lstrip("@")
        if handle in seen:
            continue
        seen.add(handle)
        out.append(handle)

    return out


async def resolve_username_channels(
    client: Any,
    usernames: list[str],
    known_keys: set[str],
    seed_keys: set[str],
) -> list[ChatConfig]:
    from telethon.tl.types import Channel as ChannelType, Chat, User

    out: list[ChatConfig] = []
    seen: set[str] = set()

    for username in usernames:
        u = username.lower().lstrip("@")
        key = f"u:{u}"
        if key in known_keys or key in seed_keys or key in seen:
            continue
        seen.add(key)
        try:
            entity = await client.get_entity(u)
        except Exception as exc:
            LOG.debug("cross-mention entity miss username=%s error=%s", u, exc)
            continue
        if isinstance(entity, User):
            # Cross-mention targets channels/groups only; skip user DMs.
            continue
        if not isinstance(entity, (ChannelType, Chat)):
            continue
        pub_username = getattr(entity, "username", None)
        if pub_username:
            pub_username = pub_username.lower()
        else:
            pub_username = u
        title = getattr(entity, "title", pub_username) or pub_username
        if not channel_icp_relevant([title]):
            LOG.debug("cross-mention skip non-icp title=%s", title)
            continue
        cfg = ChatConfig(
            name=getattr(entity, "title", pub_username) or pub_username,
            username=pub_username,
            geo="global",
            enabled=True,
            chat_id=getattr(entity, "id", None),
        )
        out.append(cfg)

    return out


def _append_domain_entries(
    pending: list[RegistryEntry],
    seen_domains: set[str],
    domains: list[str],
    channel_name: str,
    source: str,
    kind: str,
    discovered_via: str = "",
) -> None:
    # discovered_via records which seed channel surfaced the URL.
    for domain in domains:
        if domain in seen_domains:
            continue
        seen_domains.add(domain)
        row: dict[str, str] = {
            "domain": domain,
            "channel": channel_name,
            "source": source,
            "kind": kind,
        }
        if discovered_via:
            row["discovered_via"] = discovered_via
        pending.append(row)


async def discover_cross_mentions(
    client: Any,
    seeds: list[ChatConfig],
    cfg: CrossMentionConfig,
    known_keys: set[str],
) -> tuple[list[ChatConfig], list[ChatConfig], list[RegistryEntry]]:
    """Return (text-discovered channels, forward-chased channels, domain registry rows)."""
    if not cfg.enabled or not seeds:
        return [], [], []

    seed_keys = {c.channel_key() for c in seeds}
    pending_handles: list[str] = []
    pending_forward_handles: list[str] = []
    pending_invites: list[str] = []
    pending_domains: list[RegistryEntry] = []
    seen_handles: set[str] = set()
    seen_forward_handles: set[str] = set()
    seen_invites: set[str] = set()
    seen_domains: set[str] = set()

    scanned = 0
    for seed in seeds:
        entity = await resolve_seed_entity(client, seed)
        if entity is None:
            continue

        texts, messages, about_text = await collect_seed_content(
            client, entity, cfg.messages_per_channel
        )
        handles, invites = extract_from_texts(texts)

        for h in handles:
            key = f"u:{h}"
            if key in known_keys or key in seed_keys or h in seen_handles:
                continue
            seen_handles.add(h)
            pending_handles.append(h)

        for inv in invites:
            key = f"i:{inv}"
            if key in known_keys or key in seed_keys or inv in seen_invites:
                continue
            seen_invites.add(inv)
            pending_invites.append(inv)

        channel_name = (seed.username or seed.name or "").strip().lstrip("@").lower()
        seed_label = channel_name or seed.channel_key()

        # Split about vs message URLs so Go tgweb can rank channel-about links higher.
        if about_text:
            _append_domain_entries(
                pending_domains,
                seen_domains,
                web_domains(about_text),
                channel_name,
                "cross_mention",
                "mentioned_in_about",
                discovered_via=seed_label,
            )

        message_only_text = "\n".join(t for t in texts if t != about_text)
        if message_only_text:
            _append_domain_entries(
                pending_domains,
                seen_domains,
                web_domains(message_only_text),
                channel_name,
                "cross_mention",
                "mentioned_in_message",
                discovered_via=seed_label,
            )

        for fwd_handle in await extract_forward_channel_usernames(client, messages):
            key = f"u:{fwd_handle}"
            if (
                key in known_keys
                or key in seed_keys
                or fwd_handle in seen_forward_handles
            ):
                continue
            seen_forward_handles.add(fwd_handle)
            # Separate from @handles in message text; discover.py tags these cross_mention_forward.
            pending_forward_handles.append(fwd_handle)

        scanned += 1
        LOG.info(
            "cross-mention seed scanned key=%s texts=%d handles=%d forwards=%d invites=%d",
            seed.channel_key(),
            len(texts),
            len(handles),
            len(pending_forward_handles),
            len(invites),
        )

    resolved = await resolve_username_channels(
        client, pending_handles, known_keys, seed_keys
    )
    forward_resolved = await resolve_username_channels(
        client, pending_forward_handles, known_keys, seed_keys
    )
    if pending_invites:
        resolved.extend(await discover_invite_hashes(client, pending_invites))

    LOG.info(
        "cross-mention finished seeds_scanned=%d new_channels=%d forward_channels=%d handles=%d invites=%d domains=%d",
        scanned,
        len(resolved),
        len(forward_resolved),
        len(pending_handles),
        len(pending_invites),
        len(pending_domains),
    )
    return resolved, forward_resolved, pending_domains
