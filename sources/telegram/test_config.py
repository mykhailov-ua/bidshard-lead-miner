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

    def test_telegram_session_env_overrides_yaml(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sources.telegram.yaml"
            path.write_text(
                "session: data/runtime/telethon.session\n"
                "discover:\n  enabled: false\n  queries: []\n"
                "  limit_per_query: 1\n  serp_channels_path: channels.json\n",
                encoding="utf-8",
            )
            import os

            os.environ["TELEGRAM_SESSION"] = "data/runtime/telethon.session.0"
            try:
                cfg = load_config(path)
                self.assertEqual(cfg.session, "data/runtime/telethon.session.0")
            finally:
                os.environ.pop("TELEGRAM_SESSION", None)

    def test_discover_query_batch_env_slices_icp(self) -> None:
        import os

        with tempfile.TemporaryDirectory() as tmp:
            icp = Path(tmp) / "discover.icp.json"
            icp.write_text(
                '{"telegram_search":["q0","q1","q2","q3","q4"]}',
                encoding="utf-8",
            )
            path = Path(tmp) / "sources.telegram.yaml"
            path.write_text(
                f"discover:\n  enabled: true\n  icp_path: {icp}\n"
                "  limit_per_query: 1\n  serp_channels_path: channels.json\n",
                encoding="utf-8",
            )
            os.environ["TELEGRAM_DISCOVER_QUERY_BATCH"] = "2"
            os.environ["TELEGRAM_DISCOVER_QUERY_OFFSET"] = "1"
            try:
                cfg = load_config(path)
                self.assertEqual(cfg.discover.queries, ["q1", "q2"])
            finally:
                os.environ.pop("TELEGRAM_DISCOVER_QUERY_BATCH", None)
                os.environ.pop("TELEGRAM_DISCOVER_QUERY_OFFSET", None)

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
