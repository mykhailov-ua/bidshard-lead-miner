import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import load_config
from sources.telegram.cursor import CursorStore

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


if __name__ == "__main__":
    unittest.main()
