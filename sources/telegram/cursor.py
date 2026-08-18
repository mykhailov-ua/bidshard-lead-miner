from __future__ import annotations

import sqlite3
from pathlib import Path

class CursorStore:
    def __init__(self, path: str | Path) -> None:
        self.path = Path(path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._conn = sqlite3.connect(self.path)
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

    def close(self) -> None:
        self._conn.close()
