import os
import unittest
from datetime import datetime, timezone
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.history_chunk import (
    history_chunk_size,
    iter_messages_chunked,
    realtime_backfill_limit,
)


class HistoryChunkConfigTest(unittest.TestCase):
    def test_defaults(self) -> None:
        with patch.dict(os.environ, {}, clear=True):
            self.assertEqual(history_chunk_size(), 100)
            self.assertEqual(realtime_backfill_limit(), 300)


class HistoryChunkIterTest(unittest.IsolatedAsyncioTestCase):
    async def test_pauses_every_chunk(self) -> None:
        messages = [SimpleNamespace(id=100 - i) for i in range(5)]

        async def fake_iter(_entity, limit=0, search=None):
            for msg in messages[:limit]:
                yield msg

        client = SimpleNamespace(iter_messages=fake_iter)
        seen: list[int] = []
        with patch(
            "sources.telegram.history_chunk.asyncio.sleep", new=AsyncMock()
        ) as mock_sleep:
            async for msg in iter_messages_chunked(
                client,
                "chat",
                total_limit=5,
                chunk_size=2,
                delay_min=1.5,
                delay_max=1.5,
            ):
                seen.append(msg.id)
        self.assertEqual(seen, [100, 99, 98, 97, 96])
        self.assertEqual(mock_sleep.await_count, 2)

    async def test_stops_at_cursor(self) -> None:
        messages = [SimpleNamespace(id=i) for i in (50, 40, 30, 20)]

        async def fake_iter(_entity, limit=0, search=None):
            for msg in messages:
                yield msg

        client = SimpleNamespace(iter_messages=fake_iter)
        seen: list[int] = []
        async for msg in iter_messages_chunked(
            client,
            "chat",
            total_limit=10,
            stop_before_id=30,
            chunk_size=100,
        ):
            seen.append(msg.id)
        self.assertEqual(seen, [50, 40])

    async def test_stops_at_date(self) -> None:
        messages = [
            SimpleNamespace(
                id=3,
                date=datetime(2025, 6, 1, tzinfo=timezone.utc),
            ),
            SimpleNamespace(
                id=2,
                date=datetime(2025, 4, 1, tzinfo=timezone.utc),
            ),
            SimpleNamespace(
                id=1,
                date=datetime(2025, 1, 1, tzinfo=timezone.utc),
            ),
        ]

        async def fake_iter(_entity, limit=0, search=None):
            for msg in messages:
                yield msg

        client = SimpleNamespace(iter_messages=fake_iter)
        seen: list[int] = []
        since = datetime(2025, 3, 1, tzinfo=timezone.utc)
        async for msg in iter_messages_chunked(
            client,
            "chat",
            total_limit=0,
            stop_before_date=since,
            chunk_size=100,
        ):
            seen.append(msg.id)
        self.assertEqual(seen, [3, 2])


if __name__ == "__main__":
    unittest.main()
