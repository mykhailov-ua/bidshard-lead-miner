import io
import json
import os
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.config import GlobalSearchConfig, ScraperConfig, DiscoverConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.global_search import run_global_search


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


def _cfg() -> ScraperConfig:
    return ScraperConfig(
        chats=[],
        session="data/runtime/telethon.session",
        cursor_db="data/runtime/crawler.db",
        poll_delay_sec=1,
        message_limit=100,
        discover=DiscoverConfig(
            enabled=False,
            queries=[],
            limit_per_query=1,
            serp_channels_path="data/runtime/discovered_telegram_channels.json",
        ),
        global_search=GlobalSearchConfig(
            enabled=True,
            terms=["voluum alternative"],
            messages_per_query=5,
        ),
    )


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class GlobalSearchTest(unittest.IsolatedAsyncioTestCase):
    async def test_emits_global_source(self) -> None:
        message = SimpleNamespace(
            id=7,
            chat_id=55,
            reply_to=None,
            text="need voluum alternative after postback failures",
        )
        message.get_sender = AsyncMock(return_value=SimpleNamespace(username="buyer1"))

        async def iter_messages(entity, search="", limit=0):
            self.assertIsNone(entity)
            self.assertEqual(search, "voluum alternative")
            yield message

        client = SimpleNamespace(iter_messages=iter_messages)
        out = io.StringIO()

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                with patch.dict(
                    os.environ,
                    {"TELEGRAM_GLOBAL_SEARCH": "1", "TELEGRAM_GLOBAL_SEARCH_LIMIT": "5"},
                    clear=False,
                ):
                    emitted = await run_global_search(client, _cfg(), store, out)
            finally:
                store.close()

        self.assertEqual(emitted, 1)
        row = json.loads(out.getvalue().strip())
        self.assertEqual(row["source"], "telegram:global:voluum_alternative")
        self.assertEqual(row["chat_type"], "global_search")

    async def test_hourly_budget_blocks_second_run(self) -> None:
        client = SimpleNamespace(iter_messages=AsyncMock())
        out = io.StringIO()

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                with patch.dict(
                    os.environ,
                    {"TELEGRAM_GLOBAL_SEARCH": "1", "TELEGRAM_GLOBAL_SEARCH_LIMIT": "1"},
                    clear=False,
                ):
                    store.record_global_search(1)
                    emitted = await run_global_search(client, _cfg(), store, out)
            finally:
                store.close()

        self.assertEqual(emitted, 0)
        self.assertEqual(out.getvalue(), "")


if __name__ == "__main__":
    unittest.main()
