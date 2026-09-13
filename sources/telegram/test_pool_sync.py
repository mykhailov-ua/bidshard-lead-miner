import json
import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import ChatConfig, PoolConfig, DiscoverConfig, ScraperConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.pool_sync import sync_registry_to_store


class PoolSyncTest(unittest.TestCase):
    def test_imports_registry_and_applies_denylist(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            registry = Path(tmp) / "channels.json"
            registry.write_text(
                json.dumps(
                    {
                        "channels": [
                            {
                                "username": "buyermedia",
                                "title": "Buyer media WW",
                                "query": "voluum",
                            },
                            {
                                "username": "maximaffiliate",
                                "title": "Maxim",
                                "query": "cpa",
                            },
                        ]
                    }
                ),
                encoding="utf-8",
            )
            db = Path(tmp) / "crawler.db"
            store = CursorStore(str(db))
            try:
                stats = sync_registry_to_store(
                    registry,
                    store,
                    denylist=["maximaffiliate"],
                    max_active=10,
                )
                self.assertEqual(stats["imported"], 1)
                enabled = {c.username for c in store.list_enabled_chats()}
                self.assertIn("buyermedia", enabled)
                self.assertNotIn("maximaffiliate", enabled)
            finally:
                store.close()

    def test_pool_sync_prefers_higher_priority_when_capped(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            registry = Path(tmp) / "channels.json"
            registry.write_text(
                json.dumps(
                    {
                        "channels": [
                            {
                                "username": "voluum_pain_chat",
                                "title": "Voluum pain",
                                "query": "site:t.me voluum",
                                "source": "registry_sync",
                            },
                            {
                                "username": "mediabuy_arb_team",
                                "title": "Media buy arb",
                                "query": "cross_mention",
                                "source": "cross_mention",
                            },
                        ]
                    }
                ),
                encoding="utf-8",
            )
            db = Path(tmp) / "crawler.db"
            store = CursorStore(str(db))
            try:
                stats = sync_registry_to_store(
                    registry,
                    store,
                    denylist=[],
                    max_active=1,
                )
                self.assertEqual(stats["imported"], 1)
                enabled = store.list_enabled_chats()
                self.assertEqual(len(enabled), 1)
                self.assertEqual(enabled[0].username, "mediabuy_arb_team")
            finally:
                store.close()


if __name__ == "__main__":
    unittest.main()
