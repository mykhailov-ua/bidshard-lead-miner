import unittest
from unittest.mock import patch

from sources.telegram.config import ChatConfig
from sources.telegram.shard import chat_shard, filter_chats_for_shard, stable_shard


class ShardTest(unittest.TestCase):
    def test_stable_shard_deterministic(self) -> None:
        a = stable_shard("u:aff_chat", 2)
        b = stable_shard("u:aff_chat", 2)
        self.assertEqual(a, b)
        self.assertIn(a, (0, 1))

    def test_explicit_shard_override(self) -> None:
        chat = ChatConfig(name="x", username="aff", geo="eu", shard=1)
        self.assertEqual(chat_shard(chat, 2), 1)

    def test_filter_disjoint_shards(self) -> None:
        chats = [
            ChatConfig(name=f"c{i}", username=f"chat_{i}", geo="eu")
            for i in range(6)
        ]
        with patch.dict("os.environ", {"TELEGRAM_SHARD": "0", "TELEGRAM_SHARD_COUNT": "2"}):
            shard0 = {c.username for c in filter_chats_for_shard(chats)}
        with patch.dict("os.environ", {"TELEGRAM_SHARD": "1", "TELEGRAM_SHARD_COUNT": "2"}):
            shard1 = {c.username for c in filter_chats_for_shard(chats)}
        self.assertFalse(shard0 & shard1)
        self.assertEqual(shard0 | shard1, {c.username for c in chats})


if __name__ == "__main__":
    unittest.main()
