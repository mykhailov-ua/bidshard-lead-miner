import json
import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import ChatConfig, load_config
from sources.telegram.discover import load_serp_entries, merge_chat_lists


class DiscoverTest(unittest.TestCase):
    def test_load_serp_entries(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "channels.json"
            path.write_text(
                json.dumps(
                    {
                        "channels": [
                            {"username": "Aff_Lead"},
                            {"invite_hash": "AbCdEfGhIjKlMn"},
                            {"username": "aff_lead"},
                        ]
                    }
                ),
                encoding="utf-8",
            )
            got = load_serp_entries(path)
            keys = {c.channel_key() for c in got}
            self.assertIn("u:aff_lead", keys)
            self.assertIn("i:AbCdEfGhIjKlMn", keys)

    def test_merge_prefers_manual_geo(self) -> None:
        manual = [ChatConfig(name="m", username="foo", geo="eu")]
        discovered = [ChatConfig(name="d", username="foo", geo="global")]
        merged = merge_chat_lists(manual, discovered)
        self.assertEqual(len(merged), 1)
        self.assertEqual(merged[0].geo, "eu")

    def test_discover_defaults_in_config(self) -> None:
        yaml_text = """
discover:
  enabled: true
chats: []
session: data/s.session
cursor_db: data/c.db
"""
        with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as f:
            f.write(yaml_text)
            path = f.name
        cfg = load_config(path)
        self.assertTrue(cfg.discover.enabled)
        self.assertGreaterEqual(len(cfg.discover.queries), 4)


if __name__ == "__main__":
    unittest.main()
