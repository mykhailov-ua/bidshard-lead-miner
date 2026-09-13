from __future__ import annotations

import os
from dataclasses import dataclass, field
from pathlib import Path

import yaml

from .icp import load_icp_queries


def slice_discover_queries(queries: list[str]) -> list[str]:
    """Optional batch window for MTProto Contacts.Search (cron-friendly)."""
    batch_raw = os.environ.get("TELEGRAM_DISCOVER_QUERY_BATCH", "").strip()
    if not batch_raw:
        return queries
    try:
        batch = max(1, int(batch_raw))
    except ValueError:
        return queries
    offset = 0
    offset_raw = os.environ.get("TELEGRAM_DISCOVER_QUERY_OFFSET", "").strip()
    if offset_raw:
        try:
            offset = int(offset_raw)
        except ValueError:
            offset = 0
    if not queries:
        return queries
    offset = offset % len(queries)
    rotated = queries[offset:] + queries[:offset]
    if batch >= len(rotated):
        return rotated
    return rotated[:batch]


CHAT_ROLES = frozenset(
    {"buyer_supergroup", "vendor_support", "supply", "intel_only", "buyer"}
)


@dataclass
class ChatConfig:
    name: str
    username: str = ""
    invite_hash: str = ""
    geo: str = "global"
    enabled: bool = True
    chat_id: int | None = None
    shard: int | None = None
    # buyer_supergroup | vendor_support | supply | intel_only (M4 curation)
    role: str = "buyer_supergroup"

    def normalized_role(self) -> str:
        role = (self.role or "buyer_supergroup").strip().lower()
        if role not in CHAT_ROLES:
            return "buyer_supergroup"
        return role

    def channel_key(self) -> str:
        if self.username:
            return f"u:{self.username.lower().lstrip('@')}"
        if self.invite_hash:
            return f"i:{self.invite_hash}"
        if self.chat_id is not None:
            return f"c:{self.chat_id}"
        return f"n:{self.name}"


@dataclass
class CrossMentionConfig:
    enabled: bool = True
    max_seeds: int = 60
    messages_per_channel: int = 50
    rescan_days: int = 30


@dataclass
class ChannelSearchConfig:
    enabled: bool = False
    terms: list[str] = field(default_factory=list)
    messages_per_term: int = 20
    channel_usernames: list[str] = field(default_factory=list)


@dataclass
class DiscoverConfig:
    enabled: bool
    queries: list[str]
    limit_per_query: int
    serp_channels_path: str
    employer_tg_queries_path: str = (
        "data/runtime/discovered_employer_tg_queries.json"
    )
    domains_path: str = "data/runtime/discovered_telegram_domains.json"
    icp_path: str = "config/discover.icp.json"
    cross_mention: CrossMentionConfig = field(default_factory=CrossMentionConfig)


@dataclass
class GlobalSearchConfig:
    enabled: bool = False
    terms: list[str] = field(default_factory=list)
    messages_per_query: int = 20


def _parse_chat_role(raw: object) -> str:
    role = str(raw or "buyer_supergroup").strip().lower()
    if role not in CHAT_ROLES:
        raise ValueError(f"invalid chat role: {role}")
    return role


@dataclass
class PoolConfig:
    auto_from_registry: bool = True
    max_active: int = 80
    denylist: list[str] = field(default_factory=list)


@dataclass
class ScraperConfig:
    chats: list[ChatConfig]
    session: str
    cursor_db: str
    poll_delay_sec: float
    message_limit: int
    discover: DiscoverConfig
    pool: PoolConfig = field(default_factory=PoolConfig)
    channel_search: ChannelSearchConfig = field(default_factory=ChannelSearchConfig)
    global_search: GlobalSearchConfig = field(default_factory=GlobalSearchConfig)


