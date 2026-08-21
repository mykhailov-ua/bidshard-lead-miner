import tempfile
import unittest
from pathlib import Path

from sources.telegram.config import load_config


class ConfigLoadTest(unittest.TestCase):
    def test_null_chats_key_loads_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sources.telegram.yaml"
            path.write_text(
                "discover:\n  enabled: false\n  queries: []\n  limit_per_query: 1\n"
                "  serp_channels_path: channels.json\nchats:\n  # commented only\n",
                encoding="utf-8",
            )
            cfg = load_config(path)
            self.assertEqual(cfg.chats, [])


if __name__ == "__main__":
    unittest.main()
