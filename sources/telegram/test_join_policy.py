import os
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.join_policy import resolve_invite_entity


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class JoinPolicyTest(unittest.IsolatedAsyncioTestCase):
    async def test_check_only_does_not_import(self) -> None:
        from telethon.tl.functions.messages import (
            CheckChatInviteRequest,
            ImportChatInviteRequest,
        )

        chat = ChatConfig(name="preview", invite_hash="abc123", geo="global")
        checked = SimpleNamespace(
            title="preview channel",
            chat=SimpleNamespace(id=99, title="preview channel"),
        )
        calls: list[object] = []

        async def fake_client(req: object) -> object:
            calls.append(req)
            if isinstance(req, CheckChatInviteRequest):
                return checked
            raise AssertionError("ImportChatInviteRequest must not be called")

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                entity = await resolve_invite_entity(fake_client, chat, store)
            finally:
                store.close()

        self.assertEqual(entity.id, 99)
        self.assertEqual(len(calls), 1)
        self.assertIsInstance(calls[0], CheckChatInviteRequest)
        self.assertNotIsInstance(calls[0], ImportChatInviteRequest)

    async def test_preview_without_join_raises(self) -> None:
        from telethon.tl.functions.messages import CheckChatInviteRequest

        chat = ChatConfig(name="preview", invite_hash="abc123", geo="global")
        checked = SimpleNamespace(title="preview only")

        async def fake_client(req: object) -> object:
            if isinstance(req, CheckChatInviteRequest):
                return checked
            raise AssertionError("unexpected request")

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                with self.assertRaises(ValueError):
                    await resolve_invite_entity(fake_client, chat, store)
            finally:
                store.close()

    async def test_join_when_enabled_and_under_cap(self) -> None:
        from telethon.tl.functions.messages import (
            CheckChatInviteRequest,
            ImportChatInviteRequest,
        )

        chat = ChatConfig(name="join me", invite_hash="joinhash", geo="global")
        checked = SimpleNamespace(title="join me")
        imported = SimpleNamespace(chats=[SimpleNamespace(id=42, title="join me")])
        calls: list[object] = []

        async def fake_client(req: object) -> object:
            calls.append(req)
            if isinstance(req, CheckChatInviteRequest):
                return checked
            if isinstance(req, ImportChatInviteRequest):
                return imported
            raise AssertionError("unexpected request")

        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                with patch.dict(os.environ, {"TELEGRAM_INVITE_JOIN": "1"}, clear=False):
                    entity = await resolve_invite_entity(fake_client, chat, store)
            finally:
                store.close()

        self.assertEqual(entity.id, 42)
        self.assertEqual(len(calls), 2)
        self.assertIsInstance(calls[1], ImportChatInviteRequest)
        self.assertTrue(store.can_invite_join(3))


if __name__ == "__main__":
    unittest.main()
