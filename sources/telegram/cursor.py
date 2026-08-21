from __future__ import annotations

import sqlite3
from pathlib import Path

from .config import ChatConfig


class CursorStore:
    def __init__(self, path: str | Path) -> None:
        self.path = Path(path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._conn = sqlite3.connect(self.path)
        self._migrate_channels_table()
        self._conn.execute(
            """
            CREATE TABLE IF NOT EXISTS telegram_cursors (
                chat_key TEXT PRIMARY KEY,
                last_message_id INTEGER NOT NULL DEFAULT 0,
                updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """
        )
        self._conn.commit()

    def _migrate_channels_table(self) -> None:
        row = self._conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='telegram_channels'"
        ).fetchone()
        if row:
            cols = {
                r[1] for r in self._conn.execute("PRAGMA table_info(telegram_channels)")
            }
            if "channel_key" not in cols:
                self._conn.execute("DROP TABLE telegram_channels")
            elif "cross_mention_at" not in cols:
                self._conn.execute(
                    "ALTER TABLE telegram_channels ADD COLUMN cross_mention_at TEXT"
                )
                self._conn.commit()
        self._conn.execute(
            """
            CREATE TABLE IF NOT EXISTS telegram_channels (
                channel_key TEXT PRIMARY KEY,
                username TEXT,
                invite_hash TEXT,
                chat_id INTEGER,
                title TEXT NOT NULL DEFAULT '',
                source TEXT NOT NULL DEFAULT 'manual',
                geo TEXT NOT NULL DEFAULT 'global',
                enabled INTEGER NOT NULL DEFAULT 1,
                discovered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
                cross_mention_at TEXT
            )
            """
        )
        cols = {
            r[1] for r in self._conn.execute("PRAGMA table_info(telegram_channels)")
        }
        if "cross_mention_at" not in cols:
            self._conn.execute(
                "ALTER TABLE telegram_channels ADD COLUMN cross_mention_at TEXT"
            )
            self._conn.commit()

    def get_last_message_id(self, chat_key: str) -> int:
        row = self._conn.execute(
            "SELECT last_message_id FROM telegram_cursors WHERE chat_key = ?",
            (chat_key,),
        ).fetchone()
        if not row:
            return 0
        return int(row[0])

    def set_last_message_id(self, chat_key: str, message_id: int) -> None:
        self._conn.execute(
            """
            INSERT INTO telegram_cursors (chat_key, last_message_id, updated_at)
            VALUES (?, ?, CURRENT_TIMESTAMP)
            ON CONFLICT(chat_key) DO UPDATE SET
                last_message_id = excluded.last_message_id,
                updated_at = CURRENT_TIMESTAMP
            """,
            (chat_key, message_id),
        )
        self._conn.commit()

    def upsert_channel(
        self,
        chat: ChatConfig,
        source: str,
    ) -> None:
        key = chat.channel_key()
        username = chat.username.lower().lstrip("@") if chat.username else None
        invite_hash = chat.invite_hash or None
        if not username and not invite_hash and chat.chat_id is None:
            return
        self._conn.execute(
            """
            INSERT INTO telegram_channels (
                channel_key, username, invite_hash, chat_id, title, source, geo, enabled,
                discovered_at, updated_at
            )
            VALUES (?, ?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
            ON CONFLICT(channel_key) DO UPDATE SET
                username = excluded.username,
                invite_hash = excluded.invite_hash,
                chat_id = COALESCE(excluded.chat_id, telegram_channels.chat_id),
                title = excluded.title,
                source = CASE
                    WHEN telegram_channels.source = 'manual' THEN telegram_channels.source
                    ELSE excluded.source
                END,
                geo = excluded.geo,
                enabled = 1,
                updated_at = CURRENT_TIMESTAMP
            """,
            (
                key,
                username,
                invite_hash,
                chat.chat_id,
                chat.name or username or invite_hash or key,
                source,
                chat.geo or "global",
            ),
        )
        self._conn.commit()

    def set_chat_id(self, channel_key: str, chat_id: int) -> None:
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET chat_id = ?, updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
            """,
            (chat_id, channel_key),
        )
        self._conn.commit()

    def list_enabled_chats(self) -> list[ChatConfig]:
        rows = self._conn.execute(
            """
            SELECT channel_key, username, invite_hash, chat_id, title, geo
            FROM telegram_channels
            WHERE enabled = 1
            ORDER BY updated_at DESC
            """
        ).fetchall()
        out: list[ChatConfig] = []
        for key, username, invite_hash, chat_id, title, geo in rows:
            out.append(
                ChatConfig(
                    name=str(title or username or invite_hash or key),
                    username=str(username or ""),
                    invite_hash=str(invite_hash or ""),
                    geo=str(geo or "global"),
                    enabled=True,
                    chat_id=int(chat_id) if chat_id is not None else None,
                )
            )
        return out

    def list_cross_mention_seeds(
        self, limit: int, rescan_days: int = 30
    ) -> list[ChatConfig]:
        rows = self._conn.execute(
            """
            SELECT channel_key, username, invite_hash, chat_id, title, geo
            FROM telegram_channels
            WHERE enabled = 1
              AND (username IS NOT NULL OR chat_id IS NOT NULL OR invite_hash IS NOT NULL)
              AND (
                cross_mention_at IS NULL
                OR cross_mention_at < datetime('now', ?)
              )
            ORDER BY cross_mention_at IS NULL DESC, cross_mention_at ASC
            LIMIT ?
            """,
            (f"-{int(rescan_days)} days", limit),
        ).fetchall()
        out: list[ChatConfig] = []
        for key, username, invite_hash, chat_id, title, geo in rows:
            out.append(
                ChatConfig(
                    name=str(title or username or invite_hash or key),
                    username=str(username or ""),
                    invite_hash=str(invite_hash or ""),
                    geo=str(geo or "global"),
                    enabled=True,
                    chat_id=int(chat_id) if chat_id is not None else None,
                )
            )
        return out

    def mark_cross_mention_scanned(self, channel_key: str) -> None:
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET cross_mention_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
            """,
            (channel_key,),
        )
        self._conn.commit()

    def close(self) -> None:
        self._conn.close()
