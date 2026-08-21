import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock

from telethon.tl.types import PeerChannel

from sources.telegram.crossmention import extract_forward_channel_usernames


class CrossMentionForwardsTest(unittest.IsolatedAsyncioTestCase):
    async def test_extract_forward_channel_usernames(self) -> None:
        client = AsyncMock()
        client.get_entity = AsyncMock(
            side_effect=lambda peer: SimpleNamespace(username="forwarded_ops")
        )

        msg = SimpleNamespace(
            fwd_from=SimpleNamespace(
                from_id=PeerChannel(channel_id=123),
            )
        )
        got = await extract_forward_channel_usernames(client, [msg])
        self.assertEqual(got, ["forwarded_ops"])

    async def test_skips_messages_without_forward(self) -> None:
        client = AsyncMock()
        got = await extract_forward_channel_usernames(client, [SimpleNamespace()])
        self.assertEqual(got, [])


if __name__ == "__main__":
    unittest.main()
