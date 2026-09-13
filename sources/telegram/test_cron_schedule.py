import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path


def _load_schedule_module():
    root = Path(__file__).resolve().parents[2]
    path = root / "scripts" / "ops" / "telegram-cron-schedule.py"
    spec = importlib.util.spec_from_file_location("telegram_cron_schedule", path)
    mod = importlib.util.module_from_spec(spec)
    sys.modules["telegram_cron_schedule"] = mod
    spec.loader.exec_module(mod)
    return mod


class CronScheduleTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.mod = _load_schedule_module()

    def test_shared_session_two_shards_staggered(self) -> None:
        pool = {
            "shard_count": 2,
            "cron_shared_session": True,
            "sessions": [
                {"id": 0, "role": "hot", "shard": 0, "runtime": "data/runtime/telethon.session.0"},
                {"id": 1, "role": "hot", "shard": 1, "runtime": "data/runtime/telethon.session.1"},
            ],
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sessions.pool.json"
            path.write_text(json.dumps(pool), encoding="utf-8")
            lines = self.mod.build_cron_lines(str(path), "/opt/app")
        cron = [ln for ln in lines if ln.startswith(("7 ", "37 "))]
        self.assertEqual(len(cron), 2)
        self.assertIn("TELEGRAM_SHARD=0", cron[0])
        self.assertIn("TELEGRAM_SHARD=1", cron[1])
        self.assertIn("telethon.session ", cron[0])

    def test_per_session_offsets(self) -> None:
        pool = {
            "shard_count": 2,
            "cron_shared_session": False,
            "sessions": [
                {"id": 0, "role": "hot", "shard": 0, "runtime": "data/runtime/telethon.session.0"},
                {"id": 1, "role": "hot", "shard": 1, "runtime": "data/runtime/telethon.session.1"},
            ],
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "sessions.pool.json"
            path.write_text(json.dumps(pool), encoding="utf-8")
            lines = self.mod.build_cron_lines(str(path), "/opt/app")
        self.assertTrue(any(ln.startswith("7 ") for ln in lines))
        self.assertTrue(any(ln.startswith("22 ") for ln in lines))


if __name__ == "__main__":
    unittest.main()
