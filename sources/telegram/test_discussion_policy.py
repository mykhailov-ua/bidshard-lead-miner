import os
import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.discussion_policy import discussion_allowed_for_chat


class DiscussionPolicyTest(unittest.TestCase):
    def test_only_top_buyer_supergroups(self) -> None:
        os.environ["TELEGRAM_DISCUSSION_MAX_CHATS"] = "2"
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "c.db")
            for i, pain in enumerate((50, 10, 1)):
                store.upsert_channel(
                    ChatConfig(
                        name=f"chat{i}",
                        username=f"buyer{i}",
                        role="buyer_supergroup",
                    ),
                    "manual",
                )
                key = f"u:buyer{i}"
                store._conn.execute(
                    "UPDATE telegram_channels SET pain_hits_30d = ? WHERE channel_key = ?",
                    (pain, key),
                )
            store._conn.commit()
            top = ChatConfig(name="chat0", username="buyer0", role="buyer_supergroup")
            low = ChatConfig(name="chat2", username="buyer2", role="buyer_supergroup")
            self.assertTrue(discussion_allowed_for_chat(top, store))
            self.assertFalse(discussion_allowed_for_chat(low, store))
            store.close()


if __name__ == "__main__":
    unittest.main()
