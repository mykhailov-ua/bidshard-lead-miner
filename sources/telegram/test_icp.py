import json
import tempfile
import unittest
from pathlib import Path

from sources.telegram.icp import load_icp_queries


class ICPTest(unittest.TestCase):
    def test_load_icp_queries(self) -> None:
        data = {
            "telegram_search": ["voluum alternative", "keitaro"],
            "serp_dorks": ["site:t.me affiliate"],
        }
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
            json.dump(data, f)
            path = f.name
        tg, serp = load_icp_queries(path)
        self.assertEqual(tg, ["voluum alternative", "keitaro"])
        self.assertEqual(serp, ["site:t.me affiliate"])

    def test_repo_icp_file(self) -> None:
        root = Path(__file__).resolve().parents[2]
        tg, serp = load_icp_queries(root / "config" / "discover.icp.json")
        self.assertGreater(len(tg), 5)
        self.assertGreater(len(serp), 5)


if __name__ == "__main__":
    unittest.main()
