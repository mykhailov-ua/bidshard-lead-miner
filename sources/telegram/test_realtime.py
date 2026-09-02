import os
import tempfile
import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.realtime import realtime_env_enabled, resolve_listen_targets


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


class RealtimeEnvTest(unittest.TestCase):
    def test_disabled_by_default(self) -> None:
        with patch.dict(os.environ, {}, clear=True):
            self.assertFalse(realtime_env_enabled())

    def test_enabled_when_set(self) -> None:
        with patch.dict(os.environ, {"TELEGRAM_REALTIME": "1"}):
            self.assertTrue(realtime_env_enabled())


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class RealtimeListenerTest(unittest.IsolatedAsyncioTestCase):
    async def test_resolve_listen_targets_skips_failures(self) -> None:
        from telethon.tl.types import Channel

        ok_chat = ChatConfig(name="ok", username="aff_ok", geo="eu")
        bad_chat = ChatConfig(name="bad", username="aff_bad", geo="eu")
        channel = Channel(
            id=100,
            title="Aff OK",
            username="aff_ok",
            megagroup=True,
            access_hash=1,
        )

        async def get_entity(entity: object) -> object:
            if entity == "aff_bad":
                raise ValueError("not found")
            return channel

        client = SimpleNamespace(get_entity=AsyncMock(side_effect=get_entity))
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(os.path.join(tmp, "c.db"))
            with patch(
                "sources.telegram.realtime.fetch_channel_about",
                new=AsyncMock(return_value="about"),
            ):
                targets = await resolve_listen_targets(
                    client, [ok_chat, bad_chat], store
                )
            store.close()

        self.assertEqual(len(targets), 1)
        self.assertEqual(targets[0][1].username, "aff_ok")
        self.assertEqual(targets[0][3], "supergroup")


if __name__ == "__main__":
    unittest.main()
