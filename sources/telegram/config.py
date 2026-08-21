from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml

from .icp import load_icp_queries


@dataclass
class ChatConfig:
    name: str
    username: str = ""
    invite_hash: str = ""
    geo: str = "global"
    enabled: bool = True
    chat_id: int | None = None

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
class DiscoverConfig:
    enabled: bool
    queries: list[str]
    limit_per_query: int
    serp_channels_path: str
    domains_path: str = "data/runtime/discovered_telegram_domains.json"
    icp_path: str = "config/discover.icp.json"
    cross_mention: CrossMentionConfig = field(default_factory=CrossMentionConfig)


@dataclass
class ScraperConfig:
    chats: list[ChatConfig]
    session: str
    cursor_db: str
    poll_delay_sec: float
    message_limit: int
    discover: DiscoverConfig


def load_config(path: str | Path) -> ScraperConfig:
    data = yaml.safe_load(Path(path).read_text(encoding="utf-8"))
    chats: list[ChatConfig] = []
    for entry in data.get("chats", []):
        geo = str(entry.get("geo", "global")).lower()
        if geo == "ru":
            continue
        if not entry.get("enabled", True):
            continue
        chat_id = entry.get("chat_id")
        parsed_chat_id = int(chat_id) if chat_id is not None else None
        chats.append(
            ChatConfig(
                name=str(entry.get("name", entry.get("username", "chat"))),
                username=str(entry.get("username", "")).lstrip("@"),
                invite_hash=str(entry.get("invite_hash", "")).strip(),
                geo=geo,
                enabled=True,
                chat_id=parsed_chat_id,
            )
        )

    discover_raw = data.get("discover", {}) or {}
    icp_path = discover_raw.get("icp_path", "config/discover.icp.json")
    icp_telegram, _ = load_icp_queries(icp_path)

    yaml_queries = [
        str(q).strip() for q in discover_raw.get("queries", []) if str(q).strip()
    ]
    queries = yaml_queries if yaml_queries else icp_telegram

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

    return ScraperConfig(
        chats=chats,
        session=str(data.get("session", "data/runtime/telethon.session")),
        cursor_db=str(data.get("cursor_db", "data/runtime/crawler.db")),
        poll_delay_sec=float(data.get("poll_delay_sec", 2)),
        message_limit=int(data.get("message_limit", 500)),
        discover=discover,
    )
