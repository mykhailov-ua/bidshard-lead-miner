from __future__ import annotations

import json
import sqlite3
from datetime import date
from pathlib import Path
from typing import Any

from .config import ChatConfig

# Job-board / news channel names: scrape less often (processor may still intel_only reject).
_INTEL_DEPRIORITIZE_SUBSTRINGS = (
    "_news",
    "_jobs",
    "_job_",
    "jobboard",
    "job_board",
    "vacancy",
    "partneroff",
    "recruit",
    "hiring",
)


def _intel_deprioritize_channel(username: str | None, title: str) -> bool:
    blob = f"{username or ''} {title or ''}".lower()
    return any(sub in blob for sub in _INTEL_DEPRIORITIZE_SUBSTRINGS)


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
        self._conn.execute(
            """
            CREATE TABLE IF NOT EXISTS telegram_runtime (
                key TEXT PRIMARY KEY,
                value TEXT NOT NULL DEFAULT ''
            )
            """
        )
        # telegram_runtime: invite_join_* (daily), global_search_hour:* (hourly UTC).
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
        for col, ddl in (
            ("cross_mention_at", "ALTER TABLE telegram_channels ADD COLUMN cross_mention_at TEXT"),
            ("last_emitted_at", "ALTER TABLE telegram_channels ADD COLUMN last_emitted_at TEXT"),
            ("pain_hits_30d", "ALTER TABLE telegram_channels ADD COLUMN pain_hits_30d INTEGER NOT NULL DEFAULT 0"),
            ("last_scraped_at", "ALTER TABLE telegram_channels ADD COLUMN last_scraped_at TEXT"),
            ("scrape_interval_days", "ALTER TABLE telegram_channels ADD COLUMN scrape_interval_days INTEGER NOT NULL DEFAULT 1"),
            ("channel_search_day", "ALTER TABLE telegram_channels ADD COLUMN channel_search_day TEXT"),
            ("channel_search_count", "ALTER TABLE telegram_channels ADD COLUMN channel_search_count INTEGER NOT NULL DEFAULT 0"),
            ("emitted_last_run", "ALTER TABLE telegram_channels ADD COLUMN emitted_last_run INTEGER NOT NULL DEFAULT 0"),
            ("last_flood_wait_sec", "ALTER TABLE telegram_channels ADD COLUMN last_flood_wait_sec INTEGER NOT NULL DEFAULT 0"),
        ):
            cols = {
                r[1] for r in self._conn.execute("PRAGMA table_info(telegram_channels)")
            }
            if col not in cols:
                self._conn.execute(ddl)
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
        if _intel_deprioritize_channel(username, chat.name or ""):
            self._conn.execute(
                """
                UPDATE telegram_channels
                SET scrape_interval_days = CASE
                    WHEN scrape_interval_days < 14 THEN 14
                    ELSE scrape_interval_days
                END,
                updated_at = CURRENT_TIMESTAMP
                WHERE channel_key = ?
                """,
                (key,),
            )
            self._conn.commit()

    def set_channel_enabled(self, channel_key: str, enabled: bool) -> None:
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET enabled = ?, updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
            """,
            (1 if enabled else 0, channel_key),
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

    def _row_to_chat(
        self, key: str, username: str | None, invite_hash: str | None, chat_id: int | None, title: str, geo: str
    ) -> ChatConfig:
        return ChatConfig(
            name=str(title or username or invite_hash or key),
            username=str(username or ""),
            invite_hash=str(invite_hash or ""),
            geo=str(geo or "global"),
            enabled=True,
            chat_id=int(chat_id) if chat_id is not None else None,
        )

    def list_enabled_chats(self) -> list[ChatConfig]:
        rows = self._conn.execute(
            """
            SELECT channel_key, username, invite_hash, chat_id, title, geo
            FROM telegram_channels
            WHERE enabled = 1
            ORDER BY updated_at DESC
            """
        ).fetchall()
        return [self._row_to_chat(*row) for row in rows]

    def list_due_chats(self) -> list[ChatConfig]:
        rows = self._conn.execute(
            """
            SELECT channel_key, username, invite_hash, chat_id, title, geo
            FROM telegram_channels
            WHERE enabled = 1
              AND (
                last_scraped_at IS NULL
                OR datetime(last_scraped_at) <= datetime('now', printf('-%d days', scrape_interval_days))
              )
            ORDER BY
              CASE WHEN pain_hits_30d > 0 THEN 0 ELSE 1 END,
              pain_hits_30d DESC,
              updated_at DESC
            """
        ).fetchall()
        return [self._row_to_chat(*row) for row in rows]

    def mark_scraped(self, channel_key: str, flood_wait_sec: int = 0) -> None:
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET last_scraped_at = CURRENT_TIMESTAMP,
                emitted_last_run = 0,
                last_flood_wait_sec = ?,
                updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
            """,
            (max(0, int(flood_wait_sec)), channel_key),
        )
        self._conn.commit()

    def record_emit(self, channel_key: str, had_pain: bool) -> None:
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET last_emitted_at = CURRENT_TIMESTAMP,
                emitted_last_run = emitted_last_run + 1,
                pain_hits_30d = CASE WHEN ? THEN pain_hits_30d + 1 ELSE pain_hits_30d END,
                scrape_interval_days = CASE WHEN ? THEN 1 ELSE scrape_interval_days END,
                updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
            """,
            (1 if had_pain else 0, 1 if had_pain else 0, channel_key),
        )
        self._conn.commit()
        if not had_pain:
            self._maybe_slow_cold_channel(channel_key)

    def _maybe_slow_cold_channel(self, channel_key: str) -> None:
        row = self._conn.execute(
            """
            SELECT pain_hits_30d, last_emitted_at, scrape_interval_days
            FROM telegram_channels WHERE channel_key = ?
            """,
            (channel_key,),
        ).fetchone()
        if not row:
            return
        pain_hits, last_emitted_at, interval = row
        if int(pain_hits or 0) > 0:
            return
        if not last_emitted_at:
            return
        if int(interval or 1) >= 7:
            return
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET scrape_interval_days = 7,
                updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
              AND pain_hits_30d = 0
              AND last_emitted_at IS NOT NULL
              AND datetime(last_emitted_at) < datetime('now', '-30 days')
            """,
            (channel_key,),
        )
        self._conn.commit()

    def can_channel_search(self, channel_key: str, daily_limit: int) -> bool:
        if daily_limit <= 0:
            return False
        today = date.today().isoformat()
        row = self._conn.execute(
            """
            SELECT channel_search_day, channel_search_count
            FROM telegram_channels WHERE channel_key = ?
            """,
            (channel_key,),
        ).fetchone()
        if not row:
            return True
        day, count = row
        if day != today:
            return True
        return int(count or 0) < daily_limit

    def record_channel_search(self, channel_key: str, searches: int) -> None:
        if searches <= 0:
            return
        today = date.today().isoformat()
        self._conn.execute(
            """
            UPDATE telegram_channels
            SET channel_search_day = ?,
                channel_search_count = CASE
                    WHEN channel_search_day = ? THEN channel_search_count + ?
                    ELSE ?
                END,
                updated_at = CURRENT_TIMESTAMP
            WHERE channel_key = ?
            """,
            (today, today, searches, searches, channel_key),
        )
        self._conn.commit()

    def top_channels_by_pain(self, limit: int = 10) -> list[dict[str, Any]]:
        rows = self._conn.execute(
            """
            SELECT username, title, pain_hits_30d, emitted_last_run, last_flood_wait_sec, scrape_interval_days
            FROM telegram_channels
            WHERE enabled = 1
            ORDER BY pain_hits_30d DESC, emitted_last_run DESC
            LIMIT ?
            """,
            (limit,),
        ).fetchall()
        out: list[dict[str, Any]] = []
        for username, title, pain, emitted, flood, interval in rows:
            label = username or title or "?"
            out.append(
                {
                    "channel": label,
                    "pain_hits_30d": int(pain or 0),
                    "emitted_last_run": int(emitted or 0),
                    "last_flood_wait_sec": int(flood or 0),
                    "scrape_interval_days": int(interval or 1),
                }
            )
        return out

    def list_channel_export_rows(self) -> list[dict[str, str]]:
        rows = self._conn.execute(
            """
            SELECT username, invite_hash, title, source
            FROM telegram_channels
            WHERE enabled = 1
            ORDER BY pain_hits_30d DESC, updated_at DESC
            """
        ).fetchall()
        out: list[dict[str, str]] = []
        for username, invite_hash, title, source in rows:
            entry: dict[str, str] = {"title": str(title or "")}
            if username:
                entry["username"] = str(username)
            if invite_hash:
                entry["invite_hash"] = str(invite_hash)
            if source:
                entry["source"] = str(source)
            out.append(entry)
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
        return [self._row_to_chat(*row) for row in rows]

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

    def _runtime_get(self, key: str) -> str:
        row = self._conn.execute(
            "SELECT value FROM telegram_runtime WHERE key = ?",
            (key,),
        ).fetchone()
        if not row:
            return ""
        return str(row[0] or "")

    def _runtime_set(self, key: str, value: str) -> None:
        self._conn.execute(
            """
            INSERT INTO telegram_runtime (key, value)
            VALUES (?, ?)
            ON CONFLICT(key) DO UPDATE SET value = excluded.value
            """,
            (key, value),
        )
        self._conn.commit()

    def can_invite_join(self, daily_limit: int) -> bool:
        if daily_limit <= 0:
            return False
        today = date.today().isoformat()
        if self._runtime_get("invite_join_day") != today:
            return True
        try:
            count = int(self._runtime_get("invite_join_count") or "0")
        except ValueError:
            count = 0
        return count < daily_limit

    def record_invite_join(self) -> None:
        today = date.today().isoformat()
        if self._runtime_get("invite_join_day") != today:
            self._runtime_set("invite_join_day", today)
            self._runtime_set("invite_join_count", "1")
            return
        try:
            count = int(self._runtime_get("invite_join_count") or "0")
        except ValueError:
            count = 0
        self._runtime_set("invite_join_count", str(count + 1))

    def can_global_search(self, hourly_limit: int) -> bool:
        if hourly_limit <= 0:
            return False
        return self.global_search_count_this_hour() < hourly_limit

    def global_search_count_this_hour(self) -> int:
        from datetime import datetime, timezone

        hour = datetime.now(timezone.utc).strftime("%Y%m%d%H")
        key = f"global_search_hour:{hour}"
        try:
            return int(self._runtime_get(key) or "0")
        except ValueError:
            return 0

    def record_global_search(self, queries_run: int) -> None:
        if queries_run <= 0:
            return
        from datetime import datetime, timezone

        hour = datetime.now(timezone.utc).strftime("%Y%m%d%H")
        key = f"global_search_hour:{hour}"
        try:
            count = int(self._runtime_get(key) or "0")
        except ValueError:
            count = 0
        self._runtime_set(key, str(count + queries_run))

    def close(self) -> None:
        self._conn.close()
