import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore


class CursorStoreTest(unittest.TestCase):
    def test_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            self.assertEqual(store.get_last_message_id("chat_a"), 0)
            store.set_last_message_id("chat_a", 42)
            self.assertEqual(store.get_last_message_id("chat_a"), 42)
            store.close()

    def test_set_channel_enabled(self) -> None:
        chat = ChatConfig(name="jobs", username="job_board", geo="global")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(chat, "test")
            self.assertEqual(len(store.list_enabled_chats()), 1)
            store.set_channel_enabled(chat.channel_key(), False)
            self.assertEqual(store.list_enabled_chats(), [])
            store.set_channel_enabled(chat.channel_key(), True)
            self.assertEqual(len(store.list_enabled_chats()), 1)
            store.close()

    def test_list_enabled_chats_skips_disabled(self) -> None:
        enabled = ChatConfig(name="good", username="good_chat", geo="eu")
        disabled = ChatConfig(name="bad", username="bad_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(enabled, "test")
            store.upsert_channel(disabled, "test")
            store.set_channel_enabled(disabled.channel_key(), False)
            usernames = {c.username for c in store.list_enabled_chats()}
            self.assertEqual(usernames, {"good_chat"})
            store.close()

    def test_list_due_chats_respects_interval(self) -> None:
        hot = ChatConfig(name="hot", username="hot_chat", geo="eu")
        cold = ChatConfig(name="cold", username="cold_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(hot, "test")
            store.upsert_channel(cold, "test")
            store._conn.execute(
                """
                UPDATE telegram_channels
                SET last_scraped_at = CURRENT_TIMESTAMP, scrape_interval_days = 7
                WHERE channel_key = ?
                """,
                (cold.channel_key(),),
            )
            store._conn.commit()
            usernames = {c.username for c in store.list_due_chats()}
            self.assertIn("hot_chat", usernames)
            self.assertNotIn("cold_chat", usernames)
            store.close()

    def test_record_emit_pain_resets_interval(self) -> None:
        chat = ChatConfig(name="pain", username="pain_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(chat, "test")
            store.record_emit(chat.channel_key(), True)
            row = store._conn.execute(
                "SELECT pain_hits_30d, scrape_interval_days FROM telegram_channels WHERE channel_key = ?",
                (chat.channel_key(),),
            ).fetchone()
            self.assertEqual(row, (1, 1))
            store.close()

    def test_channel_search_budget(self) -> None:
        chat = ChatConfig(name="search", username="search_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(chat, "test")
            key = chat.channel_key()
            self.assertTrue(store.can_channel_search(key, 3))
            store.record_channel_search(key, 3)
            self.assertFalse(store.can_channel_search(key, 3))
            store.close()

    def test_invite_join_budget(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            self.assertTrue(store.can_invite_join(2))
            store.record_invite_join()
            store.record_invite_join()
            self.assertFalse(store.can_invite_join(2))
            store.close()

    def test_global_search_budget(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            self.assertTrue(store.can_global_search(2))
            store.record_global_search(2)
            self.assertFalse(store.can_global_search(2))
            store.close()

    def test_intel_deprioritize_interval(self) -> None:
        chat = ChatConfig(name="Partner Jobs", username="partneroff_pro", geo="global")
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            store.upsert_channel(chat, "discover")
            row = store._conn.execute(
                "SELECT scrape_interval_days FROM telegram_channels WHERE channel_key = ?",
                (chat.channel_key(),),
            ).fetchone()
            self.assertEqual(row, (14,))
            store.close()


if __name__ == "__main__":
    unittest.main()
