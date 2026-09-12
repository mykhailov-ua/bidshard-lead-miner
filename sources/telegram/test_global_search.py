import io
import json
import os
import tempfile
import unittest
from datetime import datetime, timezone
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.config import GlobalSearchConfig, ScraperConfig, DiscoverConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.global_search import (
    in_global_search_window,
    parse_utc_hours_window,
    run_global_search,
)


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
            terms=["postback failing"],
            messages_per_query=5,
        ),
    )


_GLOBAL_SEARCH_ENV = {
    "TELEGRAM_GLOBAL_SEARCH": "1",
    "TELEGRAM_GLOBAL_SEARCH_LIMIT": "5",
    "TELEGRAM_GLOBAL_SEARCH_DAILY_LIMIT": "5",
    "TELEGRAM_GLOBAL_SEARCH_UTC_HOURS": "*",
}


class ParseUtcHoursTest(unittest.TestCase):
    def test_range(self) -> None:
        self.assertEqual(parse_utc_hours_window("2-6"), frozenset({2, 3, 4, 5, 6}))

    def test_list(self) -> None:
        self.assertEqual(parse_utc_hours_window("2,4,6"), frozenset({2, 4, 6}))


class GlobalSearchWindowTest(unittest.TestCase):
    def test_inside_window(self) -> None:
        with patch.dict(os.environ, {"TELEGRAM_GLOBAL_SEARCH_UTC_HOURS": "2-6"}, clear=False):
            now = datetime(2026, 1, 1, 4, 0, tzinfo=timezone.utc)
            self.assertTrue(in_global_search_window(now))

    def test_outside_window(self) -> None:
        with patch.dict(os.environ, {"TELEGRAM_GLOBAL_SEARCH_UTC_HOURS": "2-6"}, clear=False):
            now = datetime(2026, 1, 1, 12, 0, tzinfo=timezone.utc)
            self.assertFalse(in_global_search_window(now))

    def test_wildcard_allows_all_hours(self) -> None:
        with patch.dict(os.environ, {"TELEGRAM_GLOBAL_SEARCH_UTC_HOURS": "*"}, clear=False):
            now = datetime(2026, 1, 1, 12, 0, tzinfo=timezone.utc)
            self.assertTrue(in_global_search_window(now))


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
            self.assertEqual(search, "postback failing")
            yield message

        client = SimpleNamespace(iter_messages=iter_messages)
        out = io.StringIO()

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                with patch.dict(os.environ, _GLOBAL_SEARCH_ENV, clear=False):
                    emitted = await run_global_search(client, _cfg(), store, out)
            finally:
                store.close()

        self.assertEqual(emitted, 1)
        row = json.loads(out.getvalue().strip())
        self.assertEqual(row["source"], "telegram:global:postback_failing")
        self.assertEqual(row["chat_type"], "global_search")

    async def test_hourly_budget_blocks_second_run(self) -> None:
        client = SimpleNamespace(iter_messages=AsyncMock())
        out = io.StringIO()

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                env = {**_GLOBAL_SEARCH_ENV, "TELEGRAM_GLOBAL_SEARCH_LIMIT": "1"}
                with patch.dict(os.environ, env, clear=False):
                    store.record_global_search(1)
                    emitted = await run_global_search(client, _cfg(), store, out)
            finally:
                store.close()

        self.assertEqual(emitted, 0)
        self.assertEqual(out.getvalue(), "")

    async def test_daily_cap_blocks_second_run(self) -> None:
        client = SimpleNamespace(iter_messages=AsyncMock())
        out = io.StringIO()

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                env = {
                    **_GLOBAL_SEARCH_ENV,
                    "TELEGRAM_GLOBAL_SEARCH_DAILY_LIMIT": "1",
                }
                with patch.dict(os.environ, env, clear=False):
                    store.record_global_search(1)
                    emitted = await run_global_search(client, _cfg(), store, out)
            finally:
                store.close()

        self.assertEqual(emitted, 0)
        self.assertEqual(out.getvalue(), "")

    async def test_window_blocks_outside_hours(self) -> None:
        client = SimpleNamespace(iter_messages=AsyncMock())
        out = io.StringIO()

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                env = {
                    **_GLOBAL_SEARCH_ENV,
                    "TELEGRAM_GLOBAL_SEARCH_UTC_HOURS": "2-6",
                }
                with patch.dict(os.environ, env, clear=False):
                    now = datetime(2026, 1, 1, 12, 0, tzinfo=timezone.utc)
                    with patch(
                        "sources.telegram.global_search.in_global_search_window",
                        return_value=False,
                    ):
                        emitted = await run_global_search(client, _cfg(), store, out)
            finally:
                store.close()

        self.assertEqual(emitted, 0)
        self.assertEqual(out.getvalue(), "")


if __name__ == "__main__":
    unittest.main()
