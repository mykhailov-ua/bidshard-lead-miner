import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.config import ChatConfig, load_config
from sources.telegram.discover import discover_via_search, load_serp_entries, merge_chat_lists


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


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


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class DiscoverFloodWaitTest(unittest.IsolatedAsyncioTestCase):
    async def test_search_continues_after_flood_wait(self) -> None:
        from telethon.tl.types import Channel

        channel = Channel(
            id=1,
            title="Affiliate EU",
            username="aff_eu",
            megagroup=True,
            access_hash=0,
        )
        ok_result = SimpleNamespace(chats=[channel])
        calls = {"n": 0}

        async def fake_client(req: object) -> object:
            calls["n"] += 1
            if calls["n"] == 1:
                from telethon.errors import FloodWaitError

                raise FloodWaitError(10)
            return ok_result

        with patch(
            "sources.telegram.telethon_retry.asyncio.sleep", new=AsyncMock()
        ):
            got = await discover_via_search(fake_client, ["affiliate"], 5)

        self.assertEqual(len(got), 1)
        self.assertEqual(got[0].username, "aff_eu")


if __name__ == "__main__":
    unittest.main()
