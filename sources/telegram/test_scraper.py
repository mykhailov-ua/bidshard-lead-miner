import io
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.config import ChatConfig, load_config
from sources.telegram.cursor import CursorStore
from sources.telegram.scraper import emit_line, fetch_reply_context, process_scrape_message


class ConfigTest(unittest.TestCase):
    def test_excludes_ru_and_disabled(self) -> None:
        yaml_text = """
chats:
  - name: latam
    username: latam_chat
    geo: latam
    enabled: true
  - name: ru_only
    username: arb_rf
    geo: ru
    enabled: true
  - name: disabled
    username: off_chat
    geo: eu
    enabled: false
"""
        with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as f:
            f.write(yaml_text)
            path = f.name
        cfg = load_config(path)
        usernames = {c.username for c in cfg.chats}
        self.assertEqual(usernames, {"latam_chat"})


class CursorStoreTest(unittest.TestCase):
    def test_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            store = CursorStore(db)
            self.assertEqual(store.get_last_message_id("chat_a"), 0)
            store.set_last_message_id("chat_a", 42)
            self.assertEqual(store.get_last_message_id("chat_a"), 42)
            store.close()


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


class EmitLineTest(unittest.TestCase):
    def test_caption_only_emits_reply_context_fields(self) -> None:
        buf = io.StringIO()
        chat = ChatConfig(name="latam", username="aff_latam", geo="latam")
        pain = "voluum alternative needed; postback failing on FTD again."
        emit_line(
            buf,
            chat,
            pain,
            "media_buyer",
            42,
            reply_to_message_id=41,
            reply_context="parent asked about tracker migration",
        )
        row = json.loads(buf.getvalue().strip())
        self.assertEqual(row["text"], pain)
        self.assertEqual(row["message_id"], 42)
        self.assertEqual(row["reply_to_message_id"], 41)
        self.assertEqual(row["reply_context"], "parent asked about tracker migration")


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class ScraperMessageTest(unittest.IsolatedAsyncioTestCase):
    async def test_process_caption_only_message(self) -> None:
        buf = io.StringIO()
        chat = ChatConfig(name="latam", username="aff_latam", geo="latam")
        message = SimpleNamespace(
            id=99,
            text="voluum alternative needed; postback failing on FTD again.",
            message="",
            photo=None,
            entities=None,
            reply_to=None,
            get_sender=AsyncMock(return_value=SimpleNamespace(username="buyer1")),
        )
        client = SimpleNamespace(get_messages=AsyncMock())
        with patch.dict("os.environ", {"TELEGRAM_PREFILTER": "false"}):
            emitted = await process_scrape_message(
                client,
                "entity",
                message,
                chat,
                "",
                "channel",
                buf,
            )
        self.assertTrue(emitted)
        row = json.loads(buf.getvalue().strip())
        self.assertIn("voluum", row["text"])

    async def test_reply_parent_fetch_once(self) -> None:
        buf = io.StringIO()
        chat = ChatConfig(name="latam", username="aff_latam", geo="latam")
        parent = SimpleNamespace(
            text="parent: need keitaro migration help",
            message="parent: need keitaro migration help",
        )
        client = SimpleNamespace(get_messages=AsyncMock(return_value=[parent]))
        message = SimpleNamespace(
            id=100,
            text="voluum alternative needed; postback failing on FTD again.",
            message="voluum alternative needed; postback failing on FTD again.",
            entities=None,
            reply_to=SimpleNamespace(reply_to_msg_id=55),
            get_sender=AsyncMock(return_value=SimpleNamespace(username="buyer1")),
        )
        with patch.dict("os.environ", {"TELEGRAM_PREFILTER": "false"}):
            await process_scrape_message(
                client, "entity", message, chat, "", "supergroup", buf
            )
        client.get_messages.assert_awaited_once_with("entity", ids=55)
        row = json.loads(buf.getvalue().strip())
        self.assertEqual(row["reply_to_message_id"], 55)
        self.assertIn("keitaro", row["reply_context"])

    async def test_reply_parent_skipped_on_flood_wait(self) -> None:
        from telethon.errors import FloodWaitError

        client = SimpleNamespace(
            get_messages=AsyncMock(side_effect=FloodWaitError(120))
        )
        ctx = await fetch_reply_context(client, "entity", 77)
        self.assertEqual(ctx, "")
        client.get_messages.assert_awaited_once()


if __name__ == "__main__":
    unittest.main()
