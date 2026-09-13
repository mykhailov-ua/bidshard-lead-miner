import tempfile
import unittest
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.participants import (
    export_participants_json,
    harvest_chat_participants,
    participant_harvest_enabled,
)


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


class ParticipantHarvestTest(unittest.IsolatedAsyncioTestCase):
    async def test_skips_broadcast_channel(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            chat = ChatConfig(name="ch", username="mychan", geo="global")
            store.upsert_channel(chat, "test")
            client = MagicMock()
            n = await harvest_chat_participants(client, object(), chat, store, "channel")
            self.assertEqual(n, 0)
            client.iter_participants.assert_not_called()
            store.close()

    @unittest.skipIf(_telethon_missing(), "telethon not installed")
    async def test_harvest_supergroup_upserts(self) -> None:
        from telethon.tl.types import User

        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            chat = ChatConfig(name="grp", username="mygroup", geo="global")
            store.upsert_channel(chat, "test")
            user = User(id=99, is_self=False, contact=False, mutual_contact=False, deleted=False, bot=False, bot_chat_history=False, bot_nochats=False, verified=False, restricted=False, min=False, bot_inline_geo=False, support=False, scam=False, fake=False, bot_attach_menu=False, premium=False, attach_menu_enabled=False, bot_can_edit=False, close_friend=False, stories_hidden=False, stories_unavailable=True, access_hash=0, first_name="Ann", last_name="", username="ann_u", phone=None, photo=None, status=None, bot_info_version=None, restriction_reason=[], bot_inline_placeholder=None, lang_code=None, emoji_status=None, usernames=[], stories_max_id=None, color=None, profile_color=None, bot_active_users=None, bot_verification_icon=None, send_paid_messages_stars=None)
            users = [user]

            async def _aiter(*_a, **_k):
                for u in users:
                    yield u

            client = MagicMock()
            client.iter_participants = MagicMock(return_value=_aiter())

            n = await harvest_chat_participants(client, object(), chat, store, "supergroup")
            self.assertEqual(n, 1)
            self.assertEqual(store.count_chat_members(chat.channel_key()), 1)
            row = store.list_all_chat_members()[0]
            self.assertEqual(row["username"], "ann_u")
            self.assertEqual(row["user_id"], 99)
            store.close()

    def test_export_json(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            out = Path(tmp) / "members.json"
            store = CursorStore(db)
            chat = ChatConfig(name="grp", username="mygroup", geo="global")
            store.upsert_channel(chat, "test")
            store.upsert_chat_members(
                chat.channel_key(),
                "mygroup",
                [{"user_id": 1, "username": "u1", "first_name": "A", "last_name": "", "is_bot": False}],
            )
            n = export_participants_json(store, out)
            self.assertEqual(n, 1)
            self.assertTrue(out.is_file())
            store.close()

    def test_harvest_enabled_default(self) -> None:
        self.assertTrue(participant_harvest_enabled())


class CursorParticipantsTest(unittest.TestCase):
    def test_refresh_due_after_touch(self) -> None:
        chat = ChatConfig(name="g", username="g_chat", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            store.upsert_channel(chat, "test")
            key = chat.channel_key()
            self.assertTrue(store.participants_harvest_due(key, 7))
            store.touch_participants_harvest(key)
            self.assertFalse(store.participants_harvest_due(key, 7))
            store.close()
