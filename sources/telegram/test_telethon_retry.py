import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.telethon_retry import (
    MAX_FLOOD_WAIT_SEC,
    call_with_flood_wait,
    capped_flood_wait_seconds,
    is_flood_wait,
    sleep_flood_wait,
)


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


class TelethonRetryUnitTest(unittest.TestCase):
    def test_capped_flood_wait_seconds(self) -> None:
        self.assertEqual(capped_flood_wait_seconds(900), MAX_FLOOD_WAIT_SEC)
        self.assertEqual(capped_flood_wait_seconds(-5), 0)
        self.assertEqual(capped_flood_wait_seconds(30), 30)


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class TelethonRetryAsyncTest(unittest.IsolatedAsyncioTestCase):
    async def test_sleep_flood_wait_caps(self) -> None:
        from telethon.errors import FloodWaitError

        exc = FloodWaitError(900)
        self.assertTrue(is_flood_wait(exc))
        with patch(
            "sources.telegram.telethon_retry.asyncio.sleep", new=AsyncMock()
        ) as mock_sleep:
            waited = await sleep_flood_wait(900, label="test")
        self.assertEqual(waited, MAX_FLOOD_WAIT_SEC)
        self.assertTrue(mock_sleep.called)
        slept = mock_sleep.call_args[0][0]
        self.assertGreaterEqual(slept, MAX_FLOOD_WAIT_SEC)

    async def test_call_with_flood_wait_retries(self) -> None:
        from telethon.errors import FloodWaitError

        calls = {"n": 0}

        async def fn() -> str:
            calls["n"] += 1
            if calls["n"] == 1:
                raise FloodWaitError(5)
            return "ok"

        with patch(
            "sources.telegram.telethon_retry.asyncio.sleep", new=AsyncMock()
        ):
            got = await call_with_flood_wait("job", fn, attempts=2)
        self.assertEqual(got, "ok")
        self.assertEqual(calls["n"], 2)


if __name__ == "__main__":
    unittest.main()