def load_config(path: str | Path) -> ScraperConfig:
    data = yaml.safe_load(Path(path).read_text(encoding="utf-8")) or {}
    chats: list[ChatConfig] = []
    for entry in data.get("chats") or []:
        geo = str(entry.get("geo", "global")).lower()
        if geo == "ru":
            continue
        chat_id = entry.get("chat_id")
        parsed_chat_id = int(chat_id) if chat_id is not None else None
        shard_raw = entry.get("shard")
        parsed_shard = int(shard_raw) if shard_raw is not None else None
        chats.append(
            ChatConfig(
                name=str(entry.get("name", entry.get("username", "chat"))),
                username=str(entry.get("username", "")).lstrip("@"),
                invite_hash=str(entry.get("invite_hash", "")).strip(),
                geo=geo,
                enabled=bool(entry.get("enabled", True)),
                chat_id=parsed_chat_id,
                shard=parsed_shard,
                role=_parse_chat_role(entry.get("role") or entry.get("channel_class")),
            )
        )

    discover_raw = data.get("discover", {}) or {}
    icp_path = discover_raw.get("icp_path", "config/discover.icp.json")
    icp_telegram, _ = load_icp_queries(icp_path)

    yaml_queries = [
        str(q).strip() for q in discover_raw.get("queries", []) if str(q).strip()
    ]
    queries = yaml_queries or icp_telegram

    cross_raw = discover_raw.get("cross_mention", {}) or {}
    cross_mention = CrossMentionConfig(
        enabled=bool(cross_raw.get("enabled", True)),
        max_seeds=int(cross_raw.get("max_seeds", 60)),
        messages_per_channel=int(cross_raw.get("messages_per_channel", 50)),
        rescan_days=int(cross_raw.get("rescan_days", 30)),
    )

    discover = DiscoverConfig(
        enabled=bool(discover_raw.get("enabled", True)),
        queries=queries,
        limit_per_query=int(discover_raw.get("limit_per_query", 15)),
        serp_channels_path=str(
            discover_raw.get(
                "serp_channels_path",
                "data/runtime/discovered_telegram_channels.json",
            )
        ),
        employer_tg_queries_path=str(
            discover_raw.get(
                "employer_tg_queries_path",
                "data/runtime/discovered_employer_tg_queries.json",
            )
        ),
        domains_path=str(
            discover_raw.get(
                "domains_path",
                "data/runtime/discovered_telegram_domains.json",
            )
        ),
        icp_path=str(icp_path),
        cross_mention=cross_mention,
    )
    if discover.enabled and not discover.queries and icp_telegram:
        discover.queries = icp_telegram
    discover.queries = slice_discover_queries(discover.queries)

    search_raw = data.get("channel_search", {}) or {}
    search_terms = [
        str(t).strip() for t in search_raw.get("terms", []) if str(t).strip()
    ]
    if not search_terms:
        search_terms = [
            "postback",
            "voluum",
            "keitaro alternative",
            "tracker alternative",
            "migrate from",
        ]
    channel_search = ChannelSearchConfig(
        enabled=bool(search_raw.get("enabled", True)),
        terms=search_terms,
        messages_per_term=int(search_raw.get("messages_per_term", 20)),
        channel_usernames=[
            str(u).strip().lstrip("@").lower()
            for u in search_raw.get("channels", [])
            if str(u).strip()
        ],
    )

    global_raw = data.get("global_search", {}) or {}
    global_terms = [
        str(t).strip() for t in global_raw.get("terms", []) if str(t).strip()
    ]
    if not global_terms:
        global_terms = [
            "voluum alternative",
            "keitaro alternative",
            "postback failing",
            "migrate from voluum",
            "tracker alternative",
        ]
    global_search = GlobalSearchConfig(
        enabled=bool(global_raw.get("enabled", False)),
        terms=global_terms,
        messages_per_query=int(global_raw.get("messages_per_query", 20)),
    )

    pool_raw = data.get("pool", {}) or {}
    denylist = [
        str(u).strip().lstrip("@").lower()
        for u in pool_raw.get("denylist", []) or []
        if str(u).strip()
    ]
    pool = PoolConfig(
        auto_from_registry=bool(pool_raw.get("auto_from_registry", True)),
        max_active=int(pool_raw.get("max_active", 80)),
        denylist=denylist,
    )

    session = str(data.get("session", "data/runtime/telethon.session"))
    session_env = os.environ.get("TELEGRAM_SESSION", "").strip()
    if session_env:
        session = session_env

    return ScraperConfig(
        chats=chats,
        session=session,
        cursor_db=str(data.get("cursor_db", "data/runtime/crawler.db")),
        poll_delay_sec=float(data.get("poll_delay_sec", 2)),
        message_limit=int(data.get("message_limit", 500)),
        discover=discover,
        pool=pool,
        channel_search=channel_search,
        global_search=global_search,
    )
