import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock

from sources.telegram.channel_search import iter_search_hits


class ChannelSearchTest(unittest.IsolatedAsyncioTestCase):
    async def test_iter_search_hits_emits(self) -> None:
        message = SimpleNamespace(id=99, text="voluum alternative postback failing")
        calls: list[tuple[str, int]] = []

        async def iter_messages(entity, search="", limit=0):
            calls.append((search, limit))
            yield message

        client = SimpleNamespace(iter_messages=iter_messages)
        emitted: list[int] = []

        async def on_message(msg: object) -> bool:
            emitted.append(int(msg.id))
            return True

        searches, count = await iter_search_hits(
            client,
            SimpleNamespace(),
            ["voluum", "keitaro"],
            10,
            2,
            on_message,
        )
        self.assertEqual(searches, 2)
        self.assertEqual(count, 2)
        self.assertEqual(emitted, [99, 99])
        self.assertEqual(calls[0][0], "voluum")


if __name__ == "__main__":
    unittest.main()
