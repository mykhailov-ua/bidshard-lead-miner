import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore


class CrossMentionStoreTest(unittest.TestCase):
    def test_cross_mention_seed_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(
                ChatConfig(name="seed", username="seed_chat", geo="eu"),
                "discover",
            )
            seeds = store.list_cross_mention_seeds(limit=10)
            self.assertEqual(len(seeds), 1)
            self.assertEqual(seeds[0].username, "seed_chat")
            store.mark_cross_mention_scanned(seeds[0].channel_key())
            again = store.list_cross_mention_seeds(limit=10, rescan_days=30)
            self.assertEqual(len(again), 0)
            store.close()


if __name__ == "__main__":
    unittest.main()
