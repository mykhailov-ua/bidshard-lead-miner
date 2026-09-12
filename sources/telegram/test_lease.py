import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.lease import lease_settings


class LeaseTest(unittest.TestCase):
    def _store_with_chats(self, tmp: str, n: int = 4) -> CursorStore:
        db = Path(tmp) / "crawler.db"
        store = CursorStore(db)
        for i in range(n):
            chat = ChatConfig(name=f"c{i}", username=f"chat_{i}", geo="eu")
            store.upsert_channel(chat, "test")
        return store

    def test_claim_disjoint_workers(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = self._store_with_chats(tmp, 6)
            a = store.claim_due_chats("worker-a", 3, 300, 120)
            b = store.claim_due_chats("worker-b", 3, 300, 120)
            keys_a = {c.channel_key() for c in a}
            keys_b = {c.channel_key() for c in b}
            self.assertEqual(len(keys_a), 3)
            self.assertEqual(len(keys_b), 3)
            self.assertFalse(keys_a & keys_b)
            store.close()

    def test_release_on_flood_wait_records_seconds(self) -> None:
        chat = ChatConfig(name="hot", username="hot_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            store.upsert_channel(chat, "test")
            key = chat.channel_key()
            store.claim_due_chats("worker-a", 1, 300, 120)
            store.release_lease(key, "worker-a", flood_wait_sec=90)
            row = store._conn.execute(
                "SELECT lease_worker_id, last_flood_wait_sec FROM telegram_channels WHERE channel_key = ?",
                (key,),
            ).fetchone()
            self.assertIsNone(row[0])
            self.assertEqual(row[1], 90)
            store.close()

    def test_stale_lease_reclaimed(self) -> None:
        chat = ChatConfig(name="stale", username="stale_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            store.upsert_channel(chat, "test")
            key = chat.channel_key()
            store.claim_due_chats("worker-old", 1, 60, 120)
            store._conn.execute(
                """
                UPDATE telegram_channels
                SET lease_heartbeat_at = datetime('now', '-10 minutes')
                WHERE channel_key = ?
                """,
                (key,),
            )
            store._conn.commit()
            reclaimed = store.claim_due_chats("worker-new", 1, 300, 120)
            self.assertEqual(len(reclaimed), 1)
            self.assertEqual(reclaimed[0].channel_key(), key)
            row = store._conn.execute(
                "SELECT lease_worker_id FROM telegram_channels WHERE channel_key = ?",
                (key,),
            ).fetchone()
            self.assertEqual(row[0], "worker-new")
            store.close()

    def test_heartbeat_extends_lease(self) -> None:
        chat = ChatConfig(name="beat", username="beat_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            store.upsert_channel(chat, "test")
            key = chat.channel_key()
            store.claim_due_chats("worker-a", 1, 300, 120)
            self.assertTrue(store.heartbeat_lease(key, "worker-a", 600))
            row = store._conn.execute(
                "SELECT lease_worker_id FROM telegram_channels WHERE channel_key = ?",
                (key,),
            ).fetchone()
            self.assertEqual(row[0], "worker-a")
            store.close()

    def test_lease_settings_env(self) -> None:
        with patch.dict(
            "os.environ",
            {
                "TELEGRAM_WORKER_ID": "vps-0",
                "TELEGRAM_LEASE_TTL_SEC": "120",
                "TELEGRAM_LEASE_MAX_CHATS": "5",
                "TELEGRAM_LEASE_STALE_SEC": "60",
            },
            clear=False,
        ):
            wid, ttl, max_claim, stale = lease_settings()
            self.assertEqual(wid, "vps-0")
            self.assertEqual(ttl, 120)
            self.assertEqual(max_claim, 5)
            self.assertEqual(stale, 60)


if __name__ == "__main__":
    unittest.main()
