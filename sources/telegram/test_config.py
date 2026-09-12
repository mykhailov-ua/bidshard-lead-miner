import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import load_config


class ConfigLoadTest(unittest.TestCase):
    def test_chat_role_must_be_known(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sources.telegram.yaml"
            path.write_text(
                "chats:\n"
                "  - name: bad\n"
                "    username: bad_chat\n"
                "    role: not_a_role\n"
                "discover:\n  enabled: false\n  queries: []\n"
                "  limit_per_query: 1\n  serp_channels_path: channels.json\n",
                encoding="utf-8",
            )
            with self.assertRaises(ValueError):
                load_config(path)

    def test_chat_role_parses_buyer_supergroup(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sources.telegram.yaml"
            path.write_text(
                "chats:\n"
                "  - name: ok\n"
                "    username: ok_chat\n"
                "    role: vendor_support\n"
                "    enabled: false\n"
                "discover:\n  enabled: false\n  queries: []\n"
                "  limit_per_query: 1\n  serp_channels_path: channels.json\n",
                encoding="utf-8",
            )
            cfg = load_config(path)
            self.assertEqual(cfg.chats[0].role, "vendor_support")
            self.assertFalse(cfg.chats[0].enabled)

    def test_null_chats_key_loads_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sources.telegram.yaml"
            path.write_text(
                "discover:\n  enabled: false\n  queries: []\n  limit_per_query: 1\n"
                "  serp_channels_path: channels.json\nchats:\n  # commented only\n",
                encoding="utf-8",
            )
            cfg = load_config(path)
            self.assertEqual(cfg.chats, [])


if __name__ == "__main__":
    unittest.main()
