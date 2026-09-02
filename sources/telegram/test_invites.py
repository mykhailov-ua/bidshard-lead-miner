from typing import Any
import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.invites import discover_invite_hashes


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class InvitesTest(unittest.IsolatedAsyncioTestCase):
    async def test_flood_wait_continues(self) -> None:
        from telethon.errors import FloodWaitError

        checked_ok = SimpleNamespace(
            title="ok channel",
            chat=SimpleNamespace(id=1, title="ok channel"),
        )
        calls: list[Any] = []

        async def fake_client(req: Any) -> Any:
            calls.append(req)
            if len(calls) == 1:
                raise FloodWaitError(120)
            return checked_ok

        with patch("sources.telegram.telethon_retry.asyncio.sleep", new=AsyncMock()) as mock_sleep:
            got = await discover_invite_hashes(fake_client, ["hash1", "hash2"], rate_limit_qps=0)

        self.assertEqual(len(got), 1)
        self.assertEqual(got[0].name, "ok channel")
        self.assertTrue(mock_sleep.called)
        sleeps = [call.args[0] for call in mock_sleep.call_args_list]
        self.assertGreaterEqual(sleeps[0], 120)


if __name__ == "__main__":
    unittest.main()
