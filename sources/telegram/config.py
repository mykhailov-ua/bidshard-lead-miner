from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

@dataclass
class ChatConfig:
    name: str
    username: str
    geo: str
    enabled: bool = True
    chat_id: str | None = None


@dataclass
class ScraperConfig:
    chats: list[ChatConfig]
    session: str
    cursor_db: str
    poll_delay_sec: float
    message_limit: int


def load_config(path: str | Path) -> ScraperConfig:
    data = yaml.safe_load(Path(path).read_text(encoding="utf-8"))
    chats: list[ChatConfig] = []
    for entry in data.get("chats", []):
        geo = str(entry.get("geo", "global")).lower()
        if geo == "ru":
            continue
        if not entry.get("enabled", True):
            continue
        chats.append(
            ChatConfig(
                name=str(entry.get("name", entry.get("username", "chat"))),
                username=str(entry.get("username", "")).lstrip("@"),
                geo=geo,
                enabled=True,
                chat_id=entry.get("chat_id"),
            )
        )
    return ScraperConfig(
        chats=chats,
        session=str(data.get("session", "data/telethon.session")),
        cursor_db=str(data.get("cursor_db", "data/crawler.db")),
        poll_delay_sec=float(data.get("poll_delay_sec", 2)),
        message_limit=int(data.get("message_limit", 500)),
    )
