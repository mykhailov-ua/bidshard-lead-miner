import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock

from sources.telegram.discussion import (
    discussion_cursor_key,
    linked_discussion_id,
)


def _telethon_missing() -> bool:
    try:
        import telethon  # noqa: F401
    except ImportError:
        return True
    return False


class DiscussionHelpersTest(unittest.TestCase):
    def test_discussion_cursor_key(self) -> None:
        self.assertEqual(discussion_cursor_key("u:aff_chat"), "u:aff_chat#discussion")


@unittest.skipIf(_telethon_missing(), "telethon not installed")
class LinkedDiscussionTest(unittest.IsolatedAsyncioTestCase):
    async def test_linked_discussion_id_from_full_channel(self) -> None:
        from telethon.tl.types import Channel

        channel = Channel(id=100, access_hash=1, title="news", megagroup=False)
        full = SimpleNamespace(full_chat=SimpleNamespace(linked_chat_id=200))
        client = AsyncMock(return_value=full)

        linked = await linked_discussion_id(client, channel)
        self.assertEqual(linked, 200)

    async def test_megagroup_skips_discussion_lookup(self) -> None:
        from telethon.tl.types import Channel

        group = Channel(id=100, access_hash=1, title="chat", megagroup=True)
        client = AsyncMock()

        linked = await linked_discussion_id(client, group)
        self.assertIsNone(linked)
        client.assert_not_called()


if __name__ == "__main__":
    unittest.main()
