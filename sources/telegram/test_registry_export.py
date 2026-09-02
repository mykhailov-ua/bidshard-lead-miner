import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import ChatConfig
from sources.telegram.cursor import CursorStore
from sources.telegram.registry_export import export_channels_json


class RegistryExportTest(unittest.TestCase):
    def test_export_channels_json(self) -> None:
        chat = ChatConfig(name="Alpha", username="alpha", geo="eu")
        with tempfile.TemporaryDirectory() as tmp:
            db = Path(tmp) / "crawler.db"
            out = Path(tmp) / "channels.json"
            store = CursorStore(db)
            store.upsert_channel(chat, "discover")
            export_channels_json(store, out)
            raw = out.read_text(encoding="utf-8")
            self.assertIn("alpha", raw)
            store.close()


if __name__ == "__main__":
    unittest.main()
